package router

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/gorm"
	"go-fly-muti/middleware"
	"go-fly-muti/models"
	"go-fly-muti/setting"
	"go-fly-muti/tools"
	"go-fly-muti/types"
	"go-fly-muti/ws"
)

func TestVerifiedVisitorHTTPAndWebSocket(t *testing.T) {
	browserScript, _ := filepath.Abs("../tests/file-visitor-access.browser.cjs")
	path := os.Getenv("GOFLY_TEST_MYSQL_CONFIG")
	if path == "" {
		t.Skip("isolated MySQL integration test")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg types.Mysql
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "127.0.0.1" && cfg.Server != "localhost" {
		t.Fatal("visitor access test requires local MySQL")
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", cfg.Username, cfg.Password, cfg.Server, cfg.Port)
	root, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("open MySQL")
	}
	defer root.Close()
	schema := fmt.Sprintf("gofly_visitor_test_%d", time.Now().UnixNano())
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
	oldDB, oldConfigs := models.DB, models.CustomConfigs
	models.DB, models.CustomConfigs = db, nil
	defer func() { models.DB, models.CustomConfigs = oldDB, oldConfigs }()
	if err := db.AutoMigrate(&models.User{}, &models.User_role{}, &models.Role{}, &models.Visitor{}, &models.Ipblack{}, &models.VisitorBlack{},
		&models.Message{}, &models.VisitorExt{}, &models.EntConfig{}, &models.ConversationEvent{}, &models.Config{},
		&models.User_client{}, &models.Oauth{}, &models.Conversation{}, &models.Article{}, &models.ReplyGroup{}, &models.ReplyItem{}, &models.Welcome{}).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Name: "visitor-handler", AuthVersion: 1}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	const id, name = "fixture_id", "张三 & Co"
	verify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("username") == "upstream-error" {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		fmt.Fprint(w, r.Method == "GET" && r.URL.Path == "/test/customer_verify" &&
			query.Get("code") == id && query.Get("username") == name)
	}))
	defer verify.Close()
	configDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(configDir, "config"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"qqzeng-ip-utf8.dat", "GeoLite2-Country.mmdb"} {
		geo, err := os.ReadFile(filepath.Join("..", "config", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "config", name), geo, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(configDir)
	config := []byte(fmt.Sprintf("app:\n  trusted_proxies: ['127.0.0.1']\n  customer_verify_url: %q\n", verify.URL+"/test/customer_verify"))
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), config, 0600); err != nil {
		t.Fatal(err)
	}
	if err := setting.Init(); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.CrossSite)
	engine.SetHTMLTemplate(template.Must(template.New("chat_page.html").Parse("{{.visitorId}}")))
	InitViewRouter(engine)
	InitApiRouter(engine)
	app := httptest.NewServer(engine)
	defer app.Close()
	makeClient := func() *http.Client {
		jar, _ := cookiejar.New(nil)
		return &http.Client{Jar: jar, Timeout: 5 * time.Second}
	}
	client := makeClient()
	otherBrowser := makeClient()
	changeSetting := func(role int, required bool, want int) {
		t.Helper()
		req, err := http.NewRequest("POST", app.URL+"/system/visitor-access",
			strings.NewReader(fmt.Sprintf("required=%t", required)))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if role != 0 {
			token, err := tools.MakeToken(map[string]interface{}{
				"name": user.Name, "create_time": time.Now().Unix(), "kefu_id": user.ID,
				"auth_version": 1, "role_id": role, "pid": 0,
			})
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("token", token)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var result struct{ Code int }
		if err := json.NewDecoder(res.Body).Decode(&result); err != nil || res.StatusCode != 200 || result.Code != want {
			t.Fatalf("visitor access settings returned HTTP %d code %d, want code %d: %v", res.StatusCode, result.Code, want, err)
		}
	}
	request := func(client *http.Client, method, path string, form url.Values, want int) {
		t.Helper()
		req, err := http.NewRequest(method, app.URL+path, strings.NewReader(form.Encode()))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != want {
			t.Fatalf("%s %s returned %d, want %d", method, path, res.StatusCode, want)
		}
		if method == "POST" && (path == "/visitor_login" || path == "/2/message") && want == 200 {
			var result struct{ Code int }
			if err := json.NewDecoder(res.Body).Decode(&result); err != nil || result.Code != 200 {
				t.Fatalf("%s did not succeed: %+v %v", path, result, err)
			}
		}
	}
	ent := fmt.Sprint(user.ID)
	visitorID := ent + "|" + id
	entry := url.Values{"ent_id": {ent}, "kefu_id": {user.Name}, "visitor_id": {id}, "visitor_name": {name}}
	request(client, "GET", "/chatIndex?ent_id="+ent, nil, 200)
	changeSetting(0, true, 401)
	changeSetting(2, true, int(types.ApiCode.NO_ADMIN_AUTH))
	request(client, "GET", "/chatIndex?ent_id="+ent, nil, 200)
	changeSetting(1, true, 200)
	request(client, "GET", "/chatIndex?ent_id="+ent, nil, 404)
	rejected := url.Values{"ent_id": {ent}, "visitor_id": {id}, "visitor_name": {"wrong"}}
	request(client, "GET", "/chatIndex?"+rejected.Encode(), nil, 404)
	rejected.Set("visitor_name", "upstream-error")
	request(client, "GET", "/chatIndex?"+rejected.Encode(), nil, 404)
	request(client, "GET", "/chatIndex?"+entry.Encode(), nil, 200)
	request(otherBrowser, "GET", "/chatIndex?"+entry.Encode(), nil, 200)
	login := url.Values{"visitor_id": {id}, "visitor_name": {"wrong"}, "ent_id": {ent}, "to_id": {user.Name}}
	request(client, "POST", "/visitor_login", login, 404)
	request(client, "POST", "/uc/v1/visitorLogin", login, 404)
	request(client, "POST", "/visitor/v2/login", login, 404)
	login.Set("visitor_name", name)
	request(client, "POST", "/visitor_login", login, 200)
	loginReady := time.Now().Add(2100 * time.Millisecond)

	query := url.Values{"visitor_id": {visitorID}, "ent_id": {ent}, "page": {"1"}, "pagesize": {"5"}}
	request(client, "GET", "/2/messages_page?"+query.Encode(), nil, 200)
	request(otherBrowser, "GET", "/2/messages_page?"+query.Encode(), nil, 404)
	form := url.Values{"from_id": {visitorID}, "to_id": {user.Name}, "content": {"test message"}}
	request(otherBrowser, "POST", "/2/message", form, 404)
	request(client, "POST", "/2/message", form, 200)
	assertHistory := func(browser *http.Client) {
		t.Helper()
		res, err := browser.Get(app.URL + "/2/messages_page?" + query.Encode() + "&page=1&pagesize=5")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var result struct {
			Code   int `json:"code"`
			Result struct {
				List []struct {
					Content string `json:"content"`
				} `json:"list"`
			} `json:"result"`
		}
		if err := json.NewDecoder(res.Body).Decode(&result); err != nil || res.StatusCode != 200 || result.Code != 200 {
			t.Fatalf("verified history request failed: HTTP %d code %d: %v", res.StatusCode, result.Code, err)
		}
		for _, item := range result.Result.List {
			if item.Content == "test message" {
				return
			}
		}
		t.Fatal("verified browser did not receive the persisted fixture message")
	}
	assertHistory(client)
	request(otherBrowser, "GET", "/2/messages_page?"+query.Encode()+"&page=1&pagesize=5", nil, 404)

	wsURL := "ws" + strings.TrimPrefix(app.URL, "http") + "/ws_visitor?"
	dial := func(client *http.Client, query url.Values, status int) *websocket.Conn {
		t.Helper()
		dialer := websocket.Dialer{Jar: client.Jar, HandshakeTimeout: 3 * time.Second}
		conn, res, err := dialer.Dial(wsURL+query.Encode(), nil)
		if res == nil || res.StatusCode != status {
			t.Fatalf("websocket handshake wanted %d: %v", status, err)
		}
		if status != 101 {
			res.Body.Close()
		} else if err != nil {
			t.Fatal(err)
		}
		return conn
	}
	dial(otherBrowser, query, 404)
	dial(client, url.Values{"visitor_id": {ent + "|another"}}, 404)
	conn := dial(client, query, 101)
	defer conn.Close()
	if err := conn.WriteJSON(ws.TypeMessage{Type: "ping"}); err != nil {
		t.Fatal(err)
	}
	go ws.WsServerBackend()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var packet ws.TypeMessage
	if err := conn.ReadJSON(&packet); err != nil || packet.Type != "pong" {
		t.Fatalf("websocket ping/pong failed: %+v %v", packet, err)
	}
	_ = conn.Close()
	time.Sleep(time.Until(loginReady))
	request(otherBrowser, "POST", "/visitor_login", login, 200)
	request(otherBrowser, "GET", "/2/messages_page?"+query.Encode(), nil, 200)
	assertHistory(otherBrowser)
	t.Run("portable file origin", func(t *testing.T) {
		portableLogin := url.Values{}
		for key, value := range login {
			portableLogin[key] = value
		}
		portableLogin.Set("access_mode", "token")
		res, err := http.PostForm(app.URL+"/visitor_login", portableLogin)
		if err != nil {
			t.Fatal(err)
		}
		var result struct {
			Code  int    `json:"code"`
			Token string `json:"visitor_access_token"`
		}
		err = json.NewDecoder(res.Body).Decode(&result)
		res.Body.Close()
		if err != nil || result.Code != 200 || result.Token == "" {
			t.Fatalf("portable login failed: code %d, %v", result.Code, err)
		}
		fetch := func(token, visitor string, want int) {
			t.Helper()
			req, _ := http.NewRequest("GET", app.URL+"/2/messages_page?"+url.Values{"visitor_id": {visitor}, "ent_id": {ent}, "page": {"1"}, "pagesize": {"5"}}.Encode(), nil)
			req.Header.Set("Origin", "null")
			req.Header.Set("X-Gofly-Visitor-Token", token)
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != want {
				t.Fatalf("portable request HTTP %d, want %d", res.StatusCode, want)
			}
			if want == 200 {
				var payload struct {
					Result struct {
						List []struct {
							Content string `json:"content"`
						} `json:"list"`
					} `json:"result"`
				}
				if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				found := false
				for _, message := range payload.Result.List {
					found = found || message.Content == "test message"
				}
				if !found {
					t.Fatal("portable client lost persisted history")
				}
			}
		}
		fetch(result.Token, visitorID, 200)
		fetch("", visitorID, 404)
		fetch(result.Token+"tampered", visitorID, 404)
		fetch(result.Token, ent+"|other", 404)
		expired, _ := tools.MakeToken(map[string]interface{}{"purpose": "visitor-portable", "visitor_id": visitorID, "exp": time.Now().Add(-time.Hour).Unix()})
		fetch(expired, visitorID, 404)
		if os.Getenv("GOFLY_TEST_FILE_PAGE") != "" {
			cmd := exec.Command("node", browserScript)
			cmd.Env = append(os.Environ(), "GOFLY_TEST_APP_URL="+app.URL)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("file browser test: %v\n%s", err, out)
			} else {
				t.Log(string(out))
			}
		}
		t.Setenv("GOFLY_ALLOWED_ORIGINS", "https://trusted.example")
		dialer := websocket.Dialer{Subprotocols: []string{"gofly-visitor." + result.Token}, HandshakeTimeout: 3 * time.Second}
		portableWS, _, err := dialer.Dial(wsURL+query.Encode(), http.Header{"Origin": {"null"}})
		if err != nil {
			t.Fatal("portable file websocket:", err)
		}
		defer portableWS.Close()
		_ = portableWS.SetReadDeadline(time.Now().Add(3 * time.Second))
		if err := portableWS.WriteJSON(ws.TypeMessage{Type: "ping"}); err != nil {
			t.Fatal(err)
		}
		var pong ws.TypeMessage
		if err := portableWS.ReadJSON(&pong); err != nil || pong.Type != "pong" {
			t.Fatalf("portable websocket pong: %v", err)
		}
	})
	t.Run("legacy file client without cookies or tokens", func(t *testing.T) {
		const ua = "LegacyAPK-Compatibility-Test/1.0"
		call := func(method, path string, form url.Values, agent string, want int) {
			t.Helper()
			req, _ := http.NewRequest(method, app.URL+path, strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Origin", "null")
			req.Header.Set("User-Agent", agent)
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != want {
				t.Fatalf("%s returned %d, want %d", path, res.StatusCode, want)
			}
			if want == 200 {
				var body struct{ Code int }
				if err := json.NewDecoder(res.Body).Decode(&body); err != nil || body.Code != 200 {
					t.Fatalf("legacy business response: %+v %v", body, err)
				}
			}
		}
		history := "/2/messages_page?" + query.Encode()
		call("GET", history, nil, ua, 404)
		invalidLogin := url.Values{"visitor_id": {id}, "visitor_name": {"wrong"}, "ent_id": {ent}, "to_id": {user.Name}}
		call("POST", "/visitor_login", invalidLogin, ua, 404)
		call("GET", history, nil, ua, 404)
		call("POST", "/visitor_login", login, ua, 200)
		call("GET", history, nil, ua, 200)
		call("GET", history, nil, ua+"other-browser", 404)
		call("GET", "/2/messages_page?visitor_id="+url.QueryEscape(ent+"|unknown")+"&ent_id="+ent, nil, ua, 404)
		t.Setenv("GOFLY_ALLOWED_ORIGINS", "https://trusted.example")
		oldWS, _, err := (&websocket.Dialer{HandshakeTimeout: 3 * time.Second}).Dial(wsURL+query.Encode(), http.Header{"Origin": {"null"}, "User-Agent": {ua}})
		if err != nil {
			t.Fatal("legacy websocket:", err)
		}
		defer oldWS.Close()
		_ = oldWS.SetReadDeadline(time.Now().Add(3 * time.Second))
		if err := oldWS.WriteJSON(ws.TypeMessage{Type: "ping"}); err != nil {
			t.Fatal(err)
		}
		var pong ws.TypeMessage
		if err := oldWS.ReadJSON(&pong); err != nil || pong.Type != "pong" {
			t.Fatalf("legacy pong failed: %v", err)
		}
	})
	changeSetting(1, false, 200)
	request(client, "GET", "/chatIndex?ent_id="+ent, nil, 200)

	t.Run("global and merchant IP allowlists", func(t *testing.T) {
		another := models.User{Name: "another-merchant"}
		if err := db.Create(&another).Error; err != nil {
			t.Fatal(err)
		}
		anotherID := fmt.Sprint(another.ID)
		for _, visitor := range []models.Visitor{
			{VisitorId: "allowlist-fixture-a", EntId: ent, ToId: user.Name},
			{VisitorId: "allowlist-fixture-b", EntId: anotherID, ToId: another.Name},
		} {
			if err := db.Create(&visitor).Error; err != nil {
				t.Fatal(err)
			}
		}

		setAllowlist := func(path string, role int, body string) {
			t.Helper()
			token, err := tools.MakeToken(map[string]interface{}{
				"name": user.Name, "create_time": time.Now().Unix(), "kefu_id": user.ID,
				"auth_version": 1, "role_id": role, "pid": 0,
			})
			if err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequest("POST", app.URL+path, strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("token", token)
			res, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			var result struct{ Code int }
			if err := json.NewDecoder(res.Body).Decode(&result); err != nil || result.Code != 200 {
				t.Fatalf("saving allowlist: HTTP %d, code %d, %v", res.StatusCode, result.Code, err)
			}
		}
		const mainlandIP = "114.114.114.114"
		const explicitIP = "2001:db8::20"
		const deniedIP = "198.51.100.30"
		setAllowlist("/system/visitor-allowlist", 1, `{"enabled":true,"ips":["2001:db8::20"],"regions":["CN"]}`)
		setAllowlist("/kefu/visitor-allowlist?ent_id="+anotherID, 2, `{"enabled":true,"ips":["2001:db8::20","198.51.100.30"]}`)

		httpStatus := func(method, path, ip string, form url.Values, want int) {
			t.Helper()
			req, err := http.NewRequest(method, app.URL+path, strings.NewReader(form.Encode()))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("X-Forwarded-For", ip)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			res, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != want {
				t.Fatalf("%s %s from %s returned %d, want %d", method, path, ip, res.StatusCode, want)
			}
		}
		for _, tc := range []struct {
			ip, tenant, visitor string
			want                int
		}{
			{explicitIP, ent, "allowlist-fixture-a", 200},
			{mainlandIP, ent, "allowlist-fixture-a", 404},
			{deniedIP, ent, "allowlist-fixture-a", 404},
			{explicitIP, anotherID, "allowlist-fixture-b", 200},
			{mainlandIP, anotherID, "allowlist-fixture-b", 200},
			{deniedIP, anotherID, "allowlist-fixture-b", 404},
		} {
			httpStatus("GET", "/chatIndex?ent_id="+tc.tenant, tc.ip, nil, tc.want)
			// The merchant is resolved from the stored visitor when ent_id is absent.
			httpStatus("GET", "/2/messages?visitor_id="+tc.visitor, tc.ip, nil, tc.want)
			dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
			conn, res, err := dialer.Dial(wsURL+url.Values{"visitor_id": {tc.visitor}}.Encode(),
				http.Header{"X-Forwarded-For": {tc.ip}})
			if tc.want == 404 {
				if res == nil || res.StatusCode != 404 || conn != nil {
					t.Fatalf("websocket from %s to tenant %s not rejected: %v", tc.ip, tc.tenant, err)
				}
				res.Body.Close()
				continue
			}
			if err != nil {
				t.Fatalf("websocket from %s to tenant %s: %v", tc.ip, tc.tenant, err)
			}
			_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			var packet ws.TypeMessage
			err = conn.WriteJSON(ws.TypeMessage{Type: "ping"})
			if err == nil {
				err = conn.ReadJSON(&packet)
			}
			_ = conn.Close()
			if err != nil || packet.Type != "pong" {
				t.Fatalf("websocket from %s to tenant %s failed ping/pong: %v", tc.ip, tc.tenant, err)
			}
		}
		httpStatus("POST", "/visitor_login", deniedIP, login, 404)
		httpStatus("POST", "/2/message", mainlandIP, url.Values{
			"from_id": {"allowlist-fixture-a"}, "to_id": {user.Name}, "content": {"denied test"},
		}, 404)
		httpStatus("GET", "/2/messages_page?visitor_id=allowlist-fixture-a&ent_id="+anotherID, mainlandIP, nil, 404)
		// An untrusted hop cannot use an earlier spoofed address.
		httpStatus("GET", "/chatIndex?ent_id="+ent, explicitIP+", "+deniedIP, nil, 404)

		dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
		active, _, err := dialer.Dial(wsURL+"visitor_id=allowlist-fixture-a",
			http.Header{"X-Forwarded-For": {explicitIP}})
		if err != nil {
			t.Fatal(err)
		}
		defer active.Close()
		_ = active.SetReadDeadline(time.Now().Add(3 * time.Second))
		if err := active.WriteJSON(ws.TypeMessage{Type: "ping"}); err != nil {
			t.Fatal(err)
		}
		var activePacket ws.TypeMessage
		if err := active.ReadJSON(&activePacket); err != nil || activePacket.Type != "pong" {
			t.Fatalf("websocket was not active before updating policy: %+v %v", activePacket, err)
		}
		setAllowlist("/kefu/visitor-allowlist", 2, `{"enabled":true,"regions":["CN"]}`)
		if _, _, err := active.ReadMessage(); err == nil {
			t.Fatal("websocket survived a rule change that removed its IP")
		} else if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
			t.Fatal("server did not disconnect the now-disallowed websocket")
		}
		httpStatus("GET", "/chatIndex?ent_id="+ent, mainlandIP, nil, 200)
		httpStatus("GET", "/chatIndex?ent_id="+ent, explicitIP, nil, 404)
		httpStatus("GET", "/chatIndex?ent_id="+anotherID, explicitIP, nil, 200)
	})
}
