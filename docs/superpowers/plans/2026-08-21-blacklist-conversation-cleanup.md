# Blacklist Conversation Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop hidden SLA timeout alerts immediately when a merchant blacklists a visitor or IP.

**Architecture:** A controller service resolves matching existing conversations, disconnects matching live visitor sessions, and broadcasts a dedicated resolution event to the assigned agent. The shared admin JavaScript consumes that event and clears all local SLA state for the visitor.

**Tech Stack:** Go, Gin, Gorm v1, Gorilla WebSocket, Vue 2 browser client, Go tests.

---

### Task 1: Model transition for blocked conversations

**Files:**
- Modify: `models/conversations.go`
- Test: `models/conversations_test.go`

- [ ] **Step 1: Write the failing model tests**

Create tests asserting that `ResolveExistingConversation` changes an existing `open` conversation to `resolved`, clears `waiting_since`, and returns `false` without creating a row when no conversation exists.

- [ ] **Step 2: Run the focused test and verify failure**

Run: `go test ./models -run TestResolveExistingConversation -count=1`

Expected: FAIL because `ResolveExistingConversation` is undefined.

- [ ] **Step 3: Add the minimal conditional transition**

Implement:

```go
func ResolveExistingConversation(entID, visitorID, kefuID string, at time.Time) bool {
    conversation := FindConversation(entID, visitorID)
    if conversation.ID == 0 {
        return false
    }
    DB.Model(&conversation).Updates(map[string]interface{}{
        "kefu_id": kefuID, "status": ConversationStatusResolved,
        "waiting_since": nil, "resolved_at": at,
    })
    return true
}
```

- [ ] **Step 4: Run the focused model test**

Run: `go test ./models -run TestResolveExistingConversation -count=1`

Expected: PASS.

### Task 2: Atomically disconnect a blacklisted visitor

**Files:**
- Modify: `ws/ws.go`
- Test: `ws/user_test.go`

- [ ] **Step 1: Write failing WebSocket registry tests**

Test that `DisconnectVisitor` removes the currently registered visitor even when its socket is unavailable, returns that visitor state, and retries when the registered connection is replaced before removal.

- [ ] **Step 2: Run focused tests and verify failure**

Run: `go test ./ws -run TestDisconnectVisitor -count=1`

Expected: FAIL because `DisconnectVisitor` is undefined.

- [ ] **Step 3: Implement current-session removal**

Add a bounded retry loop that reads `VisitorConnection`, removes that exact pointer with `RemoveVisitorConnection`, sends a marshalled `force_close` payload best-effort, closes the socket, and returns the removed `UserState`.

- [ ] **Step 4: Run focused WebSocket tests**

Run: `go test ./ws -run TestDisconnectVisitor -count=1`

Expected: PASS.

### Task 3: Apply cleanup from both blacklist endpoints

**Files:**
- Create: `controller/blacklist_cleanup.go`
- Modify: `controller/visitor_black.go`
- Modify: `controller/ip.go`
- Modify: `controller/ip_test.go`
- Test: `controller/visitor_black_test.go`

- [ ] **Step 1: Write failing handler tests**

Stub the cleanup entry point and assert that visitor blacklisting passes exactly one authenticated-enterprise visitor, while IP blacklisting queries only `ent_id = ? AND (client_ip = ? OR source_ip = ?)` and passes all matches after the blacklist row is successfully created.

- [ ] **Step 2: Run focused controller tests and verify failure**

Run: `go test ./controller -run 'TestPost(VisitorBlack|Ipblack).*Cleanup' -count=1`

Expected: FAIL because the endpoints do not call cleanup.

- [ ] **Step 3: Implement the cleanup coordinator**

For each tenant-owned visitor, call `ResolveExistingConversation`; create a `ConversationEventResolved` event when transitioned; send `conversationResolved` to the assigned agent; call `ws.DisconnectVisitor`; and call `ws.VisitorOffline` only when a live session was actually removed.

- [ ] **Step 4: Run focused controller tests**

Run: `go test ./controller -run 'TestPost(VisitorBlack|Ipblack).*Cleanup' -count=1`

Expected: PASS.

### Task 4: Clear client-side SLA state immediately

**Files:**
- Modify: `static/js/chat-main.js`
- Test: `setting/chat_delivery_template_test.go`

- [ ] **Step 1: Add a failing shared-script contract assertion**

Assert that the shared admin client recognizes `conversationResolved` and clears `service_status`, `waiting_since`, `waiting_seconds`, and `slaAlertedVisitors` for the supplied visitor ID.

- [ ] **Step 2: Run the contract test and verify failure**

Run: `go test ./setting -run TestChatMainHandlesConversationResolved -count=1`

Expected: FAIL because the event handler is absent.

- [ ] **Step 3: Implement the Vue event handler**

Add a `conversationResolved` WebSocket switch case and a method that updates matching items in both lists, clears the alert map entry, and updates the open conversation when it is selected.

- [ ] **Step 4: Run the contract test**

Run: `go test ./setting -run TestChatMainHandlesConversationResolved -count=1`

Expected: PASS.

### Task 5: Regression and browser simulation

**Files:**
- Verify only; no planned production edits.

- [ ] **Step 1: Format and run Go regression tests**

Run: `gofmt -w controller/blacklist_cleanup.go controller/visitor_black.go controller/ip.go controller/ip_test.go controller/visitor_black_test.go models/conversations.go models/conversations_test.go ws/ws.go ws/user_test.go`

Run: `go test ./controller ./models ./setting ./ws -count=1`

Expected: PASS.

- [ ] **Step 2: Check the working tree patch**

Run: `git diff --check`

Expected: no output.

- [ ] **Step 3: Restart and simulate locally**

Create an open visitor conversation from `127.0.0.1`, add that IP to the blacklist in the admin UI, and verify the visitor is disconnected, its subsequent message is rejected, the admin item is resolved with zero waiting time, and no SLA sound repeats.
