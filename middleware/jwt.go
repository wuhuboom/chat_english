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
	c.Set("staff_page", true)
	staffJWTAuth(c, false)
}

func staffAuthError(c *gin.Context, code int, message string) {
	if c.GetBool("staff_page") {
		c.Redirect(302, "/login")
		c.Abort()
		return
	}
	c.AbortWithStatusJSON(200, gin.H{"code": code, "msg": message})
}

func JwtApiMiddleware(c *gin.Context) {
	staffJWTAuth(c, true)
}

func StaffSessionRefreshAuth(c *gin.Context) {
	RequireStaffLoginHeader(c)
	if !c.IsAborted() {
		staffJWTAuth(c, false)
	}
}

// StaffResourceAuth is for read-only resources loaded by plain HTML elements.
func StaffResourceAuth(c *gin.Context) {
	staffJWTAuth(c, false)
}

func staffJWTAuth(c *gin.Context, requireBrowserHeader bool) {
	token, _ := c.Cookie(StaffCookie)
	fromCookie := token != ""
	if fromCookie {
		// Cookie sessions are local to the dashboard. In particular, a
		// cross-origin page must not read the rendered WebSocket proof.
		c.Writer.Header().Del("Access-Control-Allow-Credentials")
		if requireBrowserHeader {
			RequireStaffLoginHeader(c)
			if c.IsAborted() {
				return
			}
		}
	} else if !c.GetBool("staff_page") && c.FullPath() != "/ws_kefu" {
		token = c.GetHeader("token")
	}
	userinfo := tools.ParseToken(token)
	name, nameOK := userinfo["name"].(string)
	createTime, createOK := numericClaim(userinfo["create_time"])
	kefuID, idOK := numericClaim(userinfo["kefu_id"])
	roleID, roleOK := numericClaim(userinfo["role_id"])
	pid, pidOK := numericClaim(userinfo["pid"])
	authVersion, versionOK := numericClaim(userinfo["auth_version"])
	if userinfo == nil || (userinfo["staff_cookie"] == true) != fromCookie || !nameOK || name == "" || !createOK || !idOK || !roleOK || !pidOK || !versionOK {
		staffAuthError(c, 401, "请重新登录")
		return
	}
	nowTime := time.Now().Unix()
	if createTime > nowTime+60 || nowTime-createTime >= StaffCookieLifetime {
		staffAuthError(c, 401, "登录已过期")
		return
	}
	if fromCookie {
		started, ok := numericClaim(userinfo["session_started"])
		if !ok || started > nowTime+60 || nowTime-started >= 7*StaffCookieLifetime {
			staffAuthError(c, 401, "登录已过期")
			return
		}
	}
	currentUser := models.FindUserByUid(uint(kefuID))
	if currentUser.ID == 0 || currentUser.Name != name || currentUser.Status == 1 || authVersion <= 0 || currentUser.TokenVersion() != authVersion {
		staffAuthError(c, 401, "登录状态已失效，请重新登录")
		return
	}
	if models.GoogleAuthenticatorEnabled() && (userinfo["mfa_verified"] != true || !models.AccountTOTPBound(currentUser.ID)) {
		staffAuthError(c, 401, "请重新登录并完成个人谷歌验证")
		return
	}
	c.Set("staff_cookie", fromCookie)
	c.Set("staff_claims", userinfo)
	c.Set("staff_auth_version", authVersion)
	c.Set("staff_mfa", userinfo["mfa_verified"] == true)
	c.Set("staff_expires", createTime+StaffCookieLifetime)
	c.Header("Cache-Control", "no-store")
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
