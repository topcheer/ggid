package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- test helpers ---

func setTestUserID_C17(parent context.Context, userID string) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, UserIDKey, userID)
}

func setTestSessionID_C17(parent context.Context, sessionID string) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, SessionIDKey, sessionID)
}

// --- session.go coverage ---

func TestSessionKey_C17(t *testing.T) {
	got := sessionKey("abc-123")
	want := "ggid:session:abc-123"
	if got != want {
		t.Errorf("sessionKey = %q, want %q", got, want)
	}
}

func TestWriteSessionError_C17(t *testing.T) {
	rr := httptest.NewRecorder()
	writeSessionError(rr, "session expired")
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "session expired" {
		t.Errorf("error = %q", body["error"])
	}
}

func TestWriteJSONError_C17(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSONError(rr, http.StatusServiceUnavailable, "unavailable")
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestSessionListHandler_NoUser_C17(t *testing.T) {
	sm := NewSessionManager(nil)
	h := sm.SessionListHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called")
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/sessions", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestSessionListHandler_NilRedis_C17(t *testing.T) {
	// FIX: UserIDFromRequest calls uuid.Parse — must pass valid UUID to get past auth check.
	sm := NewSessionManager(nil)
	h := sm.SessionListHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called")
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/sessions", nil)
	ctx := setTestUserID_C17(req.Context(), "550e8400-e29b-41d4-a716-446655440000")
	req = req.WithContext(ctx)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestSessionRevokeHandler_BadMethod_C17(t *testing.T) {
	sm := NewSessionManager(nil)
	h := sm.SessionRevokeHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/sessions/123", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestSessionRevokeHandler_NoUser_C17(t *testing.T) {
	sm := NewSessionManager(nil)
	h := sm.SessionRevokeHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/sessions/123", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestSessionRevokeHandler_InvalidUUID_C17(t *testing.T) {
	sm := NewSessionManager(nil)
	h := sm.SessionRevokeHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/sessions/not-a-uuid", nil)
	ctx := setTestUserID_C17(req.Context(), "550e8400-e29b-41d4-a716-446655440000")
	req = req.WithContext(ctx)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestSessionRevokeHandler_ValidUUID_NilRedis_C17(t *testing.T) {
	sm := NewSessionManager(nil)
	h := sm.SessionRevokeHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/sessions/550e8400-e29b-41d4-a716-446655440000", nil)
	ctx := setTestUserID_C17(req.Context(), "550e8400-e29b-41d4-a716-446655440000")
	req = req.WithContext(ctx)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestSessionMiddleware_PublicPath_C17(t *testing.T) {
	sm := NewSessionManager(nil)
	called := false
	h := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/verify", nil)
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("public path should pass through")
	}
}

func TestSessionMiddleware_Healthz_C17(t *testing.T) {
	sm := NewSessionManager(nil)
	called := false
	h := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("healthz should be public")
	}
}

func TestSessionMiddleware_NilRedis_C17(t *testing.T) {
	sm := NewSessionManager(nil)
	called := false
	h := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("X-Session-ID", "sess-123")
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("nil Redis should pass through")
	}
}

func TestSessionIDFromContext_C17(t *testing.T) {
	ctx := setTestSessionID_C17(nil, "test-session")
	id, ok := SessionIDFromContext(ctx)
	if !ok || id != "test-session" {
		t.Errorf("SessionIDFromContext = %q, ok=%v", id, ok)
	}
}

func TestTouchSessionTTL_NilRedis_C17(t *testing.T) {
	sm := NewSessionManager(nil)
	sm.touchSessionTTL(nil, "sess-123", time.Minute)
}

// --- grpcweb.go coverage ---

func TestGRPCWebTrailers_Status_C17(t *testing.T) {
	trailer := append([]byte{0x80}, []byte("grpc-status: 5\r\ngrpc-message: not found\r\n")...)
	status, msg := GRPCWebTrailers(trailer)
	if status != 5 {
		t.Errorf("status = %d, want 5", status)
	}
	if msg != "not found" {
		t.Errorf("message = %q, want %q", msg, "not found")
	}
}

func TestGRPCWebTrailers_BinaryFrame_C17(t *testing.T) {
	trailer := append([]byte{0x00}, []byte("grpc-status: 14\r\ngrpc-message: unavailable\r\n")...)
	status, msg := GRPCWebTrailers(trailer)
	if status != 14 {
		t.Errorf("status = %d, want 14", status)
	}
	if msg != "unavailable" {
		t.Errorf("message = %q", msg)
	}
}

func TestGRPCWebTrailers_NoTrailer_C17(t *testing.T) {
	status, msg := GRPCWebTrailers([]byte{0x01, 0x02, 0x03})
	if status != 0 || msg != "" {
		t.Errorf("expected 0/empty, got %d/%q", status, msg)
	}
}

func TestGRPCWebResponseWriter_Header_C17(t *testing.T) {
	rec := &grpcWebResponseWriter{
		headers: http.Header{"X-Custom": []string{"val"}},
	}
	h := rec.Header()
	if h.Get("X-Custom") != "val" {
		t.Error("expected custom header from headers field")
	}
}

func TestGRPCWebResponseWriter_WriteHeader_C17(t *testing.T) {
	rr := httptest.NewRecorder()
	rec := &grpcWebResponseWriter{ResponseWriter: rr}
	rec.WriteHeader(http.StatusTeapot)
	if rec.status != http.StatusTeapot {
		t.Errorf("status = %d, want %d", rec.status, http.StatusTeapot)
	}
}

func TestGRPCWebResponseWriter_Write_C17(t *testing.T) {
	rec := &grpcWebResponseWriter{}
	n, err := rec.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Errorf("Write returned %d, %v", n, err)
	}
	if string(rec.body) != "hello" {
		t.Errorf("body = %q", rec.body)
	}
}

