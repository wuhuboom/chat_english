package setting

import (
	"testing"
	"time"
)

func TestConfigureTimezone(t *testing.T) {
	originalLocation := Location()
	t.Cleanup(func() {
		currentLocation.Store(originalLocation)
	})

	if err := ConfigureTimezone("America/New_York"); err != nil {
		t.Fatalf("configure valid timezone: %v", err)
	}
	if CurrentTimezone() != "America/New_York" {
		t.Fatalf("timezone = %q", CurrentTimezone())
	}
}

func TestConfigureTimezoneDefaultsToShanghai(t *testing.T) {
	originalLocation := Location()
	t.Cleanup(func() {
		currentLocation.Store(originalLocation)
	})

	if err := ConfigureTimezone(""); err != nil {
		t.Fatalf("configure default timezone: %v", err)
	}
	if CurrentTimezone() != DefaultTimezone {
		t.Fatalf("timezone = %q, want %q", CurrentTimezone(), DefaultTimezone)
	}
}

func TestConfigureTimezoneRejectsInvalidName(t *testing.T) {
	if err := ConfigureTimezone("Mars/Olympus"); err == nil {
		t.Fatal("expected invalid timezone error")
	}
}

func TestFormatUsesConfiguredTimezone(t *testing.T) {
	originalLocation := Location()
	t.Cleanup(func() {
		currentLocation.Store(originalLocation)
	})
	if err := ConfigureTimezone("America/New_York"); err != nil {
		t.Fatal(err)
	}
	value := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	if got := Format(value); got != "2026-07-29 08:00:00" {
		t.Fatalf("Format() = %q", got)
	}
}
