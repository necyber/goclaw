package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/logger"
	"github.com/gorilla/websocket"
)

func testWSLogger() logger.Logger {
	return logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "json",
		Output: "stdout",
	})
}

func wsURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

func TestWebSocketHandler_RejectsNonUpgrade(t *testing.T) {
	handler := NewWebSocketHandler(testWSLogger(), WebSocketConfig{})

	req := httptest.NewRequest(http.MethodGet, "/ws/events", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestWebSocketHandler_SubscribeAndBroadcast(t *testing.T) {
	handler := NewWebSocketHandler(testWSLogger(), WebSocketConfig{
		MaxConnections: 5,
	})

	server := httptest.NewServer(handler)
	defer server.Close()
	defer handler.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type":        "subscribe",
		"workflow_id": "wf-1",
	}); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	if err := handler.Broadcast(EventMessage{
		Type: "workflow.state_changed",
		Payload: map[string]any{
			"workflow_id": "wf-1",
			"new_state":   "running",
		},
	}); err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var got EventMessage
	if err := conn.ReadJSON(&got); err != nil {
		t.Fatalf("failed to read broadcast event: %v", err)
	}
	if got.Type != "workflow.state_changed" {
		t.Fatalf("type = %q, want workflow.state_changed", got.Type)
	}
}

func TestWebSocketHandler_ConnectionLimit(t *testing.T) {
	handler := NewWebSocketHandler(testWSLogger(), WebSocketConfig{
		MaxConnections: 1,
	})

	server := httptest.NewServer(handler)
	defer server.Close()
	defer handler.Close()

	first, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("failed to open first websocket: %v", err)
	}
	defer first.Close()

	_, resp, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err == nil {
		t.Fatal("expected second websocket dial to fail")
	}
	var handshakeErr websocket.HandshakeError
	if !errors.As(err, &handshakeErr) {
		t.Logf("dial returned non-handshake error type: %T", err)
	}
	if resp == nil {
		t.Fatal("expected HTTP response for failed upgrade")
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestWebSocketHandler_OriginCheck(t *testing.T) {
	handler := NewWebSocketHandler(testWSLogger(), WebSocketConfig{
		AllowedOrigins: []string{"http://allowed.example"},
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	defer handler.Close()

	dialer := websocket.Dialer{}
	headers := http.Header{}
	headers.Set("Origin", "http://blocked.example")

	_, resp, err := dialer.Dial(wsURL(server.URL), headers)
	if err == nil {
		t.Fatal("expected websocket dial with blocked origin to fail")
	}
	if resp == nil {
		t.Fatal("expected HTTP response for blocked origin")
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestConnectionManager_RegisterUnregisterBroadcast(t *testing.T) {
	manager := NewConnectionManager(2)
	clientA := newWSClient(nil)
	clientB := newWSClient(nil)

	clientA.subscribe("wf-1")

	if err := manager.Register(clientA); err != nil {
		t.Fatalf("register clientA failed: %v", err)
	}
	if err := manager.Register(clientB); err != nil {
		t.Fatalf("register clientB failed: %v", err)
	}
	if manager.Count() != 2 {
		t.Fatalf("count = %d, want 2", manager.Count())
	}

	eventWf1 := EventMessage{
		Type: "task.state_changed",
		Payload: map[string]any{
			"workflow_id": "wf-1",
		},
	}
	if err := manager.Broadcast(eventWf1); err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}

	select {
	case <-clientA.send:
	case <-time.After(time.Second):
		t.Fatal("expected subscribed clientA to receive wf-1 event")
	}
	select {
	case <-clientB.send:
	case <-time.After(time.Second):
		t.Fatal("expected global clientB to receive wf-1 event")
	}

	eventWf2 := EventMessage{
		Type: "task.state_changed",
		Payload: map[string]any{
			"workflow_id": "wf-2",
		},
	}
	if err := manager.Broadcast(eventWf2); err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}

	select {
	case <-clientA.send:
		t.Fatal("did not expect clientA subscription to receive wf-2 event")
	case <-time.After(200 * time.Millisecond):
	}
	select {
	case <-clientB.send:
	case <-time.After(time.Second):
		t.Fatal("expected global clientB to receive wf-2 event")
	}

	manager.Unregister(clientA)
	if manager.Count() != 1 {
		t.Fatalf("count after unregister = %d, want 1", manager.Count())
	}
}

func TestEventMessageJSONFormat(t *testing.T) {
	event := EventMessage{
		Type:      "workflow.state_changed",
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"workflow_id": "wf-1",
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := decoded["type"]; !ok {
		t.Fatal("missing type field")
	}
	if _, ok := decoded["timestamp"]; !ok {
		t.Fatal("missing timestamp field")
	}
	if _, ok := decoded["payload"]; !ok {
		t.Fatal("missing payload field")
	}
}
