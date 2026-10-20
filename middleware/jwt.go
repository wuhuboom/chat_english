package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-fly-muti/models"
	"go-fly-muti/tools"
	"strconv"
	"time"
)

func JwtPageMiddleware(c *gin.Context) {
	//暂时不处理
	//token := c.Query("token")
	//userinfo := tools.ParseToken(token)
	//if userinfo == nil {
	//	c.Redirect(302,"/login")
	//	c.Abort()
	//}
}

func JwtApiMiddleware(c *gin.Context) {
	token := c.GetHeader("token")
	if token == "" {
		token = c.Query("token")
	}
	userinfo := tools.ParseToken(token)
	name, nameOK := userinfo["name"].(string)
	createTime, createOK := numericClaim(userinfo["create_time"])
	kefuID, idOK := numericClaim(userinfo["kefu_id"])
	roleID, roleOK := numericClaim(userinfo["role_id"])
	pid, pidOK := numericClaim(userinfo["pid"])
	authVersion, versionOK := numericClaim(userinfo["auth_version"])
	if userinfo == nil || !nameOK || name == "" || !createOK || !idOK || !roleOK || !pidOK || !versionOK {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "验证失败",
		})
		c.Abort()
		return
	}
	var expire int64 = 24 * 60 * 60
	nowTime := time.Now().Unix()
	if createTime > nowTime+60 || (nowTime-createTime) >= expire {
		c.JSON(200, gin.H{
			"code": 401,
			"msg":  "token失效",
		})
		c.Abort()
		return
	}
	currentUser := models.FindUserByUid(uint(kefuID))
	if currentUser.ID == 0 || currentUser.Name != name || currentUser.Status == 1 || authVersion <= 0 || currentUser.TokenVersion() != authVersion {
		c.JSON(200, gin.H{"code": 401, "msg": "登录状态已失效，请重新登录"})
		c.Abort()
		return
	}
	if models.GoogleAuthenticatorEnabled() && (userinfo["mfa_verified"] != true || !models.AccountTOTPBound(currentUser.ID)) {
		c.JSON(200, gin.H{"code": 401, "msg": "请重新登录并完成个人谷歌验证"})
		c.Abort()
		return
	}
	c.Set("user", name)
	c.Set("kefu_id", float64(kefuID))
	c.Set("kefu_name", name)
	c.Set("role_id", float64(roleID))
	c.Set("pid", float64(pid))
	entId := ""
	if roleID == 3 {
		entId = fmt.Sprintf("%d", pid)
	} else {
		entId = fmt.Sprintf("%d", kefuID)
	}
	c.Set("ent_id", entId)
}

func numericClaim(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), v == float64(int64(v))
	case int64:
		return v, true
	case int:
		return int64(v), true
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
