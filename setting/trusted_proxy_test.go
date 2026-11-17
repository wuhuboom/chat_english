package setting

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStaffProxyRulesReloadAfterConfigReplacement(t *testing.T) {
	original := currentStaffProxyPolicy.Load()
	t.Cleanup(func() { currentStaffProxyPolicy.Store(original) })
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(path, "app:\n  trusted_proxies: ['192.0.2.4']\n  require_trusted_proxy: true\n")
	if err := reloadStaffProxyPolicy(path); err != nil {
		t.Fatal(err)
	}
	stop, err := watchStaffProxyConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stop)

	fromA := httptest.NewRequest(http.MethodPost, "http://backend.test/check", nil)
	fromA.RemoteAddr = "192.0.2.4:1234"
	fromB := fromA.Clone(fromA.Context())
	fromB.RemoteAddr = "198.51.100.7:1234"
	fromB.Header.Set("X-Forwarded-For", "192.0.2.4")
	if !StaffProxyAllowed(fromA) || StaffProxyAllowed(fromB) {
		t.Fatal("initial trust must be based on the connection peer, not a forwarding header")
	}

	replacement := filepath.Join(dir, "new-config.yaml")
	write(replacement, "app:\n  trusted_proxies: ['198.51.100.0/24']\n  require_trusted_proxy: true\n")
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) && (StaffProxyAllowed(fromA) || !StaffProxyAllowed(fromB)) {
		time.Sleep(25 * time.Millisecond)
	}
	if StaffProxyAllowed(fromA) || !StaffProxyAllowed(fromB) {
		t.Fatal("config replacement did not reload the trust policy")
	}

	write(path, "app:\n  trusted_proxies: ['not-an-ip']\n  require_trusted_proxy: true\n")
	if err := reloadStaffProxyPolicy(path); err == nil {
		t.Fatal("invalid proxy must be rejected")
	}
	if StaffProxyAllowed(fromA) || !StaffProxyAllowed(fromB) {
		t.Fatal("invalid configuration changed the trusted peers")
	}
}
