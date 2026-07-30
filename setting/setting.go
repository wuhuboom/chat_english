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
	"strings"
	"sync/atomic"
	"time"
	_ "time/tzdata"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const DefaultTimezone = "Asia/Shanghai"

var currentLocation atomic.Pointer[time.Location]

func init() {
	location, _ := time.LoadLocation(DefaultTimezone)
	currentLocation.Store(location)
}

func Init() error {
	//指定配置文件 如果是 json 就写json
	viper.SetConfigFile("config.yaml")
	//指定文件路径
	viper.AddConfigPath(".")
	//读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}
	//监控配置文件变化
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Println("配置文件修改了!")
	})

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
