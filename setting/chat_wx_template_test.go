package setting

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWechatH5TemplateRendersSharedPage(t *testing.T) {
	t.Parallel()

	templates, err := template.ParseGlob(filepath.Join("..", "static", "templates", "default", "*.html"))
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	var rendered bytes.Buffer
	if err := templates.ExecuteTemplate(&rendered, "chat_wx.html", map[string]interface{}{
		"H5ChatTemplate": "chat_wx.html",
	}); err != nil {
		t.Fatalf("render chat_wx.html: %v", err)
	}
	page := rendered.String()
	for _, marker := range []string{
		`<body class="visitorChatBody visitorWechatSkin">`,
		`/static/css/chat-wx.css?v=1.0.0`,
		`id="app"`,
	} {
		if !strings.Contains(page, marker) {
			t.Fatalf("rendered WeChat H5 page missing %q", marker)
		}
	}
}

func TestWechatH5TemplateUsesSharedChatCore(t *testing.T) {
	t.Parallel()

	templatePath := filepath.Join("..", "static", "templates", "default", "chat_wx.html")
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("read %s: %v", templatePath, err)
	}
	if !strings.Contains(string(templateContent), `{{template "chat_page.html" .}}`) {
		t.Fatal("wechat H5 template must reuse the maintained visitor chat core")
	}

	basePath := filepath.Join("..", "static", "templates", "default", "chat_page.html")
	baseContent, err := os.ReadFile(basePath)
	if err != nil {
		t.Fatalf("read %s: %v", basePath, err)
	}
	baseSource := string(baseContent)
	for _, marker := range []string{
		`chat-wx.css?v=1.0.0`,
		`visitorWechatSkin`,
		`visitorMessageSelfAvatar`,
	} {
		if !strings.Contains(baseSource, marker) {
			t.Fatalf("shared visitor template missing WeChat skin marker %q", marker)
		}
	}
}

func TestWechatH5StylesMobileSafeAreaAndNaturalWordWrapping(t *testing.T) {
	t.Parallel()

	cssPath := filepath.Join("..", "static", "css", "chat-wx.css")
	content, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("read %s: %v", cssPath, err)
	}
	source := string(content)
	for _, marker := range []string{
		`env(safe-area-inset-top, 0px)`,
		`env(safe-area-inset-bottom, 0px)`,
		`overflow-wrap: break-word`,
		`word-break: normal`,
		`#95ec69`,
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("wechat H5 stylesheet missing %q", marker)
		}
	}
}
