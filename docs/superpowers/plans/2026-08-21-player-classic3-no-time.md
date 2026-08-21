# Classic 3 Player Skin Without Time Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a selectable “经典界面 3-无时间” player H5 skin that preserves Classic 3 behavior while removing message timestamps and their layout slots.

**Architecture:** Copy the current working-tree `chat_page3.html` into an isolated template so all existing delivery and compatibility fixes are inherited. Register the copy in the server allowlist and merchant skin selector, then alter only the copied message-row markup so sender metadata and read status remain without using `.chatTime` inside the message loop.

**Tech Stack:** Go 1.23 tests and template selection, Gin HTML templates, Vue 2 Options API, Element UI, HTML/CSS, browser-based local verification.

---

### Task 1: Define the Classic 3 no-time template contract

**Files:**
- Modify: `setting/chat_template_test.go`
- Modify: `setting/player_no_time_template_test.go`
- Test: `setting/chat_template_test.go`
- Test: `setting/player_no_time_template_test.go`

- [ ] **Step 1: Add the failing allowlist case**

Add this row to `TestValidateH5ChatTemplate`:

```go
{name: "classic three without time", value: "chat_page3_notime.html", want: "chat_page3_notime.html"},
```

- [ ] **Step 2: Add the failing isolation and layout tests**

Append these tests to `setting/player_no_time_template_test.go`:

```go
func TestClassicThreeNoTimePlayerTemplateIsIsolated(t *testing.T) {
	original := readPlayerTemplate(t, "chat_page3.html")
	if !strings.Contains(original, "formatTime(v.time)") {
		t.Fatal("classic interface 3 must keep its existing message timestamps")
	}

	noTime := readPlayerTemplate(t, "chat_page3_notime.html")
	if strings.Contains(noTime, "formatTime(v.time)") {
		t.Fatal("classic interface 3 without time must not render message timestamps")
	}
	for _, marker := range []string{
		"syncUnreadVisitorMessages(this);",
		"visibleUnreadVisitorMessageIDs(_this)",
		`v-html="v.content"`,
		"v.read_status",
		"flyLang.moremessage",
	} {
		if !strings.Contains(noTime, marker) {
			t.Fatalf("classic interface 3 without time lost required behavior %q", marker)
		}
	}
}

func TestClassicThreeNoTimeMessageRowsDoNotReserveTimeSlot(t *testing.T) {
	template := readPlayerTemplate(t, "chat_page3_notime.html")
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
		t.Fatal("classic interface 3 no-time rows must not keep timestamp spacing")
	}
	for _, marker := range []string{"noTimeMessageMeta", "chatReadStatus"} {
		if !strings.Contains(messageRows, marker) {
			t.Fatalf("classic interface 3 no-time rows missing %q", marker)
		}
	}
}

func TestClassicThreeNoTimePlayerTemplateIsSelectable(t *testing.T) {
	settings := readPlayerTemplate(t, "setting_bottom.html")
	for _, marker := range []string{"经典界面 3-无时间", "chat_page3_notime.html"} {
		if !strings.Contains(settings, marker) {
			t.Fatalf("player skin settings missing %q", marker)
		}
	}
}
```

- [ ] **Step 3: Run the focused tests and verify RED**

Run:

```bash
GOCACHE=/tmp/chat-english-go-cache go test ./setting -run 'Test(ValidateH5ChatTemplate|ClassicThreeNoTime)' -count=1
```

Expected: FAIL because `chat_page3_notime.html` is not allowed, does not exist, and is not listed in settings.

- [ ] **Step 4: Record the original template checksum**

Run:

```bash
shasum static/templates/default/chat_page3.html
```

Expected current checksum: `2d91321441734bc05702c76a40c2af6143753d12`.

### Task 2: Add and register the isolated no-time skin

**Files:**
- Create: `static/templates/default/chat_page3_notime.html`
- Modify: `setting/chat_template.go`
- Modify: `static/templates/default/setting_bottom.html`
- Test: `setting/chat_template_test.go`
- Test: `setting/player_no_time_template_test.go`
- Test: `setting/chat_delivery_template_test.go`

- [ ] **Step 1: Copy the current Classic 3 template**

Run the mechanical copy:

```bash
cp static/templates/default/chat_page3.html static/templates/default/chat_page3_notime.html
```

This must copy the working-tree version, not a historical Git revision.

- [ ] **Step 2: Add an isolated compact metadata style**

Add to the inline `<style>` block in the new template only:

