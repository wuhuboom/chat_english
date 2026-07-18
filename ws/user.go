package ws

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-fly-muti/models"
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
	kefu.Conn = conn
	AddKefuToList(&kefu)
	go models.UpdateUserRecNumZero(kefuInfo.Name)
	for {
		//接受消息
		var receive []byte
		messageType, receive, err := conn.ReadMessage()
		if err != nil {
			log.Println("ws/user.go ", err)
			conn.Close()
			RemoveKefuConnection(kefu.Id, &kefu)
			return
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

func RemoveKefuConnection(kefuID string, target *User) {
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
		return
	}
	KefuList[kefuID] = active
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

// 给超管发消息
func SuperAdminMessage(str []byte) {
	return
	//给超管发
	for _, kefuUsers := range KefuList {
		for _, kefuUser := range kefuUsers {
			if kefuUser.Role_id == "2" {
				kefuUser.Conn.WriteMessage(websocket.TextMessage, str)
			}
		}
	}
}

// 给指定客服发消息
func OneKefuMessage(toId string, str []byte) {
	//新版
	mKefuConns := KefuConnections(toId)
	if len(mKefuConns) > 0 {
		for _, kefu := range mKefuConns {
			error := writeUserMessage(kefu, websocket.TextMessage, str)
			if error != nil {
				//if websocket.IsCloseError(error, websocket.CloseGoingAway) {
				//	// 连接已关闭，不再进行写入操作
				//	log.Println("连接已关闭，不再进行写入操作", error, string(str))
				//	return
				//}
				log.Println("send_kefu_message", error, string(str))
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
			Time:    time.Now().Format("2006-01-02 15:04:05"),
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
			if err := writeUserMessage(kefuConn, websocket.TextMessage, str); err != nil {
				failed[kefuConn] = struct{}{}
			}
		}
	}
	for kefuID, connections := range KefuListSnapshot() {
		for _, connection := range connections {
			if _, ok := failed[connection]; ok {
				RemoveKefuConnection(kefuID, connection)
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
