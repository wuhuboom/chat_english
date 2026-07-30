package ws

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-fly-muti/models"
	"go-fly-muti/setting"
	"log"
	"time"
)

func NewKefuServer(c *gin.Context) {

	fmt.Println("======")
	kefuId, _ := c.Get("kefu_id")
	kefuInfo := models.FindUserById(kefuId)
	if kefuInfo.ID == 0 {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "用户不存在",
		})
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	//获取GET参数,创建WS
	var kefu User
	kefu.Id = kefuInfo.Name
	kefu.Name = kefuInfo.Nickname
	kefu.Avator = kefuInfo.Avator
	kefu.Role_id = kefuInfo.RoleId
	kefu.Ent_id = fmt.Sprintf("%d", kefuInfo.ID)
	if kefuInfo.Pid != 0 {
		kefu.Ent_id = fmt.Sprintf("%d", kefuInfo.Pid)
	}
	kefu.Conn = conn
	kefu.UpdateTime = time.Now()
	if err := configureWebSocketHeartbeat(&kefu); err != nil {
		log.Printf("websocket setup failed role=kefu id=%q error=%q", kefu.Id, err)
		_ = conn.Close()
		return
	}
	AddKefuToList(&kefu)
	log.Printf(
		"websocket connected role=kefu id=%q session=%d active=%d remote=%q",
		kefu.Id, kefu.SessionID, len(KefuConnections(kefu.Id)), c.ClientIP(),
	)
	go models.UpdateUserRecNumZero(kefuInfo.Name)
	for {
		//接受消息
		var receive []byte
		messageType, receive, err := conn.ReadMessage()
		if err != nil {
			code, reason := websocketCloseInfo(err)
			_ = conn.Close()
			removedLast := RemoveKefuConnection(kefu.Id, &kefu)
			log.Printf(
				"websocket disconnected role=kefu id=%q session=%d code=%d reason=%q active=%d",
				kefu.Id, kefu.SessionID, code, reason, len(KefuConnections(kefu.Id)),
			)
			if removedLast {
				scheduleKefuFailover(kefu.Id, kefu.Ent_id)
			}
			return
		}
		if err := refreshWebSocketReadDeadline(&kefu); err != nil {
			log.Printf("websocket deadline refresh failed role=kefu id=%q session=%d error=%q", kefu.Id, kefu.SessionID, err)
		}

		message <- &Message{
			user:        &kefu,
			content:     receive,
			context:     c,
			messageType: messageType,
		}
	}
}
func AddKefuToList(kefu *User) {
	Mux.Lock()
	KefuList[kefu.Id] = append(KefuList[kefu.Id], kefu)
	Mux.Unlock()
}

func RemoveKefuConnection(kefuID string, target *User) bool {
	Mux.Lock()
	defer Mux.Unlock()
	connections := KefuList[kefuID]
	active := make([]*User, 0, len(connections))
	for _, connection := range connections {
		if connection != target {
			active = append(active, connection)
		}
	}
	if len(active) == 0 {
		delete(KefuList, kefuID)
		return len(connections) > 0
	}
	KefuList[kefuID] = active
	return false
}

func KefuConnections(kefuID string) []*User {
	Mux.RLock()
	defer Mux.RUnlock()
	return append([]*User(nil), KefuList[kefuID]...)
}

func IsKefuOnline(kefuID string) bool {
	return len(KefuConnections(kefuID)) > 0
}

func KefuListSnapshot() map[string][]*User {
	Mux.RLock()
	defer Mux.RUnlock()
	snapshot := make(map[string][]*User, len(KefuList))
	for kefuID, connections := range KefuList {
		snapshot[kefuID] = append([]*User(nil), connections...)
	}
	return snapshot
}

func removeBrokenKefuConnection(kefu *User) bool {
	if kefu == nil {
		return false
	}
	if kefu.Conn != nil {
		_ = kefu.Conn.Close()
	}
	state := kefu.State()
	return RemoveKefuConnection(state.Id, kefu)
}

// 给超管发消息
func SuperAdminMessage(str []byte) {
}

// 给指定客服发消息
func OneKefuMessage(toId string, str []byte) {
	//新版
	mKefuConns := KefuConnections(toId)
	if len(mKefuConns) > 0 {
		for _, kefu := range mKefuConns {
			error := writeUserMessage(kefu, websocket.TextMessage, str)
			if error != nil {
				state := kefu.State()
				removedLast := removeBrokenKefuConnection(kefu)
				log.Printf(
					"websocket write failed role=kefu id=%q session=%d error=%q active=%d",
					state.Id, state.SessionID, error, len(KefuConnections(state.Id)),
				)
				if removedLast {
					scheduleKefuFailover(state.Id, state.EntID)
				}
			}
		}
	}
	SuperAdminMessage(str)
}
func KefuMessage(visitorId, content string, kefuInfo models.User) {
	msg := TypeMessage{
		Type: "message",
		Data: ClientMessage{
			Name:    kefuInfo.Nickname,
			Avator:  kefuInfo.Avator,
			Id:      visitorId,
			Time:    setting.Now().Format("2006-01-02 15:04:05"),
			ToId:    visitorId,
			Content: content,
			IsKefu:  "yes",
		},
	}
	str, _ := json.Marshal(msg)
	OneKefuMessage(kefuInfo.Name, str)
}

// 给客服客户端发送消息判断客户端是否在线
func SendPingToKefuClient() {
	msg := TypeMessage{
		Type: "many pong",
	}
	str, _ := json.Marshal(msg)
	failed := make(map[*User]struct{})
	for _, kfConns := range KefuListSnapshot() {
		for _, kefuConn := range kfConns {
			if kefuConn == nil {
				continue
			}
			state := kefuConn.State()
			if isKefuConnectionStale(state.UpdateTime, time.Now()) {
				failed[kefuConn] = struct{}{}
				if kefuConn.Conn != nil {
					_ = kefuConn.Conn.Close()
				}
				continue
			}
			if err := writeUserControlMessage(kefuConn, websocket.PingMessage, nil); err != nil {
				failed[kefuConn] = struct{}{}
				continue
			}
			if err := writeUserMessage(kefuConn, websocket.TextMessage, str); err != nil {
				failed[kefuConn] = struct{}{}
			}
		}
	}
	for kefuID, connections := range KefuListSnapshot() {
		for _, connection := range connections {
			if _, ok := failed[connection]; ok {
				state := connection.State()
				if removeBrokenKefuConnection(connection) {
					scheduleKefuFailover(kefuID, state.EntID)
				}
			}
		}
	}
}

// GetEntOnlineKefuId 获取企业下在线的客服
func GetEntOnlineKefuId(entId string) string {
	for kefuId, kefuConn := range KefuListSnapshot() {
		if len(kefuConn) > 0 {
			if kefuConn[0].Ent_id == entId {
				return kefuId
			}
		}
	}
	return ""
}
