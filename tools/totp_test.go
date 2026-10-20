package tools

import (
	"testing"
	"time"
)

func TestValidateTOTPRFCVector(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	now := time.Unix(59, 0)
	if !ValidateTOTP(secret, "287082", now) {
		t.Fatal("expected RFC TOTP vector to validate")
	}
	if ValidateTOTP(secret, "287083", now) {
		t.Fatal("unexpected invalid TOTP code to validate")
	}
}

func TestGenerateTOTPSecret(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("generate secret: %v", err)
	}
	if len(secret) != 32 {
		t.Fatalf("secret length = %d, want 32", len(secret))
	}
}
