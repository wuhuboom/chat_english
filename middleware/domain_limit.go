package middleware

import (
	"github.com/gin-gonic/gin"
	"go-fly-muti/common"
	"go-fly-muti/tools"
	"go-fly-muti/types"
	"net/url"
	"strings"
)

// DomainLimitMiddleware /**  TODO  域名白名单
func DomainLimitMiddleware(c *gin.Context) {
	if common.DomainWhiteList == "*" {
		return
	}
	host := ""
	if origin, err := url.Parse(staffOrigin(c.Request)); err == nil {
		host = origin.Hostname()
	}

	allowHost := strings.Split(common.DomainWhiteList, ",")
	if exist, _ := tools.InArray(host, allowHost); host == "" || !exist {
		c.JSON(200, gin.H{
			"code": types.ApiCode.DOMAIN_LIMIT,
			"msg":  types.ApiCode.GetMessage(types.ApiCode.DOMAIN_LIMIT),
		})
		c.Abort()
		return
	}
}
