package ws

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-fly-muti/models"
	"go-fly-muti/setting"
)

const defaultAgentOfflineGrace = 15 * time.Second

var scheduledKefuFailovers sync.Map

func routingProtection(entID string) (bool, time.Duration) {
	autoReassign := true
	autoConfig := models.FindEntConfig(entID, "AutoReassignOnOffline")
	if autoConfig.ConfValue != "" {
		if parsed, err := strconv.ParseBool(strings.TrimSpace(autoConfig.ConfValue)); err == nil {
			autoReassign = parsed
		}
	}

	grace := defaultAgentOfflineGrace
	graceConfig := models.FindEntConfig(entID, "AgentOfflineGraceSeconds")
	if seconds, err := strconv.Atoi(strings.TrimSpace(graceConfig.ConfValue)); err == nil && seconds >= 3 && seconds <= 300 {
		grace = time.Duration(seconds) * time.Second
	}
	return autoReassign, grace
}

func scheduleKefuFailover(kefuID, entID string) {
	if kefuID == "" || entID == "" {
		return
	}
	enabled, grace := routingProtection(entID)
	if !enabled {
		return
	}
	key := entID + ":" + kefuID
	if _, loaded := scheduledKefuFailovers.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	go func() {
		defer scheduledKefuFailovers.Delete(key)
		timer := time.NewTimer(grace)
		defer timer.Stop()
		<-timer.C
		if IsKefuOnline(kefuID) {
			return
		}
		reassignVisitorsFromOfflineKefu(kefuID, entID)
	}()
}

func availableFailoverKefus(users []models.User, offlineKefu string, online func(string) bool, loads map[string]int) []models.User {
	available := make([]models.User, 0, len(users))
	for _, user := range users {
		if user.Name == offlineKefu || user.OnlineStatus != 1 || !online(user.Name) {
			continue
		}
		available = append(available, user)
	}
	sort.SliceStable(available, func(left, right int) bool {
		leftLoad := loads[available[left].Name]
		rightLoad := loads[available[right].Name]
		if leftLoad != rightLoad {
			return leftLoad < rightLoad
		}
		if available[left].RecNum != available[right].RecNum {
			return available[left].RecNum < available[right].RecNum
		}
		return available[left].Name < available[right].Name
	})
	return available
}

func reassignVisitorsFromOfflineKefu(offlineKefu, entID string) {
	loads := ActiveVisitorCountsByKefu(entID)
	users := models.FindUsersWhere("pid = ? or id=?", entID, entID)
	visitors := VisitorConnectionsSnapshot()

	for _, connection := range visitors {
		if connection == nil {
			continue
		}
		state := connection.State()
		if state.EntID != entID || state.ToID != offlineKefu {
			continue
		}
		available := availableFailoverKefus(users, offlineKefu, IsKefuOnline, loads)
		if len(available) == 0 {
			return
		}
		nextKefu := available[0]
		visitor := models.FindVisitorByVistorId(state.Id)
		if visitor.ID == 0 {
			continue
		}

		models.UpdateVisitorKefu(state.Id, nextKefu.Name)
		models.UpdateConversationKefu(entID, state.Id, nextKefu.Name)
		models.CreateConversationEvent(
			entID, state.Id, models.ConversationEventAutoFailover,
			"system", offlineKefu, nextKefu.Name,
			"原客服持续离线，系统自动转接", setting.Now(),
		)
		connection.SetTarget(nextKefu.Name)
		models.UpdateUserRecNum(offlineKefu, -1)
		models.UpdateUserRecNum(nextKefu.Name, 1)
		visitor.ToId = nextKefu.Name
		notifyVisitorOnline(nextKefu.Name, visitor)
		VisitorTransfer(state.Id, nextKefu.Name)
		loads[offlineKefu]--
		loads[nextKefu.Name]++
	}
}
