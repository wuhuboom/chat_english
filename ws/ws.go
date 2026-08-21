package ws

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-fly-muti/common"
	"go-fly-muti/models"
	"go-fly-muti/tools"
	"log"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
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
	SessionID  uint64
	Mux        sync.Mutex
	stateMux   sync.RWMutex
	UpdateTime time.Time
}

type UserState struct {
	Name       string
	Id         string
	Avator     string
	ToID       string
	EntID      string
	RoleID     string
	SessionID  uint64
	UpdateTime time.Time
}

func (u *User) State() UserState {
	if u == nil {
		return UserState{}
	}
	u.stateMux.RLock()
	defer u.stateMux.RUnlock()
	return UserState{
		Name:       u.Name,
		Id:         u.Id,
		Avator:     u.Avator,
		ToID:       u.To_id,
		EntID:      u.Ent_id,
		RoleID:     u.Role_id,
		SessionID:  u.SessionID,
		UpdateTime: u.UpdateTime,
	}
}

func (u *User) SetTarget(toID string) {
	if u == nil {
		return
	}
	u.stateMux.Lock()
	u.To_id = toID
	u.stateMux.Unlock()
}

func (u *User) Touch() {
	if u == nil {
		return
	}
	u.stateMux.Lock()
	u.UpdateTime = time.Now()
	u.stateMux.Unlock()
}

