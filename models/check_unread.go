package models

import (
	"time"
)

type Check_unread struct {
	ID           uint      `gorm:"primary_key" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	KeFuUsername string    `json:"ke_fu_username"`
	Visitor      string    `json:"visitor"`
	TimeOut      int64     `json:"time_out"`
}

func (cu *Check_unread) Created() {
	cu.CreatedAt = time.Now()
	DB.Create(cu)
}
