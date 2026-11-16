package router

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/gorm"
	"go-fly-muti/middleware"
	"go-fly-muti/models"
	"go-fly-muti/tools"
	"go-fly-muti/types"
	"go-fly-muti/ws"
)

func TestStaffCookieLifecycle(t *testing.T) {
	config := os.Getenv("GOFLY_TEST_MYSQL_CONFIG")
	if config == "" {
		t.Skip("isolated MySQL integration test")
	}
	raw, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var cfg types.Mysql
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "127.0.0.1" && cfg.Server != "localhost" {
		t.Fatal("staff cookie test requires local MySQL")
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", cfg.Username, cfg.Password, cfg.Server, cfg.Port)
	root, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("open MySQL")
	}
	defer root.Close()
	schema := fmt.Sprintf("gofly_staff_test_%d", time.Now().UnixNano())
	if _, err := root.Exec("CREATE DATABASE " + schema); err != nil {
		t.Fatal("create isolated schema:", err)
	}
	defer root.Exec("DROP DATABASE " + schema)
	db, err := gorm.Open("mysql", dsn+schema+"?parseTime=true")
	if err != nil {
		t.Fatal("open isolated schema")
	}
	defer db.Close()
	db.SingularTable(true)
	previous, configs := models.DB, models.CustomConfigs
	models.DB, models.CustomConfigs = db, nil
	defer func() { models.DB, models.CustomConfigs = previous, configs }()
	if err := db.AutoMigrate(&models.User{}, &models.User_role{}, &models.Role{}, &models.AccountTOTP{}, &models.EntConfig{}, &models.Config{}).Error; err != nil {
		t.Fatal(err)
	}
	const password = "fixture-only"
	user := models.User{Name: "staff-cookie-fixture", Password: tools.Md5(password), AuthVersion: 1}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Role{Id: "1", Name: "fixture", Method: "*", Path: "*"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.User_role{UserId: user.ID, RoleId: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Config{ConfKey: models.LegacySystemKefuKey, ConfValue: "preserved evidence fixture"}).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	InitApiRouter(engine)
	engine.GET("/staff-test-page", middleware.JwtPageMiddleware, func(c *gin.Context) { c.Status(204) })
	server := httptest.NewServer(engine)
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request := func(method, path, origin string, form url.Values, replay *http.Cookie) (*http.Response, map[string]interface{}) {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+path, strings.NewReader(form.Encode()))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if replay != nil {
			req.AddCookie(replay)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		payload := map[string]interface{}{}
		_ = json.NewDecoder(res.Body).Decode(&payload)
		return res, payload
	}
	dial := func(origin string, cookie *http.Cookie) (*websocket.Conn, error) {
		header := http.Header{"Origin": []string{origin}, "Cookie": []string{cookie.Name + "=" + cookie.Value}}
		conn, res, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws_kefu", header)
		if err != nil && res != nil {
			res.Body.Close()
		}
		return conn, err
	}
	form := url.Values{"username": {user.Name}, "password": {password}}
	wrong := url.Values{"username": {user.Name}, "password": {"incorrect"}}
	if res, data := request("POST", "/check", server.URL, wrong, nil); data["code"] == float64(200) || len(res.Cookies()) != 0 {
		t.Fatal("incorrect password authenticated")
	}
	if res, _ := request("POST", "/check", "http://other.invalid", form, nil); res.StatusCode != 403 {
		t.Fatal("cross-origin login accepted")
	}
	res, data := request("POST", "/check", server.URL, form, nil)
	if data["code"] != float64(200) || data["result"] != nil || len(res.Cookies()) != 1 {
		t.Fatal("login did not return a cookie-only session")
	}
	cookie := res.Cookies()[0]
	if cookie.Name != middleware.StaffCookie || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Domain != "" {
		t.Fatal("wrong session cookie attributes")
	}
	if page, _ := request("GET", "/staff-test-page", "", nil, nil); page.StatusCode != 204 {
		t.Fatal("page reload did not authenticate")
	}
	for _, key := range []string{"SystemKefu", " systemkefu ", "SYSTEMKEFU"} {
		res, _ := request("POST", "/config", server.URL, url.Values{"key": {key}, "value": {"replacement fixture"}}, nil)
		if res.StatusCode != 403 {
			t.Fatalf("retired config write accepted for %q", key)
		}
	}
	if res, _ := request("GET", "/config?key=SystemKefu", "", nil, nil); res.StatusCode != 403 {
		t.Fatal("retired config exposed through read API")
	}
	if res, _ := request("GET", "/config?key=NoticeEmailPassword", "", nil, nil); res.StatusCode != 403 {
		t.Fatal("private config exposed through read API")
	}
	for _, test := range []struct {
		key, value string
		status     int
	}{
		{"IndexJumpUrl", "mailto:fixture", 400},
		{"LandHost", "example.invalid/path", 400},
		{"version", "999", 403},
		{"SystemTitle", "Safe fixture", 200},
	} {
		res, data := request("POST", "/config", server.URL, url.Values{"key": {test.key}, "value": {test.value}}, nil)
		if res.StatusCode != test.status || data["code"] != float64(test.status) {
			t.Fatalf("unexpected result for %s: %d", test.key, res.StatusCode)
		}
	}
	if _, data := request("GET", "/configs", "", nil, nil); strings.Contains(fmt.Sprint(data["result"]), "preserved evidence fixture") {
		t.Fatal("retired config exposed through settings list")
	}
	var evidence models.Config
	if err := db.Where("conf_key = ?", models.LegacySystemKefuKey).First(&evidence).Error; err != nil || evidence.ConfValue != "preserved evidence fixture" {
		t.Fatal("retired config evidence changed")
	}
	if denied, _ := request("POST", "/staff/session/refresh", "http://other.invalid", nil, nil); denied.StatusCode != 403 {
		t.Fatal("cross-origin refresh accepted")
	}
	res, data = request("POST", "/staff/session/refresh", server.URL, nil, nil)
	if data["code"] != float64(200) || data["result"] != nil || len(res.Cookies()) != 1 {
		t.Fatal("refresh did not renew cookie")
	}
	cookie = res.Cookies()[0]
	if bad, err := dial("http://other.invalid", cookie); err == nil {
		bad.Close()
		t.Fatal("cross-origin WebSocket accepted")
	}
	first, err := dial(server.URL, cookie)
	if err != nil {
		t.Fatal("WebSocket initial handshake:", err)
	}
	first.Close()
	active, err := dial(server.URL, cookie)
	if err != nil {
		t.Fatal("WebSocket reconnect:", err)
	}
	defer active.Close()
	deadline := time.Now().Add(2 * time.Second)
	for !ws.IsKefuOnline(user.Name) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	res, data = request("POST", "/staff/session/logout", server.URL, nil, nil)
	if data["code"] != float64(200) || len(res.Cookies()) != 1 || res.Cookies()[0].MaxAge != -1 {
		t.Fatal("logout did not clear cookie")
	}
	_ = active.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := active.ReadMessage(); err == nil {
		t.Fatal("existing WebSocket remained open")
	} else if nerr, ok := err.(interface{ Timeout() bool }); ok && nerr.Timeout() {
		t.Fatal("existing WebSocket did not close")
	}
	if page, _ := request("GET", "/staff-test-page", "", nil, cookie); page.StatusCode != 302 {
		t.Fatal("revoked cookie still authenticated")
	}
	if stale, err := dial(server.URL, cookie); err == nil {
		stale.Close()
		t.Fatal("revoked cookie reconnected")
	}
	models.CustomConfigs = []models.Config{{ConfKey: models.GoogleAuthenticatorEnabledKey, ConfValue: "true"}}
	if res, data := request("POST", "/check", server.URL, form, nil); data["code"] != float64(428) || len(res.Cookies()) != 0 {
		t.Fatal("TOTP required but issued session before enrollment")
	}
}
