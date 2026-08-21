package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go-fly-muti/models"
)

func TestPostVisitorBlackTerminatesAuthenticatedVisitorConversation(t *testing.T) {
	originalAdd := addVisitorBlack
	originalFind := findVisitorForBlacklist
	originalCleanup := cleanupBlacklistedVisitorsFn
	t.Cleanup(func() {
		addVisitorBlack = originalAdd
		findVisitorForBlacklist = originalFind
		cleanupBlacklistedVisitorsFn = originalCleanup
	})

	addVisitorBlack = func(model *models.VisitorBlack) error { return nil }
	findVisitorForBlacklist = func(visitorID string) models.Visitor {
		return models.Visitor{Model: models.Model{ID: 1}, VisitorId: visitorID, EntId: "42", ToId: "agent-a", Name: "访客A"}
	}
	cleanupCalled := false
	cleanupBlacklistedVisitorsFn = func(entID, actor, detail string, visitors []models.Visitor) {
		cleanupCalled = entID == "42" && actor == "agent-a" &&
			strings.Contains(detail, "访客黑名单") && len(visitors) == 1 && visitors[0].VisitorId == "visitor-a"
	}

	form := url.Values{"visitor_id": {"visitor-a"}, "name": {"访客A"}}
	request := httptest.NewRequest(http.MethodPost, "/visitor-black", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	ipblackTestEngine(http.MethodPost, PostVisitorBlack).ServeHTTP(recorder, request)

	if !cleanupCalled {
		t.Fatal("successful visitor blacklist did not terminate its conversation")
	}
	assertIpblackResponseCode(t, recorder, 20000)
}

func TestPostVisitorBlackDoesNotTerminateVisitorFromAnotherEnterprise(t *testing.T) {
	originalAdd := addVisitorBlack
	originalFind := findVisitorForBlacklist
	originalCleanup := cleanupBlacklistedVisitorsFn
	t.Cleanup(func() {
		addVisitorBlack = originalAdd
		findVisitorForBlacklist = originalFind
		cleanupBlacklistedVisitorsFn = originalCleanup
	})

	addVisitorBlack = func(model *models.VisitorBlack) error { return nil }
	findVisitorForBlacklist = func(visitorID string) models.Visitor {
		return models.Visitor{Model: models.Model{ID: 2}, VisitorId: visitorID, EntId: "99", ToId: "other-agent"}
	}
	cleanupBlacklistedVisitorsFn = func(string, string, string, []models.Visitor) {
		t.Fatal("another enterprise visitor must not be terminated")
	}

	form := url.Values{"visitor_id": {"visitor-a"}, "name": {"访客A"}}
	request := httptest.NewRequest(http.MethodPost, "/visitor-black", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	ipblackTestEngine(http.MethodPost, PostVisitorBlack).ServeHTTP(recorder, request)

	assertIpblackResponseCode(t, recorder, 20000)
}
