package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMerchantAuth(t *testing.T) {
	tests := []struct {
		name      string
		roleID    interface{}
		wantAllow bool
	}{
		{name: "merchant", roleID: float64(2), wantAllow: true},
		{name: "merchant uint", roleID: uint(2), wantAllow: true},
		{name: "super administrator", roleID: float64(1)},
		{name: "agent", roleID: float64(3)},
		{name: "invalid", roleID: "bad"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodPost, "/kefu/messages/cleanup", nil)
			context.Set("role_id", test.roleID)

			MerchantAuth(context)
			if test.wantAllow && context.IsAborted() {
				t.Fatal("merchant request was aborted")
			}
			if !test.wantAllow && !context.IsAborted() {
				t.Fatal("non-merchant request was allowed")
			}
		})
	}
}
