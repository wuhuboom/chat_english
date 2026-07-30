// Package middleware TODO  跨域处理
package middleware

import (
	"github.com/gin-gonic/gin"
	"go-fly-muti/common"
	"net/http"
)

func CrossSite(c *gin.Context) {
	origin := c.GetHeader("Origin")
	allowedOrigin, allowCredentials := common.AllowedOrigin(origin)
	if allowedOrigin != "" {
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
	}
	if allowedOrigin != "*" {
		c.Header("Vary", "Origin")
	}
	//服务器支持的所有跨域请求的方法
	c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	//允许跨域设置可以返回其他子段，可以自定义字段
	c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session")
	// 允许浏览器（客户端）可以解析的头部 （重要）
	c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers")
	if allowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}
	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	c.Next()
}
