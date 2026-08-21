package ws

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-fly-muti/common"
	"go-fly-muti/models"
	"go-fly-muti/setting"
	"log"
	"time"
)

func NewVisitorServer(c *gin.Context) {

	//升級協議   get=>ws
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	//获取GET参数,创建WS
	requestedToID := c.Query("to_id")
	visitorId := c.Query("visitor_id")
	if visitorId == "" {
		log.Println("访客ws参数为空")
		conn.Close()
		return
	}

	//獲取訪客
	vistorInfo := models.FindVisitorByVistorId(visitorId)
	toId := visitorWebSocketTarget(requestedToID, vistorInfo)
	if vistorInfo.VisitorId == "" || toId == "" {
		log.Println("访客visitorId不存在:", visitorId)
		conn.Close()
		return
	}
	if requestedToID != "" && requestedToID != toId {
		log.Printf("visitor websocket target ignored visitor_id=%q requested=%q stored=%q", visitorId, requestedToID, toId)
	}
	user := &User{
		Conn:       conn,
		Name:       fmt.Sprintf("#%d%s", vistorInfo.ID, vistorInfo.Name),
		Avator:     vistorInfo.Avator,
		Id:         vistorInfo.VisitorId,
		To_id:      toId,
		Ent_id:     vistorInfo.EntId,
		UpdateTime: setting.Now(),
	}
	if err := configureWebSocketHeartbeat(user); err != nil {
		log.Printf("websocket setup failed role=visitor id=%q error=%q", user.Id, err)
		_ = conn.Close()
		return
	}

	if AddVisitorToList(user) {
		VisitorOnline(toId, vistorInfo)
	} else {
		notifyVisitorOnline(toId, vistorInfo)
	}
	log.Printf(
		"websocket connected role=visitor id=%q session=%d kefu=%q remote=%q",
		user.Id, user.SessionID, user.State().ToID, c.ClientIP(),
	)
	//判断是否有room_id,代表聊天室
	roomId := c.Query("room_id")
	if roomId != "" {
		AddVisitorToRoom(roomId, vistorInfo.VisitorId)
	}
	for {
		//接受消息
		var receive []byte
		messageType, receive, err := conn.ReadMessage()
		if err != nil {
			code, reason := websocketCloseInfo(err)
			_ = conn.Close()
			if RemoveVisitorConnection(user.Id, user) {
				state := user.State()
				log.Printf(
					"websocket disconnected role=visitor id=%q session=%d kefu=%q code=%d reason=%q",
					state.Id, state.SessionID, state.ToID, code, reason,
				)
				VisitorOffline(state.ToID, state.Id, state.Name)
			} else {
				log.Printf(
					"websocket replaced role=visitor id=%q session=%d code=%d reason=%q",
					user.Id, user.SessionID, code, reason,
				)
			}
			return
		}
		if err := refreshWebSocketReadDeadline(user); err != nil {
			log.Printf("websocket deadline refresh failed role=visitor id=%q session=%d error=%q", user.Id, user.SessionID, err)
		}

		message <- &Message{
			user:        user,
			content:     receive,
			context:     c,
			messageType: messageType,
		}
	}
}

func visitorWebSocketTarget(_ string, visitor models.Visitor) string {
	return visitor.ToId
}

// AddVisitorToList returns true only when this is a newly-online visitor.
// Replacing a stale connection must not decrement and increment rec_num in
// competing goroutines, otherwise an active visitor can be counted offline.
func AddVisitorToList(user *User) bool {
	oldUser := storeVisitorConnection(user)
	if oldUser != nil {
		oldState := oldUser.State()
		user.SetUpdateTime(oldState.UpdateTime)
		closemsg := TypeMessage{
			Type: "close",
			Data: user.Id,
		}
		closeStr, _ := json.Marshal(closemsg)
		if err := writeUserMessage(oldUser, websocket.TextMessage, closeStr); err != nil {
			if oldUser.Conn != nil {
				_ = oldUser.Conn.Close()
			}
		}
		return false
	}
	return true
}
func AddVisitorToRoom(roomId, visitorId string) {
	user, ok := VisitorConnection(visitorId)
	if user != nil && ok {
		Room.addMember(roomId, user)
	}
}
func VisitorOnline(kefuId string, visitor models.Visitor) {
	go models.UpdateUserRecNum(kefuId, 1)
	notifyVisitorOnline(kefuId, visitor)
}

// ActiveVisitorCountsByKefu returns current websocket visitor counts for one
// enterprise. It is used for routing instead of the eventually-consistent
// rec_num database field, which may be reset when an agent reconnects.
func ActiveVisitorCountsByKefu(entId string) map[string]int {
	return activeVisitorCountsByKefu(entId, VisitorConnectionsSnapshot())
}

func activeVisitorCountsByKefu(entId string, visitors map[string]*User) map[string]int {
	counts := make(map[string]int)
	for _, visitor := range visitors {
		if visitor == nil {
			continue
		}
		state := visitor.State()
		if state.EntID == entId && state.ToID != "" {
			counts[state.ToID]++
		}
	}
	return counts
}

