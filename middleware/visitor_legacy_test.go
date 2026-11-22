package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestLegacyVisitorGrantBindingAndExpiry(t *testing.T) {
	store := newLegacyVisitorStore(2)
	now := time.Now()
	context := func(ip, ua string) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.RemoteAddr = ip + ":1234"
		c.Request.Header.Set("User-Agent", ua)
		c.Request.Header.Set("Origin", "null")
		return c
	}
	a := context("192.0.2.1", "legacy-browser")
	if store.allowed(a, "2|alice", now) {
		t.Fatal("unverified access")
	}
	store.issue(a, "2|alice", now)
	if !store.allowed(a, "2|alice", now.Add(time.Minute)) {
		t.Fatal("verified access denied")
	}
	for _, other := range []*gin.Context{context("192.0.2.2", "legacy-browser"), context("192.0.2.1", "other-browser")} {
		if store.allowed(other, "2|alice", now) {
			t.Fatal("grant crossed IP or user agent")
		}
	}
	if store.allowed(a, "3|alice", now) || store.allowed(a, "2|other", now) {
		t.Fatal("grant crossed visitor identity")
	}
	if store.allowed(a, "2|alice", now.Add(12*time.Hour)) {
		t.Fatal("expired grant accepted")
	}
	store.issue(a, "2|alice", now)
	store.issue(a, "2|bob", now)
	store.issue(a, "2|carol", now)
	if len(store.grants) > 2 {
		t.Fatal("unbounded cache")
	}
	if store.allowed(context("192.0.2.1", ""), "2|alice", now) {
		t.Fatal("empty UA accepted")
	}
}
