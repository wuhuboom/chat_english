package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go-fly-muti/models"
	"go-fly-muti/setting"
	"go-fly-muti/ws"
)

func GetVisitorAccessSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "result": gin.H{
		"required": models.VisitorVerificationRequired(),
	}})
}

func PostVisitorAccessSettings(c *gin.Context) {
	var form struct {
		Required *bool `form:"required" json:"required"`
	}
	if c.ShouldBind(&form) != nil || form.Required == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "请选择是否关闭普通访客"})
		return
	}
	if *form.Required && setting.CustomerVerifyURL() == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "请先在 config.yaml 设置 app.customer_verify_url"})
		return
	}
	if err := models.SaveConfig("关闭普通访客", models.VisitorAccessConfigKey, strconv.FormatBool(*form.Required)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存访客设置失败"})
		return
	}
	if *form.Required {
		ws.CloseUnverifiedVisitors()
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "result": gin.H{"required": *form.Required}})
}
