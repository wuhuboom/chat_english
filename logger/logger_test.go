package logger

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeQuery(t *testing.T) {
	query := url.Values{
		"keyword":      {"visitor"},
		"token":        {"secret-token"},
		"access_token": {"another-secret"},
		"password":     {"123456"},
	}
	sanitized := sanitizeQuery(query)
	if strings.Contains(sanitized, "secret-token") ||
		strings.Contains(sanitized, "another-secret") ||
		strings.Contains(sanitized, "123456") {
		t.Fatalf("sensitive value leaked: %s", sanitized)
	}
	if !strings.Contains(sanitized, "keyword=visitor") {
		t.Fatalf("regular query value missing: %s", sanitized)
	}
}

func TestResolveRequestID(t *testing.T) {
	if got := resolveRequestID("client-request_123"); got != "client-request_123" {
		t.Fatalf("valid request ID changed: %s", got)
	}
	if got := resolveRequestID("bad request id"); got == "bad request id" || len(got) < 8 {
		t.Fatalf("invalid request ID was not replaced: %s", got)
	}
}

func TestParseEntry(t *testing.T) {
	raw := `{"time":"2026-07-31T01:00:00+08:00","level":"warn","message":"GET /test","component":"http","request_id":"request-1","status":429,"duration":12.5}`
	entry := parseEntry(raw)
	if entry.Level != "warn" || entry.Status != 429 || entry.RequestID != "request-1" {
		t.Fatalf("unexpected parsed entry: %#v", entry)
	}
	if entry.Duration != "12.50ms" {
		t.Fatalf("unexpected duration: %s", entry.Duration)
	}
}

func TestTailLinesDropsPartialFirstLine(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "app.log")
	content := strings.Join([]string{
		`{"level":"info","message":"first"}`,
		`{"level":"warn","message":"second"}`,
		`{"level":"error","message":"third"}`,
	}, "\n")
	if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	lines, truncated, err := tailLines(filename, int64(len(content)/2))
	if err != nil {
		t.Fatal(err)
	}
	if !truncated {
		t.Fatal("expected truncated tail")
	}
	if len(lines) == 0 || !strings.Contains(lines[len(lines)-1], "third") {
		t.Fatalf("latest complete line missing: %#v", lines)
	}
}
