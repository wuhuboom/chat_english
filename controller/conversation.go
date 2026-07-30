package controller

import (
	"fmt"
	"go-fly-muti/models"
	"go-fly-muti/setting"
	"strings"

	"github.com/gin-gonic/gin"
)

func enrichConversationUsers(users []*VisitorOnline, entId string) {
	visitorIds := make([]string, 0, len(users))
	for _, user := range users {
		visitorIds = append(visitorIds, user.VisitorId)
	}
	conversations := models.FindConversationMap(entId, visitorIds)
	for _, user := range users {
		conversation := conversations[user.VisitorId]
		if conversation.ID == 0 {
			user.Priority = models.ConversationPriorityNormal
			if user.UnreadNum > 0 {
				user.ServiceStatus = models.ConversationStatusOpen
			} else {
				user.ServiceStatus = models.ConversationStatusPending
			}
			continue
		}
		user.ServiceStatus = conversation.Status
		user.Priority = conversation.Priority
		if conversation.WaitingSince != nil {
			user.WaitingSince = setting.Format(*conversation.WaitingSince)
			waitingSeconds := int64(setting.Now().Sub(*conversation.WaitingSince).Seconds())
			if waitingSeconds > 0 {
				user.WaitingSeconds = waitingSeconds
			}
		}
	}
}

func enrichConversationUserValues(users []VisitorOnline, entId string) {
	pointers := make([]*VisitorOnline, 0, len(users))
	for i := range users {
		pointers = append(pointers, &users[i])
	}
	enrichConversationUsers(pointers, entId)
}

func GetConversation(c *gin.Context) {
	entId, _ := c.Get("ent_id")
	visitorId := strings.TrimSpace(c.Query("visitor_id"))
	visitor := models.FindVisitorByVistorId(visitorId)
	if visitor.ID == 0 || visitor.EntId != fmt.Sprintf("%v", entId) {
		c.JSON(200, gin.H{"code": 400, "msg": "会话不存在"})
		return
	}

	conversation := models.FindConversation(visitor.EntId, visitorId)
	if conversation.ID == 0 {
		conversation = models.UpdateConversationWorkflow(
			visitor.EntId,
			visitorId,
			visitor.ToId,
			models.ConversationStatusPending,
			models.ConversationPriorityNormal,
			setting.Now(),
		)
	}
	c.JSON(200, gin.H{"code": 200, "msg": "ok", "result": conversation})
}

func GetConversationEvents(c *gin.Context) {
	entIdValue, _ := c.Get("ent_id")
	entId := fmt.Sprintf("%v", entIdValue)
	visitorId := strings.TrimSpace(c.Query("visitor_id"))
	visitor := models.FindVisitorByVistorId(visitorId)
	if visitor.ID == 0 || visitor.EntId != entId {
		c.JSON(200, gin.H{"code": 400, "msg": "会话不存在"})
		return
	}
	events := models.FindConversationEvents(entId, visitorId, 100)
	c.JSON(200, gin.H{"code": 200, "msg": "ok", "result": events})
}

func UpdateConversation(c *gin.Context) {
	entIdValue, _ := c.Get("ent_id")
	kefuIdValue, _ := c.Get("kefu_name")
	entId := fmt.Sprintf("%v", entIdValue)
	kefuId := fmt.Sprintf("%v", kefuIdValue)
	visitorId := strings.TrimSpace(c.PostForm("visitor_id"))
	status := strings.TrimSpace(c.PostForm("status"))
	priority := strings.TrimSpace(c.PostForm("priority"))

	visitor := models.FindVisitorByVistorId(visitorId)
	if visitor.ID == 0 || visitor.EntId != entId || visitor.ToId != kefuId {
		c.JSON(200, gin.H{"code": 400, "msg": "只能操作分配给自己的会话"})
		return
	}
	if status != "" && status != models.ConversationStatusOpen &&
		status != models.ConversationStatusPending && status != models.ConversationStatusResolved {
		c.JSON(200, gin.H{"code": 400, "msg": "无效的会话状态"})
		return
	}
	if priority != "" && priority != models.ConversationPriorityNormal &&
		priority != models.ConversationPriorityHigh && priority != models.ConversationPriorityUrgent {
		c.JSON(200, gin.H{"code": 400, "msg": "无效的优先级"})
		return
	}
	if status == "" && priority == "" {
		c.JSON(200, gin.H{"code": 400, "msg": "没有需要更新的内容"})
		return
	}

	before := models.FindConversation(entId, visitorId)
	now := setting.Now()
	conversation := models.UpdateConversationWorkflow(entId, visitorId, kefuId, status, priority, now)
	if status != "" && before.Status != conversation.Status {
		models.CreateConversationEvent(
			entId, visitorId, models.ConversationEventStatusChanged,
			kefuId, kefuId, kefuId,
			before.Status+" → "+conversation.Status, now,
		)
	}
	if priority != "" && before.Priority != conversation.Priority {
		models.CreateConversationEvent(
			entId, visitorId, models.ConversationEventPriorityChanged,
			kefuId, kefuId, kefuId,
			before.Priority+" → "+conversation.Priority, now,
		)
	}
	c.JSON(200, gin.H{"code": 200, "msg": "保存成功", "result": conversation})
}
