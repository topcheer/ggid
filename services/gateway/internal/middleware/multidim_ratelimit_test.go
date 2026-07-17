package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestMultiDimRateLimit_Allow(t *testing.T) {
	limiter := NewMultiDimRateLimiter(DefaultMultiDimConfigs())
	result := limiter.Check(TierFree, "tenant-1", "user-1", "key-1", "10.0.0.1", "/api/v1/users")
	if !result.Allowed {
		t.Fatal("first request should be allowed")
	}
}

func TestMultiDimRateLimit_BurstLimit(t *testing.T) {
	limiter := NewMultiDimRateLimiter(DefaultMultiDimConfigs())
	// Free tier user burst = 30/min.
	for i := 0; i < 30; i++ {
		limiter.Check(TierFree, "t1", "u1", "", "10.0.0.1", "/api/users")
	}
	result := limiter.Check(TierFree, "t1", "u1", "", "10.0.0.1", "/api/users")
	if result.Allowed {
		t.Fatal("expected rate limit after 30 requests (free tier user burst)")
	}
	if result.Dimension != DimUser {
		t.Fatalf("expected user dimension, got %s", result.Dimension)
	}
}

func TestMultiDimRateLimit_EnterpriseUnlimited(t *testing.T) {
	limiter := NewMultiDimRateLimiter(DefaultMultiDimConfigs())
	// Enterprise tenant = 0 (unlimited).
	for i := 0; i < 100; i++ {
		result := limiter.Check(TierEnterprise, "ent-1", "", "", "10.0.0.1", "/api/v1/data")
		if !result.Allowed {
			t.Fatalf("enterprise should be unlimited, blocked at request %d", i+1)
		}
	}
}

func TestMultiDimRateLimit_UpdateTier(t *testing.T) {
	limiter := NewMultiDimRateLimiter(DefaultMultiDimConfigs())
	// Update free tier to very low limit.
	limiter.UpdateTier(TierFree, MultiDimTierConfig{
		User:     MultiDimRateLimit{BurstPerMin: 2, SustainedPerHour: 10},
		Tenant:   MultiDimRateLimit{BurstPerMin: 0, SustainedPerHour: 0},
		IP:       MultiDimRateLimit{BurstPerMin: 0, SustainedPerHour: 0},
		APIKey:   MultiDimRateLimit{BurstPerMin: 0, SustainedPerHour: 0},
		Endpoint: MultiDimRateLimit{BurstPerMin: 0, SustainedPerHour: 0},
	})

	for i := 0; i < 2; i++ {
		r := limiter.Check(TierFree, "t1", "u1", "", "10.0.0.1", "/api/v1/test")
		if !r.Allowed {
			t.Fatalf("request %d should be allowed (limit 2)", i+1)
		}
	}
	r := limiter.Check(TierFree, "t1", "u1", "", "10.0.0.1", "/api/v1/test")
	if r.Allowed {
		t.Fatal("3rd request should be blocked (limit 2)")
	}
}

func TestMultiDimRateLimit_DifferentDimensions(t *testing.T) {
	limiter := NewMultiDimRateLimiter(DefaultMultiDimConfigs())
	// 30 requests from user-1 (hits user burst).
	for i := 0; i < 30; i++ {
		limiter.Check(TierFree, "t1", "u1", "", "10.0.0.1", "/api/v1/data")
	}
	// user-1 is blocked.
	r1 := limiter.Check(TierFree, "t1", "u1", "", "10.0.0.1", "/api/v1/data")
	if r1.Allowed {
		t.Fatal("user-1 should be blocked")
	}
	// Different user on same IP should still work (IP has higher limit).
	r2 := limiter.Check(TierFree, "t1", "u2", "", "10.0.0.1", "/api/v1/data")
	if !r2.Allowed {
		t.Fatal("user-2 should be allowed (different dimension)")
	}
}

func TestMultiDimRateLimit_GetUsage(t *testing.T) {
	limiter := NewMultiDimRateLimiter(DefaultMultiDimConfigs())
	limiter.Check(TierFree, "t1", "u1", "k1", "10.0.0.1", "/api/v1/data")
	limiter.Check(TierFree, "t1", "u1", "k1", "10.0.0.1", "/api/v1/data")

	usage := limiter.GetUsage(TierFree, "t1", "u1", "k1", "10.0.0.1", "/api/v1/data")
	if len(usage) != 5 {
		t.Fatalf("expected 5 dimension usages, got %d", len(usage))
	}
	// User burst should show 2 used.
	for _, u := range usage {
		if u.Dimension == DimUser && u.BurstUsed != 2 {
			t.Fatalf("expected user burst_used=2, got %d", u.BurstUsed)
		}
	}
}

func TestMultiDimRateLimitMiddleware(t *testing.T) {
	limiter := NewMultiDimRateLimiter(map[Tier]MultiDimTierConfig{
		TierFree: {
			User:     MultiDimRateLimit{BurstPerMin: 2, SustainedPerHour: 100},
			Tenant:   MultiDimRateLimit{BurstPerMin: 100, SustainedPerHour: 1000},
			IP:       MultiDimRateLimit{BurstPerMin: 100, SustainedPerHour: 1000},
			APIKey:   MultiDimRateLimit{BurstPerMin: 100, SustainedPerHour: 1000},
			Endpoint: MultiDimRateLimit{BurstPerMin: 100, SustainedPerHour: 1000},
		},
	})

	chain := MultiDimRateLimitMiddleware(limiter, func(r *http.Request) (Tier, string, string, string, string, string) {
		return TierFree, "t1", "u1", "", "10.0.0.1", r.URL.Path
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 2 requests should pass.
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		chain.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/test", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d should pass, got %d", i+1, rec.Code)
		}
	}
	// 3rd should be rate limited.
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/test", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd request should be 429, got %d", rec.Code)
	}
	if rec.Header().Get("X-RateLimit-Dimension") == "" {
		t.Fatal("expected X-RateLimit-Dimension header")
	}
}

func TestMultiDimRateLimit_Concurrent(t *testing.T) {
	limiter := NewMultiDimRateLimiter(DefaultMultiDimConfigs())
	var wg sync.WaitGroup
	blocked := 0
	var mu sync.Mutex

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := limiter.Check(TierFree, "t1", "u1", "", "10.0.0.1", "/api/v1/data")
			if !r.Allowed {
				mu.Lock()
				blocked++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if blocked == 0 {
		t.Fatal("expected some requests to be blocked at 50 concurrent with limit 30")
	}
}
