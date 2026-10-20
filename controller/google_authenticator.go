package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"go-fly-muti/models"
	"strconv"
)

func googleAuthenticatorState() gin.H {
	return gin.H{"enabled": models.GoogleAuthenticatorEnabled()}
}
func GetGoogleAuthenticatorSettings(c *gin.Context) {
	c.JSON(200, gin.H{"code": 200, "msg": "ok", "result": googleAuthenticatorState()})
}
func PostGoogleAuthenticatorSettings(c *gin.Context) {
	action := c.PostForm("action")
	if action == "reset-account" {
		id, err := strconv.ParseUint(c.PostForm("user_id"), 10, 32)
		if err != nil || id == 0 || models.FindUserByUid(uint(id)).ID == 0 {
			c.JSON(200, gin.H{"code": 400, "msg": "账号不存在"})
			return
		}
		tx := models.DB.Begin()
		if tx.Error != nil {
			c.JSON(200, gin.H{"code": 400, "msg": "重置失败"})
			return
		}
		if err := tx.Where("user_id = ?", id).Delete(&models.AccountTOTP{}).Error; err != nil {
			tx.Rollback()
			c.JSON(200, gin.H{"code": 400, "msg": "重置失败"})
			return
		}
		if err := tx.Model(&models.User{}).Where("id = ?", id).UpdateColumn("auth_version", gorm.Expr("auth_version + 1")).Error; err != nil {
			tx.Rollback()
			c.JSON(200, gin.H{"code": 400, "msg": "重置失败"})
			return
		}
		if tx.Commit().Error != nil {
			c.JSON(200, gin.H{"code": 400, "msg": "重置失败"})
			return
		}
		c.JSON(200, gin.H{"code": 200, "msg": "已重置，此账号需重新登录绑定", "result": googleAuthenticatorState()})
		return
	}
	if action != "enable" && action != "disable" {
		c.JSON(200, gin.H{"code": 400, "msg": "仅支持开启或关闭全局验证，各账号请在登录时绑定自己的验证器"})
		return
	}
	value := "false"
	if action == "enable" {
		value = "true"
	}
	if err := models.SaveConfig("谷歌登录验证", models.GoogleAuthenticatorEnabledKey, value); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "保存失败"})
		return
	}
	c.JSON(200, gin.H{"code": 200, "msg": "保存成功，请重新登录", "result": googleAuthenticatorState()})
}
