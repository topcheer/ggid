package middleware

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- gRPC-Web coverage ---

func TestIsGRPCWebRequest_Variants(t *testing.T) {
	cases := []struct {
		ct   string
		want bool
	}{
		{"application/grpc-web", true},
		{"application/grpc-web+text", true},
		{"application/grpc-web+proto", true},
		{"application/json", false},
		{"", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Set("Content-Type", c.ct)
		if got := IsGRPCWebRequest(r); got != c.want {
			t.Errorf("IsGRPCWebRequest(ct=%q) = %v, want %v", c.ct, got, c.want)
		}
	}
}

func TestNewGRPCWebTranslator(t *testing.T) {
	tr := NewGRPCWebTranslator("localhost:9070")
	if tr.BackendAddr != "localhost:9070" {
		t.Errorf("expected localhost:9070, got %s", tr.BackendAddr)
	}
}

func TestGRPCWebHandler_NonGRPC_PassThrough(t *testing.T) {
	tr := NewGRPCWebTranslator("localhost:9070")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	h := GRPCWebHandler(tr, next)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)
	if w.Body.String() != "ok" {
		t.Errorf("expected ok, got %s", w.Body.String())
	}
}

func TestGRPCWebHandler_TextMode(t *testing.T) {
	tr := NewGRPCWebTranslator("localhost:9070")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("grpc-response"))
	})
	h := GRPCWebHandler(tr, next)

	encodedBody := base64.StdEncoding.EncodeToString([]byte("request"))
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/svc.Method", strings.NewReader(encodedBody))
	r.Header.Set("Content-Type", "application/grpc-web+text")
	h.ServeHTTP(w, r)
	if w.Body.Len() == 0 {
		t.Error("expected non-empty response")
	}
}

func TestGRPCWebHandler_ProtoMode(t *testing.T) {
	tr := NewGRPCWebTranslator("localhost:9070")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("binary"))
	})
	h := GRPCWebHandler(tr, next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/svc.Method", strings.NewReader("request"))
	r.Header.Set("Content-Type", "application/grpc-web+proto")
	h.ServeHTTP(w, r)
	if w.Body.Len() == 0 {
		t.Error("expected non-empty response")
	}
}

func TestGRPCWebHandler_NilBody(t *testing.T) {
	tr := NewGRPCWebTranslator("localhost:9070")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := GRPCWebHandler(tr, next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/svc.Method", nil)
	r.Header.Set("Content-Type", "application/grpc-web")
	h.ServeHTTP(w, r)
}

func TestGRPCWebTrailers(t *testing.T) {
	cases := []struct {
		name       string
		body       []byte
		wantStatus int
		wantMsg    string
	}{
		{"valid_ok", []byte("data\x80grpc-status: 0\r\ngrpc-message: OK\r\n"), 0, "OK"},
		{"valid_error", []byte("data\x80grpc-status: 13\r\ngrpc-message: internal\r\n"), 13, "internal"},
		{"too_short", []byte("x"), 0, ""},
		{"no_trailer", []byte("just data"), 0, ""},
		{"empty", []byte{}, 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, msg := GRPCWebTrailers(c.body)
			if status != c.wantStatus {
				t.Errorf("status = %d, want %d", status, c.wantStatus)
			}
			if msg != c.wantMsg {
				t.Errorf("msg = %q, want %q", msg, c.wantMsg)
			}
		})
	}
}

func TestGRPCWebResponseWriter(t *testing.T) {
	inner := httptest.NewRecorder()
	g := &grpcWebResponseWriter{ResponseWriter: inner, headers: http.Header{}}

	g.WriteHeader(http.StatusCreated)
	if g.status != http.StatusCreated {
		t.Errorf("expected 201, got %d", g.status)
	}

	n, err := g.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Errorf("Write failed: n=%d, err=%v", n, err)
	}
	if string(g.body) != "hello" {
		t.Errorf("expected hello in body, got %s", g.body)
	}

	// Header() should return the internal header map
	h := g.Header()
	h.Set("X-Custom", "test")
	if g.headers.Get("X-Custom") != "test" {
		t.Error("Header() should return internal map")
	}
}

func TestGRPCWebResponseWriter_NilHeaders(t *testing.T) {
	inner := httptest.NewRecorder()
	g := &grpcWebResponseWriter{ResponseWriter: inner}
	// When headers is nil, should fall back to ResponseWriter's Header
	h := g.Header()
	h.Set("X-Test", "val")
	if inner.Header().Get("X-Test") != "val" {
		t.Error("expected fallback to ResponseWriter.Header()")
	}
}

// --- gzip WriteHeader coverage ---

func TestGzipResponseWriter_WriteHeader_Normal(t *testing.T) {
	inner := httptest.NewRecorder()
	gw := &compressWriter{ResponseWriter: inner}
	gw.WriteHeader(http.StatusTeapot)
	if inner.Code != http.StatusTeapot {
		t.Errorf("expected 418, got %d", inner.Code)
	}
}

func TestGzipResponseWriter_WriteHeader_SkipCompression(t *testing.T) {
	inner := httptest.NewRecorder()
	gw := &compressWriter{ResponseWriter: inner}
	gw.Header().Set("Content-Type", "image/png")
	gw.WriteHeader(http.StatusOK)
	if inner.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", inner.Code)
	}
}
