package models

import "time"

const (
	ConversationEventAssigned        = "assigned"
	ConversationEventRerouted        = "rerouted"
	ConversationEventManualTransfer  = "manual_transfer"
	ConversationEventAutoFailover    = "auto_failover"
	ConversationEventStatusChanged   = "status_changed"
	ConversationEventPriorityChanged = "priority_changed"
	ConversationEventResolved        = "resolved"
)

// ConversationEvent is an immutable audit record for assignment and workflow
// changes. It is deliberately separate from customer-visible chat messages.
type ConversationEvent struct {
	ID          uint      `gorm:"primary_key" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	EntId       string    `gorm:"type:varchar(32);index:idx_conversation_event" json:"ent_id"`
	VisitorId   string    `gorm:"type:varchar(255);index:idx_conversation_event" json:"visitor_id"`
	EventType   string    `gorm:"type:varchar(40);index" json:"event_type"`
	ActorKefuId string    `gorm:"type:varchar(100)" json:"actor_kefu_id"`
	FromKefuId  string    `gorm:"type:varchar(100)" json:"from_kefu_id"`
	ToKefuId    string    `gorm:"type:varchar(100)" json:"to_kefu_id"`
	Detail      string    `gorm:"type:varchar(500)" json:"detail"`
}

func CreateConversationEvent(entId, visitorId, eventType, actorKefuId, fromKefuId, toKefuId, detail string, at time.Time) ConversationEvent {
	event := ConversationEvent{
		CreatedAt:   at,
		EntId:       entId,
		VisitorId:   visitorId,
		EventType:   eventType,
		ActorKefuId: actorKefuId,
		FromKefuId:  fromKefuId,
		ToKefuId:    toKefuId,
		Detail:      detail,
	}
	DB.Create(&event)
	return event
}

func FindConversationEvents(entId, visitorId string, limit int) []ConversationEvent {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	var events []ConversationEvent
	DB.Where("ent_id = ? AND visitor_id = ?", entId, visitorId).
		Order("id desc").
		Limit(limit).
		Find(&events)
	return events
}
