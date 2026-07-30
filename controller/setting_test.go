package controller

import "testing"

func TestParseConversationSLASettings(t *testing.T) {
	tests := []struct {
		name        string
		warning     string
		overdue     string
		wantError   bool
		wantWarning int
		wantOverdue int
	}{
		{name: "valid", warning: "5", overdue: "15", wantWarning: 5, wantOverdue: 15},
		{name: "trim spaces", warning: " 10 ", overdue: " 30 ", wantWarning: 10, wantOverdue: 30},
		{name: "warning not integer", warning: "five", overdue: "15", wantError: true},
		{name: "warning zero", warning: "0", overdue: "15", wantError: true},
		{name: "overdue not greater", warning: "15", overdue: "15", wantError: true},
		{name: "overdue too large", warning: "15", overdue: "1441", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			warning, overdue, err := parseConversationSLASettings(test.warning, test.overdue)
			if test.wantError {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if warning != test.wantWarning || overdue != test.wantOverdue {
				t.Fatalf("got (%d, %d), want (%d, %d)", warning, overdue, test.wantWarning, test.wantOverdue)
			}
		})
	}
}

func TestParseRoutingSettings(t *testing.T) {
	tests := []struct {
		name      string
		enabled   string
		grace     string
		wantError bool
	}{
		{name: "enabled", enabled: "true", grace: "15"},
		{name: "disabled", enabled: "false", grace: "300"},
		{name: "trim spaces", enabled: " true ", grace: " 10 "},
		{name: "invalid boolean", enabled: "yes", grace: "15", wantError: true},
		{name: "too short", enabled: "true", grace: "2", wantError: true},
		{name: "too long", enabled: "true", grace: "301", wantError: true},
		{name: "not integer", enabled: "true", grace: "soon", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := parseRoutingSettings(test.enabled, test.grace)
			if test.wantError && err == nil {
				t.Fatal("expected an error")
			}
			if !test.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
