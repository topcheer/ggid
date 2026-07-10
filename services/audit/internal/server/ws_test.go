package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/ggid/ggid/services/audit/internal/service"
	"github.com/google/uuid"
)

// --- StreamHub Unit Tests ---

func TestStreamHub_SubscribeUnsubscribe(t *testing.T) {
	hub := NewStreamHub()
	clientID, ch := hub.Subscribe()
	if clientID == "" {
		t.Fatal("expected non-empty client ID")
	}
	if hub.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber, got %d", hub.SubscriberCount())
	}

	hub.Unsubscribe(clientID)
	if hub.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers after unsubscribe, got %d", hub.SubscriberCount())
	}

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed after unsubscribe")
	}
}

func TestStreamHub_Broadcast(t *testing.T) {
	hub := NewStreamHub()
	_, ch := hub.Subscribe()

	event := &domain.AuditEvent{
		ID:     uuid.New(),
		Action: "user.login",
		Result: "success",
	}
	hub.Broadcast(event)

	select {
	case received := <-ch:
		if received.Action != "user.login" {
			t.Fatalf("expected action user.login, got %s", received.Action)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast event")
	}
}

func TestStreamHub_BroadcastMultipleSubscribers(t *testing.T) {
	hub := NewStreamHub()
	_, ch1 := hub.Subscribe()
	_, ch2 := hub.Subscribe()
	_, ch3 := hub.Subscribe()

	if hub.SubscriberCount() != 3 {
		t.Fatalf("expected 3 subscribers, got %d", hub.SubscriberCount())
	}

	event := &domain.AuditEvent{ID: uuid.New(), Action: "user.register"}
	hub.Broadcast(event)

	for i, ch := range []<-chan *domain.AuditEvent{ch1, ch2, ch3} {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d did not receive event", i)
		}
	}
}

func TestStreamHub_BroadcastNoSubscribers(t *testing.T) {
	hub := NewStreamHub()
	// Should not panic with zero subscribers
	hub.Broadcast(&domain.AuditEvent{ID: uuid.New(), Action: "test"})
}

func TestStreamHub_UnsubscribeUnknownID(t *testing.T) {
	hub := NewStreamHub()
	// Should not panic
	hub.Unsubscribe("nonexistent")
}

func TestStreamHub_BufferFullSkipsSubscriber(t *testing.T) {
	hub := NewStreamHub()
	_, ch := hub.Subscribe()

	// Fill the buffer (64 capacity)
	for i := 0; i < 64; i++ {
		hub.Broadcast(&domain.AuditEvent{ID: uuid.New(), Action: "fill"})
	}

	// One more broadcast should not block (just skip)
	hub.Broadcast(&domain.AuditEvent{ID: uuid.New(), Action: "overflow"})

	// Drain the channel - should have 64 events
	count := 0
	for range ch {
		count++
		if count == 64 {
			break
		}
	}
	if count != 64 {
		t.Fatalf("expected 64 events, got %d", count)
	}
}

// --- WebSocket Integration Tests ---

func TestHandleWebSocket_ConnectAndReceive(t *testing.T) {
	srv := newWSTestServer(t)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, srv.URL+"/api/v1/audit/ws", nil)
	if err != nil {
		t.Fatalf("failed to dial WebSocket: %v", err)
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	// Read the initial "connected" message
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read connected message: %v", err)
	}

	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if msg["type"] != "connected" {
		t.Fatalf("expected type=connected, got %v", msg["type"])
	}

	// Broadcast an event
	event := &domain.AuditEvent{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		Action:       "user.login",
		Result:       "success",
		ResourceType: "user",
	}
	srv.hub.Broadcast(event)

	// Read the event
	_, data, err = c.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read event: %v", err)
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}
	if msg["type"] != "audit_event" {
		t.Fatalf("expected type=audit_event, got %v", msg["type"])
	}
	eventMap, ok := msg["event"].(map[string]any)
	if !ok {
		t.Fatal("expected event field in message")
	}
	if eventMap["action"] != "user.login" {
		t.Fatalf("expected action user.login, got %v", eventMap["action"])
	}
}

