package middleware

import (
	"crypto/hmac"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go-fly-muti/setting"
	"go-fly-muti/tools"
)

const StaffCookie = "gofly_staff"
const StaffCookieLifetime = 24 * 60 * 60
const StaffWebSocketProtocol = "gofly-staff-v1"

func StaffSocketProtocol(c *gin.Context) string {
	raw, _ := c.Get("staff_claims")
	claims, ok := raw.(map[string]interface{})
	if !ok || !c.GetBool("staff_cookie") {
		return ""
	}
	id, idOK := numericClaim(claims["kefu_id"])
	start, startOK := numericClaim(claims["session_started"])
	version, versionOK := numericClaim(claims["auth_version"])
	if !idOK || !startOK || !versionOK {
		return ""
	}
	return "gofly-staff." + tools.StaffSocketProof(id, start, version)
}

func StaffWebSocketAuth(c *gin.Context) {
	staffJWTAuth(c, false)
	if c.IsAborted() || !c.GetBool("staff_cookie") {
		c.Abort()
		return
	}
	expected := StaffSocketProtocol(c)
	var verified, offersPublicProtocol bool
	for _, header := range c.Request.Header.Values("Sec-WebSocket-Protocol") {
		for _, protocol := range strings.Split(header, ",") {
			protocol = strings.TrimSpace(protocol)
			offersPublicProtocol = offersPublicProtocol || protocol == StaffWebSocketProtocol
			verified = verified || (expected != "" && hmac.Equal([]byte(protocol), []byte(expected)))
		}
	}
	if verified {
		if offersPublicProtocol {
			c.Set("staff_websocket_protocol", StaffWebSocketProtocol)
		} else {
			// Version 9 only offers its verified session-specific protocol.
			c.Set("staff_websocket_protocol", expected)
		}
		return
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "请刷新客服后台后重新连接"})
}

func staffOrigin(r *http.Request) string {
	if origin := staffProxyOrigin(r); origin != "" {
		return origin
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Behind a reverse proxy, configure the public origin explicitly.
	if public, err := url.Parse(os.Getenv("GOFLY_BROWSER_ORIGIN")); r.TLS == nil && err == nil &&
		(public.Scheme == "https" || public.Scheme == "http") && public.User == nil &&
		strings.EqualFold(public.Host, r.Host) && public.Path == "" && public.RawQuery == "" && public.Fragment == "" {
		scheme = public.Scheme
	}
	return scheme + "://" + r.Host
}

// Only the directly connected, trusted proxy can supply the public origin.
// The proxy must overwrite these headers on every request.
func staffProxyOrigin(r *http.Request) string {
	if !setting.IsTrustedProxy(r.RemoteAddr) {
		return ""
	}

	schemes := r.Header.Values("X-Forwarded-Proto")
	hosts := r.Header.Values("X-Forwarded-Host")
	if len(schemes) != 1 || len(hosts) > 1 {
		return ""
	}
	scheme := strings.ToLower(strings.TrimSpace(schemes[0]))
	if scheme != "https" && scheme != "http" {
		return ""
	}
	host := r.Host
	if len(hosts) == 1 {
		host = strings.TrimSpace(hosts[0])
	}
	if strings.ContainsAny(host, " \t\r\n,\\/#?@") {
		return ""
	}
	u, err := url.Parse(scheme + "://" + host)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	if (scheme == "https" && u.Port() == "443") || (scheme == "http" && u.Port() == "80") {
		host = strings.TrimSuffix(host, ":"+u.Port())
	}
	return scheme + "://" + host
}

const StaffLoginHeader = "X-Gofly-Login"

func requireStaffProxy(c *gin.Context) bool {
	if setting.StaffProxyAllowed(c.Request) {
		return true
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "请求来源不受信任"})
	return false
}

// Keep this header out of the CORS allowlist. Browsers can only attach it
// without preflight when posting from a page on the same origin.
func RequireStaffLoginHeader(c *gin.Context) {
	if !requireStaffProxy(c) {
		return
	}
	if c.GetHeader(StaffLoginHeader) != "1" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "请刷新登录页面后重试"})
	}
}

func SetStaffCookie(c *gin.Context, token string, age int) {
	secure := c.Request.TLS != nil || strings.HasPrefix(staffOrigin(c.Request), "https://")
	if !secure && c.GetHeader("Sec-Fetch-Site") == "same-origin" {
		origin, err := url.Parse(c.GetHeader("Origin"))
		secure = err == nil && origin.Scheme == "https" && origin.Hostname() != ""
	}
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(StaffCookie, token, age, "/", "", secure, true)
	c.Header("Cache-Control", "no-store")
}
