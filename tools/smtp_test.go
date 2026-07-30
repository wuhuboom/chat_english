package tools

import (
	"fmt"
	"log"
	"os"
	"testing"
)

func TestSendSmtp(t *testing.T) {
	if os.Getenv("GOFLY_RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set GOFLY_RUN_INTEGRATION_TESTS=1 to run SMTP integration test")
	}
	username := os.Getenv("GOFLY_TEST_SMTP_USERNAME")
	password := os.Getenv("GOFLY_TEST_SMTP_PASSWORD")
	recipient := os.Getenv("GOFLY_TEST_SMTP_RECIPIENT")
	if username == "" || password == "" || recipient == "" {
		t.Fatal("SMTP integration test credentials are not configured")
	}
	for i := 0; i < 1; i++ {
		body := fmt.Sprintf("<h1>hello,body %d</h1>", i)
		subject := fmt.Sprintf("hello subject %d", i)
		err := SendSmtp("smtp.qq.com:465", username, password, []string{recipient}, subject, body)
		log.Println(err)
	}
}
