package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
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
		RejectVisitorAccess(c)
		return false
	}
	return true
}

func RejectVisitorAccess(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": 404, "msg": "Not Found"})
}

// IssueVisitorAccess is called only after a verified visitor login succeeds.
func IssueVisitorAccess(c *gin.Context, id string) {
	const age = 12 * time.Hour
	token, err := tools.MakeToken(map[string]interface{}{
		"purpose": "visitor-access", "visitor_id": id, "exp": time.Now().Add(age).Unix(),
	})
	if err == nil {
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
		return true
	}
	value, _ := c.Cookie(visitorAccessCookie(id))
	if id != "" && value != "" {
		claims := tools.ParseToken(value)
		if claims["purpose"] == "visitor-access" && claims["visitor_id"] == id {
			c.Set("verified_visitor_session", true)
			return true
		}
	}
	RejectVisitorAccess(c)
	return false
}

func OrdinaryVisitorOnly(c *gin.Context) {
	if models.VisitorVerificationRequired() {
		c.AbortWithStatus(http.StatusNotFound)
	}
}
