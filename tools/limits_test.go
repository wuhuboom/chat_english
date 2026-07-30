package tools

import (
	"testing"

	"github.com/spf13/viper"
)

func TestLimitFreqSingleDisabled(t *testing.T) {
	viper.Set("rate_limit.enabled", false)
	t.Cleanup(viper.Reset)
	LimitQueue.reset()

	for i := 0; i < 100; i++ {
		if !LimitFreqSingle("same-client", 1, 60) {
			t.Fatalf("disabled limiter rejected request %d", i+1)
		}
	}
}

func TestLimitQueueEnabled(t *testing.T) {
	viper.Set("rate_limit.enabled", true)
	t.Cleanup(viper.Reset)
	LimitQueue.reset()

	if !LimitQueue.allow("same-client", 2, 10, 100) {
		t.Fatal("first request should pass")
	}
	if !LimitQueue.allow("same-client", 2, 10, 101) {
		t.Fatal("second request should pass")
	}
	if LimitQueue.allow("same-client", 2, 10, 102) {
		t.Fatal("third request inside the window should be rejected")
	}
	if !LimitQueue.allow("same-client", 2, 10, 111) {
		t.Fatal("request should pass after the oldest entry expires")
	}
}
