package ws

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketCloseInfo(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   int
		wantReason string
	}{
		{
			name:       "abnormal websocket close",
			err:        &websocket.CloseError{Code: websocket.CloseAbnormalClosure, Text: "unexpected EOF"},
			wantCode:   websocket.CloseAbnormalClosure,
			wantReason: "unexpected EOF",
		},
		{
			name:       "generic transport error",
			err:        errors.New("connection reset by peer"),
			wantCode:   0,
			wantReason: "connection reset by peer",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, reason := websocketCloseInfo(test.err)
			if code != test.wantCode || reason != test.wantReason {
				t.Fatalf("websocketCloseInfo() = (%d, %q), want (%d, %q)", code, reason, test.wantCode, test.wantReason)
			}
		})
	}
}

func TestRemoveBrokenKefuConnectionPreservesOtherSession(t *testing.T) {
	const kefuID = "broken-session-test"
	broken := &User{Id: kefuID, SessionID: 1}
	healthy := &User{Id: kefuID, SessionID: 2}
	AddKefuToList(broken)
	AddKefuToList(healthy)
	t.Cleanup(func() {
		RemoveKefuConnection(kefuID, broken)
		RemoveKefuConnection(kefuID, healthy)
	})

	if removedLast := removeBrokenKefuConnection(broken); removedLast {
		t.Fatal("removing one broken session must not mark the agent offline")
	}
	connections := KefuConnections(kefuID)
	if len(connections) != 1 || connections[0] != healthy {
		t.Fatalf("healthy session was not preserved: %#v", connections)
	}
}

func TestKefuConnectionLifecycle(t *testing.T) {
	const kefuID = "test-kefu"
	first := &User{Id: kefuID}
	second := &User{Id: kefuID}

	AddKefuToList(first)
	AddKefuToList(second)
	t.Cleanup(func() {
		RemoveKefuConnection(kefuID, first)
		RemoveKefuConnection(kefuID, second)
	})

	if !IsKefuOnline(kefuID) {
		t.Fatal("kefu should be online after connections are added")
	}
	if got := len(KefuConnections(kefuID)); got != 2 {
		t.Fatalf("connection count = %d, want 2", got)
	}

	RemoveKefuConnection(kefuID, first)
	if !IsKefuOnline(kefuID) {
		t.Fatal("removing an old connection must preserve the newer connection")
	}

	RemoveKefuConnection(kefuID, second)
	if IsKefuOnline(kefuID) {
		t.Fatal("kefu should be offline after all connections are removed")
	}
}

