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

func TestStaffPublicOriginBehindReverseProxy(t *testing.T) {
	t.Setenv("GOFLY_BROWSER_ORIGIN", "")
	for _, tc := range []struct {
		name, peer, proto, host, fetchSite string
		wantProxyOrigin, wantSecure        bool
	}{
		{"local proxy", "127.0.0.1:1234", "https", "support.example.com:8443", "", true, true},
		{"unlisted remote proxy", "10.20.30.4:1234", "https", "support.example.com:8443", "", false, false},
		{"untrusted client", "192.0.2.4:1234", "https", "support.example.com:8443", "", false, false},
		{"ambiguous scheme", "127.0.0.1:1234", "http, https", "support.example.com:8443", "", false, false},
		{"browser indicates HTTPS", "127.0.0.1:1234", "", "", "same-origin", false, true},
		{"browser indicates only same site", "127.0.0.1:1234", "", "", "same-site", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8081/check", nil)
			r.RemoteAddr = tc.peer
			r.Header.Set("Origin", "https://support.example.com:8443")
			if tc.proto != "" {
				r.Header.Set("X-Forwarded-Proto", tc.proto)
				r.Header.Set("X-Forwarded-Host", tc.host)
			}
			if tc.fetchSite != "" {
				r.Header.Set("Sec-Fetch-Site", tc.fetchSite)
			}
			if got := staffProxyOrigin(r); (got != "") != tc.wantProxyOrigin || (got != "" && got != "https://support.example.com:8443") {
				t.Errorf("unexpected trusted proxy origin %q", got)
			}
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = r
			SetStaffCookie(c, "fixture", StaffCookieLifetime)
			cookies := rec.Result().Cookies()
			if len(cookies) != 1 || cookies[0].Secure != tc.wantSecure {
				t.Errorf("wrong Secure flag, want %v", tc.wantSecure)
			}
		})
	}
}
