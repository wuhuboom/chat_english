package v2

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"go-fly-muti/models"
	"go-fly-muti/tools"
	"go-fly-muti/types"
	"strconv"
	"strings"
	"time"
)

type LoginForm struct {
	Username   string `form:"username" json:"username" uri:"username" xml:"username" binding:"required"`
	Password   string `form:"password" json:"password" uri:"password" xml:"password" binding:"required"`
	GoogleCode string `form:"google_code" json:"google_code" uri:"google_code" xml:"google_code"`
}

func PostUcLogin(c *gin.Context) {
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
	if !tools.SecurityLimitFreqSingle("login-v2:"+c.ClientIP()+":"+strings.ToLower(form.Username), 10, 300) {
		c.JSON(200, gin.H{"code": 429, "msg": "登录尝试过多，请5分钟后重试"})
		return
	}
	c.JSON(200, UcLogin(form))
}
func UcLogin(form LoginForm) gin.H {
	var user *models.User
	user = &models.User{
		Name: form.Username,
	}
	*user = user.GetOneUser("*")
	md5Pass := tools.Md5(form.Password)
	if user.ID == 0 || user.Password != md5Pass {
		return gin.H{
			"code": types.ApiCode.LOGIN_FAILED,
			"msg":  types.ApiCode.GetMessage(types.ApiCode.LOGIN_FAILED),
		}
	}
	ok, errCode, errMsg := user.CheckStatusExpired()
	if !ok {
		return gin.H{
			"code": errCode,
			"msg":  errMsg,
		}
	}
	if binding, err := models.CheckAccountTOTP(*user, form.GoogleCode); err != nil {
		return gin.H{"code": types.ApiCode.GOOGLE_AUTH_FAILED, "msg": "谷歌验证失败，请检查动态码后重试"}
	} else if binding != nil {
		return gin.H{"code": 428, "msg": "请绑定此账号自己的谷歌验证器", "result": binding}
	}
	roleId, _ := strconv.Atoi(user.RoleId)
	tokenCliams := tools.UserClaims{
		Id:          user.ID,
		MFAVerified: models.GoogleAuthenticatorEnabled(),
		Username:    user.Name,
		RoleId:      uint(roleId),
		Pid:         user.Pid,
		RoleName:    user.RoleName,
		CreateTime:  time.Now().Format("2006-01-02 15:04:05"),
		AuthVersion: user.TokenVersion(),
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Unix() + 24*3600,
		},
	}
	token, err := tools.MakeCliamsToken(tokenCliams)
	if err != nil {
		return gin.H{
			"code": types.ApiCode.LOGIN_FAILED,
			"msg":  err.Error(),
		}
	}
	return gin.H{
		"code": types.ApiCode.SUCCESS,
		"msg":  types.ApiCode.GetMessage(types.ApiCode.SUCCESS),
		"result": gin.H{
			"token": token,
		},
	}
}
