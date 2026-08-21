package controller

import (
	"testing"
	"time"

	"go-fly-muti/models"
	"go-fly-muti/ws"
)

func TestCleanupBlacklistedVisitorsResolvesNotifiesAndDisconnectsWithinTenant(t *testing.T) {
	originalResolve := resolveExistingConversation
	originalEvent := createBlacklistConversationEvent
	originalDisconnect := disconnectBlacklistedVisitor
	originalOffline := notifyBlacklistedVisitorOffline
	originalResolvedNotice := sendConversationResolved
	t.Cleanup(func() {
		resolveExistingConversation = originalResolve
		createBlacklistConversationEvent = originalEvent
		disconnectBlacklistedVisitor = originalDisconnect
		notifyBlacklistedVisitorOffline = originalOffline
		sendConversationResolved = originalResolvedNotice
	})

	resolved := false
	eventCreated := false
	disconnected := false
	offlineNotified := false
	resolutionNotified := false
	resolveExistingConversation = func(entID, visitorID, kefuID string, _ time.Time) bool {
		resolved = entID == "42" && visitorID == "visitor-a" && kefuID == "agent-a"
		return true
	}
	createBlacklistConversationEvent = func(entID, visitorID, eventType, actor, from, to, detail string, _ time.Time) models.ConversationEvent {
		eventCreated = entID == "42" && visitorID == "visitor-a" &&
			eventType == models.ConversationEventResolved && actor == "agent-a" && detail != ""
		return models.ConversationEvent{}
	}
	sendConversationResolved = func(kefuID, visitorID string) {
		resolutionNotified = kefuID == "agent-a" && visitorID == "visitor-a"
	}
	disconnectBlacklistedVisitor = func(visitorID string, payload []byte) (ws.UserState, bool, error) {
		disconnected = visitorID == "visitor-a" && len(payload) > 0
		return ws.UserState{Id: visitorID, ToID: "agent-a", Name: "访客A"}, true, nil
	}
	notifyBlacklistedVisitorOffline = func(kefuID, visitorID, visitorName string) {
		offlineNotified = kefuID == "agent-a" && visitorID == "visitor-a" && visitorName == "访客A"
	}

	cleanupBlacklistedVisitors("42", "agent-a", "加入IP黑名单", []models.Visitor{
		{Model: models.Model{ID: 1}, VisitorId: "visitor-a", EntId: "42", ToId: "agent-a"},
		{Model: models.Model{ID: 2}, VisitorId: "visitor-other", EntId: "99", ToId: "other-agent"},
	})

	if !resolved || !eventCreated || !resolutionNotified || !disconnected || !offlineNotified {
		t.Fatalf("cleanup effects resolve=%v event=%v resolution_notice=%v disconnect=%v offline=%v",
			resolved, eventCreated, resolutionNotified, disconnected, offlineNotified)
	}
}
