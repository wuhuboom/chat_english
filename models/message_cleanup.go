package models

import (
	"fmt"
	"strings"
	"time"
)

// CountMessagesInRange returns the number of chat messages belonging to one
// merchant inside the inclusive time range.
func CountMessagesInRange(entID string, start, end time.Time) (int64, error) {
	if DB == nil {
		return 0, fmt.Errorf("database connection is nil")
	}
	entID = strings.TrimSpace(entID)
	if entID == "" {
		return 0, fmt.Errorf("merchant id is empty")
	}

	var count int64
	err := DB.Model(&Message{}).
		Where("ent_id = ? AND created_at >= ? AND created_at <= ?", entID, start, end).
		Count(&count).Error
	return count, err
}

// DeleteMessagesInRange permanently removes messages for one merchant. The
// ent_id condition is deliberately part of the delete statement so one
// merchant can never clear another merchant's chat history.
func DeleteMessagesInRange(entID string, start, end time.Time) (int64, error) {
	if DB == nil {
		return 0, fmt.Errorf("database connection is nil")
	}
	entID = strings.TrimSpace(entID)
	if entID == "" {
		return 0, fmt.Errorf("merchant id is empty")
	}

	transaction := DB.Begin()
	if transaction.Error != nil {
		return 0, transaction.Error
	}
	result := transaction.
		Where("ent_id = ? AND created_at >= ? AND created_at <= ?", entID, start, end).
		Delete(&Message{})
	if result.Error != nil {
		transaction.Rollback()
		return 0, result.Error
	}
	if err := transaction.Commit().Error; err != nil {
		return 0, err
	}
	return result.RowsAffected, nil
}
