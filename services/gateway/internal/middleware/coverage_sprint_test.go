package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// --- gRPC-Web tests ---

func TestIsGRPCWebRequest_Text(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/policy.Check", nil)
	r.Header.Set("Content-Type", "application/grpc-web+text")
	if !IsGRPCWebRequest(r) {
		t.Error("expected grpc-web request to be detected")
	}
}

func TestIsGRPCWebRequest_Proto(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/policy.Check", nil)
	r.Header.Set("Content-Type", "application/grpc-web+proto")
	if !IsGRPCWebRequest(r) {
		t.Error("expected grpc-web+proto to be detected")
	}
}

func TestIsGRPCWebRequest_Plain(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/policy.Check", nil)
	r.Header.Set("Content-Type", "application/grpc-web")
	if !IsGRPCWebRequest(r) {
		t.Error("expected grpc-web to be detected")
	}
}

func TestIsGRPCWebRequest_NonGRPC(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/users", nil)
	r.Header.Set("Content-Type", "application/json")
	if IsGRPCWebRequest(r) {
		t.Error("json request should not be detected as grpc-web")
	}
}

func TestNewGRPCWebTranslator(t *testing.T) {
	tr := NewGRPCWebTranslator("localhost:9070")
	if tr.BackendAddr != "localhost:9070" {
		t.Errorf("expected localhost:9070, got %s", tr.BackendAddr)
	}
}

func TestGRPCWebHandler_PassThrough(t *testing.T) {
	tr := NewGRPCWebTranslator("localhost:9070")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	handler := GRPCWebHandler(tr, next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/users", nil)
	r.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(w, r)

	if w.Body.String() != "ok" {
		t.Errorf("expected ok, got %s", w.Body.String())
	}
}

func TestGRPCWebHandler_TextMode(t *testing.T) {
	tr := NewGRPCWebTranslator("localhost:9070")
	grpcResponse := []byte("gRPC response data")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/grpc" {
			t.Errorf("expected grpc content-type, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(grpcResponse)
	})
	handler := GRPCWebHandler(tr, next)

	encodedBody := base64.StdEncoding.EncodeToString([]byte("request data"))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/policy.Check", strings.NewReader(encodedBody))
	r.Header.Set("Content-Type", "application/grpc-web+text")
	handler.ServeHTTP(w, r)

	if w.Body.Len() == 0 {
		t.Error("expected non-empty response")
	}
}

func TestGRPCWebHandler_ProtoMode(t *testing.T) {
	tr := NewGRPCWebTranslator("localhost:9070")
	grpcResponse := []byte("binary gRPC response")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(grpcResponse)
	})
	handler := GRPCWebHandler(tr, next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/policy.Check", strings.NewReader("binary request"))
	r.Header.Set("Content-Type", "application/grpc-web+proto")
	handler.ServeHTTP(w, r)

	if w.Body.Len() == 0 {
		t.Error("expected non-empty response")
	}
}

func TestGRPCWebHandler_NilBody(t *testing.T) {
	tr := NewGRPCWebTranslator("localhost:9070")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := GRPCWebHandler(tr, next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/policy.Check", nil)
	r.Header.Set("Content-Type", "application/grpc-web")
	handler.ServeHTTP(w, r)
}

func TestGRPCWebTrailers_Valid(t *testing.T) {
	body := []byte("data\x80grpc-status: 0\r\ngrpc-message: OK\r\n")
	status, msg := GRPCWebTrailers(body)
	if status != 0 {
		t.Errorf("expected status 0, got %d", status)
	}
	if msg != "OK" {
		t.Errorf("expected OK, got %s", msg)
	}
}

func TestGRPCWebTrailers_ErrorStatus(t *testing.T) {
	body := []byte("data\x80grpc-status: 13\r\ngrpc-message: internal error\r\n")
	status, msg := GRPCWebTrailers(body)
	if status != 13 {
		t.Errorf("expected status 13, got %d", status)
	}
	if !strings.Contains(msg, "internal error") {
		t.Errorf("expected internal error, got %s", msg)
	}
}

func TestGRPCWebTrailers_TooShort(t *testing.T) {
	body := []byte("x")
	status, _ := GRPCWebTrailers(body)
	if status != 0 {
		t.Errorf("expected status 0 for too short body, got %d", status)
	}
}

func TestGRPCWebTrailers_NoTrailer(t *testing.T) {
	body := []byte("just some data without trailer")
	status, msg := GRPCWebTrailers(body)
	if status != 0 || msg != "" {
		t.Errorf("expected 0/empty for no trailer, got %d/%s", status, msg)
	}
}

func TestGRPCWebTrailers_Empty(t *testing.T) {
	status, msg := GRPCWebTrailers([]byte{})
	if status != 0 || msg != "" {
		t.Errorf("expected 0/empty, got %d/%s", status, msg)
	}
}

// --- apiScopeCtxKey String test ---

func TestAPIScopeCtxKey_String(t *testing.T) {
	k := apiScopeCtxKey("test_key")
	if k.String() != "test_key" {
		t.Errorf("expected test_key, got %s", k.String())
	}
}

// --- Gzip WriteHeader test ---

func TestGzipResponseWriter_WriteHeader(t *testing.T) {
	inner := httptest.NewRecorder()
	gw := &gzipResponseWriter{
		ResponseWriter: inner,
	}

	gw.WriteHeader(http.StatusNotFound)

	if inner.Code != http.StatusNotFound {
		t.Errorf("expected 404 forwarded, got %d", inner.Code)
	}
}

func TestGzipResponseWriter_WriteHeader_SkipCompression(t *testing.T) {
	inner := httptest.NewRecorder()
	gw := &gzipResponseWriter{
		ResponseWriter: inner,
	}
	gw.Header().Set("Content-Type", "image/png")

	gw.WriteHeader(http.StatusOK)

	if inner.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", inner.Code)
	}
}

// --- Session handler tests (nil redis paths) ---

func TestSessionListHandler_NoUserID(t *testing.T) {
	sm := NewSessionManager(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/sessions", nil)
	sm.SessionListHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSessionRevokeHandler_NoUserID(t *testing.T) {
	sm := NewSessionManager(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/api/v1/sessions/550e8400-e29b-41d4-a716-446655440000", nil)
	sm.SessionRevokeHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSessionRevokeHandler_BadSessionID(t *testing.T) {
	sm := NewSessionManager(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/api/v1/sessions/not-a-uuid", nil)

	uid := uuid.New()
	ctx := context.WithValue(r.Context(), UserIDKey, uid.String())
	r = r.WithContext(ctx)

	sm.SessionRevokeHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad session ID, got %d", w.Code)
	}
}

func TestSessionRevokeHandler_WrongMethod(t *testing.T) {
	sm := NewSessionManager(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/sessions/123", nil)
	sm.SessionRevokeHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}
