package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

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
	<-connected

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

	SendPingToKefuClient()
	_ = newClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	var heartbeat TypeMessage
	if err := newClient.ReadJSON(&heartbeat); err != nil {
		t.Fatalf("read heartbeat: %v", err)
	}
	if heartbeat.Type != "many pong" {
		t.Fatalf("heartbeat type = %v, want many pong", heartbeat.Type)
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
