package setting

import "testing"

func TestConfigDestinations(t *testing.T) {
	for _, value := range []string{"/login", "https://example.invalid/login?lang=en"} {
		if !SafeIndexDestination(value) {
			t.Errorf("valid index URL rejected: %q", value)
		}
	}
	for _, value := range []string{"//example.invalid", "mailto:example", "https://user@example.invalid", "/\\example.invalid", "/a\nb"} {
		if SafeIndexDestination(value) {
			t.Errorf("unsafe index URL accepted: %q", value)
		}
	}
	for _, value := range []string{"example.invalid", "example.invalid:8443", "[::1]:8081"} {
		if !SafeLandingHost(value) {
			t.Errorf("valid landing host rejected: %q", value)
		}
	}
	for _, value := range []string{"https://example.invalid", "user@example.invalid", "example.invalid/query", "example.invalid?x=y"} {
		if SafeLandingHost(value) {
			t.Errorf("unsafe landing host accepted: %q", value)
		}
	}
}
