package models

import (
	"encoding/json"
	"strings"
)

const VisitorAllowlistKey = "VisitorIPAllowlist"

type VisitorAllowlist struct {
	Enabled bool     `json:"enabled"`
	IPs     []string `json:"ips"`
	Regions []string `json:"regions"`
}

func GetVisitorAllowlist(entID string) (VisitorAllowlist, error) {
	value := FindConfig(VisitorAllowlistKey)
	if entID != "" {
		var config EntConfig
		query := DB.Where("ent_id = ? AND conf_key = ?", entID, VisitorAllowlistKey).Find(&config)
		if query.Error != nil && !query.RecordNotFound() {
			return VisitorAllowlist{}, query.Error
		}
		value = config.ConfValue
	}
	policy := VisitorAllowlist{}
	if strings.TrimSpace(value) != "" {
		if err := json.Unmarshal([]byte(value), &policy); err != nil {
			return VisitorAllowlist{}, err
		}
	}
	return policy, nil
}
