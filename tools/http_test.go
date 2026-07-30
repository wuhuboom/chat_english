package tools

import (
	"log"
	"os"
	"testing"
)

func TestPostHeader(t *testing.T) {
	if os.Getenv("GOFLY_RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set GOFLY_RUN_INTEGRATION_TESTS=1 to run HTTP integration test")
	}
	url := os.Getenv("GOFLY_TEST_HTTP_URL")
	if url == "" {
		t.Fatal("GOFLY_TEST_HTTP_URL is not configured")
	}
	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	res, err := PostHeader(url, []byte("health=check"), headers)
	log.Println(res, err)
}
func TestIsMobile(t *testing.T) {
	IsMobile("aaaa")
}
