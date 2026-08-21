package controller

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"go-fly-muti/ws"
)

func TestVisitorMessageLimitKey(t *testing.T) {
	t.Parallel()

	first := visitorMessageLimitKey("visitor-a", "127.0.0.1")
	second := visitorMessageLimitKey("visitor-b", "127.0.0.1")
	if first == second {
		t.Fatalf("different visitors behind one IP must not share a rate-limit key: %q", first)
	}

	if got := visitorMessageLimitKey("visitor-a", "10.0.0.2"); got != first {
		t.Fatalf("one visitor must keep the same key across IP changes: got %q, want %q", got, first)
	}

	if got := visitorMessageLimitKey("", "127.0.0.1"); got != "sendmessage:127.0.0.1" {
		t.Fatalf("empty visitor id must fall back to client IP: got %q", got)
	}
}

func TestParseVisitorReadMessageIDs(t *testing.T) {
	t.Parallel()

	got := parseVisitorReadMessageIDs(" 17,18,17,bad,0, 19 ")
	want := []uint{17, 18, 19}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseVisitorReadMessageIDs() = %#v, want %#v", got, want)
	}
}

func TestParseVisitorReadMessageIDsLimitsBatchSize(t *testing.T) {
	t.Parallel()

	parts := make([]string, 0, maxVisitorReadMessageIDs+20)
	for id := 1; id <= maxVisitorReadMessageIDs+20; id++ {
		parts = append(parts, strconv.Itoa(id))
	}
	if got := parseVisitorReadMessageIDs(strings.Join(parts, ",")); len(got) != maxVisitorReadMessageIDs {
		t.Fatalf("read acknowledgement size = %d, want capped at %d", len(got), maxVisitorReadMessageIDs)
	}
}

func TestValidateVisitorReadSessionUsesCurrentServerState(t *testing.T) {
	const visitorID = "visitor-read-session"
	visitor := &ws.User{Id: visitorID, Ent_id: "2", To_id: "WGG2"}
	ws.AddVisitorToList(visitor)
	t.Cleanup(func() { ws.RemoveVisitorConnection(visitorID, visitor) })

	state, ok := validateVisitorReadSession(visitorID, "2")
	if !ok || state.ToID != "WGG2" || state.EntID != "2" {
		t.Fatalf("validated state = %#v, ok %v", state, ok)
	}
	if _, ok := validateVisitorReadSession(visitorID, "1"); ok {
		t.Fatal("visitor read acknowledgement accepted a different enterprise")
	}
}

func TestParseVisitorReadMessageIDsRejectsEmptyInput(t *testing.T) {
	t.Parallel()

	if got := parseVisitorReadMessageIDs(""); len(got) != 0 {
		t.Fatalf("empty read acknowledgement must not select every message: %#v", got)
	}
}

func TestVisitorReadNotificationCarriesAcknowledgedMessageIDs(t *testing.T) {
	t.Parallel()

	message := visitorReadNotification("visitor-a", "WGG1", []uint{17, 19})
	data, ok := message.Data.(ws.ClientMessage)
	if !ok {
		t.Fatalf("notification data type = %T, want ws.ClientMessage", message.Data)
	}
	if !reflect.DeepEqual(data.MsgIds, []uint{17, 19}) {
		t.Fatalf("notification msg_ids = %#v, want [17 19]", data.MsgIds)
	}
}
