package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCrossSiteWildcardDoesNotAllowCredentials(t *testing.T) {
	t.Setenv("GOFLY_ALLOWED_ORIGINS", "")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	context.Request.Header.Set("Origin", "https://example.com")

	CrossSite(context)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow origin = %q, want *", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("credentials header = %q, want empty", got)
	}
}

func TestCrossSiteAllowsPackagedChatHeaders(t *testing.T) {
	t.Setenv("GOFLY_ALLOWED_ORIGINS", "")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodOptions, "/2/messages_page", nil)
	context.Request.Header.Set("Origin", "null")
	context.Request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	context.Request.Header.Set("Access-Control-Request-Headers", "lang,x-requested-with,content-type")

	CrossSite(context)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow origin = %q, want *", got)
	}
	allowedHeaders := strings.ToLower(recorder.Header().Get("Access-Control-Allow-Headers"))
	for _, header := range []string{"lang", "x-requested-with", "content-type"} {
		if !strings.Contains(allowedHeaders, header) {
			t.Errorf("allowed headers %q does not contain %q", allowedHeaders, header)
		}
	}
}

func TestCrossSiteConfiguredOriginAndPreflight(t *testing.T) {
	t.Setenv("GOFLY_ALLOWED_ORIGINS", "https://support.example.com")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodOptions, "/", nil)
	context.Request.Header.Set("Origin", "https://support.example.com")

	CrossSite(context)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://support.example.com" {
		t.Fatalf("allow origin = %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("credentials header = %q, want true", got)
	}
	if !context.IsAborted() {
		t.Fatal("preflight request was not aborted")
	}
}
