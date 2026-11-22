package middleware

import (
	"crypto/sha256"
	"encoding/json"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go-fly-muti/setting"
)

// Legacy file clients cannot send cookies or tokens. This is deliberately a
// weaker, bounded compatibility grant, not proof of a unique device.
type legacyVisitorStore struct {
	mu     sync.Mutex
	grants map[[32]byte]time.Time
	limit  int
}

func newLegacyVisitorStore(limit int) *legacyVisitorStore {
	return &legacyVisitorStore{grants: make(map[[32]byte]time.Time), limit: limit}
}

var legacyVisitors = newLegacyVisitorStore(50000)

func legacyVisitorKey(c *gin.Context, id string) ([32]byte, bool) {
	ip := setting.VisitorClientIP(c.Request)
	ua := c.Request.UserAgent()
	reason := ""
	switch {
	case id == "" || len(id) > 512:
		reason = "invalid_visitor_id"
	case !ip.IsValid():
		reason = "invalid_client_ip"
	case ua == "" || len(ua) > 2048:
		reason = "invalid_user_agent"
	case c.GetHeader("Origin") != "" && c.GetHeader("Origin") != "null":
		reason = "origin_not_legacy"
	}
	if reason != "" {
		c.Set("visitor_diagnostic_legacy", reason)
		return [32]byte{}, false
	}
	verifyURL := ""
	if u := setting.CustomerVerifyURL(); u != nil {
		verifyURL = u.String()
	}
	encoded, _ := json.Marshal([]string{id, ip.String(), ua, verifyURL})
	return sha256.Sum256(encoded), true
}

func (s *legacyVisitorStore) issue(c *gin.Context, id string, now time.Time) {
	key, ok := legacyVisitorKey(c, id)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.grants) >= s.limit {
		var oldest [32]byte
		var earliest time.Time
		for k, expiry := range s.grants {
			if !now.Before(expiry) {
				delete(s.grants, k)
				continue
			}
			if earliest.IsZero() || expiry.Before(earliest) {
				oldest, earliest = k, expiry
			}
		}
		if len(s.grants) >= s.limit {
			delete(s.grants, oldest)
		}
	}
	s.grants[key] = now.Add(12 * time.Hour)
	c.Set("visitor_diagnostic_legacy", "issued")
}

func (s *legacyVisitorStore) allowed(c *gin.Context, id string, now time.Time) bool {
	key, ok := legacyVisitorKey(c, id)
	if !ok {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry, found := s.grants[key]
	if found && !now.Before(expiry) {
		delete(s.grants, key)
		c.Set("visitor_diagnostic_legacy", "expired")
		return false
	}
	if found {
		c.Set("visitor_diagnostic_legacy", "matched")
	} else {
		c.Set("visitor_diagnostic_legacy", "not_found_for_ip_ua_id")
	}
	return found
}
