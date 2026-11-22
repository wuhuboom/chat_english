package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-fly-muti/models"
	"go-fly-muti/setting"
	"go-fly-muti/tools"
)

func visitorAccessCookie(id string) string {
	sum := sha256.Sum256([]byte(id))
	return "gofly_verified_visitor_" + hex.EncodeToString(sum[:16])
}

func VisitorPageAccess(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if !VisitorLinkIPAllowed(c, c.Query("ent_id"), c.Query("kefu_id")) || !models.VisitorVerificationRequired() {
		return
	}
	query := c.Request.URL.Query()
	if len(query["visitor_id"]) != 1 || len(query["visitor_name"]) != 1 ||
		!VerifyVisitor(c, query.Get("visitor_id"), query.Get("visitor_name")) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Set("verified_visitor_entry", true)
}

func VerifyVisitor(c *gin.Context, id, name string) bool {
	if !setting.VerifyCustomer(c.Request, id, name) {
		c.Set("visitor_diagnostic_reason", "customer_verify_failed")
		RejectVisitorAccess(c)
		return false
	}
	c.Set("visitor_diagnostic_upstream", "verified")
	return true
}

func RejectVisitorAccess(c *gin.Context) {
	if c.GetString("visitor_diagnostic_reason") == "" {
		c.Set("visitor_diagnostic_reason", "visitor_access_denied")
	}
	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": 404, "msg": "Not Found"})
}

// IssueVisitorAccess is called only after a verified visitor login succeeds.
func IssueVisitorAccess(c *gin.Context, id string) {
	c.Set("visitor_diagnostic_id", id)
	// Issued only at the end of a successful upstream-verified login. Old APKs
	// omit Origin on XHR or send Origin:null, and cannot retain cookies.
	if c.PostForm("access_mode") == "" {
		legacyVisitors.issue(c, id, time.Now())
		VisitorDiagnosticEvent(c, "grant", c.GetString("visitor_diagnostic_legacy"))
	}
	const age = 12 * time.Hour
	token, err := tools.MakeToken(map[string]interface{}{
		"purpose": "visitor-access", "visitor_id": id, "exp": time.Now().Add(age).Unix(),
	})
	if err == nil {
		c.Header("Cache-Control", "no-store")
		if c.PostForm("access_mode") == "token" {
			portable, err := tools.MakeToken(map[string]interface{}{
				"purpose": "visitor-portable", "visitor_id": id,
				"exp": time.Now().Add(time.Hour).Unix(),
			})
			if err == nil {
				c.Set("visitor_access_token", portable)
			}
		}
		http.SetCookie(c.Writer, &http.Cookie{
			Name: visitorAccessCookie(id), Value: token, Path: "/", MaxAge: int(age.Seconds()),
			HttpOnly: true, Secure: c.Request.TLS != nil ||
				(setting.IsTrustedProxy(c.Request.RemoteAddr) && c.GetHeader("X-Forwarded-Proto") == "https"),
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func VisitorSessionAccess(param string, fromQuery bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var id, entID string
		if fromQuery {
			id = c.Query(param)
			entID = c.Query("ent_id")
		} else {
			id = c.PostForm(param)
			entID = c.PostForm("ent_id")
		}
		if CheckVisitorAccess(c, id) {
			VisitorConversationIPAllowed(c, entID, id)
		}
	}
}

func CheckVisitorAccess(c *gin.Context, id string) bool {
	if !models.VisitorVerificationRequired() {
		c.Set("visitor_diagnostic_auth", "verification_disabled")
		return true
	}
	// Portable clients cannot rely on third-party cookies. Never accept their
	// credentials from URL query parameters (which are logged and shared).
	portable := c.GetHeader("X-Gofly-Visitor-Token")
	if c.Request.URL.Path == "/ws_visitor" {
		for _, offered := range strings.Split(c.GetHeader("Sec-WebSocket-Protocol"), ",") {
			if strings.HasPrefix(strings.TrimSpace(offered), "gofly-visitor.") {
				portable = strings.TrimPrefix(strings.TrimSpace(offered), "gofly-visitor.")
				break
			}
		}
	}
	c.Set("visitor_diagnostic_portable_present", portable != "")
	if portable != "" && id != "" {
		claims := tools.ParseToken(portable)
		if claims["purpose"] == "visitor-portable" && claims["visitor_id"] == id {
			c.Set("verified_visitor_session", true)
			c.Set("portable_visitor_session", true)
			c.Set("visitor_diagnostic_auth", "portable_token")
			return true
		}
	}
	value, _ := c.Cookie(visitorAccessCookie(id))
	c.Set("visitor_diagnostic_cookie_present", value != "")
	if id != "" && value != "" {
		claims := tools.ParseToken(value)
		if claims["purpose"] == "visitor-access" && claims["visitor_id"] == id {
			c.Set("verified_visitor_session", true)
			c.Set("visitor_diagnostic_auth", "visitor_cookie")
			return true
		}
	}
	if portable == "" && value == "" && legacyVisitors.allowed(c, id, time.Now()) {
		c.Set("verified_visitor_session", true)
		c.Set("legacy_visitor_session", true)
		c.Set("visitor_diagnostic_auth", "legacy_grant")
		return true
	}
	c.Set("visitor_diagnostic_reason", "no_valid_visitor_session")
	RejectVisitorAccess(c)
	return false
}

func OrdinaryVisitorOnly(c *gin.Context) {
	if models.VisitorVerificationRequired() {
		c.AbortWithStatus(http.StatusNotFound)
	}
}
