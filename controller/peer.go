package controller

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"go-fly-muti/models"
	"go-fly-muti/setting"
	"go-fly-muti/ws"
)

func PostCallKefu(c *gin.Context) {
	kefuId := c.PostForm("kefu_id")
	visitorId := c.PostForm("visitor_id")
	kefuInfo := models.FindUser(kefuId)
	vistorInfo := models.FindVisitorByVistorId(visitorId)
	if kefuInfo.ID == 0 || vistorInfo.ID == 0 {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "用户不存在",
		})
		return
	}
	msg := ws.TypeMessage{
		Type: "callpeer",
		Data: ws.ClientMessage{
			Avator:  vistorInfo.Avator,
			Id:      vistorInfo.VisitorId,
			Name:    vistorInfo.Name,
			ToId:    kefuInfo.Name,
			Content: "请求通话",
			Time:    setting.Now().Format("2006-01-02 15:04:05"),
			IsKefu:  "no",
		},
	}
	str, _ := json.Marshal(msg)
	ws.OneKefuMessage(kefuInfo.Name, str)
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
	})
}
func PostKefuPeerId(c *gin.Context) {
	peerId := c.PostForm("peer_id")
	visitorId := c.PostForm("visitor_id")
	kefuName, _ := c.Get("kefu_name")
	if peerId == "" {
		c.JSON(200, gin.H{
			"code":   400,
			"msg":    "peer_id不能为空",
			"result": "",
		})
		return
	}
	msg := ws.TypeMessage{
		Type: "peerid",
		Data: peerId,
	}
	str, _ := json.Marshal(msg)
	visitor, ok := ws.VisitorConnection(visitorId)
	state := visitor.State()
	if !ok || state.Name == "" || fmt.Sprintf("%v", kefuName) != state.ToID {
		c.JSON(200, gin.H{
			"code":   400,
			"msg":    "客户不存在",
			"result": "",
		})
		return
	}
	if err := ws.SendMessageToVisitor(visitor, str); err != nil {
		c.JSON(200, gin.H{
			"code":   500,
			"msg":    "发送失败",
			"result": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
	})
}
