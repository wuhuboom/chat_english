package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go-fly-muti/models"
)

func TestPostIpblackUsesAuthenticatedEnterprise(t *testing.T) {
	originalCreate := createIpblack
	originalFind := findVisitorsByEntIP
	originalCleanup := cleanupBlacklistedVisitorsFn
	t.Cleanup(func() {
		createIpblack = originalCreate
		findVisitorsByEntIP = originalFind
		cleanupBlacklistedVisitorsFn = originalCleanup
	})

	var capturedKefuId, capturedEntId string
	createIpblack = func(ip, kefuId, entId, name string) (uint, error) {
		capturedKefuId = kefuId
		capturedEntId = entId
		return 1, nil
	}
	findVisitorsByEntIP = func(entID, ip string) []models.Visitor {
		if entID != "42" || ip != "203.0.113.10" {
			t.Fatalf("IP visitor lookup = %q/%q, want 42/203.0.113.10", entID, ip)
		}
		return []models.Visitor{{VisitorId: "visitor-a", EntId: "42", ToId: "agent-a"}}
	}
	cleanupCalled := false
	cleanupBlacklistedVisitorsFn = func(entID, actor, detail string, visitors []models.Visitor) {
		cleanupCalled = entID == "42" && actor == "agent-a" &&
			strings.Contains(detail, "IP") && len(visitors) == 1 && visitors[0].VisitorId == "visitor-a"
	}

	form := url.Values{"ip": {"203.0.113.10"}, "name": {"访客A"}, "ent_id": {"999"}}
	request := httptest.NewRequest(http.MethodPost, "/ipblack", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	ipblackTestEngine(http.MethodPost, PostIpblack).ServeHTTP(recorder, request)

	if capturedKefuId != "agent-a" || capturedEntId != "42" {
		t.Fatalf("captured kefu=%q ent=%q, want agent-a/42", capturedKefuId, capturedEntId)
	}
	if !cleanupCalled {
		t.Fatal("successful IP blacklist did not terminate matching visitor conversations")
	}
	assertIpblackResponseCode(t, recorder, 200)
}

func TestGetIpblacksUsesEnterpriseScope(t *testing.T) {
	originalFind := findIpsByEntId
	t.Cleanup(func() { findIpsByEntId = originalFind })

	var capturedEntId string
	findIpsByEntId = func(entId string) []models.Ipblack {
		capturedEntId = entId
		return []models.Ipblack{{IP: "203.0.113.10", EntId: entId}}
	}

	request := httptest.NewRequest(http.MethodGet, "/ipblacks", nil)
	recorder := httptest.NewRecorder()
	ipblackTestEngine(http.MethodGet, GetIpblacksByKefuId).ServeHTTP(recorder, request)

	if capturedEntId != "42" {
		t.Fatalf("captured ent=%q, want 42", capturedEntId)
	}
	assertIpblackResponseCode(t, recorder, 200)
}

func TestDelIpblackCannotDeleteAnotherEnterprise(t *testing.T) {
	originalScopedDelete := deleteIpblackByIpAndEntId
	originalGlobalDelete := deleteIpblackByIp
	t.Cleanup(func() {
		deleteIpblackByIpAndEntId = originalScopedDelete
		deleteIpblackByIp = originalGlobalDelete
	})

	var capturedIP, capturedEntId string
	deleteIpblackByIpAndEntId = func(ip, entId string) error {
		capturedIP = ip
		capturedEntId = entId
		return nil
	}
	deleteIpblackByIp = func(ip string) error {
		t.Fatal("merchant delete must not use global IP deletion")
		return nil
	}

	request := httptest.NewRequest(http.MethodDelete, "/ipblack?ip=203.0.113.10", nil)
	recorder := httptest.NewRecorder()
	ipblackTestEngine(http.MethodDelete, DelIpblack).ServeHTTP(recorder, request)

	if capturedIP != "203.0.113.10" || capturedEntId != "42" {
		t.Fatalf("captured ip=%q ent=%q, want 203.0.113.10/42", capturedIP, capturedEntId)
	}
	assertIpblackResponseCode(t, recorder, 200)
}

func TestSeatCanListAndDeleteEnterpriseIpblacks(t *testing.T) {
	originalFind := findIpsByEntId
	originalDelete := deleteIpblackByIpAndEntId
	t.Cleanup(func() {
		findIpsByEntId = originalFind
		deleteIpblackByIpAndEntId = originalDelete
	})

	findCalled := false
	deleteCalled := false
	findIpsByEntId = func(entId string) []models.Ipblack {
		findCalled = entId == "42"
		return []models.Ipblack{}
	}
	deleteIpblackByIpAndEntId = func(ip, entId string) error {
		deleteCalled = ip == "203.0.113.10" && entId == "42"
		return nil
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/ipblacks", nil)
	listRecorder := httptest.NewRecorder()
	ipblackTestEngineWithRole(http.MethodGet, GetIpblacksByKefuId, 3).ServeHTTP(listRecorder, listRequest)
	assertIpblackResponseCode(t, listRecorder, 200)

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/ipblack?ip=203.0.113.10", nil)
	deleteRecorder := httptest.NewRecorder()
	ipblackTestEngineWithRole(http.MethodDelete, DelIpblack, 3).ServeHTTP(deleteRecorder, deleteRequest)
	assertIpblackResponseCode(t, deleteRecorder, 200)

	if !findCalled || !deleteCalled {
		t.Fatalf("seat scope calls: find=%v delete=%v, want both true", findCalled, deleteCalled)
	}
}

func ipblackTestEngine(method string, handler gin.HandlerFunc) *gin.Engine {
	return ipblackTestEngineWithRole(method, handler, 2)
}

func ipblackTestEngineWithRole(method string, handler gin.HandlerFunc, roleId float64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	context := func(c *gin.Context) {
		c.Set("role_id", roleId)
		c.Set("ent_id", "42")
		c.Set("kefu_name", "agent-a")
	}
	engine.Handle(method, "/*path", context, handler)
	return engine
}

func assertIpblackResponseCode(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	var response struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != want {
		t.Fatalf("response code=%d body=%s, want %d", response.Code, recorder.Body.String(), want)
	}
}
