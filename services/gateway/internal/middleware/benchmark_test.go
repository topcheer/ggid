package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// BenchmarkMiddlewareChain measures overhead of a basic middleware chain.
func BenchmarkMiddlewareChain(b *testing.B) {
	logger := NoopLogger{}
	handler := RequestLogging(logger)(
		RequestIDMiddleware(
			SecurityHeadersConfigurable(nil)(
				PanicRecovery(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})),
			),
		),
	)

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkJWTValidation measures JWT middleware overhead.
func BenchmarkJWTValidation(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequestIDMiddleware(handler)

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer some.jwt.token")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		wrapped.ServeHTTP(rr, req)
	}
}

// BenchmarkRateLimiter measures token bucket allow/deny decisions.
func BenchmarkRateLimiter(b *testing.B) {
	tb := NewTokenBucket(1000, 100)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = tb.Allow()
	}
}

// BenchmarkCircuitBreaker measures circuit breaker state checks.
func BenchmarkCircuitBreaker(b *testing.B) {
	registry := NewCircuitRegistry(CircuitConfig{
		MaxFailures: 5,
		Timeout:     30 * time.Second,
	})
	cb := registry.Get("backend-1")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cb.Allow()
	}
}

// BenchmarkRouteTimeoutMatch measures per-route timeout lookup.
func BenchmarkRouteTimeoutMatch(b *testing.B) {
	cfg := DefaultRouteTimeoutConfig()
	paths := []string{
		"/api/v1/users/550e8400-e29b-41d4-a716-446655440000",
		"/api/v1/scim/v2/Users",
		"/api/v1/audit/events?tenant_id=xxx",
		"/api/v1/auth/login",
		"/api/v1/oauth/token",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cfg.MatchTimeout(paths[i%len(paths)])
	}
}

// BenchmarkHealthScoreRecompute measures health score recalculation.
func BenchmarkHealthScoreRecompute(b *testing.B) {
	hs := NewHealthScore(5*time.Minute, 0.95)
	for i := 0; i < 100; i++ {
		hs.RecordSuccess("backend-1", time.Duration(i)*time.Millisecond)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bh := hs.getOrCreate("backend-1")
		hs.mu.Lock()
		hs.recomputeScore(bh)
		hs.mu.Unlock()
	}
}
