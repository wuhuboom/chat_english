package models

import (
	"fmt"
	"time"
)

type Check_unread struct {
	ID           uint      `gorm:"primary_key" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	KeFuUsername string    `json:"ke_fu_username"`
	Visitor      string    `json:"visitor"`
	TimeOut      int64     `json:"time_out"`
}

func CheckIsExistModelAdmin() {
	if DB.AutoMigrate().HasTable(&Check_unread{}) {
		fmt.Println("数据库已经存在了!")
		DB.AutoMigrate(&Check_unread{})

	} else {
		fmt.Println("数据不存在,所以我要先创建数据库")
		DB.AutoMigrate().CreateTable(&Check_unread{})
	}

}

func (cu *Check_unread) Created() {
	cu.CreatedAt = time.Now()
	DB.Create(cu)
}
