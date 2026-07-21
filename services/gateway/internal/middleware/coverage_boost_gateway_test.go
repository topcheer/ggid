package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

// === Session Manager Tests ===

func TestSessionManager_NilRedis(t *testing.T) {
	sm := NewSessionManager(nil)

	if sm.IsSessionRevoked(context.Background(), "s1") {
		t.Error("nil Redis: IsSessionRevoked should return false")
	}
	if err := sm.MarkSessionRevoked(context.Background(), "s1"); err != nil {
		t.Errorf("nil Redis: MarkSessionRevoked error: %v", err)
	}
	sm.touchSessionTTL(context.Background(), "s1", 0)
}

func TestSessionMiddleware_PublicPath(t *testing.T) {
	sm := NewSessionManager(nil)

	called := false
	handler := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("Public path should bypass session check")
	}
}

func TestSessionMiddleware_NoSessionID(t *testing.T) {
	sm := NewSessionManager(nil)

	called := false
	handler := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("No session ID should pass through")
	}
}

func TestSessionMiddleware_SessionInContext(t *testing.T) {
	sm := NewSessionManager(nil)

	called := false
	handler := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	ctx := context.WithValue(req.Context(), SessionIDKey, "sess-123")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("Session with nil Redis should pass through")
	}
}

func TestSessionRevokeHandler_BadMethod(t *testing.T) {
	sm := NewSessionManager(nil)

	handler := sm.SessionRevokeHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call next")
	}))

	req := httptest.NewRequest("GET", "/api/v1/sessions/123", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET on revoke: want 405, got %d", rr.Code)
	}
}

func TestIsPublicPath_Comprehensive(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/healthz", true},
		{"/.well-known/jwks.json", true},
		{"/api/v1/auth/verify", true},
		{"/api/v1/auth/register", true},
		{"/api/v1/auth/social/github", true},
		{"/oauth/authorize", true},
		{"/saml/login", true},
		{"/docs", true},
		{"/login", true},
		{"/api/v1/users", false},
		{"/api/v1/orgs", false},
	}

	for _, tt := range tests {
		if got := isPublicPath(tt.path); got != tt.want {
			t.Errorf("isPublicPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestSessionIDFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), SessionIDKey, "test-sess")
	id, ok := SessionIDFromContext(ctx)
	if !ok || id != "test-sess" {
		t.Errorf("SessionIDFromContext: got '%s', ok=%v", id, ok)
	}

	_, ok = SessionIDFromContext(context.Background())
	if ok {
		t.Error("Background context should not have session")
	}
}

// === extractTenantFromJWT Tests ===

func makeJWTWithClaims(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + payloadB64 + ".sig"
}

func TestExtractTenantFromJWT_ValidToken(t *testing.T) {
	token := makeJWTWithClaims(map[string]any{
		"tenant_id": "t-12345",
		"sub":       "user1",
	})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	tid := extractTenantFromJWT(req)
	if tid != "t-12345" {
		t.Errorf("tenant_id: want 't-12345', got '%s'", tid)
	}
}

func TestExtractTenantFromJWT_NoAuthHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	tid := extractTenantFromJWT(req)
	if tid != "" {
		t.Errorf("No auth header: want '', got '%s'", tid)
	}
}

func TestExtractTenantFromJWT_WrongScheme(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	tid := extractTenantFromJWT(req)
	if tid != "" {
		t.Errorf("Wrong scheme: want '', got '%s'", tid)
	}
}

func TestExtractTenantFromJWT_NoTenantClaim(t *testing.T) {
	token := makeJWTWithClaims(map[string]any{
		"sub": "user1",
	})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	tid := extractTenantFromJWT(req)
	if tid != "" {
		t.Errorf("No tenant_id claim: want '', got '%s'", tid)
	}
}

func TestExtractTenantFromJWT_SinglePartToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer onepart")
	tid := extractTenantFromJWT(req)
	if tid != "" {
		t.Errorf("Single part: want '', got '%s'", tid)
	}
}

func TestExtractTenantFromJWT_EmptyBearer(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	tid := extractTenantFromJWT(req)
	if tid != "" {
		t.Errorf("Empty bearer: want '', got '%s'", tid)
	}
}

// === extractTenantFromSubdomain Tests ===

func TestExtractTenantFromSubdomain_ValidUUID(t *testing.T) {
	id := uuid.New()
	result := extractTenantFromSubdomain(id.String()+".iam.example.com", ".iam.example.com")
	if result != id {
		t.Errorf("Valid UUID subdomain: got %v, want %v", result, id)
	}
}

func TestExtractTenantFromSubdomain_WWW(t *testing.T) {
	result := extractTenantFromSubdomain("www.iam.example.com", ".iam.example.com")
	if result != uuid.Nil {
		t.Error("www subdomain should return Nil")
	}
}