func TestKefuConnectionStaleThreshold(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		lastSeen time.Time
		want     bool
	}{
		{name: "zero timestamp remains compatible", lastSeen: time.Time{}, want: false},
		{name: "recent heartbeat", lastSeen: now.Add(-time.Minute), want: false},
		{name: "threshold boundary", lastSeen: now.Add(-kefuConnectionStaleAfter), want: false},
		{name: "past threshold", lastSeen: now.Add(-kefuConnectionStaleAfter - time.Second), want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isKefuConnectionStale(test.lastSeen, now); got != test.want {
				t.Fatalf("isKefuConnectionStale() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestKefuConnectionsConcurrentAccess(t *testing.T) {
	const kefuID = "test-kefu-concurrent"
	connections := make([]*User, 100)
	var wait sync.WaitGroup
	for index := range connections {
		connections[index] = &User{Id: kefuID}
		wait.Add(1)
		go func(user *User) {
			defer wait.Done()
			AddKefuToList(user)
			_ = IsKefuOnline(kefuID)
		}(connections[index])
	}
	wait.Wait()

	if got := len(KefuConnections(kefuID)); got != len(connections) {
		t.Fatalf("connection count = %d, want %d", got, len(connections))
	}

	for _, connection := range connections {
		wait.Add(1)
		go func(user *User) {
			defer wait.Done()
			RemoveKefuConnection(kefuID, user)
		}(connection)
	}
	wait.Wait()
	if IsKefuOnline(kefuID) {
		t.Fatal("kefu should be offline after concurrent removals")
	}
}

func TestVisitorReconnectKeepsNewestConnection(t *testing.T) {
	const visitorID = "visitor-reconnect"
	first := &User{Id: visitorID, To_id: "WGG1", UpdateTime: time.Now()}
	second := &User{Id: visitorID, To_id: "WGG2", UpdateTime: time.Now()}

	if !AddVisitorToList(first) {
		t.Fatal("first connection was not marked newly online")
	}
	t.Cleanup(func() {
		RemoveVisitorConnection(visitorID, first)
		RemoveVisitorConnection(visitorID, second)
	})

	if AddVisitorToList(second) {
		t.Fatal("reconnect was incorrectly marked newly online")
	}
	if RemoveVisitorConnection(visitorID, first) {
		t.Fatal("closing the old connection removed the new connection")
	}
	current, ok := VisitorConnection(visitorID)
	if !ok || current != second {
		t.Fatal("new visitor connection was not preserved")
	}
}

func TestVisitorConnectionsConcurrentAccess(t *testing.T) {
	const total = 100
	users := make([]*User, total)
	var wait sync.WaitGroup

	for index := range users {
		users[index] = &User{Id: fmt.Sprintf("visitor-%d", index)}
		wait.Add(1)
		go func(user *User) {
			defer wait.Done()
			storeVisitorConnection(user)
			user.Touch()
			_ = user.State()
			_, _ = VisitorConnection(user.Id)
			_ = VisitorConnectionsSnapshot()
		}(users[index])
	}
	wait.Wait()

	for _, user := range users {
		wait.Add(1)
		go func(user *User) {
			defer wait.Done()
			RemoveVisitorConnection(user.Id, user)
		}(user)
	}
	wait.Wait()
}

func TestRoomConcurrentMembershipAndSnapshot(t *testing.T) {
	room := &ChatRoom{members: make(map[string][]*User)}
	const roomID = "room-concurrent"
	const total = 100
	var wait sync.WaitGroup

	for index := 0; index < total; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			room.addMember(roomID, &User{Id: fmt.Sprintf("member-%d", index)})
		}(index)
	}
	wait.Wait()

	members, ok := room.GetMembers(roomID)
	if !ok || len(members) != total {
		t.Fatalf("member count = %d, want %d", len(members), total)
	}
	members[0] = nil
	current, _ := room.GetMembers(roomID)
	if current[0] == nil {
		t.Fatal("GetMembers returned the room's mutable backing slice")
	}
}

func TestKefuReconnectHeartbeatAndNotification(t *testing.T) {
	const kefuID = "WGG1-simulation"
	connected := make(chan *User, 2)
	serverErrors := make(chan error, 2)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			serverErrors <- err
			return
		}
		user := &User{Id: kefuID, Name: "WGG1", Conn: conn}
		if err := configureWebSocketHeartbeat(user); err != nil {
			serverErrors <- err
			_ = conn.Close()
			return
		}
		AddKefuToList(user)
		connected <- user
		defer func() {
			RemoveKefuConnection(kefuID, user)
			_ = conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	oldClient, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial old WGG1 connection: %v", err)
	}
	defer oldClient.Close()
	<-connected

	newClient, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial reconnected WGG1 connection: %v", err)
	}
	defer newClient.Close()
	reconnectedUser := <-connected

	if got := len(KefuConnections(kefuID)); got != 2 {
		t.Fatalf("connections before old disconnect = %d, want 2", got)
	}
	t.Log("WGG1 automatic reconnect registered alongside the old connection")

	if err := oldClient.Close(); err != nil {
		t.Fatalf("close old WGG1 connection: %v", err)
	}
	waitForKefuConnections(t, kefuID, 1)
	if !IsKefuOnline(kefuID) {
		t.Fatal("WGG1 went offline when only the old connection closed")
	}
	t.Log("old connection closed; the reconnected WGG1 session stayed online")

	staleTime := time.Now().Add(-time.Minute)
	reconnectedUser.SetUpdateTime(staleTime)
	SendPingToKefuClient()
	_ = newClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	var heartbeat TypeMessage
	if err := newClient.ReadJSON(&heartbeat); err != nil {
		t.Fatalf("read heartbeat: %v", err)
	}
	if heartbeat.Type != "many pong" {
		t.Fatalf("heartbeat type = %v, want many pong", heartbeat.Type)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if reconnectedUser.State().UpdateTime.After(staleTime) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !reconnectedUser.State().UpdateTime.After(staleTime) {
		t.Fatal("protocol pong did not refresh the reconnected session heartbeat")
	}
	t.Log("reconnected WGG1 session received the server heartbeat")

	notification, err := json.Marshal(TypeMessage{
		Type: "userOnline",
		Data: map[string]string{"uid": "simulated-customer", "username": "模拟客户"},
	})
	if err != nil {
		t.Fatalf("marshal notification: %v", err)
	}
	OneKefuMessage(kefuID, notification)
	_ = newClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	var received TypeMessage
	if err := newClient.ReadJSON(&received); err != nil {
		t.Fatalf("read new-customer notification: %v", err)
	}
	if received.Type != "userOnline" {
		t.Fatalf("notification type = %v, want userOnline", received.Type)
	}
	t.Log("WGG1 received the simulated new-customer notification without refreshing")
	if err := newClient.Close(); err != nil {
		t.Fatalf("close reconnected WGG1 connection: %v", err)
	}
	waitForKefuConnections(t, kefuID, 0)

	select {
	case err := <-serverErrors:
		t.Fatalf("websocket test server: %v", err)
	default:
	}
}

