package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- gzip.go coverage ---

func TestGzip_NoAcceptEncoding_C17(t *testing.T) {
	called := false
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte("response"))
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("should pass through without gzip")
	}
	if rr.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not set Content-Encoding: gzip")
	}
}

func TestGzip_WithAcceptEncoding_C17(t *testing.T) {
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(strings.Repeat("hello world ", 100)))
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Error("expected Content-Encoding: gzip")
	}
	if rr.Body.Len() == 0 {
		t.Error("expected non-empty compressed body")
	}
}

func TestGzip_SkipsBinaryContent_C17(t *testing.T) {
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("binary-image-data"))
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not gzip binary content")
	}
}

func TestGzipBrotli_Alias_C17(t *testing.T) {
	called := false
	h := GzipBrotli(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte("ok"))
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("GzipBrotli should behave like Gzip")
	}
}

func TestShouldSkipCompression_C17(t *testing.T) {
	skip := []string{"image/png", "video/mp4", "audio/mpeg", "application/zip", "application/pdf", "application/octet-stream"}
	noskip := []string{"", "text/plain", "application/json", "text/html; charset=utf-8"}
	for _, ct := range skip {
		if !shouldSkipCompression(ct) {
			t.Errorf("shouldSkipCompression(%q) = false, want true", ct)
		}
	}
	for _, ct := range noskip {
		if shouldSkipCompression(ct) {
			t.Errorf("shouldSkipCompression(%q) = true, want false", ct)
		}
	}
	if !shouldSkipCompression("IMAGE/PNG") {
		t.Error("should be case insensitive")
	}
}

func TestGzipResponseWriter_MultipleWrites_C17(t *testing.T) {
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("chunk1-"))
		w.Write([]byte("chunk2-"))
		w.Write([]byte("chunk3"))
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	h.ServeHTTP(rr, req)
	// Verify gzip encoding header was set (content was compressed)
	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("expected Content-Encoding: gzip")
	}
	// Verify body is non-empty (compressed data present)
	if rr.Body.Len() == 0 {
		t.Error("expected non-empty compressed body")
	}
}

// --- grpc.go coverage ---

// FIX #3: isGRPCRequest uses strings.HasPrefix(ct, "application/grpc")
// which returns true for "application/grpc-web" as well.
func TestIsGRPCRequest_C17(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"application/grpc", true},
		{"application/grpc+proto", true},
		{"application/grpc-web", true}, // HasPrefix("application/grpc") matches!
		{"application/grpc-web+text", true},
		{"application/json", false},
		{"", false},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("POST", "/svc.Method", nil)
		if tt.ct != "" {
			req.Header.Set("Content-Type", tt.ct)
		}
		if got := isGRPCRequest(req); got != tt.want {
			t.Errorf("isGRPCRequest(ct=%q) = %v, want %v", tt.ct, got, tt.want)
		}
	}
}

func TestExtractGRPCService_C17(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/package.Service/Method", "package.Service"},
		{"/myapp.UserService/GetUser", "myapp.UserService"},
		{"/Simple/Method", "Simple"},
		{"/invalid", "invalid"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := extractGRPCService(tt.path); got != tt.want {
			t.Errorf("extractGRPCService(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

// FIX #2: GRPCProxy has no `backends` field — use NewGRPCProxy with GRPCProxyConfig.Backends
func TestGRPCHTTPHandler_NonGRPC_C17(t *testing.T) {
	proxy := NewGRPCProxy(GRPCProxyConfig{
		Backends: make(map[string]string),
	})
	called := false
	h := proxy.GRPCHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("non-gRPC request should pass through")
	}
}

func TestGRPCHTTPHandler_GRPCNoBackend_C17(t *testing.T) {
	proxy := NewGRPCProxy(GRPCProxyConfig{
		Backends: make(map[string]string),
	})
	h := proxy.GRPCHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called")
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/pkg.Service/Method", nil)
	req.Header.Set("Content-Type", "application/grpc")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadGateway)
	}
}

func TestGRPCHTTPHandler_NilNext_C17(t *testing.T) {
	proxy := NewGRPCProxy(GRPCProxyConfig{
		Backends: make(map[string]string),
	})
	h := proxy.GRPCHTTPHandler(nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req) // should not panic
}

func TestGRPCProxy_GetBackend_C17(t *testing.T) {
	proxy := NewGRPCProxy(GRPCProxyConfig{
		Backends: map[string]string{
			"pkg.Service": "localhost:9070",
		},
	})
	if addr := proxy.GetBackend("pkg.Service"); addr != "localhost:9070" {
		t.Errorf("GetBackend = %q", addr)
	}
	if addr := proxy.GetBackend("unknown"); addr != "" {
		t.Errorf("GetBackend unknown = %q, want empty", addr)
	}
}
