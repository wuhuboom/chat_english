package middleware

import (
	"github.com/gin-gonic/gin"
	"go-fly-muti/models"
	"go-fly-muti/tools"
	"go-fly-muti/types"
)

/*
*
jwt验证中间件
*/
func JwtApiV2Middleware(c *gin.Context) {

	token := c.GetHeader("token")
	if token == "" {
		token = c.Query("token")
	}
	orgToken, err := tools.ParseCliamsToken(token, true)
	if err != nil {
		c.JSON(200, gin.H{
			"code": types.ApiCode.TOKEN_FAILED,
			"msg":  err.Error(),
		})
		c.Abort()
		return
	}
	currentUser := models.FindUserByUid(orgToken.Id)
	if currentUser.ID == 0 || currentUser.Name != orgToken.Username || currentUser.Status == 1 || orgToken.AuthVersion <= 0 || currentUser.TokenVersion() != orgToken.AuthVersion {
		c.JSON(200, gin.H{"code": types.ApiCode.TOKEN_FAILED, "msg": "登录状态已失效，请重新登录"})
		c.Abort()
		return
	}
	if models.GoogleAuthenticatorEnabled() && (!orgToken.MFAVerified || !models.AccountTOTPBound(currentUser.ID)) {
		c.JSON(200, gin.H{"code": 401, "msg": "请重新登录并完成个人谷歌验证"})
		c.Abort()
		return
	}
	c.Set("user", orgToken.Username)
	c.Set("kefu_id", orgToken.Id)
	c.Set("kefu_name", orgToken.Username)
	c.Set("role_id", orgToken.RoleId)
	c.Set("pid", orgToken.Pid)
	if orgToken.Pid <= 1 {
		c.Set("ent_id", orgToken.Id)
	} else {
		c.Set("ent_id", orgToken.Pid)
	}
}
