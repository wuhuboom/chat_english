package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go-fly-muti/models"
)

func TestGetHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/healthz", GetHealth)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if body := recorder.Body.String(); body != `{"status":"ok"}` {
		t.Fatalf("health body = %s", body)
	}
}

func TestGetReadinessWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalDB := models.DB
	models.DB = nil
	t.Cleanup(func() {
		models.DB = originalDB
	})

	engine := gin.New()
	engine.GET("/readyz", GetReadiness)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if body := recorder.Body.String(); body != `{"status":"unavailable"}` {
		t.Fatalf("readiness body = %s", body)
	}
}
