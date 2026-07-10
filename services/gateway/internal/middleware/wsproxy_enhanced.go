package middleware

import (
	"strings"
	"sync"
	"time"
)

// --- WebSocket Subprotocol Negotiation ---

// NegotiateSubprotocol selects the best subprotocol from client-offered
// Sec-WebSocket-Protocol values. If none match the server-supported list,
// an empty string is returned (no subprotocol).
func NegotiateSubprotocol(clientProtos []string, serverProtos []string) string {
	server := make(map[string]bool, len(serverProtos))
	for _, s := range serverProtos {
		server[strings.ToLower(strings.TrimSpace(s))] = true
	}
	for _, c := range clientProtos {
		proto := strings.ToLower(strings.TrimSpace(c))
		if server[proto] {
			return proto
		}
	}
	return ""
}

// ParseSubprotocols splits the Sec-WebSocket-Protocol header value into
// individual protocol names.
func ParseSubprotocols(headerValue string) []string {
	if headerValue == "" {
		return nil
	}
	parts := strings.Split(headerValue, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// WebSocketConfig holds configuration for the enhanced WebSocket proxy.
type WebSocketConfig struct {
	// SupportedSubprotocols is the list of subprotocols the server supports.
	SupportedSubprotocols []string

	// PingInterval controls how often to send WebSocket ping frames.
	// 0 disables keepalive pings.
	PingInterval time.Duration

	// PongTimeout is how long to wait for a pong response before closing.
	PongTimeout time.Duration

	// HandshakeTimeout is the max time for the backend handshake.
	HandshakeTimeout time.Duration
}

// DefaultWebSocketConfig returns sensible defaults.
func DefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		SupportedSubprotocols: []string{"chat", "notification", "graphql-ws"},
		PingInterval:         30 * time.Second,
		PongTimeout:          10 * time.Second,
		HandshakeTimeout:     10 * time.Second,
	}
}

// --- WebSocket Keepalive Manager ---

// WSKeepalive manages ping/pong heartbeats for a WebSocket tunnel.
// It runs in a background goroutine sending periodic pings. If pong
// is not received within PongTimeout, the connection is terminated.
type WSKeepalive struct {
	mu       sync.Mutex
	interval time.Duration
	timeout  time.Duration
	lastPong time.Time
	stop     chan struct{}
}

// NewWSKeepalive creates a keepalive manager with the given interval
// and pong timeout.
func NewWSKeepalive(interval, timeout time.Duration) *WSKeepalive {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &WSKeepalive{
		interval: interval,
		timeout:  timeout,
		lastPong: time.Now(),
		stop:     make(chan struct{}),
	}
}

// Start begins the keepalive loop. onTimeout is called when a pong
// timeout is detected.
func (k *WSKeepalive) Start(onTimeout func()) {
	go func() {
		ticker := time.NewTicker(k.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				k.mu.Lock()
				if time.Since(k.lastPong) > k.interval+k.timeout {
					k.mu.Unlock()
					if onTimeout != nil {
						onTimeout()
					}
					return
				}
				k.mu.Unlock()
			case <-k.stop:
				return
			}
		}
	}()
}

// RecordPong updates the last pong time to now.
func (k *WSKeepalive) RecordPong() {
	k.mu.Lock()
	k.lastPong = time.Now()
	k.mu.Unlock()
}

// Stop terminates the keepalive loop.
func (k *WSKeepalive) Stop() {
	select {
	case <-k.stop:
		// already closed
	default:
		close(k.stop)
	}
}

// LastPong returns the timestamp of the last pong received.
func (k *WSKeepalive) LastPong() time.Time {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.lastPong
}
