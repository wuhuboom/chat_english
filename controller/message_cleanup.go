package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	applogger "go-fly-muti/logger"
	"go-fly-muti/models"
	"go-fly-muti/setting"
)

const (
	messageCleanupTimeLayout   = "2006-01-02 15:04:05"
	messageCleanupConfirmation = "CLEAR_CHAT_MESSAGES"
)

type messageCleanupForm struct {
	StartTime    string `form:"start_time" json:"start_time" binding:"required"`
	EndTime      string `form:"end_time" json:"end_time" binding:"required"`
	Confirmation string `form:"confirmation" json:"confirmation"`
}

var (
	countMessagesInRange  = models.CountMessagesInRange
	deleteMessagesInRange = models.DeleteMessagesInRange
)

func parseMessageCleanupRange(startValue, endValue string) (time.Time, time.Time, error) {
	start, err := time.ParseInLocation(messageCleanupTimeLayout, strings.TrimSpace(startValue), setting.Location())
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("开始时间格式无效，请使用 YYYY-MM-DD HH:mm:ss")
	}
	end, err := time.ParseInLocation(messageCleanupTimeLayout, strings.TrimSpace(endValue), setting.Location())
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("结束时间格式无效，请使用 YYYY-MM-DD HH:mm:ss")
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("结束时间必须晚于开始时间")
	}
	return start, end, nil
}

func GetMessageCleanupPreview(c *gin.Context) {
	start, end, err := parseMessageCleanupRange(c.Query("start_time"), c.Query("end_time"))
	if err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	entID := c.GetString("ent_id")
	count, err := countMessagesInRange(entID, start, end)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "统计聊天记录失败"})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
		"result": gin.H{
			"count":      count,
			"start_time": setting.Format(start),
			"end_time":   setting.Format(end),
			"timezone":   setting.CurrentTimezone(),
		},
	})
}

func DeleteMessagesByTimeRange(c *gin.Context) {
	var form messageCleanupForm
	if err := c.ShouldBind(&form); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "请选择完整的开始时间和结束时间"})
		return
	}
	if form.Confirmation != messageCleanupConfirmation {
		c.JSON(200, gin.H{"code": 400, "msg": "清理确认无效，请重新操作"})
		return
	}
	start, end, err := parseMessageCleanupRange(form.StartTime, form.EndTime)
	if err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	entID := c.GetString("ent_id")
	deleted, err := deleteMessagesInRange(entID, start, end)
	if err != nil {
		zap.L().Error("merchant chat cleanup failed",
			zap.String("component", "audit"),
			zap.String("ent_id", entID),
			zap.String("start_time", setting.Format(start)),
			zap.String("end_time", setting.Format(end)),
			zap.Error(err),
		)
		c.JSON(200, gin.H{"code": 500, "msg": "清理聊天记录失败，请查看系统日志"})
		return
	}

	requestID, _ := c.Get(applogger.RequestIDKey)
	actor, _ := c.Get("kefu_name")
	zap.L().Warn("merchant chat messages permanently deleted",
		zap.String("component", "audit"),
		zap.String("operation", "message_cleanup"),
		zap.String("ent_id", entID),
		zap.Any("actor", actor),
		zap.Any("request_id", requestID),
		zap.String("start_time", setting.Format(start)),
		zap.String("end_time", setting.Format(end)),
		zap.Int64("deleted_count", deleted),
	)
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "聊天记录清理完成",
		"result": gin.H{
			"deleted_count": deleted,
			"start_time":    setting.Format(start),
			"end_time":      setting.Format(end),
			"timezone":      setting.CurrentTimezone(),
		},
	})
}
