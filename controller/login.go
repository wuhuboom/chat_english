package controller

import (
	"github.com/gin-gonic/gin"
	"go-fly-muti/models"
	"go-fly-muti/tools"
	"go-fly-muti/types"
	"go.uber.org/zap"
	"strings"
	"time"
)

type LoginForm struct {
	Username   string `form:"username" json:"username" uri:"username" xml:"username" binding:"required"`
	Password   string `form:"password" json:"password" uri:"password" xml:"password" binding:"required"`
	GoogleCode string `form:"google_code" json:"google_code" uri:"google_code" xml:"google_code"`
}

// LoginCheckPass @Summary 登陆验证接口
// @Produce  json
// @Accept multipart/form-data
// @Param username formData   string true "用户名"
// @Param password formData   string true "密码"
// @Param type formData   string true "类型"
// @Success 200 {object} controller.Response
// @Failure 200 {object} controller.Response
// @Router /check [post]
// 验证接口
func LoginCheckPass(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	var form LoginForm
	err := c.Bind(&form)

	if err != nil {
		c.JSON(200, gin.H{
			"code":   types.ApiCode.FAILED,
			"msg":    types.ApiCode.GetMessage(types.ApiCode.INVALID),
			"result": err.Error(),
		})
		return
	}
	form.Username = strings.TrimSpace(form.Username)
	limitKey := c.ClientIP() + ":" + strings.ToLower(form.Username)
	if !tools.SecurityLimitFreqSingle("login:"+limitKey, 10, 300) {
		zap.L().Warn("login rate limited", zap.String("ip", c.ClientIP()), zap.String("username", form.Username))
		c.JSON(200, gin.H{"code": 429, "msg": "登录尝试过多，请5分钟后重试"})
		return
	}

	info, ok := CheckKefuPass(form.Username, form.Password)
	if !ok {
		zap.L().Warn("login failed", zap.String("ip", c.ClientIP()), zap.String("username", form.Username))
		c.JSON(200, gin.H{
			"code": types.ApiCode.LOGIN_FAILED,
			"msg":  types.ApiCode.GetMessage(types.ApiCode.LOGIN_FAILED),
		})
		return
	}
	if info.Status == 1 {
		c.JSON(200, gin.H{
			"code": types.ApiCode.ACCOUNT_FORBIDDEN,
			"msg":  types.ApiCode.GetMessage(types.ApiCode.ACCOUNT_FORBIDDEN),
		})
		return
	}
	if binding, err := models.CheckAccountTOTP(info, form.GoogleCode); err != nil {
		c.JSON(200, gin.H{"code": types.ApiCode.GOOGLE_AUTH_FAILED, "msg": "谷歌验证失败，请检查动态码后重试"})
		return
	} else if binding != nil {
		c.JSON(200, gin.H{"code": 428, "msg": "请绑定此账号自己的谷歌验证器，再输入动态码登录", "result": binding})
		return
	}
	//获取远程结果
	err, ok = CheckServerAddress()
	if !ok {
		c.JSON(200, gin.H{
			"code": types.ApiCode.LOGIN_FAILED,
			"msg":  err.Error(),
		})
		return
	}

	token := GenUserToken(info)
	zap.L().Info("login succeeded", zap.String("ip", c.ClientIP()), zap.String("username", info.Name), zap.Uint("kefu_id", info.ID))
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "验证成功,正在跳转",
		"result": gin.H{
			"token": token,
		},
	})
}
func GenUserToken(info models.User) string {
	userinfo := make(map[string]interface{})
	userinfo["name"] = info.Name
	userinfo["kefu_id"] = info.ID
	userinfo["type"] = "kefu"
	userinfo["mfa_verified"] = models.GoogleAuthenticatorEnabled()
	uRole := models.FindRoleByUserId(info.ID)
	if uRole.RoleId != 0 {
		userinfo["role_id"] = uRole.RoleId
	} else {
		userinfo["role_id"] = 2
	}
	userinfo["create_time"] = time.Now().Unix()
	userinfo["auth_version"] = info.TokenVersion()
	userinfo["pid"] = info.Pid

	token, _ := tools.MakeToken(userinfo)
	return token
}
