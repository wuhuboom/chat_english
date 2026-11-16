package models

import (
	"errors"
	"strings"
	"sync"

	"github.com/jinzhu/gorm"
)

const (
	GoogleAuthenticatorEnabledKey = "GoogleAuthenticatorEnabled"
	GoogleAuthenticatorSecretKey  = "GoogleAuthenticatorSecret"
	SystemRegisterKey             = "SystemRegister"
	CustomerServiceVersionKey     = "version"
	LegacySystemKefuKey           = "SystemKefu"
)

var ErrRetiredConfig = errors.New("该配置已停用")

func IsRetiredConfigKey(key string) bool {
	return strings.EqualFold(strings.TrimSpace(key), LegacySystemKefuKey)
}

var CustomConfigs []Config
var customConfigsMux sync.RWMutex

type Config struct {
	ID        uint   `gorm:"primary_key" json:"id"`
	ConfName  string `json:"conf_name"`
	ConfKey   string `json:"conf_key"`
	ConfValue string `json:"conf_value"`
}

func UpdateConfig(key string, value string) {
	_ = SaveConfig("", key, value)
}

func SaveConfig(name, key, value string) error {
	if IsRetiredConfigKey(key) {
		return ErrRetiredConfig
	}
	var config Config
	err := DB.Where("conf_key = ?", key).First(&config).Error
	// MySQL can match alternate spellings under its column collation.
	if err == nil && IsRetiredConfigKey(config.ConfKey) {
		return ErrRetiredConfig
	}
	if gorm.IsRecordNotFoundError(err) {
		if name == "" {
			name = key
		}
		err = DB.Create(&Config{
			ConfName:  name,
			ConfKey:   key,
			ConfValue: value,
		}).Error
	} else if err == nil {
		updates := map[string]string{"conf_value": value}
		if name != "" {
			updates["conf_name"] = name
		}
		err = DB.Model(&Config{}).Where("conf_key = ?", key).Updates(updates).Error
	}
	if err == nil {
		InitConfig()
	}
	return err
}
func FindConfigs() []Config {
	var config []Config
	DB.Find(&config)

	return config
}
func InitConfig() {
	configs := FindConfigs()
	customConfigsMux.Lock()
	CustomConfigs = configs
	customConfigsMux.Unlock()
}
func FindConfig(key string) string {
	customConfigsMux.RLock()
	defer customConfigsMux.RUnlock()
	for _, config := range CustomConfigs {
		if key == config.ConfKey {
			return config.ConfValue
		}
	}
	return ""
}

func CustomerServiceVersion() string {
	return strings.TrimSpace(FindConfig(CustomerServiceVersionKey))
}

func GoogleAuthenticatorEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(FindConfig(GoogleAuthenticatorEnabledKey)), "true")
}

// RegistrationEnabled is fail-closed: public registration is allowed only
// when an administrator explicitly stores the value "1".
func RegistrationEnabled() bool {
	return strings.TrimSpace(FindConfig(SystemRegisterKey)) == "1"
}
