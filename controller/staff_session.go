package controller

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"go-fly-muti/middleware"
	"go-fly-muti/models"
	"go-fly-muti/tools"
	"go-fly-muti/ws"
)

func RefreshStaffSession(c *gin.Context) {
	if !c.GetBool("staff_cookie") {
		c.JSON(401, gin.H{"code": 401})
		return
	}
	raw, _ := c.Get("staff_claims")
	claims := raw.(map[string]interface{})
	now := time.Now().Unix()
	start, ok := claims["session_started"].(float64)
	if !ok || now-int64(start) >= 7*middleware.StaffCookieLifetime {
		c.JSON(401, gin.H{"code": 401, "msg": "请重新登录"})
		return
	}
	claims["create_time"] = now
	token, err := tools.MakeToken(claims)
	if err != nil {
		c.JSON(500, gin.H{"code": 500})
		return
	}
	middleware.SetStaffCookie(c, token, middleware.StaffCookieLifetime)
	c.JSON(200, gin.H{"code": 200})
}

func LogoutStaffSession(c *gin.Context) {
	if !c.GetBool("staff_cookie") {
		c.JSON(401, gin.H{"code": 401})
		return
	}
	id, _ := c.Get("kefu_id")
	version, _ := c.Get("staff_auth_version")
	if err := models.DB.Model(&models.User{}).Where("id = ? AND auth_version = ?", id, version).
		UpdateColumn("auth_version", gorm.Expr("auth_version + 1")).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "退出失败，请重试"})
		return
	}
	for _, conn := range ws.KefuConnections(c.GetString("kefu_name")) {
		if conn.Conn != nil {
			_ = conn.Conn.Close()
		}
	}
	middleware.SetStaffCookie(c, "", -1)
	c.JSON(200, gin.H{"code": 200})
}
