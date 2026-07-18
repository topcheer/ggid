package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// === Timeout Middleware Tests ===

func TestTimeoutMiddleware_DefaultConfig(t *testing.T) {
	cfg := DefaultTimeoutConfig()
	if cfg.Default != 30*time.Second {
		t.Errorf("Default: want 30s, got %v", cfg.Default)
	}
	if d := cfg.GetTimeoutForRoute("/api/v1/auth/login"); d != 10*time.Second {
		t.Errorf("Login timeout: want 10s, got %v", d)
	}
	if d := cfg.GetTimeoutForRoute("/api/v1/audit/events"); d != 60*time.Second {
		t.Errorf("Audit timeout: want 60s, got %v", d)
	}
	// Unknown route uses default
	if d := cfg.GetTimeoutForRoute("/api/v1/users"); d != 30*time.Second {
		t.Errorf("Unknown route: want 30s, got %v", d)
	}
}

func TestTimeoutMiddleware_RequestCompletesInTime(t *testing.T) {
	handler := TimeoutMiddleware(&TimeoutConfig{
		Default: 5 * time.Second,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status: want 200, got %d", rr.Code)
	}
	if rr.Body.String() != "ok" {
		t.Errorf("Body: want 'ok', got '%s'", rr.Body.String())
	}
}

func TestTimeoutMiddleware_RequestTimesOut(t *testing.T) {
	handler := TimeoutMiddleware(&TimeoutConfig{
		Default: 50 * time.Millisecond,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))

	req := httptest.NewRequest("GET", "/api/v1/slow", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusGatewayTimeout {
		t.Errorf("Status: want 504, got %d", rr.Code)
	}
	if rr.Header().Get("X-Request-Timeout") == "" {
		t.Error("Missing X-Request-Timeout header")
	}
}

func TestTimeoutMiddleware_PerRouteTimeout(t *testing.T) {
	var completed int32

	handler := TimeoutMiddleware(&TimeoutConfig{
		Default: 5 * time.Second,
		RouteConfigs: map[string]time.Duration{
			"/api/v1/slow": 50 * time.Millisecond,
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			atomic.AddInt32(&completed, 1)
			return
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))

	req := httptest.NewRequest("GET", "/api/v1/slow/resource", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusGatewayTimeout {
		t.Errorf("Status: want 504, got %d", rr.Code)
	}
}

func TestTimeoutMiddleware_HealthCheckSkipped(t *testing.T) {
	handler := TimeoutMiddleware(&TimeoutConfig{
		Default: 1 * time.Millisecond,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // would time out if timeout applied
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Health check should not time out: want 200, got %d", rr.Code)
	}
}

func TestTimeoutMiddleware_WebSocketSkipped(t *testing.T) {
	handler := TimeoutMiddleware(&TimeoutConfig{
		Default: 1 * time.Millisecond,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("WebSocket should not time out: want 200, got %d", rr.Code)
	}
}

func TestTimeoutMiddleware_NilConfig(t *testing.T) {
	handler := TimeoutMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Nil config should use defaults: want 200, got %d", rr.Code)
	}
}

func TestTimeoutMiddleware_ContextCancelled(t *testing.T) {
	var ctxCancelled bool

	handler := TimeoutMiddleware(&TimeoutConfig{
		Default: 50 * time.Millisecond,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		ctxCancelled = true
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Give goroutine time to observe cancellation
	time.Sleep(50 * time.Millisecond)
	if !ctxCancelled {
		t.Error("Context should have been cancelled")
	}
}

// === Per-Tenant CORS Tests ===

func TestTenantCORSStore_SetGetDelete(t *testing.T) {
	store := NewTenantCORSStore(CORSConfig{AllowedOrigins: []string{"*"}})
	store.SetOrigins("tenant-1", []string{"https://app1.com", "https://app2.com"})

	origins := store.GetOrigins("tenant-1")
	if len(origins) != 2 {
		t.Fatalf("Want 2 origins, got %d", len(origins))
	}
	if origins[0] != "https://app1.com" {
		t.Errorf("First origin: want 'https://app1.com', got '%s'", origins[0])
	}

	// Fallback for unknown tenant
	origins = store.GetOrigins("unknown")
	if len(origins) != 1 || origins[0] != "*" {
		t.Errorf("Unknown tenant should get fallback: got %v", origins)
	}

	store.DeleteOrigins("tenant-1")
	origins = store.GetOrigins("tenant-1")
	if len(origins) != 1 || origins[0] != "*" {
		t.Errorf("After delete should fallback: got %v", origins)
	}
}

func TestPerTenantCORS_AllowedOrigin(t *testing.T) {
	store := NewTenantCORSStore(CORSConfig{AllowedOrigins: []string{"*"}})
	store.SetOrigins("tenant-1", []string{"https://app1.com"})

	handler := PerTenantCORS(store, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://app1.com")
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant-1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "https://app1.com" {
		t.Errorf("Allow-Origin: got '%s'", rr.Header().Get("Access-Control-Allow-Origin"))
	}
	if rr.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("Missing credentials header")
	}
}

func TestPerTenantCORS_DisallowedOrigin(t *testing.T) {
	store := NewTenantCORSStore(CORSConfig{AllowedOrigins: []string{}})
	store.SetOrigins("tenant-1", []string{"https://allowed.com"})

	handler := PerTenantCORS(store, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Non-OPTIONS request with disallowed origin still passes through (no Allow-Origin set)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant-1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Disallowed origin should not get Allow-Origin: got '%s'",
			rr.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestPerTenantCORS_PreflightAllowed(t *testing.T) {
	store := NewTenantCORSStore(CORSConfig{AllowedOrigins: []string{"*"}})
	store.SetOrigins("t1", []string{"https://app.com"})

	handler := PerTenantCORS(store, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://app.com")
	ctx := context.WithValue(req.Context(), TenantIDKey, "t1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Preflight: want 204, got %d", rr.Code)
	}
}

func TestPerTenantCORS_PreflightDisallowed(t *testing.T) {
	store := NewTenantCORSStore(CORSConfig{AllowedOrigins: []string{}})
	store.SetOrigins("t1", []string{"https://app.com"})

	handler := PerTenantCORS(store, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	ctx := context.WithValue(req.Context(), TenantIDKey, "t1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Disallowed preflight: want 403, got %d", rr.Code)
	}
}

func TestPerTenantCORS_NoOrigin(t *testing.T) {
	store := NewTenantCORSStore(CORSConfig{AllowedOrigins: []string{"*"}})

	handler := PerTenantCORS(store, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("No origin: want 200, got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Should default to wildcard: got '%s'", rr.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestPerTenantCORS_WildcardFallback(t *testing.T) {
	store := NewTenantCORSStore(CORSConfig{AllowedOrigins: []string{"*"}})

	// Unknown tenant → wildcard
	handler := PerTenantCORS(store, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://anything.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Wildcard fallback: got '%s'", rr.Header().Get("Access-Control-Allow-Origin"))
	}
}

// === Token Bucket Rate Limiter Tests ===

func TestTokenBucket_Allow(t *testing.T) {
	tb := NewTokenBucket(5, 1) // 5 burst, 1/sec
	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Errorf("Request %d should be allowed", i)
		}
	}
	if tb.Allow() {
		t.Error("6th request should be rate limited")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	tb := NewTokenBucket(2, 100) // 2 burst, 100/sec refill
	// Consume all tokens
	tb.Allow()
	tb.Allow()

	// Wait for refill
	time.Sleep(50 * time.Millisecond) // ~5 tokens refilled

	if !tb.Allow() {
		t.Error("Should have refilled tokens after wait")
	}
}

func TestTokenBucket_RetryAfter(t *testing.T) {
	tb := NewTokenBucket(1, 1) // 1 burst, 1/sec
	tb.Allow()                 // consume the only token

	ra := tb.RetryAfter()
	if ra <= 0 {
		t.Errorf("RetryAfter should be positive, got %d", ra)
	}
}

func TestTokenBucket_Tokens(t *testing.T) {
	tb := NewTokenBucket(10, 0) // no refill
	if tb.Tokens() != 10 {
		t.Errorf("Initial tokens: want 10, got %.2f", tb.Tokens())
	}
	tb.Allow()
	if tb.Tokens() != 9 {
		t.Errorf("After 1 allow: want 9, got %.2f", tb.Tokens())
	}
}

func TestDefaultBucketRateLimitConfig(t *testing.T) {
	cfg := DefaultBucketRateLimitConfig()
	if cfg.DefaultMaxTokens != 100 {
		t.Errorf("Default max tokens: want 100, got %.0f", cfg.DefaultMaxTokens)
	}
	if free, ok := cfg.TierOverrides["free"]; !ok || free.MaxTokens != 20 {
		t.Error("Free tier should have 20 tokens")
	}
	if ent, ok := cfg.TierOverrides["enterprise"]; !ok || ent.MaxTokens != 1000 {
		t.Error("Enterprise tier should have 1000 tokens")
	}
}

func TestTenantBucketLimiter_Middleware_Allowed(t *testing.T) {
	limiter := NewTenantBucketLimiter(&BucketRateLimitConfig{
		DefaultMaxTokens:    100,
		DefaultRefillPerSec: 10,
	})

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("First request: want 200, got %d", rr.Code)
	}
	if rr.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Missing X-RateLimit-Limit header")
	}
}

func TestTenantBucketLimiter_Middleware_RateLimited(t *testing.T) {
	limiter := NewTenantBucketLimiter(&BucketRateLimitConfig{
		DefaultMaxTokens:    2,
		DefaultRefillPerSec: 0, // no refill to ensure exhaustion
	})

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 2 requests succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("X-Tenant-ID", "t1")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("Request %d: want 200, got %d", i, rr.Code)
		}
	}

	// 3rd request rate limited
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("3rd request: want 429, got %d", rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Error("Missing Retry-After header")
	}
}

func TestTenantBucketLimiter_HealthCheckSkipped(t *testing.T) {
	limiter := NewTenantBucketLimiter(&BucketRateLimitConfig{
		DefaultMaxTokens:    1,
		DefaultRefillPerSec: 0,
	})

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Health checks should bypass rate limiting
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/healthz", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("Health check %d: want 200, got %d", i, rr.Code)
		}
	}
}

func TestTenantBucketLimiter_PerTenantIsolation(t *testing.T) {
	limiter := NewTenantBucketLimiter(&BucketRateLimitConfig{
		DefaultMaxTokens:    1,
		DefaultRefillPerSec: 0,
	})

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Tenant 1 uses the only token
	req1 := httptest.NewRequest("GET", "/api/v1/test", nil)
	ctx1 := context.WithValue(req1.Context(), TenantIDKey, "t1")
	req1 = req1.WithContext(ctx1)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	// Tenant 2 should still have their own bucket
	req2 := httptest.NewRequest("GET", "/api/v1/test", nil)
	ctx2 := context.WithValue(req2.Context(), TenantIDKey, "t2")
	req2 = req2.WithContext(ctx2)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("Tenant 2 should have independent bucket: want 200, got %d", rr2.Code)
	}
	if limiter.BucketCount() != 2 {
		t.Errorf("Should have 2 buckets, got %d", limiter.BucketCount())
	}
}

func TestTenantBucketLimiter_TierOverride(t *testing.T) {
	limiter := NewTenantBucketLimiter(&BucketRateLimitConfig{
		DefaultMaxTokens:    1,
		DefaultRefillPerSec: 0,
		TierOverrides: map[string]BucketTierConfig{
			"enterprise": {MaxTokens: 5, RefillPerSec: 0},
		},
	})

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Enterprise tier should get 5 requests
	ctx := context.WithValue(context.Background(), tierContextKey{}, TierEnterprise)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("X-Tenant-ID", "t1")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("Enterprise request %d: want 200, got %d", i, rr.Code)
		}
	}
}

func TestTenantBucketLimiter_Cleanup(t *testing.T) {
	limiter := NewTenantBucketLimiter(&BucketRateLimitConfig{
		DefaultMaxTokens:    1,
		DefaultRefillPerSec: 0,
	})

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if limiter.BucketCount() != 1 {
		t.Fatalf("Should have 1 bucket, got %d", limiter.BucketCount())
	}

	time.Sleep(5 * time.Millisecond)
	limiter.Cleanup(1 * time.Millisecond) // expire everything
	if limiter.BucketCount() != 0 {
		t.Errorf("After cleanup: want 0 buckets, got %d", limiter.BucketCount())
	}
}

func TestClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	ip := ClientIP(req)
	if ip != "1.2.3.4" {
		t.Errorf("ClientIP: want '1.2.3.4', got '%s'", ip)
	}
}

func TestClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-IP", "9.9.9.9")
	ip := ClientIP(req)
	if ip != "9.9.9.9" {
		t.Errorf("ClientIP: want '9.9.9.9', got '%s'", ip)
	}
}

func TestClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	ip := ClientIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("ClientIP: want '192.168.1.1', got '%s'", ip)
	}
}

// === Request ID Propagation Tests ===

func TestRequestID_GeneratesNew(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	id := rr.Header().Get("X-Request-ID")
	if id == "" {
		t.Error("Should generate X-Request-ID")
	}
}

func TestRequestID_PreservesIncoming(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := r.Context().Value(RequestIDKey).(string)
		w.Write([]byte(id))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "my-custom-id")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Request-ID") != "my-custom-id" {
		t.Errorf("Should preserve incoming ID: got '%s'", rr.Header().Get("X-Request-ID"))
	}
	if rr.Body.String() != "my-custom-id" {
		t.Errorf("Context should have ID: got '%s'", rr.Body.String())
	}
}

func TestPropagateRequestID_GeneratesUUID(t *testing.T) {
	handler := PropagateRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	id := rr.Header().Get("X-Request-ID")
	if id == "" {
		t.Error("Should generate X-Request-ID")
	}
	// Should be a UUID v4 format (36 chars with dashes)
	if len(id) != 36 {
		t.Errorf("Should be UUID format (36 chars): got %d chars '%s'", len(id), id)
	}
}

func TestInjectRequestIDHeader(t *testing.T) {
	ctx := ContextWithRequestID(context.Background(), "ctx-id-123")
	req := httptest.NewRequest("GET", "/test", nil)
	InjectRequestIDHeader(ctx, req)

	if req.Header.Get("X-Request-ID") != "ctx-id-123" {
		t.Errorf("Should inject context ID: got '%s'", req.Header.Get("X-Request-ID"))
	}
}

func TestInjectRequestIDHeader_NoContextID(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	InjectRequestIDHeader(context.Background(), req)

	id := req.Header.Get("X-Request-ID")
	if id == "" {
		t.Error("Should generate new ID when context has none")
	}
}

func TestInjectRequestIDHeader_PreserveExisting(t *testing.T) {
	ctx := ContextWithRequestID(context.Background(), "ctx-id")
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "existing-id")
	InjectRequestIDHeader(ctx, req)

	if req.Header.Get("X-Request-ID") != "existing-id" {
		t.Error("Should preserve existing header")
	}
}

func TestRequestIDFromContext_Empty(t *testing.T) {
	id := RequestIDFromContext(context.Background())
	if id != "" {
		t.Errorf("Should be empty for background context: got '%s'", id)
	}
}
