package middleware

import (
	"net/http"
	"net/http/httptest"
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