func TestKefuHeartbeatFanoutWithManyConnections(t *testing.T) {
	const (
		kefuID          = "WGG-load-simulation"
		connectionCount = 50
	)
	connected := make(chan *User, connectionCount)
	serverErrors := make(chan error, connectionCount)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			serverErrors <- err
			return
		}
		user := &User{Id: kefuID, Name: "WGG-load", Conn: conn}
		if err := configureWebSocketHeartbeat(user); err != nil {
			serverErrors <- err
			_ = conn.Close()
			return
		}
		AddKefuToList(user)
		connected <- user
		defer func() {
			RemoveKefuConnection(kefuID, user)
			_ = conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clients := make([]*websocket.Conn, 0, connectionCount)
	for index := 0; index < connectionCount; index++ {
		client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("dial connection %d: %v", index, err)
		}
		clients = append(clients, client)
		<-connected
	}
	t.Cleanup(func() {
		for _, client := range clients {
			_ = client.Close()
		}
	})

	if got := len(KefuConnections(kefuID)); got != connectionCount {
		t.Fatalf("connection count = %d, want %d", got, connectionCount)
	}

	SendPingToKefuClient()
	for index, client := range clients {
		_ = client.SetReadDeadline(time.Now().Add(3 * time.Second))
		var heartbeat TypeMessage
		if err := client.ReadJSON(&heartbeat); err != nil {
			t.Fatalf("read heartbeat for connection %d: %v", index, err)
		}
		if heartbeat.Type != "many pong" {
			t.Fatalf("heartbeat type for connection %d = %v, want many pong", index, heartbeat.Type)
		}
	}

	half := connectionCount / 2
	for index := 0; index < half; index++ {
		if err := clients[index].Close(); err != nil {
			t.Fatalf("close connection %d: %v", index, err)
		}
	}
	waitForKefuConnections(t, kefuID, connectionCount-half)
	if !IsKefuOnline(kefuID) {
		t.Fatal("agent went offline while healthy sessions remained")
	}

	notification, err := json.Marshal(TypeMessage{
		Type: "userOnline",
		Data: map[string]string{"uid": "load-customer", "username": "并发客户"},
	})
	if err != nil {
		t.Fatalf("marshal fanout notification: %v", err)
	}
	OneKefuMessage(kefuID, notification)
	for index := half; index < connectionCount; index++ {
		_ = clients[index].SetReadDeadline(time.Now().Add(3 * time.Second))
		var received TypeMessage
		if err := clients[index].ReadJSON(&received); err != nil {
			t.Fatalf("read notification for connection %d: %v", index, err)
		}
		if received.Type != "userOnline" {
			t.Fatalf("notification type for connection %d = %v, want userOnline", index, received.Type)
		}
	}

	for index := half; index < connectionCount; index++ {
		if err := clients[index].Close(); err != nil {
			t.Fatalf("close connection %d: %v", index, err)
		}
	}
	waitForKefuConnections(t, kefuID, 0)

	select {
	case err := <-serverErrors:
		t.Fatalf("websocket load test server: %v", err)
	default:
	}
}

func waitForKefuConnections(t *testing.T, kefuID string, expected int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(KefuConnections(kefuID)) == expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("connections for %s = %d, want %d", kefuID, len(KefuConnections(kefuID)), expected)
}
