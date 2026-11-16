package tmpl

import (
	"github.com/gin-gonic/gin"
	"go-fly-muti/models"

	"net/http"
)

// 登陆界面
func PageLogin(c *gin.Context) {

	SystemNotice := models.FindConfig("SystemNotice")
	SystemTitle := models.FindConfig("SystemTitle")
	SystemLoginTitle := models.FindConfig("SystemLoginTitle")
	SystemKeywords := models.FindConfig("SystemKeywords")
	SystemDesc := models.FindConfig("SystemDesc")
	CopyrightTxt := models.FindConfig("CopyrightTxt")
	CopyrightUrl := models.FindConfig("CopyrightUrl")
	googleAuthenticatorEnabled := models.GoogleAuthenticatorEnabled()
	isRegister := models.RegistrationEnabled()
	c.HTML(http.StatusOK, "login.html", gin.H{
		"SystemNotice":               SystemNotice,
		"SystemTitle":                SystemTitle,
		"SystemKeywords":             SystemKeywords,
		"SystemDesc":                 SystemDesc,
		"CopyrightTxt":               CopyrightTxt,
		"CopyrightUrl":               CopyrightUrl,
		"SystemLoginTitle":           SystemLoginTitle,
		"isRegister":                 isRegister,
		"GoogleAuthenticatorEnabled": googleAuthenticatorEnabled,
	})
}