func (u *User) SetUpdateTime(updateTime time.Time) {
	if u == nil {
		return
	}
	u.stateMux.Lock()
	u.UpdateTime = updateTime
	u.stateMux.Unlock()
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
	MsgIds    []uint `json:"msg_ids,omitempty"`
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
var clientList = make(map[string]*User)
var KefuList = make(map[string][]*User)
var message = make(chan *Message, 10)
var upgrader = websocket.Upgrader{}
var Mux sync.RWMutex
var clientMux sync.RWMutex
var websocketSessionSequence atomic.Uint64

const websocketWriteTimeout = 5 * time.Second
const kefuConnectionStaleAfter = 120 * time.Second
const websocketPongWait = 75 * time.Second
const websocketReadLimit = 1024 * 1024

func isKefuConnectionStale(lastSeen, now time.Time) bool {
	return !lastSeen.IsZero() && now.Sub(lastSeen) > kefuConnectionStaleAfter
}

func writeUserMessage(user *User, messageType int, content []byte) error {
	if user == nil || user.Conn == nil {
		return fmt.Errorf("websocket connection is unavailable")
	}
	user.Mux.Lock()
	defer user.Mux.Unlock()
	_ = user.Conn.SetWriteDeadline(time.Now().Add(websocketWriteTimeout))
	return user.Conn.WriteMessage(messageType, content)
}

func writeUserControlMessage(user *User, messageType int, content []byte) error {
	if user == nil || user.Conn == nil {
		return fmt.Errorf("websocket connection is unavailable")
	}
	user.Mux.Lock()
	defer user.Mux.Unlock()
	return user.Conn.WriteControl(messageType, content, time.Now().Add(websocketWriteTimeout))
}

func configureWebSocketHeartbeat(user *User) error {
	if user == nil || user.Conn == nil {
		return fmt.Errorf("websocket connection is unavailable")
	}
	if user.SessionID == 0 {
		user.SessionID = websocketSessionSequence.Add(1)
	}
	user.Conn.SetReadLimit(websocketReadLimit)
	if err := user.Conn.SetReadDeadline(time.Now().Add(websocketPongWait)); err != nil {
		return err
	}
	user.Conn.SetPongHandler(func(string) error {
		user.Touch()
		return user.Conn.SetReadDeadline(time.Now().Add(websocketPongWait))
	})
	return nil
}

func refreshWebSocketReadDeadline(user *User) error {
	if user == nil || user.Conn == nil {
		return fmt.Errorf("websocket connection is unavailable")
	}
	user.Touch()
	return user.Conn.SetReadDeadline(time.Now().Add(websocketPongWait))
}

func websocketCloseInfo(err error) (int, string) {
	if err == nil {
		return 0, ""
	}
	var closeError *websocket.CloseError
	if errors.As(err, &closeError) {
		return closeError.Code, closeError.Text
	}
	return 0, err.Error()
}

func init() {
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return common.IsWebSocketOriginAllowed(r.Header.Get("Origin"))
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
		message.user.Touch()

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
	if guest, ok := VisitorConnection(visitorId); ok {
		guest.SetTarget(toId)
	}
}

func VisitorConnection(visitorID string) (*User, bool) {
	clientMux.RLock()
	defer clientMux.RUnlock()
	user, ok := clientList[visitorID]
	return user, ok
}

func VisitorConnectionsSnapshot() map[string]*User {
	clientMux.RLock()
	defer clientMux.RUnlock()
	snapshot := make(map[string]*User, len(clientList))
	for visitorID, user := range clientList {
		snapshot[visitorID] = user
	}
	return snapshot
}

func SendMessageToVisitor(user *User, content []byte) error {
	return writeUserMessage(user, websocket.TextMessage, content)
}

// SendMessageToCurrentVisitor delivers through the currently registered
// visitor session. A failed write must evict that exact session so callers do
// not keep treating a dead websocket as an online customer.
func SendMessageToCurrentVisitor(visitorID string, user *User, content []byte) (UserState, bool, error) {
	return sendMessageToCurrentVisitorWith(visitorID, user, content, SendMessageToVisitor)
}

func sendMessageToCurrentVisitorWith(visitorID string, initial *User, content []byte, send func(*User, []byte) error) (UserState, bool, error) {
	candidate := initial
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		current, ok := VisitorConnection(visitorID)
		if !ok || current == nil {
			if lastErr == nil {
				lastErr = fmt.Errorf("websocket connection is unavailable")
			}
			return UserState{}, false, lastErr
		}
		if candidate != current {
			candidate = current
		}

		err := send(candidate, content)
		latest, latestOK := VisitorConnection(visitorID)
		if err == nil {
			if latestOK && latest == candidate {
				return UserState{}, false, nil
			}
			// A reconnect replaced the socket while this write was in flight.
			// Deliver once more to the newest session so the new page cannot miss it.
			candidate = latest
			continue
		}

		lastErr = err
		removedState := candidate.State()
		removed := RemoveVisitorConnection(visitorID, candidate)
		if candidate.Conn != nil {
			_ = candidate.Conn.Close()
		}
		latest, latestOK = VisitorConnection(visitorID)
		if latestOK && latest != nil && latest != candidate {
			candidate = latest
			continue
		}
		if removed {
			return removedState, true, err
		}
		return UserState{}, false, err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("visitor websocket changed repeatedly during delivery")
	}
	return UserState{}, false, lastErr
}

func storeVisitorConnection(user *User) *User {
	clientMux.Lock()
	defer clientMux.Unlock()
	oldUser := clientList[user.Id]
	clientList[user.Id] = user
	return oldUser
}

func RemoveVisitorConnection(visitorID string, target *User) bool {
	clientMux.Lock()
	defer clientMux.Unlock()
	current, ok := clientList[visitorID]
	if !ok || current != target {
		return false
	}
	delete(clientList, visitorID)
	return true
}

// DisconnectVisitor atomically removes the currently registered visitor
// session before closing its socket. Removing first prevents the read loop
// from deleting a newer replacement connection.
func DisconnectVisitor(visitorID string, content []byte) (UserState, bool, error) {
	for attempt := 0; attempt < 4; attempt++ {
		visitor, ok := VisitorConnection(visitorID)
		if !ok || visitor == nil {
			return UserState{}, false, nil
		}
		if !RemoveVisitorConnection(visitorID, visitor) {
			continue
		}

		state := visitor.State()
		err := writeUserMessage(visitor, websocket.TextMessage, content)
		if visitor.Conn != nil {
			_ = visitor.Conn.Close()
		}
		return state, true, err
	}
	return UserState{}, false, fmt.Errorf("visitor websocket changed repeatedly during disconnect")
}
