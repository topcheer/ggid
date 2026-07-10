package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/ggid/ggid/services/audit/internal/domain"
)

// StreamHub manages WebSocket client subscriptions for real-time audit events.
// When a new audit event is recorded, Broadcast sends it to all connected clients.
type StreamHub struct {
	mu          sync.RWMutex
	subscribers map[string]chan *domain.AuditEvent
	nextID      int
}

// NewStreamHub creates a new StreamHub.
func NewStreamHub() *StreamHub {
	return &StreamHub{
		subscribers: make(map[string]chan *domain.AuditEvent),
	}
}

// Subscribe registers a new client and returns its channel + client ID.
// The caller must call Unsubscribe when done.
func (h *StreamHub) Subscribe() (string, <-chan *domain.AuditEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	clientID := formatClientID(h.nextID)
	ch := make(chan *domain.AuditEvent, 64) // buffered to avoid blocking broadcaster
	h.subscribers[clientID] = ch
	return clientID, ch
}

// Unsubscribe removes a client and closes its channel.
func (h *StreamHub) Unsubscribe(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if ch, ok := h.subscribers[clientID]; ok {
		close(ch)
		delete(h.subscribers, clientID)
	}
}

// Broadcast sends an audit event to all connected subscribers.
// Non-blocking: subscribers with full buffers are skipped.
func (h *StreamHub) Broadcast(event *domain.AuditEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ch := range h.subscribers {
		select {
		case ch <- event:
		default:
			// Subscriber buffer full — skip to avoid blocking
		}
	}
}

// SubscriberCount returns the number of connected clients.
func (h *StreamHub) SubscriberCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}

func formatClientID(n int) string {
	return formatClientIDImpl(n)
}

// formatClientIDImpl generates a unique client ID.
func formatClientIDImpl(n int) string {
	return "ws-" + jsonNumber(n)
}

// jsonNumber converts an int to its string representation.
func jsonNumber(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

// HandleWebSocket upgrades an HTTP connection to a WebSocket and streams
// audit events in real-time. The client receives JSON-encoded events.
//
// GET /api/v1/audit/stream (WebSocket upgrade)
func (s *HTTPServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	clientID, eventCh := s.hub.Subscribe()
	defer s.hub.Unsubscribe(clientID)

	ctx := r.Context()

	// Send initial connection acknowledgment
	ack, _ := json.Marshal(map[string]any{
		"type":       "connected",
		"client_id":  clientID,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"message":    "Subscribed to audit event stream",
	})
	_ = c.Write(ctx, websocket.MessageText, ack)

	// Main loop: forward events to WebSocket client
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			data, err := json.Marshal(map[string]any{
				"type":  "audit_event",
				"event": eventToWSJSON(event),
			})
			if err != nil {
				continue
			}
			err = c.Write(ctx, websocket.MessageText, data)
			if err != nil {
				return // client disconnected
			}
		}
	}
}

// eventToWSJSON converts an AuditEvent to a JSON-friendly map.
func eventToWSJSON(e *domain.AuditEvent) map[string]any {
	return map[string]any{
		"id":            e.ID.String(),
		"tenant_id":     e.TenantID.String(),
		"actor_id":      e.ActorID,
		"actor_name":    e.ActorName,
		"action":        e.Action,
		"resource_type": e.ResourceType,
		"resource_id":   e.ResourceID,
		"result":        e.Result,
		"ip_address":    e.IPAddress,
		"user_agent":    e.UserAgent,
		"created_at":    e.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// CloseWebSocket is a test helper that cleanly closes a WebSocket connection.
func CloseWebSocket(ctx context.Context, c *websocket.Conn) {
	_ = c.Close(websocket.StatusNormalClosure, "")
}
