package middleware

import (
	"github.com/gin-gonic/gin"
	"go-fly-muti/models"
	"go-fly-muti/setting"
	"go-fly-muti/types"
)

func Ipblack(c *gin.Context) {
	rejectBlacklistedIP(c, c.ClientIP())
}

func VisitorIpblack(c *gin.Context) {
	rejectBlacklistedIP(c, setting.VisitorClientIP(c.Request).String())
}

func rejectBlacklistedIP(c *gin.Context, ip string) {
	ipblack := models.FindIp(ip)
	if ipblack.IP != "" {
		c.JSON(200, gin.H{
			"code": types.ApiCode.IP_BAN,
			"msg":  types.ApiCode.GetMessage(types.ApiCode.IP_BAN),
		})
		c.Abort()
		return
	}
}
