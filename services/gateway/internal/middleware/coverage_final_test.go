package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDefaultRateLimitConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	if cfg.APILimit <= 0 {
		t.Error("expected positive API limit")
	}
}

func TestAPIKeyAuthError_Error(t *testing.T) {
	e := APIKeyAuthError{Reason: "expired", KeyID: "k1"}
	if e.Error() == "" {
		t.Error("expected non-empty error")
	}
}

func TestAPIKeyAuthError_String(t *testing.T) {
	e := APIKeyAuthError{Reason: "invalid", KeyID: "k2"}
	s := e.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestSessionManager_sessionKey(t *testing.T) {
	mgr := NewSessionManager(nil)
	key := mgr.sessionKey("sess-123")
	if key == "" {
		t.Error("expected non-empty session key")
	}
}

func TestSessionManager_writeSessionError(t *testing.T) {
	w := httptest.NewRecorder()
	writeSessionError(w, 401, "expired")
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSessionManager_writeJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSONError(w, 400, "bad")
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSessionManager_touchSessionTTL(t *testing.T) {
	mgr := NewSessionManager(nil)
	mgr.touchSessionTTL("s1")
}

func TestSessionManager_SessionListHandler(t *testing.T) {
	mgr := NewSessionManager(nil)
	req := httptest.NewRequest("GET", "/sessions", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	w := httptest.NewRecorder()
	mgr.SessionListHandler().ServeHTTP(w, req)
}

func TestSessionManager_SessionRevokeHandler(t *testing.T) {
	mgr := NewSessionManager(nil)
	req := httptest.NewRequest("POST", "/sessions/revoke", strings.NewReader(`{"session_id":"s1"}`))
	w := httptest.NewRecorder()
	mgr.SessionRevokeHandler().ServeHTTP(w, req)
}

func TestCacheResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	cw := &cacheResponseWriter{ResponseWriter: w, body: &bytes.Buffer{}}
	cw.WriteHeader(304)
}

func TestCompressResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	cw := &compressResponseWriter{ResponseWriter: w}
	cw.WriteHeader(201)
}

func TestGraphQLHandler_ValidQueryWithBackend(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"1","email":"a@b.com"}]`))
	}))
	defer backend.Close()
	r := NewGraphQLResolver(map[string]string{"users": backend.URL})
	query := `{"query":"{\n  users {\n    id\n    email\n  }\n}"}`
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(query))
	w := httptest.NewRecorder()
	r.GraphQLHandler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGraphQLHandler_WithVariables(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"123"}`))
	}))
	defer backend.Close()
	r := NewGraphQLResolver(map[string]string{"users": backend.URL})
	query := `{"query":"{\n  user(id: \"123\") {\n    id\n  }\n}","variables":{"id":"123"}}`
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(query))
	w := httptest.NewRecorder()
	r.GraphQLHandler().ServeHTTP(w, req)
}

func TestTenantRateLimitHandler_PutWithBody(t *testing.T) {
	store := NewTenantRateLimitStore(100, 10)
	body := `{"requests_per_min": 500, "burst_size": 50, "enabled": true}`
	req := httptest.NewRequest("PUT", "/api/v1/gateway/ratelimits/t1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	TenantRateLimitHandler(store).ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
