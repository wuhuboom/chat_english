package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUploadFileIsAlwaysDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/uploadfile", nil)

	UploadFile(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("got status %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if !strings.Contains(recorder.Body.String(), "仅允许上传图片") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}
