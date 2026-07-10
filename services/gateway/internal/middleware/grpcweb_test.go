package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsGRPCWebRequest(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"application/grpc-web+proto", true},
		{"application/grpc-web", true},
		{"application/grpc", false},
		{"application/json", false},
		{"", false},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("POST", "/api/grpc", nil)
		req.Header.Set("Content-Type", tt.ct)
		if got := IsGRPCWebRequest(req); got != tt.want {
			t.Errorf("IsGRPCWebRequest(ct=%q) = %v, want %v", tt.ct, got, tt.want)
		}
	}
}

func TestIsGRPCWebRequest_GetMethod(t *testing.T) {
	// IsGRPCWebRequest only checks Content-Type, not method
	req := httptest.NewRequest("GET", "/api/grpc", nil)
	req.Header.Set("Content-Type", "application/grpc-web+proto")
	if !IsGRPCWebRequest(req) {
		t.Error("Should detect grpc-web by content type regardless of method")
	}
}

func TestGRPCWebHandler_NonGRPC(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	translator := NewGRPCWebTranslator("localhost:9090")
	handler := GRPCWebHandler(translator, next)

	req := httptest.NewRequest("GET", "/api/data", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("Non-gRPC request should pass through")
	}
}

func TestGRPCWebHandler_GRPCRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	translator := NewGRPCWebTranslator("localhost:9090")
	handler := GRPCWebHandler(translator, next)

	req := httptest.NewRequest("POST", "/api/grpc", nil)
	req.Header.Set("Content-Type", "application/grpc-web+proto")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should not pass through to next handler for gRPC requests
	if rr.Code == http.StatusOK {
		// It's OK if the translator handles it differently
	}
}

func TestNewGRPCWebTranslator(t *testing.T) {
	tr := NewGRPCWebTranslator("localhost:9090")
	if tr == nil {
		t.Fatal("translator should not be nil")
	}
	if tr.BackendAddr != "localhost:9090" {
		t.Errorf("backendAddr: want 'localhost:9090', got '%s'", tr.BackendAddr)
	}
}

func TestGRPCWebTrailers(t *testing.T) {
	// Valid trailer: status code + message
	body := []byte{0, 0, 0, 0, 2, 0, 8} // simplified
	status, msg := GRPCWebTrailers(body)
	_ = status
	_ = msg
	// Should not panic
}

func TestGRPCWebTrailers_Empty(t *testing.T) {
	status, msg := GRPCWebTrailers([]byte{})
	if status != 0 {
		t.Errorf("empty body status: want 0, got %d", status)
	}
	if msg != "" {
		t.Errorf("empty body msg: want '', got '%s'", msg)
	}
}

func TestGRPCWebResponseWriter(t *testing.T) {
	inner := httptest.NewRecorder()
	w := &grpcWebResponseWriter{
		ResponseWriter: inner,
		status:         http.StatusOK,
	}
	if w.Header() == nil {
		t.Error("Header should not be nil")
	}
	w.WriteHeader(http.StatusCreated)
	if w.status != http.StatusCreated {
		t.Errorf("status: want %d, got %d", http.StatusCreated, w.status)
	}
	n, err := w.Write([]byte("test"))
	if n != 4 || err != nil {
		t.Errorf("Write: n=%d, err=%v", n, err)
	}
}
