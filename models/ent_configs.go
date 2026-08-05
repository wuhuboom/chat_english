package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

type EntConfig struct {
	ID        uint   `gorm:"primary_key" json:"id"`
	ConfName  string `json:"conf_name"`
	ConfKey   string `json:"conf_key"`
	ConfValue string `json:"conf_value"`
	EntId     string `json:"ent_id"`
}

func CreateEntConfig(kefuId interface{}, name, key, value string) {
	c := &EntConfig{
		ConfName:  name,
		ConfKey:   key,
		ConfValue: value,
		EntId:     fmt.Sprintf("%v", kefuId),
	}
	DB.Create(c)
}
func FindEntConfigs(kefuId interface{}) []EntConfig {
	var configs []EntConfig
	DB.Where("ent_id = ?", kefuId).Find(&configs)
	return configs
}
func FindEntConfigByEntid(entId interface{}) []EntConfig {
	var configs []EntConfig
	DB.Where("ent_id = ?", entId).Find(&configs)
	return configs
}
func FindEntConfig(kefuId interface{}, key string) EntConfig {
	var config EntConfig
	DB.Where("ent_id = ? and conf_key=?", kefuId, key).Find(&config)
	return config
}
func UpdateEntConfig(kefuId interface{}, name, key, value string) {
	c := map[string]string{
		"conf_name":  name,
		"conf_key":   key,
		"conf_value": value,
		"ent_id":     fmt.Sprintf("%v", kefuId),
	}
	DB.Model(&EntConfig{}).Where("ent_id = ? and conf_key = ?", fmt.Sprintf("%v", kefuId), key).Update(c)
}

func saveEntConfigWithDB(db *gorm.DB, entId, name, key, value string) error {
	var config EntConfig
	err := db.Where("ent_id = ? AND conf_key = ?", entId, key).First(&config).Error
	if gorm.IsRecordNotFoundError(err) {
		return db.Create(&EntConfig{
			ConfName:  name,
			ConfKey:   key,
			ConfValue: value,
			EntId:     entId,
		}).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&config).Updates(map[string]string{
		"conf_name":  name,
		"conf_value": value,
	}).Error
}

// SaveEntConfig creates or updates one enterprise-scoped setting.
func SaveEntConfig(entId, name, key, value string) error {
	return saveEntConfigWithDB(DB, entId, name, key, value)
}

// SaveConversationSLASettings updates both thresholds atomically so the
// workbench never observes a partially updated SLA policy.
func SaveConversationSLASettings(entId, warningMinutes, overdueMinutes string) error {
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := saveEntConfigWithDB(tx, entId, "会话SLA预警时间（分钟）", "ConversationSLAWarningMinutes", warningMinutes); err != nil {
		tx.Rollback()
		return err
	}
	if err := saveEntConfigWithDB(tx, entId, "会话SLA超时时间（分钟）", "ConversationSLAOverdueMinutes", overdueMinutes); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// SaveRoutingSettings keeps disconnect protection settings consistent.
func SaveRoutingSettings(entId, autoReassign, offlineGraceSeconds string) error {
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := saveEntConfigWithDB(tx, entId, "客服离线自动转接", "AutoReassignOnOffline", autoReassign); err != nil {
		tx.Rollback()
		return err
	}
	if err := saveEntConfigWithDB(tx, entId, "客服离线重连宽限（秒）", "AgentOfflineGraceSeconds", offlineGraceSeconds); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
