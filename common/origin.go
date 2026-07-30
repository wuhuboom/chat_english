package common

import (
	"os"
	"strings"
)

// AllowedOrigin returns the CORS origin value and whether credentialed
// cross-origin requests are allowed. An empty GOFLY_ALLOWED_ORIGINS preserves
// public visitor embedding while avoiding the invalid "* + credentials" pair.
func AllowedOrigin(origin string) (string, bool) {
	configured := strings.TrimSpace(os.Getenv("GOFLY_ALLOWED_ORIGINS"))
	if configured == "" || configured == "*" {
		return "*", false
	}
	for _, allowed := range strings.Split(configured, ",") {
		if strings.TrimSpace(allowed) == origin && origin != "" {
			return origin, true
		}
	}
	return "", false
}

func IsWebSocketOriginAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	allowedOrigin, _ := AllowedOrigin(origin)
	return allowedOrigin != ""
}
