package controller

import "testing"

func TestVisitorMessageLimitKey(t *testing.T) {
	t.Parallel()

	first := visitorMessageLimitKey("visitor-a", "127.0.0.1")
	second := visitorMessageLimitKey("visitor-b", "127.0.0.1")
	if first == second {
		t.Fatalf("different visitors behind one IP must not share a rate-limit key: %q", first)
	}

	if got := visitorMessageLimitKey("visitor-a", "10.0.0.2"); got != first {
		t.Fatalf("one visitor must keep the same key across IP changes: got %q, want %q", got, first)
	}

	if got := visitorMessageLimitKey("", "127.0.0.1"); got != "sendmessage:127.0.0.1" {
		t.Fatalf("empty visitor id must fall back to client IP: got %q", got)
	}
}