func TestGRPCWebHandler_TextMode_C17(t *testing.T) {
	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("grpc-response-data"))
	})
	translator := NewGRPCWebTranslator("localhost:9070")
	h := GRPCWebHandler(translator, backend)

	encoded := "aGVsbG8=" // base64("hello")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/grpc/service.Method", strings.NewReader(encoded))
	req.Header.Set("Content-Type", "application/grpc-web+text")
	h.ServeHTTP(rr, req)

	body := rr.Body.String()
	if body == "" {
		t.Error("expected non-empty response")
	}
}

func TestGRPCWebHandler_ProtoMode_C17(t *testing.T) {
	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("binary-response"))
	})
	translator := NewGRPCWebTranslator("localhost:9070")
	h := GRPCWebHandler(translator, backend)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/grpc/service.Method", strings.NewReader("request-body"))
	req.Header.Set("Content-Type", "application/grpc-web+proto")
	h.ServeHTTP(rr, req)

	body := rr.Body.String()
	if body == "" {
		t.Error("expected non-empty response")
	}
}

// --- graphql.go coverage ---

func TestSubstituteVariables_C17(t *testing.T) {
	field := graphqlField{
		Name: "user",
		Type: "users",
		Path: "/api/v1/users/$id",
	}
	result := substituteVariables(field, map[string]any{"id": 42})
	if result.Path != "/api/v1/users/42" {
		t.Errorf("Path = %q, want /api/v1/users/42", result.Path)
	}
}

func TestSubstituteVariables_NoVars_C17(t *testing.T) {
	field := graphqlField{Path: "/api/v1/users/1"}
	result := substituteVariables(field, nil)
	if result.Path != "/api/v1/users/1" {
		t.Errorf("Path = %q", result.Path)
	}
}

func TestSubstituteVariables_MultipleVars_C17(t *testing.T) {
	field := graphqlField{Path: "/api/v1/orgs/$org/users/$user"}
	result := substituteVariables(field, map[string]any{
		"org":  "acme",
		"user": "john",
	})
	if result.Path != "/api/v1/orgs/acme/users/john" {
		t.Errorf("Path = %q", result.Path)
	}
}

func TestInlineFragments_C17(t *testing.T) {
	query := `query {
  user {
    ...UserFields
  }
}
fragment UserFields on User {
  id
  name
}`
	result := inlineFragments(query)
	if strings.Contains(result, "...UserFields") {
		t.Error("fragment spread should be replaced")
	}
	if !strings.Contains(result, "id") || !strings.Contains(result, "name") {
		t.Error("fragment body should be inlined")
	}
}

func TestInlineFragments_NoFragments_C17(t *testing.T) {
	query := "query { user { id } }"
	result := inlineFragments(query)
	if result != query {
		t.Error("query without fragments should be unchanged")
	}
}

// --- coalesce.go coverage ---

func TestCopyResponse_C17(t *testing.T) {
	rr := httptest.NewRecorder()
	hdr := http.Header{}
	hdr.Set("X-Custom", "val")
	copyResponse(rr, http.StatusCreated, []byte("created"), hdr)
	if rr.Code != http.StatusCreated {
		t.Errorf("status = %d", rr.Code)
	}
	if rr.Header().Get("X-Custom") != "val" {
		t.Error("custom header not copied")
	}
	if rr.Body.String() != "created" {
		t.Errorf("body = %q", rr.Body.String())
	}
}

func TestCoalesceRecorder_Header_NilWriter_C17(t *testing.T) {
	rec := &coalesceRecorder{}
	h := rec.Header()
	if h == nil {
		t.Error("expected non-nil header")
	}
	h.Set("X-Test", "1")
	if rec.Header().Get("X-Test") != "1" {
		t.Error("header not set")
	}
}

func TestCoalesceRecorder_Write_C17(t *testing.T) {
	inner := httptest.NewRecorder()
	rec := &coalesceRecorder{ResponseWriter: inner}
	n, err := rec.Write([]byte("data"))
	if err != nil || n != 4 {
		t.Errorf("Write = %d, %v", n, err)
	}
	if rec.body.String() != "data" {
		t.Errorf("buffer = %q", rec.body.String())
	}
}

