package controller

import (
	"encoding/json"

	"go-fly-muti/models"
	"go-fly-muti/setting"
	"go-fly-muti/ws"
)

var resolveExistingConversation = models.ResolveExistingConversation
var createBlacklistConversationEvent = models.CreateConversationEvent
var disconnectBlacklistedVisitor = ws.DisconnectVisitor
var notifyBlacklistedVisitorOffline = ws.VisitorOffline
var sendConversationResolved = notifyConversationResolved

var cleanupBlacklistedVisitorsFn = cleanupBlacklistedVisitors

func cleanupBlacklistedVisitors(entID, actor, detail string, visitors []models.Visitor) {
	for _, visitor := range visitors {
		if visitor.VisitorId == "" || visitor.EntId != entID {
			continue
		}

		now := setting.Now()
		if resolveExistingConversation(entID, visitor.VisitorId, visitor.ToId, now) {
			createBlacklistConversationEvent(
				entID,
				visitor.VisitorId,
				models.ConversationEventResolved,
				actor,
				visitor.ToId,
				visitor.ToId,
				detail,
				now,
			)
		}
		// Always clear the agent-side timer. This also repairs a stale browser
		// whose local state still says open after the database was resolved.
		sendConversationResolved(visitor.ToId, visitor.VisitorId)

		closePayload, _ := json.Marshal(ws.TypeMessage{
			Type: "force_close",
			Data: visitor.VisitorId,
		})
		state, removed, _ := disconnectBlacklistedVisitor(visitor.VisitorId, closePayload)
		if removed {
			notifyBlacklistedVisitorOffline(state.ToID, state.Id, state.Name)
		}
	}
}

func notifyConversationResolved(kefuID, visitorID string) {
	if kefuID == "" {
		return
	}
	payload, _ := json.Marshal(ws.TypeMessage{
		Type: "conversationResolved",
		Data: map[string]string{
			"visitor_id": visitorID,
			"status":     models.ConversationStatusResolved,
		},
	})
	ws.OneKefuMessage(kefuID, payload)
}
