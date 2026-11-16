package middleware

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const StaffCookie = "gofly_staff"
const StaffCookieLifetime = 24 * 60 * 60

func IsStaffOrigin(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Origin"), staffOrigin(r))
}

func staffOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Behind a reverse proxy, configure the public origin explicitly.
	if public, err := url.Parse(os.Getenv("GOFLY_BROWSER_ORIGIN")); err == nil &&
		(public.Scheme == "https" || public.Scheme == "http") && public.User == nil &&
		strings.EqualFold(public.Host, r.Host) && public.Path == "" && public.RawQuery == "" && public.Fragment == "" {
		scheme = public.Scheme
	}
	return scheme + "://" + r.Host
}

func SetStaffCookie(c *gin.Context, token string, age int) {
	host := c.Request.Host
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	loopback := strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback())
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(StaffCookie, token, age, "/", "", !loopback || strings.HasPrefix(staffOrigin(c.Request), "https://"), true)
	c.Header("Cache-Control", "no-store")
}

// Cookie-authenticated requests stay on the exact origin, including WS upgrades.
func StaffSameOrigin(c *gin.Context) {
	origin := c.GetHeader("Origin")
	site := c.GetHeader("Sec-Fetch-Site")
	ws := strings.EqualFold(c.GetHeader("Upgrade"), "websocket")
	if (origin != "" && !strings.EqualFold(origin, staffOrigin(c.Request))) ||
		(site != "" && site != "same-origin" && site != "none") || (ws && origin == "") {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "请求来源不受信任"})
	}
}
