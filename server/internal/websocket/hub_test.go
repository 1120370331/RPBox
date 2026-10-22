package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestStaleDisconnectDoesNotCloseReplacementConnection(t *testing.T) {
	connections := make(chan *websocket.Conn, 3)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			connections <- conn
		}
	}))
	defer server.Close()
	newClient := func(userID uint) *Client {
		t.Helper()
		peer, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = peer.Close() })
		conn := <-connections
		t.Cleanup(func() { _ = conn.Close() })
		return &Client{UserID: userID, Conn: conn, Send: make(chan []byte, 4)}
	}
	hub := NewHub()
	go hub.Run()
	old, replacement, barrier := newClient(1), newClient(1), newClient(2)
	hub.Register(old)
	hub.Register(replacement)
	hub.Unregister(old)
	// Receipt of this registration proves the old disconnect has been processed.
	hub.Register(barrier)
	if !hub.IsUserOnline(1) {
		t.Fatal("old disconnect removed the replacement")
	}
	if err := hub.SendToUser(1, MessageTypeUnreadCountUpdate, 7); err != nil {
		t.Fatal(err)
	}
	select {
	case message, ok := <-replacement.Send:
		if !ok || !strings.Contains(string(message), `"data":7`) {
			t.Fatalf("replacement did not receive message: %s", message)
		}
	case <-time.After(time.Second):
		t.Fatal("replacement notification timed out")
	}
	hub.Unregister(replacement)
	hub.Unregister(replacement)
	hub.Unregister(barrier)
}
