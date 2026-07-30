package wechat

import (
	"os"
	"testing"
)

func TestGetAccessToken(t *testing.T) {
	if os.Getenv("GOFLY_RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set GOFLY_RUN_INTEGRATION_TESTS=1 to run WeChat integration test")
	}
	appID := os.Getenv("GOFLY_TEST_WECHAT_APP_ID")
	secret := os.Getenv("GOFLY_TEST_WECHAT_SECRET")
	if appID == "" || secret == "" {
		t.Fatal("WeChat integration test credentials are not configured")
	}
	GetAccessToken(appID, secret)
}

func TestCreateQrTicket(t *testing.T) {
	if os.Getenv("GOFLY_RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set GOFLY_RUN_INTEGRATION_TESTS=1 to run WeChat integration test")
	}
	appID := os.Getenv("GOFLY_TEST_WECHAT_APP_ID")
	secret := os.Getenv("GOFLY_TEST_WECHAT_SECRET")
	if appID == "" || secret == "" {
		t.Fatal("WeChat integration test credentials are not configured")
	}
	token := GetAccessToken(appID, secret)
	CreateQrImgUrl(token.AccessToken, "agent_20291289&2378278")
}
