package ws

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-fly-muti/models"
	"go-fly-muti/tools"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type User struct {
	Conn       *websocket.Conn
	Name       string
	Id         string
	Avator     string
	To_id      string
	Ent_id     string
	Role_id    string
	Mux        sync.Mutex
	UpdateTime time.Time
}
type Message struct {
	user        *User
	context     *gin.Context
	content     []byte
	messageType int
}
type TypeMessage struct {
	Type interface{} `json:"type"`
	Data interface{} `json:"data"`
}
type ClientMessage struct {
	MsgId     uint   `json:"msg_id"`
	Name      string `json:"name"`
	Avator    string `json:"avator"`
	Id        string `json:"id"`
	VisitorId string `json:"visitor_id"`
	Group     string `json:"group"`
	Time      string `json:"time"`
	ToId      string `json:"to_id"`
	Content   string `json:"content"`
	City      string `json:"city"`
	ClientIp  string `json:"client_ip"`
	Refer     string `json:"refer"`
	IsKefu    string `json:"is_kefu"`
}
type SimpleMessage struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Content string `json:"content"`
}

var Room = NewRoom()
var ClientList = make(map[string]*User)
var KefuList = make(map[string][]*User)
var message = make(chan *Message, 10)
var upgrader = websocket.Upgrader{}
var Mux sync.RWMutex

const websocketWriteTimeout = 5 * time.Second

func writeUserMessage(user *User, messageType int, content []byte) error {
	if user == nil || user.Conn == nil {
		return fmt.Errorf("websocket connection is unavailable")
	}
	user.Mux.Lock()
	defer user.Mux.Unlock()
	_ = user.Conn.SetWriteDeadline(time.Now().Add(websocketWriteTimeout))
	return user.Conn.WriteMessage(messageType, content)
}

func init() {
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// 解决跨域问题
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}
func SendServerJiang(title string, content string, domain string) string {
	noticeServerJiang, err := strconv.ParseBool(models.FindConfig("NoticeServerJiang"))
	serverJiangAPI := models.FindConfig("ServerJiangAPI")
	if err != nil || !noticeServerJiang || serverJiangAPI == "" {
		log.Println("do not notice serverjiang:", serverJiangAPI, noticeServerJiang)
		return ""
	}
	sendStr := fmt.Sprintf("%s%s", title, content)
	desp := title + ":" + content + "[登录](http://" + domain + "/main)"
	url := serverJiangAPI + "?text=" + sendStr + "&desp=" + desp
	//log.Println(url)
	res := tools.Get(url)
	return res
}

// UpdateVisitorStatusCron 定时给更新数据库状态
func UpdateVisitorStatusCron() {
	for {
		SendPingToKefuClient()
		time.Sleep(25 * time.Second)
	}
}

// WsServerBackend 后端广播发送消息
func WsServerBackend() {
	for {
		message := <-message
		var typeMsg TypeMessage
		if err := json.Unmarshal(message.content, &typeMsg); err != nil {
			log.Println("invalid websocket message:", err)
			continue
		}
		msgType, ok := typeMsg.Type.(string)
		if !ok {
			continue
		}

		switch msgType {
		//心跳
		case "ping":
			msg := TypeMessage{
				Type: "pong",
			}
			str, _ := json.Marshal(msg)
			if err := writeUserMessage(message.user, websocket.TextMessage, str); err != nil {
				log.Println("send websocket pong:", err)
			}
		case "inputing":
			data, ok := typeMsg.Data.(map[string]interface{})
			if !ok {
				continue
			}
			from, fromOK := data["from"].(string)
			to, toOK := data["to"].(string)
			if !fromOK || !toOK {
				continue
			}
			//限流
			if tools.LimitFreqSingle("inputing:"+from, 1, 2) {
				OneKefuMessage(to, message.content)
			}
		}

	}
}
func UpdateVisitorUser(visitorId string, toId string) {
	if guest, ok := ClientList[visitorId]; ok {
		guest.To_id = toId
	}

}
