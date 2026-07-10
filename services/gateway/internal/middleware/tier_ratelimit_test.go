package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var testWindow = 1 * time.Hour

func TestTierRateLimiter_FreeTierUnderLimit(t *testing.T) {
	cfg := TierRateLimitConfig{
		Limits: map[Tier]int{TierFree: 5},
		Window: testWindow,
	}
	trl := NewTierRateLimiter(cfg)
	handler := trl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req = req.WithContext(WithTier(req.Context(), TierFree))
		req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, "tenant-1"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, w.Code)
		}
	}
}

func TestTierRateLimiter_FreeTierOverLimit(t *testing.T) {
	cfg := TierRateLimitConfig{
		Limits: map[Tier]int{TierFree: 2},
		Window: testWindow,
	}
	trl := NewTierRateLimiter(cfg)
	handler := trl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req = req.WithContext(WithTier(req.Context(), TierFree))
		req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, "t-over"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req = req.WithContext(WithTier(req.Context(), TierFree))
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, "t-over"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 on 3rd request, got %d", w.Code)
	}
}

func TestTierRateLimiter_EnterpriseUnlimited(t *testing.T) {
	cfg := TierRateLimitConfig{
		Limits: map[Tier]int{TierEnterprise: 0},
		Window: testWindow,
	}
	trl := NewTierRateLimiter(cfg)
	handler := trl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req = req.WithContext(WithTier(req.Context(), TierEnterprise))
		req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, "t-ent"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("enterprise request %d: expected 200, got %d", i, w.Code)
			break
		}
	}
}

func TestTierRateLimiter_ProTier(t *testing.T) {
	cfg := TierRateLimitConfig{
		Limits: map[Tier]int{TierPro: 3},
		Window: testWindow,
	}
	trl := NewTierRateLimiter(cfg)
	handler := trl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
		req = req.WithContext(WithTier(req.Context(), TierPro))
		req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, "t-pro"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("pro request %d: expected 200, got %d", i, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
	req = req.WithContext(WithTier(req.Context(), TierPro))
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, "t-pro"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}

func TestTierRateLimiter_DifferentTenantsIndependent(t *testing.T) {
	cfg := TierRateLimitConfig{
		Limits: map[Tier]int{TierFree: 2},
		Window: testWindow,
	}
	trl := NewTierRateLimiter(cfg)
	handler := trl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Tenant A uses both slots
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(WithTier(req.Context(), TierFree))
		req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, "tenant-a"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// Tenant B should still have full quota
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(WithTier(req.Context(), TierFree))
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, "tenant-b"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("tenant B should be independent, got %d", w.Code)
	}
}

func TestTierRateLimiter_NoTierDefaultsFree(t *testing.T) {
	cfg := TierRateLimitConfig{
		Limits: map[Tier]int{TierFree: 100},
		Window: testWindow,
	}
	trl := NewTierRateLimiter(cfg)
	handler := trl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with no tier context, got %d", w.Code)
	}
	// Should have X-RateLimit-Tier header
	if tier := w.Header().Get("X-RateLimit-Tier"); tier != "free" {
		t.Errorf("expected X-RateLimit-Tier=free, got %s", tier)
	}
}

func TestTierRateLimiter_SkipHealthCheck(t *testing.T) {
	cfg := DefaultTierRateLimitConfig()
	trl := NewTierRateLimiter(cfg)
	handler := trl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for healthz, got %d", w.Code)
	}
}
