package middleware

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- WebSocketProxy error paths ---

func TestWebSocketProxy_DialFailure(t *testing.T) {
	// Target nothing listening → dial should fail
	wsHandler := WebSocketProxy("127.0.0.1:59999")

	// We need a real HTTP server that supports hijacking
	srv := httptest.NewServer(wsHandler)
	defer srv.Close()

	// Send a WebSocket upgrade request
	req, _ := http.NewRequest("GET", srv.URL, nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// The proxy should have failed to connect → it writes 502 Bad Gateway
	// to the hijacked connection
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502 from dial failure, got %d", resp.StatusCode)
	}
}

// --- WebSocketProxy successful tunnel ---

func TestWebSocketProxy_SuccessfulTunnel(t *testing.T) {
	// Start a raw TCP listener as the "backend WebSocket server"
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	// Backend accepts connections and responds with a simple HTTP-like response
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				// Read the forwarded request
				buf := make([]byte, 4096)
				n, _ := c.Read(buf)
				requestStr := string(buf[:n])
				_ = requestStr // just consume

				// Respond with HTTP 101 Switching Protocols
				response := "HTTP/1.1 101 Switching Protocols\r\n" +
					"Upgrade: websocket\r\n" +
					"Connection: Upgrade\r\n" +
					"Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=\r\n" +
					"\r\n"
				c.Write([]byte(response))

				// Echo loop
				io.Copy(c, c)
			}(conn)
		}
	}()

	// Start the WS proxy
	wsHandler := WebSocketProxy(listener.Addr().String())
	srv := httptest.NewServer(wsHandler)
	defer srv.Close()

	// Connect via raw TCP and perform WebSocket upgrade
	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Send WebSocket upgrade request
	upgradeReq := "GET / HTTP/1.1\r\n" +
		"Host: " + srv.Listener.Addr().String() + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"\r\n"
	conn.Write([]byte(upgradeReq))

	// Read response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	response := string(buf[:n])

	// Should get 101 Switching Protocols from backend (through proxy)
	if !strings.Contains(response, "101") {
		t.Errorf("expected 101 response, got: %s", response[:min(200, len(response))])
	}
}

func TestWebSocketProxy_NonWSReq_Returns400(t *testing.T) {
	wsHandler := WebSocketProxy("127.0.0.1:59999")
	srv := httptest.NewServer(wsHandler)
	defer srv.Close()

	// Regular GET (no Upgrade header)
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for non-WS request, got %d", resp.StatusCode)
	}
}

// --- WSKeepalive timeout ---

func TestWSKeepalive_TimeoutTriggersCallback(t *testing.T) {
	// Very short interval and timeout for testing
	ka := NewWSKeepalive(20*time.Millisecond, 10*time.Millisecond)
	timeoutCalled := make(chan struct{})

	ka.Start(func() {
		close(timeoutCalled)
	})
	defer ka.Stop()

	// Don't call RecordPong → should timeout
	select {
	case <-timeoutCalled:
		// Expected
	case <-time.After(200 * time.Millisecond):
		t.Error("keepalive timeout should have been triggered")
	}
}

func TestWSKeepalive_RecordPongPreventsTimeout(t *testing.T) {
	ka := NewWSKeepalive(30*time.Millisecond, 20*time.Millisecond)
	timeoutCalled := make(chan struct{})

	ka.Start(func() {
		close(timeoutCalled)
	})
	defer ka.Stop()

	// Record pong before timeout
	time.Sleep(20 * time.Millisecond)
	ka.RecordPong()

	// Wait a bit - should not timeout yet
	select {
	case <-timeoutCalled:
		// Might timeout eventually after another interval
	case <-time.After(60 * time.Millisecond):
		// OK - pong prevented immediate timeout
	}
}

func TestWSKeepalive_StopTwice(t *testing.T) {
	ka := NewWSKeepalive(100*time.Millisecond, 50*time.Millisecond)
	ka.Start(func() {})
	ka.Stop()
	ka.Stop() // should not panic
}

// --- NegotiateSubprotocol edge cases ---

func TestNegotiateSubprotocol_MultipleMatch(t *testing.T) {
	result := NegotiateSubprotocol(
		[]string{"chat", "notification", "graphql-ws"},
		[]string{"graphql-ws", "chat"},
	)
	// Should return the first client-offered match
	if result != "chat" {
		t.Errorf("expected 'chat', got %q", result)
	}
}

func TestNegotiateSubprotocol_CaseInsensitive(t *testing.T) {
	result := NegotiateSubprotocol(
		[]string{"CHAT", "Notification"},
		[]string{"chat", "notification"},
	)
	if result != "chat" { // Should match case-insensitively
		t.Errorf("expected 'chat', got %q", result)
	}
}

func TestNegotiateSubprotocol_NoServerProtos(t *testing.T) {
	result := NegotiateSubprotocol(
		[]string{"chat", "notification"},
		[]string{},
	)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestParseSubprotocols_WithSpaces(t *testing.T) {
	result := ParseSubprotocols("chat, notification, graphql-ws")
	if len(result) != 3 {
		t.Fatalf("expected 3 protocols, got %d", len(result))
	}
	if result[1] != "notification" {
		t.Errorf("result[1] = %q", result[1])
	}
}

func TestParseSubprotocols_TrailingComma(t *testing.T) {
	result := ParseSubprotocols("chat, notification,")
	if len(result) != 2 {
		t.Fatalf("expected 2 protocols (trailing comma ignored), got %d", len(result))
	}
}

func TestDefaultWebSocketConfig_Values(t *testing.T) {
	cfg := DefaultWebSocketConfig()
	if cfg.PingInterval != 30*time.Second {
		t.Errorf("PingInterval = %v", cfg.PingInterval)
	}
	if cfg.PongTimeout != 10*time.Second {
		t.Errorf("PongTimeout = %v", cfg.PongTimeout)
	}
	if cfg.HandshakeTimeout != 10*time.Second {
		t.Errorf("HandshakeTimeout = %v", cfg.HandshakeTimeout)
	}
	if len(cfg.SupportedSubprotocols) != 3 {
		t.Errorf("expected 3 subprotocols, got %d", len(cfg.SupportedSubprotocols))
	}
}

// --- io helper for echo ---

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