func TestExtractTenantFromSubdomain_NotUUID(t *testing.T) {
	result := extractTenantFromSubdomain("acme.iam.example.com", ".iam.example.com")
	if result != uuid.Nil {
		t.Error("Non-UUID subdomain should return Nil")
	}
}

func TestExtractTenantFromSubdomain_WrongSuffix(t *testing.T) {
	id := uuid.New()
	result := extractTenantFromSubdomain(id.String()+".other.com", ".iam.example.com")
	if result != uuid.Nil {
		t.Error("Wrong suffix should return Nil")
	}
}

func TestExtractTenantFromSubdomain_WithPort(t *testing.T) {
	id := uuid.New()
	result := extractTenantFromSubdomain(id.String()+".iam.example.com:8080", ".iam.example.com")
	if result != id {
		t.Errorf("With port: got %v, want %v", result, id)
	}
}

func TestExtractTenantFromSubdomain_EmptySubdomain(t *testing.T) {
	result := extractTenantFromSubdomain(".iam.example.com", ".iam.example.com")
	if result != uuid.Nil {
		t.Error("Empty subdomain should return Nil")
	}
}

// === TenantRateLimitHandler Tests (new paths not covered elsewhere) ===

func TestTenantRateLimitHandler_GetSingle_V2(t *testing.T) {
	store := NewTenantRateLimitStore(100, 20)
	store.Set(TenantRateLimitConfig{TenantID: "t1", RequestsPerMin: 200, BurstSize: 50, Enabled: true})

	handler := TenantRateLimitHandler(store)

	req := httptest.NewRequest("GET", "/api/v1/gateway/ratelimits/t1", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Get single: want 200, got %d", rr.Code)
	}
}

func TestTenantRateLimitHandler_PutValid_V2(t *testing.T) {
	store := NewTenantRateLimitStore(100, 20)

	handler := TenantRateLimitHandler(store)

	body := `{"tenant_id":"t1","requests_per_min":500,"burst_size":100,"enabled":true}`
	req := httptest.NewRequest("PUT", "/api/v1/gateway/ratelimits/t1", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("PUT: want 200, got %d", rr.Code)
	}
	cfg := store.Get("t1")
	if cfg.RequestsPerMin != 500 {
		t.Errorf("Stored RPM: want 500, got %d", cfg.RequestsPerMin)
	}
}

func TestTenantRateLimitHandler_PutDefaultValues_V2(t *testing.T) {
	store := NewTenantRateLimitStore(100, 20)

	handler := TenantRateLimitHandler(store)

	body := `{"tenant_id":"t1","enabled":true}`
	req := httptest.NewRequest("PUT", "/api/v1/gateway/ratelimits/t1", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("PUT defaults: want 200, got %d", rr.Code)
	}
	cfg := store.Get("t1")
	if cfg.RequestsPerMin != 100 {
		t.Errorf("Default RPM: want 100, got %d", cfg.RequestsPerMin)
	}
	if cfg.BurstSize != 20 {
		t.Errorf("Default burst: want 20, got %d", cfg.BurstSize)
	}
}

// === containsWildcard Tests ===

func TestContainsWildcard_AllCases(t *testing.T) {
	if !containsWildcard([]string{"*"}) {
		t.Error("'*' should return true")
	}
	if !containsWildcard([]string{"https://a.com", "*"}) {
		t.Error("List with '*' should return true")
	}
	if containsWildcard([]string{"https://a.com"}) {
		t.Error("List without '*' should return false")
	}
	if containsWildcard([]string{}) {
		t.Error("Empty list should return false")
	}
}

// === gRPC metadata helpers ===

func TestRequestIDFromIncomingMetadata_Present_V2(t *testing.T) {
	md := metadata.Pairs(GRPCRequestIDKey, "rid-123")
	id := RequestIDFromIncomingMetadata(md)
	if id != "rid-123" {
		t.Errorf("From metadata: want 'rid-123', got '%s'", id)
	}
}

func TestRequestIDFromIncomingMetadata_Absent_V2(t *testing.T) {
	md := metadata.MD{}
	id := RequestIDFromIncomingMetadata(md)
	if id != "" {
		t.Errorf("Absent: want '', got '%s'", id)
	}
}

// === token_bucket edge case tests ===

func TestTokenBucket_CapOverflow(t *testing.T) {
	tb := NewTokenBucket(5, 1000)
	tb.Allow()
	for i := 0; i < 10; i++ {
		tb.Allow()
	}
	if tb.Tokens() > 5 {
		t.Errorf("Tokens should be capped at 5, got %.2f", tb.Tokens())
	}
}

func TestTokenBucket_ZeroRefill(t *testing.T) {
	tb := NewTokenBucket(1, 0)
	if !tb.Allow() {
		t.Error("First call should succeed")
	}
	if tb.Allow() {
		t.Error("Second call with no refill should fail")
	}
	ra := tb.RetryAfter()
	if ra < 1 {
		t.Errorf("RetryAfter with 0 refill: got %d", ra)
	}
}