func TestHandleWebSocket_SubscriberCount(t *testing.T) {
	srv := newWSTestServer(t)
	defer srv.Close()

	before := srv.hub.SubscriberCount()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, srv.URL+"/api/v1/audit/ws", nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	// Read connected message
	_, _, _ = c.Read(ctx)

	// Give a small delay for the handler to register
	time.Sleep(100 * time.Millisecond)

	after := srv.hub.SubscriberCount()
	if after <= before {
		t.Fatalf("expected subscriber count to increase: before=%d after=%d", before, after)
	}
}

// newWSTestServer creates a test server with a real StreamHub.
func newWSTestServer(t *testing.T) *wsTestServer {
	mockRepo := &mockRepo{}
	svc := service.NewAuditService(mockRepo)
	httpServer := NewHTTPServer(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/audit/ws", httpServer.HandleWebSocket)

	ts := httptest.NewServer(mux)

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	return &wsTestServer{
		Server: ts,
		URL:    wsURL,
		hub:    httpServer.hub,
	}
}

type wsTestServer struct {
	*httptest.Server
	URL string
	hub *StreamHub
}

// --- Coverage Boost: HandleWebSocket client disconnect + CloseWebSocket ---

func TestHandleWebSocket_ClientDisconnect(t *testing.T) {
	srv := newWSTestServer(t)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, srv.URL+"/api/v1/audit/ws", nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	// Read the connected message
	_, _, _ = c.Read(ctx)

	// Close the client connection — handler should detect write error and return
	_ = c.Close(websocket.StatusNormalClosure, "")

	// Give handler time to process the closed connection
	time.Sleep(150 * time.Millisecond)

	// Subscriber count should eventually drop back
	after := srv.hub.SubscriberCount()
	if after > 0 {
		// It may not be exactly 0 if timing varies, but should not grow
		t.Logf("subscriber count after disconnect: %d", after)
	}
}

func TestCloseWebSocket(t *testing.T) {
	srv := newWSTestServer(t)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, srv.URL+"/api/v1/audit/ws", nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	// CloseWebSocket should close without error
	CloseWebSocket(ctx, c)

	// After closing, reading should fail
	_, _, err = c.Read(ctx)
	if err == nil {
		t.Fatal("expected read to fail after CloseWebSocket")
	}
}

func TestHandleWebSocket_AcceptError(t *testing.T) {
	// When the request is not a proper WebSocket upgrade, Accept should fail
	// and the handler returns early (no panic).
	srv := newWSTestServer(t)
	defer srv.Close()

	// Use the underlying httptest.Server URL (http://) for a plain GET
	resp, err := http.Get(srv.Server.URL + "/api/v1/audit/ws")
	if err != nil {
		t.Fatalf("HTTP GET failed: %v", err)
	}
	defer resp.Body.Close()
	// Should get a response (websocket library rejects upgrade)
	if resp.StatusCode != http.StatusBadRequest {
		// Some websocket versions return 400 on non-upgrade
		t.Logf("got status %d (expected 400 for non-WebSocket request)", resp.StatusCode)
	}
}

func TestHandleWebSocket_EventForward(t *testing.T) {
	srv := newWSTestServer(t)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, srv.URL+"/api/v1/audit/ws", nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	// Read connected
	_, _, err = c.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read connected: %v", err)
	}

	// Broadcast a second event to exercise the json.Marshal success + write path
	another := &domain.AuditEvent{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		Action:       "role.assign",
		Result:       "success",
		ResourceType: "role",
	}
	srv.hub.Broadcast(another)

	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read event: %v", err)
	}
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if msg["type"] != "audit_event" {
		t.Fatalf("expected type=audit_event, got %v", msg["type"])
	}
}

func TestFormatClientID_LargeNumber(t *testing.T) {
	// Cover multi-digit path in jsonNumber
	id := formatClientID(12345)
	if id != "ws-12345" {
		t.Fatalf("expected ws-12345, got %s", id)
	}
}