```css
.noTimeMessageMeta {
  display: flex;
  align-items: center;
  min-height: 14px;
  margin-bottom: 4px;
}

.noTimeMessageMeta .chatUser {
  margin-bottom: 0;
}
```

- [ ] **Step 3: Remove time markup from incoming messages**

Replace the incoming message metadata wrapper in the new template with:

```html
<div class="noTimeMessageMeta">
    <div class="chatUser" v-if="showKefuName!='off'">
        <{v.name}>
    </div>
</div>
```

The wrapper must not use `.chatTime` or `v.show_time` because there is no timestamp to group or hide.

- [ ] **Step 4: Remove the outgoing timestamp row**

Delete this block from the outgoing message branch in the new template only:

```html
<div class="chatTime" v-bind:class="{'chatTimeHide': v.show_time==false}"><span>
    <{formatTime(v.time)}>
</span></div>
```

Keep the existing `chatContent` and `chatReadStatus` elements unchanged.

- [ ] **Step 5: Register the server allowlist value**

Add to `allowedH5ChatTemplates` in `setting/chat_template.go`:

```go
"chat_page3_notime.html": {},
```

- [ ] **Step 6: Add the backend skin option**

Add immediately after “经典界面 3” in `static/templates/default/setting_bottom.html`:

```javascript
{label:"经典界面 3-无时间",value:"chat_page3_notime.html",description:"经典界面 3 的无时间版本，仅隐藏玩家聊天记录中的消息时间和时间占位。"},
```

- [ ] **Step 7: Format and run focused GREEN tests**

Run:

```bash
gofmt -w setting/chat_template.go setting/chat_template_test.go setting/player_no_time_template_test.go
GOCACHE=/tmp/chat-english-go-cache go test ./setting -run 'Test(ValidateH5ChatTemplate|ClassicThreeNoTime)' -count=1
```

Expected: PASS.

- [ ] **Step 8: Run all template contract tests**

Run:

```bash
GOCACHE=/tmp/chat-english-go-cache go test ./setting -count=1
```

Expected: PASS, including delivery markers for every allowed player template.

- [ ] **Step 9: Verify original Classic 3 remains unchanged**

Run:

```bash
shasum static/templates/default/chat_page3.html
```

Expected: `2d91321441734bc05702c76a40c2af6143753d12`.

- [ ] **Step 10: Commit the implementation**

```bash
git add -- setting/chat_template.go setting/chat_template_test.go setting/player_no_time_template_test.go static/templates/default/setting_bottom.html static/templates/default/chat_page3_notime.html
git commit -m "feat: add classic 3 player skin without timestamps"
```

### Task 3: Run regression and browser verification

**Files:**
- Verify: `static/templates/default/chat_page3_notime.html`
- Verify unchanged: `static/templates/default/chat_page3.html`
- Verify unchanged: `static/templates/default/chat_main.html`
- Verify unchanged: `static/templates/default/chat_main1.html`

- [ ] **Step 1: Run the complete Go test suite**

Run:

```bash
GOCACHE=/tmp/chat-english-go-cache go test ./... -count=1
```

Expected: PASS for every package. WebSocket tests require permission to bind local temporary ports.

- [ ] **Step 2: Build and check the diff**

Run:

```bash
GOCACHE=/tmp/chat-english-go-cache go build -o main .
git diff --check
```

Expected: both commands exit 0.

- [ ] **Step 3: Restart port 8081 safely**

Resolve the exact listener and daemon parent, stop only those PIDs, then run:

```bash
./main server -d -p 8081
```

Expected: a new child process listens on `*:8081` and `/login` returns HTTP 200.

- [ ] **Step 4: Verify the new skin in the browser**

Use the authenticated local backend to record the current skin, select “经典界面 3-无时间”, and open:

```text
http://127.0.0.1:8081/chatIndex?kefu_id=caonima888&ent_id=1
```

Verify the player message loop contains no `.chatTime` elements, no rendered timestamp text, and no timestamp-height row. Confirm the sender name, message bubbles, `chatReadStatus`, load-more control, input controls, and connection status still render.

- [ ] **Step 5: Compare and restore**

Select original “经典界面 3”, reload the player page, and verify timestamps still display. Restore the exact skin selection recorded in Step 4 and close temporary player tabs.

- [ ] **Step 6: Final service and repository checks**

Run:

```bash
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8081/login
curl -sS -o /dev/null -w '%{http_code}\n' 'http://127.0.0.1:8081/chatIndex?kefu_id=caonima888&ent_id=1'
git status --short
```

Expected: both HTTP responses are 200; only known local artifacts such as an untracked `.DS_Store` may remain.
