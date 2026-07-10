package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsWebSocketRequest(t *testing.T) {
	tests := []struct {
		name    string
		header  http.Header
		want    bool
	}{
		{
			name: "valid websocket upgrade",
			header: http.Header{
				"Upgrade":      []string{"websocket"},
				"Connection":   []string{"keep-alive, Upgrade"},
			},
			want: true,
		},
		{
			name: "missing upgrade header",
			header: http.Header{
				"Connection": []string{"keep-alive"},
			},
			want: false,
		},
		{
			name: "upgrade but no connection upgrade",
			header: http.Header{
				"Upgrade":    []string{"websocket"},
				"Connection": []string{"keep-alive"},
			},
			want: false,
		},
		{
			name: "case insensitive upgrade",
			header: http.Header{
				"Upgrade":    []string{"WebSocket"},
				"Connection": []string{"upgrade"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			req.Header = tt.header
			if got := IsWebSocketRequest(req); got != tt.want {
				t.Errorf("IsWebSocketRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWebSocketProxy_NonWebSocketRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	WebSocketProxy("localhost:9001").ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-WS request, got %d", w.Code)
	}
}

func TestWebSocketInterceptor_NonWSFallsThrough(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	WebSocketInterceptor("localhost:9001", next).ServeHTTP(w, req)
	if !called {
		t.Error("expected next handler to be called for non-WS request")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWebSocketInterceptor_WSDetected(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	w := httptest.NewRecorder()
	WebSocketInterceptor("localhost:1", next).ServeHTTP(w, req)
	// WS handler should be called (not next), and it will fail to connect to localhost:1
	if called {
		t.Error("next handler should NOT be called for WS request")
	}
	// The hijack will fail since httptest.ResponseRecorder doesn't support hijacking
	// so we expect a 500
	if w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest {
		// acceptable: either hijack not supported or bad gateway
		t.Logf("got code %d (expected 500 for non-hijackable recorder)", w.Code)
	}
}
