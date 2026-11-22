package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go-fly-muti/setting"
	"go.uber.org/zap"
)

func visitorDiagnosticHash(value string) string {
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

// VisitorDiagnostic observes only route metadata. Never capture request bodies,
// response bodies, query strings, cookies or WebSocket protocol headers here.
func VisitorDiagnostic(c *gin.Context) {
	start := time.Now()
	VisitorDiagnosticEvent(c, "request", "started")
	c.Next()
	reason := c.GetString("visitor_diagnostic_reason")
	if reason == "" {
		switch {
		case c.GetBool("visitor_diagnostic_login_ok"):
			reason = "login_succeeded"
		case c.Writer.Status() >= 400:
			reason = "http_rejected"
		case c.Request.URL.Path == "/visitor_login":
			reason = "login_handler_rejected"
		default:
			reason = "completed"
		}
	}
	c.Set("visitor_diagnostic_duration_ms", time.Since(start).Milliseconds())
	VisitorDiagnosticEvent(c, "complete", reason)
}

func VisitorDiagnosticEvent(c *gin.Context, stage, reason string) {
	status := c.Writer.Status()
	if c.GetBool("visitor_diagnostic_ws_connected") {
		status = 101
	}
	id := c.GetString("visitor_diagnostic_id")
	if id == "" {
		id = c.Query("visitor_id")
	}
	origin := c.GetHeader("Origin")
	if len(origin) > 256 {
		origin = origin[:256] + "[truncated]"
	}
	ua := c.Request.UserAgent()
	if len(ua) > 512 {
		ua = ua[:512] + "[truncated]"
	}
	zap.L().Info("visitor access diagnostic",
		zap.String("component", "visitor_diagnostic"),
		zap.String("stage", stage), zap.String("reason", reason),
		zap.String("request_id", c.GetString("request_id")),
		zap.Int("pid", os.Getpid()), zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method), zap.Int("http_status", status),
		zap.String("ip", setting.VisitorClientIP(c.Request).String()),
		zap.String("peer", c.Request.RemoteAddr),
		zap.String("origin", origin), zap.Bool("origin_present", origin != ""),
		zap.String("user_agent", ua), zap.String("ua_hash", visitorDiagnosticHash(c.Request.UserAgent())),
		zap.String("visitor_hash", visitorDiagnosticHash(id)),
		zap.String("auth_mode", c.GetString("visitor_diagnostic_auth")),
		zap.String("legacy_grant", c.GetString("visitor_diagnostic_legacy")),
		zap.String("upstream_verify", c.GetString("visitor_diagnostic_upstream")),
		zap.Bool("portable_present", c.GetBool("visitor_diagnostic_portable_present")),
		zap.Bool("visitor_cookie_present", c.GetBool("visitor_diagnostic_cookie_present")),
		zap.Int("record_count", c.GetInt("visitor_diagnostic_record_count")),
		zap.Int64("duration_ms", c.GetInt64("visitor_diagnostic_duration_ms")),
	)
}
