package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go-fly-muti/middleware"
	"go-fly-muti/setting"
)

func TestParseMessageCleanupRange(t *testing.T) {
	originalTimezone := setting.CurrentTimezone()
	t.Cleanup(func() { _ = setting.ConfigureTimezone(originalTimezone) })
	if err := setting.ConfigureTimezone("Asia/Shanghai"); err != nil {
		t.Fatal(err)
	}

	start, end, err := parseMessageCleanupRange("2026-08-01 10:00:00", "2026-08-01 11:30:00")
	if err != nil {
		t.Fatalf("parseMessageCleanupRange() error = %v", err)
	}
	if start.Location().String() != "Asia/Shanghai" {
		t.Fatalf("start timezone = %s", start.Location())
	}
	if end.Sub(start) != 90*time.Minute {
		t.Fatalf("range duration = %s, want 90m", end.Sub(start))
	}
}

func TestMessageCleanupPreviewUsesAuthenticatedMerchant(t *testing.T) {
	originalCount := countMessagesInRange
	t.Cleanup(func() { countMessagesInRange = originalCount })

	var capturedEntID string
	countMessagesInRange = func(entID string, start, end time.Time) (int64, error) {
		capturedEntID = entID
		return 17, nil
	}

	engine := merchantCleanupTestEngine()
	request := httptest.NewRequest(http.MethodGet,
		"/preview?start_time=2026-08-01+10%3A00%3A00&end_time=2026-08-01+11%3A00%3A00&ent_id=999",
		nil,
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if capturedEntID != "42" {
		t.Fatalf("cleanup ent_id = %q, want authenticated merchant 42", capturedEntID)
	}
	var response struct {
		Code   int `json:"code"`
		Result struct {
			Count int64 `json:"count"`
		} `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || response.Result.Count != 17 {
		t.Fatalf("response = %+v", response)
	}
}

func TestDeleteMessagesByTimeRangeUsesAuthenticatedMerchant(t *testing.T) {
	originalDelete := deleteMessagesInRange
	t.Cleanup(func() { deleteMessagesInRange = originalDelete })

	var capturedEntID string
	deleteMessagesInRange = func(entID string, start, end time.Time) (int64, error) {
		capturedEntID = entID
		return 9, nil
	}

	engine := merchantCleanupTestEngine()
	form := url.Values{
		"start_time":   {"2026-08-01 10:00:00"},
		"end_time":     {"2026-08-01 11:00:00"},
		"confirmation": {messageCleanupConfirmation},
		"ent_id":       {"999"},
	}
	request := httptest.NewRequest(http.MethodPost, "/cleanup", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if capturedEntID != "42" {
		t.Fatalf("cleanup ent_id = %q, want authenticated merchant 42", capturedEntID)
	}
	var response struct {
		Code   int `json:"code"`
		Result struct {
			DeletedCount int64 `json:"deleted_count"`
		} `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || response.Result.DeletedCount != 9 {
		t.Fatalf("response = %+v", response)
	}
}

func merchantCleanupTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	merchantContext := func(c *gin.Context) {
		c.Set("role_id", float64(2))
		c.Set("ent_id", "42")
		c.Set("kefu_name", "merchant-test")
	}
	engine.GET("/preview", merchantContext, middleware.MerchantAuth, GetMessageCleanupPreview)
	engine.POST("/cleanup", merchantContext, middleware.MerchantAuth, DeleteMessagesByTimeRange)
	return engine
}

func TestParseMessageCleanupRangeRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		start string
		end   string
	}{
		{name: "empty", start: "", end: ""},
		{name: "invalid format", start: "2026/08/01", end: "2026-08-02 00:00:00"},
		{name: "same time", start: "2026-08-01 10:00:00", end: "2026-08-01 10:00:00"},
		{name: "reversed", start: "2026-08-02 10:00:00", end: "2026-08-01 10:00:00"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := parseMessageCleanupRange(test.start, test.end); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
