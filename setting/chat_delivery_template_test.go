package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEveryH5TemplateRecoversUnreadMessagesAndAcknowledgesVisibleIDs(t *testing.T) {
	t.Parallel()

	for template := range allowedH5ChatTemplates {
		template := template
		t.Run(template, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join("..", "static", "templates", "default", template)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			source := string(content)
			if !strings.Contains(source, "syncUnreadVisitorMessages(this);") {
				t.Fatalf("%s does not recover persisted unread messages after websocket connect", template)
			}
			if !strings.Contains(source, "visibleUnreadVisitorMessageIDs(_this)") {
				t.Fatalf("%s does not acknowledge only messages visible in this browser", template)
			}
			if !strings.Contains(source, "hasVisitorChatMessageID(this, msg.msg_id)") {
				t.Fatalf("%s does not deduplicate a websocket message racing with history recovery", template)
			}
			if !strings.Contains(source, "insertVisitorChatMessageByID(_this, content)") {
				t.Fatalf("%s does not merge paged history by message id", template)
			}
			if !strings.Contains(source, "markVisitorMessagesRead(_this, data.msg_ids)") {
				t.Fatalf("%s can overwrite a newer unread state when an older acknowledgement completes", template)
			}
		})
	}
}

func TestAgentChatMarksOnlyAcknowledgedMessagesRead(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "static", "js", "chat-main.js")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(content), "msg.msg_ids.indexOf(this.msgList[i].msg_id)") {
		t.Fatal("agent chat still marks the entire conversation read for a partial visitor acknowledgement")
	}
}

func TestVisitorRecoveryIncludesRecentlyFalseReadMessages(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "static", "js", "functions.js")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	source := string(content)
	if !strings.Contains(source, `$.get("/2/messages_page"`) {
		t.Fatal("reconnect recovery cannot restore recent messages previously marked read by the old bulk acknowledgement")
	}
}

func TestAgentChatHandlesBlacklistedConversationResolution(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "static", "js", "chat-main.js")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	source := string(content)
	for _, contract := range []string{
		`case "conversationResolved":`,
		`resolveVisitorConversation(redata.data)`,
		`item.service_status="resolved"`,
		`item.waiting_since=""`,
		`item.waiting_seconds=0`,
		`clearInterval(this.alertSoundingTimer)`,
	} {
		if !strings.Contains(source, contract) {
			t.Fatalf("agent chat does not stop a blacklisted conversation alert; missing %q", contract)
		}
	}
}
