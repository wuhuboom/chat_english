package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestVisitorDiagnosticRedactsCredentialsAndBodies(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	undo := zap.ReplaceGlobals(zap.New(core))
	defer undo()
	engine := gin.New()
	engine.GET("/2/messages_page", VisitorDiagnostic, func(c *gin.Context) {
		c.Set("visitor_diagnostic_reason", "access_denied")
		c.AbortWithStatusJSON(404, gin.H{"content": "private-chat-marker"})
	})
	req := httptest.NewRequest("GET", "/2/messages_page?visitor_id=private-id&visitor_name=private-name&token=query-secret", nil)
	req.RemoteAddr = "192.0.2.5:12345"
	req.Header.Set("Origin", "null")
	req.Header.Set("User-Agent", "LegacyAPK/1")
	req.Header.Set("Cookie", "session=cookie-secret")
	req.Header.Set("X-Gofly-Visitor-Token", "header-secret")
	req.Header.Set("Sec-WebSocket-Protocol", "protocol-secret")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != 404 || logs.Len() != 2 {
		t.Fatalf("response=%d logs=%d", recorder.Code, logs.Len())
	}
	for _, entry := range logs.All() {
		encoded, _ := json.Marshal(entry.ContextMap())
		for _, secret := range []string{"private-id", "private-name", "query-secret", "cookie-secret", "header-secret", "protocol-secret", "private-chat-marker"} {
			if strings.Contains(string(encoded), secret) {
				t.Fatalf("diagnostic leaked %s", secret)
			}
		}
	}
	last := logs.All()[1].ContextMap()
	if last["reason"] != "access_denied" || last["origin"] != "null" || last["user_agent"] != "LegacyAPK/1" || last["ip"] != "192.0.2.5" {
		t.Fatalf("missing diagnostic metadata: %v", last)
	}
}
