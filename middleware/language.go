package middleware

import (
	"github.com/gin-gonic/gin"
	"go-fly-muti/types"
)

func SetLanguage(c *gin.Context) {
	lang := c.Query("lang")
	//lang := "en"

	if lang == "" {
		lang := c.GetHeader("lang")
		if lang == "" {
			lang = "cn"
		}
	}

	types.ApiCode.SetLanguage(lang)
	c.Set("lang", lang)
}
