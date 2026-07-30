package v2

import (
	"github.com/gin-gonic/gin"
	"go-fly-muti/lib"
	"go-fly-muti/types"
	"os"
	"strconv"
)

func PostEmailCode(c *gin.Context) {
	email := c.PostForm("email")
	server := os.Getenv("GOFLY_SMTP_SERVER")
	from := os.Getenv("GOFLY_SMTP_FROM")
	password := os.Getenv("GOFLY_SMTP_PASSWORD")
	port, err := strconv.Atoi(os.Getenv("GOFLY_SMTP_PORT"))
	if server == "" || from == "" || password == "" || err != nil || port < 1 || port > 65535 {
		c.JSON(200, gin.H{
			"code": types.ApiCode.FAILED,
			"msg":  "邮件服务未配置",
		})
		return
	}
	notify := &lib.Notify{
		Subject:     "测试主题",
		MainContent: "测试内容",
		EmailServer: lib.NotifyEmail{
			Server:   server,
			Port:     uint(port),
			From:     from,
			Password: password,
			To:       []string{email},
			FromName: "GOFLY客服",
		},
	}
	_, err = notify.SendMail()
	if err != nil {
		c.JSON(200, gin.H{
			"code": types.ApiCode.FAILED,
			"msg":  types.ApiCode.GetMessage(types.ApiCode.FAILED),
		})
		return
	}
	c.JSON(200, gin.H{
		"code": types.ApiCode.SUCCESS,
		"msg":  types.ApiCode.GetMessage(types.ApiCode.SUCCESS),
	})
	return
}
