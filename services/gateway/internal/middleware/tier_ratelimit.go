package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Tier represents a tenant's service tier.
type Tier string

const (
	TierFree       Tier = "free"
	TierPro        Tier = "pro"
	TierEnterprise Tier = "enterprise"
)

// TierRateLimitConfig defines rate limits per tier.
type TierRateLimitConfig struct {
	Limits map[Tier]int // requests per minute per tier
	Window time.Duration
}

// DefaultTierRateLimitConfig returns standard tier limits.
func DefaultTierRateLimitConfig() TierRateLimitConfig {
	return TierRateLimitConfig{
		Limits: map[Tier]int{
			TierFree:       100,
			TierPro:        1000,
			TierEnterprise: 0, // 0 = unlimited
		},
		Window: time.Minute,
	}
}

// TierRateLimiter enforces per-tenant-tier rate limits.
// Tier is extracted from JWT claims (set by TenantResolver middleware).
type TierRateLimiter struct {
	cfg     TierRateLimitConfig
	mu      sync.Mutex
	buckets map[string]*tierBucket // key: tenantID
}

type tierBucket struct {
	count    int
	expireAt time.Time
}

// NewTierRateLimiter creates a new tier-based rate limiter.
func NewTierRateLimiter(cfg TierRateLimitConfig) *TierRateLimiter {
	trl := &TierRateLimiter{
		cfg:     cfg,
		buckets: make(map[string]*tierBucket),
	}
	go trl.cleanup()
	return trl
}

// Middleware returns HTTP middleware that enforces tier-based rate limits.
func (trl *TierRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip health checks
		if r.URL.Path == "/healthz" || r.URL.Path == "/docs" {
			next.ServeHTTP(w, r)
			return
		}

		tenantID, _ := TenantIDFromRequest(r)
		if tenantID == "" {
			// No tenant → use free tier defaults
			tenantID = "anonymous"
		}

		// Extract tier from context (set by JWT middleware or header)
		tier := TierFromContext(r.Context())
		if tier == "" {
			tier = TierFree
		}

		limit := trl.cfg.Limits[tier]
		if limit == 0 {
			// Unlimited (enterprise)
			next.ServeHTTP(w, r)
			return
		}

		key := tenantID
		allowed := trl.allow(key, limit)
		if !allowed {
			w.Header().Set("Retry-After", "60")
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Tier", string(tier))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		w.Header().Set("X-RateLimit-Tier", string(tier))
		next.ServeHTTP(w, r)
	})
}

func (trl *TierRateLimiter) allow(key string, limit int) bool {
	trl.mu.Lock()
	defer trl.mu.Unlock()

	now := time.Now()
	b, exists := trl.buckets[key]
	if !exists || now.After(b.expireAt) {
		trl.buckets[key] = &tierBucket{count: 1, expireAt: now.Add(trl.cfg.Window)}
		return true
	}

	if b.count >= limit {
		return false
	}

	b.count++
	return true
}

func (trl *TierRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		trl.mu.Lock()
		now := time.Now()
		for k, b := range trl.buckets {
			if now.After(b.expireAt) {
				delete(trl.buckets, k)
			}
		}
		trl.mu.Unlock()
	}
}

// tierContextKey is the context key for tenant tier.
type tierContextKey struct{}

// TierFromContext extracts the tenant tier from the request context.
func TierFromContext(ctx context.Context) Tier {
	if v, ok := ctx.Value(tierContextKey{}).(Tier); ok {
		return v
	}
	return ""
}

// WithTier sets the tenant tier in the context.
func WithTier(ctx context.Context, tier Tier) context.Context {
	return context.WithValue(ctx, tierContextKey{}, tier)
}
