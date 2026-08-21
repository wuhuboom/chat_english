# Player Classic No-Time Skin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a seventh player H5 skin named “经典界面 1-无时间” without changing any of the six existing player skins or the agent chat interface.

**Architecture:** Copy the current `chat_page1.html` into a separately selectable template and remove only the two message-time spans from the new copy. Register the new filename in the existing template allowlist and skin selector; keep message timestamps in data and APIs so ordering, recovery, and read state remain unchanged.

**Tech Stack:** Go 1.21+, Gin template selection, Vue 2 Options API embedded in HTML, Go contract tests.

---

### Task 1: Add failing registration and isolation tests

**Files:**
- Modify: `setting/chat_template_test.go`
- Create: `setting/player_no_time_template_test.go`

- [ ] **Step 1: Add the allowlist test case**

Add this case to `TestValidateH5ChatTemplate`:

```go
{name: "classic one without time", value: "chat_page1_notime.html", want: "chat_page1_notime.html"},
```

- [ ] **Step 2: Add the template contract test**

Create `setting/player_no_time_template_test.go` with a test that reads both templates and asserts:

```go
func TestClassicNoTimePlayerTemplateIsIsolated(t *testing.T) {
    original := readTemplate(t, "chat_page1.html")
    noTime := readTemplate(t, "chat_page1_notime.html")

    if !strings.Contains(original, "formatTime(v.time)") {
        t.Fatal("classic interface 1 must retain its original message time")
    }
    if strings.Contains(noTime, "formatTime(v.time)") {
        t.Fatal("no-time player interface still renders a message time")
    }
    for _, marker := range []string{
        `syncUnreadVisitorMessages(this);`,
        `visibleUnreadVisitorMessageIDs(_this)`,
        `v-html="v.content"`,
        `v.read_status=='已读'`,
        `点击加载更多记录`,
    } {
        if !strings.Contains(noTime, marker) {
            t.Fatalf("no-time template lost classic behavior %q", marker)
        }
    }
}
```

The helper must read `../static/templates/default/<name>` with `os.ReadFile` and fail the test on read errors.

- [ ] **Step 3: Run the focused tests and verify red**

Run:

```bash
GOCACHE=/tmp/chat-english-go-cache go test ./setting -run 'Test(ValidateH5ChatTemplate|ClassicNoTimePlayerTemplateIsIsolated)' -count=1
```

Expected: FAIL because `chat_page1_notime.html` is not allowed and does not exist.

### Task 2: Create the isolated player template

**Files:**
- Create: `static/templates/default/chat_page1_notime.html`
- Preserve unchanged: `static/templates/default/chat_page1.html`

- [ ] **Step 1: Copy the current classic template**

Mechanically copy the current working-tree contents of `chat_page1.html` to `chat_page1_notime.html`. Do not copy from Git HEAD because the working file contains current compatibility and delivery fixes.

- [ ] **Step 2: Remove only the two message-time spans from the new copy**

In the visitor/agent message header inside `chat_page1_notime.html`, remove:

```html
<span style="margin:0 4px;">
  <{formatTime(v.time)}>
</span>
```

In the player message header, remove:

```html
<span>
  <{formatTime(v.time)}>
</span>
```

Also remove the obsolete commented example containing `formatTime(v.time)`. Preserve the surrounding `chatUser`, read icon, unread icon, load-more text, system notices, and all JavaScript `content.time` assignments.

- [ ] **Step 3: Prove the original skin was not altered by this task**

Run:

```bash
git diff -- static/templates/default/chat_page1.html
```

Expected: only changes that already existed before this task; this task adds no new hunk to the original file.

### Task 3: Register the seventh skin

**Files:**
- Modify: `setting/chat_template.go`
- Modify: `static/templates/default/setting_bottom.html`

- [ ] **Step 1: Add the safe filename to the allowlist**

Add:

```go
"chat_page1_notime.html": {},
```

to `allowedH5ChatTemplates` without removing or renaming any existing entry.

- [ ] **Step 2: Add the settings option**

Append this option after “经典界面 1”:

```javascript
{label:"经典界面 1-无时间",value:"chat_page1_notime.html",description:"经典界面 1 的无时间版本，仅隐藏玩家聊天记录中的消息时间。"},
```

Do not modify `static/js/chat-main.js` or either `chat_main*.html` agent-chat template.

- [ ] **Step 3: Run the focused tests and verify green**

Run:

```bash
GOCACHE=/tmp/chat-english-go-cache go test ./setting -run 'Test(ValidateH5ChatTemplate|ClassicNoTimePlayerTemplateIsIsolated)' -count=1
```

Expected: PASS.

### Task 4: Regression and browser verification

**Files:**
- Verify: `static/templates/default/chat_page1.html`
- Verify: `static/templates/default/chat_page1_notime.html`
- Verify: `static/templates/default/chat_main.html`
- Verify: `static/templates/default/chat_main1.html`

- [ ] **Step 1: Run setting and delivery regression tests**

Run:

```bash
GOCACHE=/tmp/chat-english-go-cache go test ./setting -count=1
```

Expected: PASS, including every existing H5 unread recovery contract.

- [ ] **Step 2: Run build and patch checks**

Run:

```bash
GOCACHE=/tmp/chat-english-go-cache go build -o /tmp/chat-english-no-time-verify .
git diff --check
```

Expected: build exits 0 and `git diff --check` prints nothing.

- [ ] **Step 3: Restart and verify in the browser**

Select “经典界面 1-无时间” in the existing merchant skin setting, open a player chat, and send one player message plus one agent reply. Verify there are no message timestamps, while load-more, connection text, agent name, read/unread icon, text messages, and image messages remain visible.

- [ ] **Step 4: Verify existing interfaces are unchanged**

Switch back to “经典界面 1” and verify its message timestamps remain visible. Open the agent chat page and verify its timestamps remain visible. Restore the merchant's original selected skin after the test.
