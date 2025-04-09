package process

import (
	"fmt"
	"github.com/spf13/viper"
	"go-fly-muti/models"
	"go-fly-muti/tools"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var AlliD []uint

func CheckUnreadMes() {

	type MessageKefu struct {
		ID        uint      `gorm:"primary_key" json:"id"`
		VisitorId string    `json:"visitor_id"`
		Name      string    `json:"name"`
		KefuId    string    `json:"kefu_id"`
		CreatedAt time.Time `json:"created_at"`
	}
	for true {
		msg := make([]MessageKefu, 0)
		models.DB.Raw("SELECT m.*, v.name \nFROM `message` m\nJOIN `visitor` v ON m.visitor_id = v.visitor_id\nWHERE m.status = 'unread'  \nAND m.mes_type = 'visitor'  \nAND m.id = (\n    SELECT MAX(m2.id) \n    FROM `message` m2 \n    WHERE m2.visitor_id = m.visitor_id\n);\n").Scan(&msg)
		for _, v := range msg {
			var msg string
			cu := models.Check_unread{}
			cu.Visitor = v.Name
			cu.KeFuUsername = v.KefuId
			//大于5分钟
			if time.Now().Unix()-v.CreatedAt.Unix() > 300 {
				b, _ := tools.InArray(v.ID, AlliD)
				if b {
					//这个五分钟已经报警过了
					fmt.Println(fmt.Sprintf("这个五分钟已经报警过了,%d", v.ID))
					continue
				}
				cu.TimeOut = 5
				msg = "\n❌已经超过5分钟已经没有回复玩家,\n❌玩家用户名: " + v.Name
				AlliD = append(AlliD, v.ID)
			} else if time.Now().Unix()-v.CreatedAt.Unix() > 60 {
				//大于1分钟 小于 5分钟
				cu.TimeOut = 1
				msg = "\n已经超过1分钟已经没有回复玩家,\n玩家用户名: " + v.Name
			}
			//生成日志
			cu.Created()
			//获取配置文件
			username := viper.GetString("app.TelegramUsername")
			usernameArray := strings.Split(username, "|")
			for _, s := range usernameArray {
				uARR := strings.Split(s, "@")
				if uARR[0] == v.KefuId {
					//飞机报警
					fmt.Println("飞机报警")
					msg = "@" + uARR[1] + msg
					fmt.Println(msg)
					msg = url.QueryEscape(msg)

					telegramSend(msg)
				}
			}
		}

		fmt.Println("检查->>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
		time.Sleep(30 * time.Second)
	}

}

func telegramSend(msg string) {

	url := "https://api.telegram.org/bot" + viper.GetString("app.TelegramToken") + "/sendmessage?chat_id=" + viper.GetString("app.TelegramChatId") + "&text=" + msg // 替换为实际 API 地址
	// 发送 GET 请求
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("请求失败:", err)
		zap.L().Debug(string(err.Error()))
		return
	}
	defer resp.Body.Close() // 确保关闭响应体

	// 读取响应数据
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		zap.L().Debug(string(err.Error()))
		return
	}

	// 输出响应内容
	//fmt.Println("响应状态码:", resp.Status)
	//fmt.Println("响应数据:", string(body))
	zap.L().Debug(string(body))
}
