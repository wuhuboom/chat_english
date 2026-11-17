/**
 * @Author $
 * @Description //TODO $
 * @Date $ $
 * @Param $
 * @return $
 **/
package setting

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	_ "time/tzdata"

	"github.com/spf13/viper"
)

const DefaultTimezone = "Asia/Shanghai"

var currentLocation atomic.Pointer[time.Location]

func init() {
	location, _ := time.LoadLocation(DefaultTimezone)
	currentLocation.Store(location)
}

func Init() error {
	configPath, err := filepath.Abs("config.yaml")
	if err != nil {
		return fmt.Errorf("无法定位配置文件: %w", err)
	}
	viper.SetConfigFile(configPath)
	//读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}
	if err := loadStaffProxyPolicy(viper.GetViper()); err != nil {
		return fmt.Errorf("代理信任配置无效: %w", err)
	}
	if _, err := watchStaffProxyConfig(configPath); err != nil {
		return fmt.Errorf("无法监听代理信任配置: %w", err)
	}
	viper.WatchConfig()

	fmt.Println("读取配置成功")
	return nil
}

func ConfigureTimezone(timezone string) error {
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		timezone = DefaultTimezone
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("无效的系统时区 %q，请使用 IANA 时区名称: %w", timezone, err)
	}
	currentLocation.Store(location)
	return nil
}

func Location() *time.Location {
	location := currentLocation.Load()
	if location == nil {
		location, _ = time.LoadLocation(DefaultTimezone)
		currentLocation.Store(location)
	}
	return location
}

func CurrentTimezone() string {
	return Location().String()
}

func Now() time.Time {
	return time.Now().In(Location())
}

func Format(value time.Time) string {
	return value.In(Location()).Format("2006-01-02 15:04:05")
}
