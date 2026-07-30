package models

import "time"

const (
	ConversationStatusOpen     = "open"
	ConversationStatusPending  = "pending"
	ConversationStatusResolved = "resolved"

	ConversationPriorityNormal = "normal"
	ConversationPriorityHigh   = "high"
	ConversationPriorityUrgent = "urgent"
)

// Conversation keeps service workflow state separate from Visitor.Status,
// which represents the visitor's transport/online state.
type Conversation struct {
	Model
	EntId          string     `gorm:"type:varchar(32);unique_index:idx_conversation_visitor" json:"ent_id"`
	VisitorId      string     `gorm:"type:varchar(255);unique_index:idx_conversation_visitor" json:"visitor_id"`
	KefuId         string     `gorm:"type:varchar(100);index" json:"kefu_id"`
	Status         string     `gorm:"type:varchar(20);index;default:'pending'" json:"status"`
	Priority       string     `gorm:"type:varchar(20);index;default:'normal'" json:"priority"`
	WaitingSince   *time.Time `json:"waiting_since"`
	LastCustomerAt *time.Time `json:"last_customer_at"`
	LastAgentAt    *time.Time `json:"last_agent_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`
}

func FindConversation(entId, visitorId string) Conversation {
	var conversation Conversation
	DB.Where("ent_id = ? AND visitor_id = ?", entId, visitorId).First(&conversation)
	return conversation
}

func FindConversationMap(entId string, visitorIds []string) map[string]Conversation {
	result := make(map[string]Conversation, len(visitorIds))
	if len(visitorIds) == 0 {
		return result
	}

	var conversations []Conversation
	DB.Where("ent_id = ? AND visitor_id IN (?)", entId, visitorIds).Find(&conversations)
	for _, conversation := range conversations {
		result[conversation.VisitorId] = conversation
	}
	return result
}

func ensureConversation(entId, visitorId, kefuId string) Conversation {
	conversation := FindConversation(entId, visitorId)
	if conversation.ID != 0 {
		return conversation
	}

	conversation = Conversation{
		EntId:     entId,
		VisitorId: visitorId,
		KefuId:    kefuId,
		Status:    ConversationStatusPending,
		Priority:  ConversationPriorityNormal,
	}
	if err := DB.Create(&conversation).Error; err != nil {
		// Another request may have created the same conversation concurrently.
		conversation = FindConversation(entId, visitorId)
	}
	return conversation
}

func MarkConversationWaiting(entId, visitorId, kefuId string, at time.Time) {
	conversation := ensureConversation(entId, visitorId, kefuId)
	DB.Model(&conversation).Updates(map[string]interface{}{
		"kefu_id":          kefuId,
		"status":           ConversationStatusOpen,
		"waiting_since":    at,
		"last_customer_at": at,
		"resolved_at":      nil,
	})
}

func MarkConversationPending(entId, visitorId, kefuId string, at time.Time) {
	conversation := ensureConversation(entId, visitorId, kefuId)
	DB.Model(&conversation).Updates(map[string]interface{}{
		"kefu_id":       kefuId,
		"status":        ConversationStatusPending,
		"waiting_since": nil,
		"last_agent_at": at,
		"resolved_at":   nil,
	})
}

func ResolveConversation(entId, visitorId, kefuId string, at time.Time) {
	conversation := ensureConversation(entId, visitorId, kefuId)
	DB.Model(&conversation).Updates(map[string]interface{}{
		"kefu_id":       kefuId,
		"status":        ConversationStatusResolved,
		"waiting_since": nil,
		"resolved_at":   at,
	})
}

func UpdateConversationWorkflow(entId, visitorId, kefuId, status, priority string, at time.Time) Conversation {
	conversation := ensureConversation(entId, visitorId, kefuId)
	updates := map[string]interface{}{"kefu_id": kefuId}
	if status != "" {
		updates["status"] = status
		switch status {
		case ConversationStatusOpen:
			if conversation.WaitingSince == nil {
				updates["waiting_since"] = at
			}
			updates["resolved_at"] = nil
		case ConversationStatusPending:
			updates["waiting_since"] = nil
			updates["resolved_at"] = nil
		case ConversationStatusResolved:
			updates["waiting_since"] = nil
			updates["resolved_at"] = at
		}
	}
	if priority != "" {
		updates["priority"] = priority
	}
	DB.Model(&conversation).Updates(updates)
	return FindConversation(entId, visitorId)
}

func UpdateConversationKefu(entId, visitorId, kefuId string) {
	conversation := FindConversation(entId, visitorId)
	if conversation.ID != 0 {
		DB.Model(&conversation).Update("kefu_id", kefuId)
	}
}