func notifyVisitorOnline(kefuId string, visitor models.Visitor) {
	lastMessage := models.FindLastMessageByVisitorId(visitor.VisitorId)
	unreadMap := models.FindUnreadMessageNumByVisitorIds([]string{visitor.VisitorId}, "visitor")
	var unreadNum uint32
	if num, ok := unreadMap[visitor.VisitorId]; ok {
		unreadNum = num
	}

	userInfo := make(map[string]string)
	userInfo["uid"] = visitor.VisitorId
	userInfo["visitor_id"] = visitor.VisitorId
	userInfo["username"] = fmt.Sprintf("#%d%s", visitor.ID, visitor.Name)
	userInfo["avator"] = visitor.Avator
	userInfo["last_message"] = lastMessage.Content
	userInfo["unread_num"] = fmt.Sprintf("%d", unreadNum)
	conversation := models.FindConversation(visitor.EntId, visitor.VisitorId)
	if conversation.ID != 0 {
		userInfo["service_status"] = conversation.Status
		userInfo["priority"] = conversation.Priority
		if conversation.WaitingSince != nil {
			userInfo["waiting_since"] = setting.Format(*conversation.WaitingSince)
			waitingSeconds := int64(setting.Now().Sub(*conversation.WaitingSince).Seconds())
			if waitingSeconds > 0 {
				userInfo["waiting_seconds"] = fmt.Sprintf("%d", waitingSeconds)
			}
		}
	} else {
		userInfo["priority"] = models.ConversationPriorityNormal
		if unreadNum > 0 {
			userInfo["service_status"] = models.ConversationStatusOpen
		} else {
			userInfo["service_status"] = models.ConversationStatusPending
		}
	}
	if userInfo["last_message"] == "" {
		userInfo["last_message"] = "新访客"
	}
	msg := TypeMessage{
		Type: "userOnline",
		Data: userInfo,
	}
	str, _ := json.Marshal(msg)
	OneKefuMessage(kefuId, str)
}
func VisitorOffline(kefuId string, visitorId string, visitorName string) {
	go models.UpdateUserRecNum(kefuId, -1)
	userInfo := make(map[string]string)
	userInfo["uid"] = visitorId
	userInfo["visitor_id"] = visitorId
	userInfo["name"] = visitorName
	msg := TypeMessage{
		Type: "userOffline",
		Data: userInfo,
	}
	str, _ := json.Marshal(msg)
	//新版
	OneKefuMessage(kefuId, str)
}
func VisitorNotice(visitorId string, notice string) {
	msg := TypeMessage{
		Type: "notice",
		Data: notice,
	}
	str, _ := json.Marshal(msg)
	visitor, ok := VisitorConnection(visitorId)
	if !ok || visitor == nil || visitor.Conn == nil {
		return
	}
	_ = writeUserMessage(visitor, websocket.TextMessage, str)
}
func VisitorCustomMessage(visitorId string, notice TypeMessage) {
	str, _ := json.Marshal(notice)
	visitor, ok := VisitorConnection(visitorId)
	if !ok || visitor == nil || visitor.Conn == nil {
		return
	}
	_ = writeUserMessage(visitor, websocket.TextMessage, str)
}
func VisitorTransfer(visitorId string, kefuId string) {
	msg := TypeMessage{
		Type: "transfer",
		Data: kefuId,
	}
	str, _ := json.Marshal(msg)
	visitor, ok := VisitorConnection(visitorId)
	if !ok || visitor == nil || visitor.Conn == nil {
		return
	}
	_ = writeUserMessage(visitor, websocket.TextMessage, str)
}
func VisitorMessage(visitorId, content string, kefuInfo models.User) {
	msg := TypeMessage{
		Type: "message",
		Data: ClientMessage{
			Name:    kefuInfo.Nickname,
			Avator:  kefuInfo.Avator,
			Id:      kefuInfo.Name,
			Time:    setting.Now().Format("2006-01-02 15:04:05"),
			ToId:    visitorId,
			Content: content,
			IsKefu:  "no",
		},
	}
	str, _ := json.Marshal(msg)
	visitor, ok := VisitorConnection(visitorId)
	if !ok || visitor == nil || visitor.Conn == nil {
		return
	}
	_ = writeUserMessage(visitor, websocket.TextMessage, str)
}
func VisitorAutoReply(vistorInfo models.Visitor, kefuInfo models.User, content string) {
	//var entInfo models.User
	//if fmt.Sprintf("%v", kefuInfo.Pid) == fmt.Sprintf("%d", 1) {
	//	entInfo = kefuInfo
	//} else {
	//	entInfo = models.FindUserByUid(kefuInfo.Pid)
	//}
	reply := models.FindArticleRow("ent_id = ? and find_in_set( ? , title)", vistorInfo.EntId, content)
	//reply = models.FindReplyItemByUserIdTitle(entInfo.Name, content)

	if reply.Content != "" {
		time.Sleep(1 * time.Second)
		VisitorMessage(vistorInfo.VisitorId, reply.Content, kefuInfo)
		KefuMessage(vistorInfo.VisitorId, reply.Content, kefuInfo)
		models.CreateMessage(kefuInfo.Name, vistorInfo.VisitorId, reply.Content, "kefu", vistorInfo.EntId, "read")
	}
}
func CleanVisitorExpire() {
	go func() {
		log.Println("cleanVisitorExpire start...")
		for {
			for _, user := range VisitorConnectionsSnapshot() {
				state := user.State()
				diff := time.Since(state.UpdateTime).Seconds()
				if diff >= common.VisitorExpire {
					entConfig := models.FindEntConfig(state.EntID, "CloseVisitorMessage")
					if entConfig.ConfValue != "" {
						kefu := models.FindUserByUid(state.EntID)
						VisitorMessage(state.Id, entConfig.ConfValue, kefu)
					}
					msg := TypeMessage{
						Type: "auto_close",
						Data: state.Id,
					}
					str, _ := json.Marshal(msg)
					if err := writeUserMessage(user, websocket.TextMessage, str); err != nil {
						_ = user.Conn.Close()
						if RemoveVisitorConnection(state.Id, user) {
							VisitorOffline(state.ToID, state.Id, state.Name)
						}
					}
					log.Println(state.Name + ":cleanVisitorExpire finished")
				}
			}
			t := time.NewTimer(time.Second * 5)
			<-t.C
		}
	}()
}
