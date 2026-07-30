package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	applogger "go-fly-muti/logger"
)

// GetSystemLogs returns a bounded, newest-first view of the primary
// application log. The route is protected by JwtApiMiddleware and AdminAuth.
func GetSystemLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	result, err := applogger.Query(applogger.LogQuery{
		Level:     c.Query("level"),
		Keyword:   c.Query("keyword"),
		RequestID: c.Query("request_id"),
		Limit:     limit,
	})
	if err != nil {
		c.JSON(200, gin.H{
			"code": 500,
			"msg":  "读取系统日志失败: " + err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": result,
	})
}
