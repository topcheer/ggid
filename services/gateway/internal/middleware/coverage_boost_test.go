package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- Additional coverage tests to push middleware past 70% ---

func TestGenerateCSRFToken_NonEmpty(t *testing.T) {
	token := generateCSRFToken()
	if token == "" {
		t.Error("expected non-empty CSRF token")
	}
	token2 := generateCSRFToken()
	if token == token2 {
		t.Error("tokens should be unique")
	}
}

func TestSetCSRFCookie_SetsCorrectFlags(t *testing.T) {
	w := httptest.NewRecorder()
	setCSRFCookie(w)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != "csrf_token" {
		t.Errorf("expected csrf_token, got %s", c.Name)
	}
	if !c.Secure {
		t.Error("expected Secure flag")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Error("expected SameSite=Lax")
	}
}

func TestWriteForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	writeForbidden(w, "test error")
	if w.Code != 403 {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestWriteSessionError(t *testing.T) {
	w := httptest.NewRecorder()
	writeSessionError(w, "bad session")
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWriteAPIKeyError(t *testing.T) {
	w := httptest.NewRecorder()
	writeAPIKeyError(w, "bad key")
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWriteJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSONError(w, 400, "bad request")
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAllowListDeny(t *testing.T) {
	w := httptest.NewRecorder()
	allowListDeny(w, "blocked")
	if w.Code != 403 {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestIsPublicPath(t *testing.T) {
	public := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/password/forgot",
		"/oauth/authorize",
		"/.well-known/openid-configuration",
		"/docs",
		"/login",
		"/healthz",
		"/.well-known/jwks.json",
	}
	for _, p := range public {
		if !isPublicPath(p) {
			t.Errorf("expected %s to be public", p)
		}
	}

	private := []string{
		"/api/v1/users",
		"/api/v1/roles",
		"/api/v1/orgs",
	}
	for _, p := range private {
		if isPublicPath(p) {
			t.Errorf("expected %s to be private", p)
		}
	}
}

func TestSessionIDFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), SessionIDKey, "sess-123")
	id, ok := SessionIDFromContext(ctx)
	if !ok || id != "sess-123" {
		t.Errorf("expected sess-123, got %s (ok=%v)", id, ok)
	}

	_, ok = SessionIDFromContext(context.Background())
	if ok {
		t.Error("expected ok=false without context")
	}
}

func TestSessionKey_Format(t *testing.T) {
	key := sessionKey("abc-123")
	if key != "ggid:session:abc-123" {
		t.Errorf("unexpected key format: %s", key)
	}
}

func TestMemoryAPIKeyValidator_AddAndValidate(t *testing.T) {
	v := NewMemoryAPIKeyValidator()
	v.AddKey("ggid_test", "t1", "u1", []string{"read"})

	tenantID, userID, scopes, err := v.Validate(context.Background(), "ggid_test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenantID != "t1" || userID != "u1" || len(scopes) != 1 {
		t.Errorf("unexpected values: %s/%s/%v", tenantID, userID, scopes)
	}

	// Invalid key
	_, _, _, err = v.Validate(context.Background(), "invalid")
	if err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestRateLimiter_BucketKey(t *testing.T) {
	rl := NewRateLimiter(DefaultRateLimitConfig())
	req := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	key := rl.bucketKey(req)
	if key == "" {
		t.Error("expected non-empty bucket key")
	}
}

func TestRateLimiter_GetLimit(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{LoginLimit: 5, RegisterLimit: 3, APILimit: 100, Window: 60000000000})
	if rl.getLimit("/api/v1/auth/login") != 5 {
		t.Error("wrong login limit")
	}
	if rl.getLimit("/api/v1/auth/register") != 3 {
		t.Error("wrong register limit")
	}
	if rl.getLimit("/api/v1/users") != 100 {
		t.Error("wrong API limit")
	}
	if rl.getLimit("/healthz") != 0 {
		t.Error("expected 0 for non-API path")
	}
}

func TestIncAuthFailure(t *testing.T) {
	// Should not panic
	IncAuthFailure("test")
}

func TestSetActiveSessions(t *testing.T) {
	// Should not panic
	SetActiveSessions(42)
}

func TestCacheKey_Format(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/data?foo=bar", nil)
	key := cacheKey(req)
	if key != "GET:/api/v1/data?foo=bar" {
		t.Errorf("unexpected cache key: %s", key)
	}
}

func TestGenerateETag(t *testing.T) {
	etag := generateETag([]byte("test data"))
	if etag == "" {
		t.Error("expected non-empty ETag")
	}
	etag2 := generateETag([]byte("test data"))
	if etag != etag2 {
		t.Error("ETags should be deterministic")
	}
	etag3 := generateETag([]byte("different data"))
	if etag == etag3 {
		t.Error("ETags should differ for different data")
	}
}

func TestShouldSkipCompression(t *testing.T) {
	// Should skip
	if !shouldSkipCompression("application/octet-stream") {
		t.Error("expected octet-stream to be skipped")
	}
	if !shouldSkipCompression("image/png") {
		t.Error("expected image/png to be skipped")
	}
	if !shouldSkipCompression("video/mp4") {
		t.Error("expected video/mp4 to be skipped")
	}

	// Should NOT skip
	if shouldSkipCompression("application/json") {
		t.Error("expected JSON to be compressed")
	}
	if shouldSkipCompression("text/html") {
		t.Error("expected HTML to be compressed")
	}
	if shouldSkipCompression("image/svg+xml") {
		t.Error("expected SVG to be compressed")
	}
	if shouldSkipCompression("application/javascript") {
		t.Error("expected JS to be compressed")
	}
}
