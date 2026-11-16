package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStaffCookieTransport(t *testing.T) {
	t.Setenv("GOFLY_BROWSER_ORIGIN", "")
	for _, tc := range []struct {
		url    string
		secure bool
	}{
		{"http://192.0.2.10:8081/check", false},
		{"https://192.0.2.10:8081/check", true},
		{"http://127.0.0.1:8081/check", false},
	} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("POST", tc.url, nil)
		SetStaffCookie(c, "fixture", StaffCookieLifetime)
		cookies := rec.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Secure != tc.secure ||
			!cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode ||
			cookies[0].Domain != "" {
			t.Errorf("unexpected cookie policy for %s", tc.url)
		}
	}
}
