package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readPlayerTemplate(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "static", "templates", "default", name))
	if err != nil {
		t.Fatalf("read player template %q: %v", name, err)
	}
	return string(content)
}

func TestClassicNoTimePlayerTemplateIsIsolated(t *testing.T) {
	original := readPlayerTemplate(t, "chat_page1.html")
	if !strings.Contains(original, "formatTime(v.time)") {
		t.Fatal("classic interface 1 must keep its existing message timestamps")
	}

	noTime := readPlayerTemplate(t, "chat_page1_notime.html")
	if strings.Contains(noTime, "formatTime(v.time)") {
		t.Fatal("classic interface 1 without time must not render message timestamps")
	}

	for _, marker := range []string{
		"syncUnreadVisitorMessages(this);",
		"visibleUnreadVisitorMessageIDs(_this)",
		`v-html="v.content"`,
		`v.read_status=='已读'`,
		"flyLang.moremessage",
	} {
		if !strings.Contains(noTime, marker) {
			t.Fatalf("classic interface 1 without time lost required behavior %q", marker)
		}
	}
}

func TestClassicNoTimePlayerTemplateIsSelectable(t *testing.T) {
	settings := readPlayerTemplate(t, "setting_bottom.html")
	for _, marker := range []string{"经典界面 1-无时间", "chat_page1_notime.html"} {
		if !strings.Contains(settings, marker) {
			t.Fatalf("player skin settings missing %q", marker)
		}
	}
}

func TestClassicNoTimeMessageRowsDoNotReserveTimeSlot(t *testing.T) {
	template := readPlayerTemplate(t, "chat_page1_notime.html")
	start := strings.Index(template, `v-for="v in msgList"`)
	if start < 0 {
		t.Fatal("message loop start marker not found")
	}
	end := strings.Index(template[start:], `<div class="clear"></div>`)
	if end < 0 {
		t.Fatal("message loop end marker not found")
	}
	messageRows := template[start : start+end]

	if strings.Contains(messageRows, `class="chatTime"`) {
		t.Fatal("no-time message rows must not keep the timestamp container spacing")
	}
	for _, marker := range []string{"noTimeMessageMeta", "noTimeMineLine", "noTimeReadStatus"} {
		if !strings.Contains(messageRows, marker) {
			t.Fatalf("no-time message rows missing compact layout %q", marker)
		}
	}
}