func TestCoalesceRecorder_WriteHeader_C17(t *testing.T) {
	inner := httptest.NewRecorder()
	rec := &coalesceRecorder{ResponseWriter: inner}
	rec.WriteHeader(http.StatusAccepted)
	if rec.status != http.StatusAccepted {
		t.Errorf("status = %d", rec.status)
	}
}

// --- health_check.go coverage ---

func TestHealthCheckMiddleware_C17(t *testing.T) {
	hc := NewHealthChecker(&HealthCheckConfig{
		Interval:         30 * time.Second,
		Timeout:          2 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
	})
	called := false
	mw := HealthCheckMiddleware(hc)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("next handler should be called")
	}
}

// --- botdetect.go coverage ---

func TestBotDetect_Suspicious_C17(t *testing.T) {
	h := BotDetect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called for suspicious bot")
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "sqlmap/1.0")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestBotDetect_KnownBot_C17(t *testing.T) {
	called := false
	h := BotDetect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "Googlebot/2.1")
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("known bots should pass through")
	}
	if rr.Header().Get("X-Bot-Detected") != "googlebot" {
		t.Error("expected X-Bot-Detected header")
	}
}

func TestBotDetect_NormalRequest_C17(t *testing.T) {
	called := false
	h := BotDetect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("normal requests should pass through")
	}
}

func TestBehavioralBotDetect_Threshold_C17(t *testing.T) {
	bd := NewBehavioralBotDetect(2, time.Minute)
	h := bd.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want %d", i, rr.Code, http.StatusOK)
		}
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("request 3: status = %d, want %d", rr.Code, http.StatusTooManyRequests)
	}
}

func TestBehavioralBotDetect_NoIP_C17(t *testing.T) {
	bd := NewBehavioralBotDetect(1, time.Minute)
	called := false
	h := bd.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = ""
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("should pass through when IP extraction fails")
	}
}

// --- cache.go coverage ---

func TestCacheMiddleware_NonGET_C17(t *testing.T) {
	cache := NewCache(time.Minute)
	called := false
	h := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/users", nil)
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("POST should not be cached")
	}
}

func TestCacheMiddleware_HIT_C17(t *testing.T) {
	cache := NewCache(time.Minute)
	count := 0
	h := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	}))

	rr1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/api/v1/data", nil)
	h.ServeHTTP(rr1, req1)
	if count != 1 {
		t.Errorf("expected handler called once, got %d", count)
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/data", nil)
	h.ServeHTTP(rr2, req2)
	if count != 1 {
		t.Errorf("expected handler called once (cached), got %d", count)
	}
}

func TestCacheInvalidate_C17(t *testing.T) {
	cache := NewCache(time.Minute)
	count := 0
	h := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data"))
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/x", nil)
	h.ServeHTTP(rr, req)

	cache.Invalidate()

	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req)
	if count != 2 {
		t.Errorf("expected handler called twice after invalidate, got %d", count)
	}
}

// --- wasm_plugin.go coverage ---

func TestFlattenHeaders_C17(t *testing.T) {
	h := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer token"},
		"X-Multi":       []string{"first", "second"},
	}
	flat := flattenHeaders(h)
	if flat["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q", flat["Content-Type"])
	}
	if flat["Authorization"] != "Bearer token" {
		t.Errorf("Authorization = %q", flat["Authorization"])
	}
	if flat["X-Multi"] != "first" {
		t.Errorf("X-Multi = %q, want first", flat["X-Multi"])
	}
}

func TestFlattenHeaders_Empty_C17(t *testing.T) {
	flat := flattenHeaders(http.Header{})
	if len(flat) != 0 {
		t.Errorf("expected empty map, got %d items", len(flat))
	}
}

func TestWasmPluginHost_ListPlugins_C17(t *testing.T) {
	host := NewWasmPluginHost()
	plugins := host.ListPlugins()
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestWasmPluginHost_UnloadPlugin_NotLoaded_C17(t *testing.T) {
	host := NewWasmPluginHost()
	err := host.UnloadPlugin(context.Background(), "nonexistent-plugin")
	if err == nil {
		t.Error("expected error for unloading non-existent plugin")
	}
}

func TestWasmPluginHost_Close_C17(t *testing.T) {
	host := NewWasmPluginHost()
	if err := host.Close(context.Background()); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestWasmMiddleware_NoPlugins_C17(t *testing.T) {
	host := NewWasmPluginHost()
	called := false
	mw := WasmMiddleware(host, nil)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("next handler should be called when no plugins loaded")
	}
}

// --- isPublicPath coverage ---

func TestIsPublicPath_C17(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/healthz", true},
		{"/.well-known/jwks.json", true},
		{"/api/v1/auth/verify", true},
		{"/api/v1/auth/social/google", false},
		{"/oauth/authorize", true},
		{"/saml/login", true},
		{"/.well-known/openid-configuration", true},
		{"/docs", true},
		{"/api/v1/users", false},
		{"/api/v1/orgs", false},
	}
	for _, tt := range tests {
		if got := isPublicPath(tt.path); got != tt.want {
			t.Errorf("isPublicPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
