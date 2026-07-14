package middleware

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/sysconfig"
)

// TokenBucket implements a classic token-bucket rate limiter.
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// NewTokenBucket creates a new token bucket.
// maxTokens is the burst capacity; refillPerSec is the sustained rate.
func NewTokenBucket(maxTokens, refillPerSec float64) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillPerSec,
		lastRefill: time.Now(),
	}
}

// Allow tries to consume one token. Returns true if allowed, false if rate limited.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// Tokens returns the current number of available tokens (for inspection/testing).
func (tb *TokenBucket) Tokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.tokens
}

// RetryAfter returns how many seconds until the next token is available.
func (tb *TokenBucket) RetryAfter() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if tb.tokens >= 1 {
		return 0
	}
	if tb.refillRate <= 0 {
		return 1 // minimum retry window when no refill configured
	}
	deficit := 1 - tb.tokens
	secs := deficit / tb.refillRate
	return int(secs) + 1
}

// BucketRateLimitConfig configures the token-bucket rate limiter per tenant tier.
type BucketRateLimitConfig struct {
	DefaultMaxTokens   float64
	DefaultRefillPerSec float64
	TierOverrides      map[string]BucketTierConfig
}

// BucketTierConfig defines rate limits for a specific tier.
type BucketTierConfig struct {
	MaxTokens    float64
	RefillPerSec float64
}

// DefaultBucketRateLimitConfig returns sensible defaults.
// Override with GATEWAY_RATE_LIMIT_TOKENS and GATEWAY_RATE_LIMIT_REFILL env vars.
func DefaultBucketRateLimitConfig() *BucketRateLimitConfig {
	cfg := &BucketRateLimitConfig{
		DefaultMaxTokens:    100,
		DefaultRefillPerSec: 10, // 600/min sustained
		TierOverrides: map[string]BucketTierConfig{
			"free":       {MaxTokens: 20, RefillPerSec: 2},    // 120/min
			"pro":        {MaxTokens: 100, RefillPerSec: 10},  // 600/min
			"enterprise": {MaxTokens: 1000, RefillPerSec: 100}, // 6000/min
		},
	}

	// Env var overrides (for test environments)
	if v := os.Getenv("GATEWAY_RATE_LIMIT_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f := float64(n)
			cfg.DefaultMaxTokens = f
			for tier := range cfg.TierOverrides {
				cfg.TierOverrides[tier] = BucketTierConfig{MaxTokens: f, RefillPerSec: f / 5}
			}
		}
	}
	if v := os.Getenv("GATEWAY_RATE_LIMIT_REFILL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f := float64(n)
			cfg.DefaultRefillPerSec = f
			for tier := range cfg.TierOverrides {
				t := cfg.TierOverrides[tier]
				t.RefillPerSec = f
				cfg.TierOverrides[tier] = t
			}
		}
	}

	return cfg
}

// TenantBucketLimiter enforces per-tenant + IP token-bucket rate limits.
type TenantBucketLimiter struct {
	cfg     *BucketRateLimitConfig
	mu      sync.RWMutex
	buckets map[string]*TokenBucket // key: tenantID:ip
	store   sysconfig.Store
}

// NewTenantBucketLimiter creates a new per-tenant bucket limiter.
func NewTenantBucketLimiter(cfg *BucketRateLimitConfig) *TenantBucketLimiter {
	if cfg == nil {
		cfg = DefaultBucketRateLimitConfig()
	}
	return &TenantBucketLimiter{
		cfg:     cfg,
		buckets: make(map[string]*TokenBucket),
	}
}

// SetSysconfigStore injects the system config store for hot-reloadable rate limits.
func (tbl *TenantBucketLimiter) SetSysconfigStore(store sysconfig.Store) {
	tbl.mu.Lock()
	defer tbl.mu.Unlock()
	tbl.store = store
}

// getBucket returns (or creates) the token bucket for the given key.
// When a sysconfig store is wired, new buckets use the current store values.
func (tbl *TenantBucketLimiter) getBucket(key string, tier string, r *http.Request) *TokenBucket {
	tbl.mu.RLock()
	if b, ok := tbl.buckets[key]; ok {
		tbl.mu.RUnlock()
		return b
	}
	tbl.mu.RUnlock()

	// Create new bucket
	maxTokens := tbl.cfg.DefaultMaxTokens
	refill := tbl.cfg.DefaultRefillPerSec
	if override, ok := tbl.cfg.TierOverrides[tier]; ok {
		maxTokens = override.MaxTokens
		refill = override.RefillPerSec
	}

	if tbl.store != nil && r != nil {
		tenantID, _ := TenantIDFromRequest(r)
		if tenantID == "" {
			tenantID = "default"
		}
		sc := tbl.store.Get(tenantID)
		if sc.GatewayRateLimitTokens > 0 {
			maxTokens = sc.GatewayRateLimitTokens
		}
		if sc.GatewayRateLimitRefillPerSec > 0 {
			refill = sc.GatewayRateLimitRefillPerSec
		}
	}

	tbl.mu.Lock()
	defer tbl.mu.Unlock()
	// Double-check after acquiring write lock
	if b, ok := tbl.buckets[key]; ok {
		return b
	}
	b := NewTokenBucket(maxTokens, refill)
	tbl.buckets[key] = b
	return b
}

// bucketKey builds the composite key from tenant ID and client IP.
func bucketKey(tenantID, ip string) string {
	return tenantID + ":" + ip
}

// Middleware returns HTTP middleware that enforces token-bucket rate limits
// keyed by tenant_id + client IP.
func (tbl *TenantBucketLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip health checks
		if r.URL.Path == "/healthz" || r.URL.Path == "/docs" || r.URL.Path == "/openapi.json" {
			next.ServeHTTP(w, r)
			return
		}

		tenantID, _ := TenantIDFromRequest(r)
		if tenantID == "" {
			tenantID = "anonymous"
		}

		ip := ClientIP(r)

		tier := string(TierFromContext(r.Context()))
		if tier == "" {
			tier = "free"
		}

		key := bucketKey(tenantID, ip)
		bucket := tbl.getBucket(key, tier, r)

		if !bucket.Allow() {
			retryAfter := bucket.RetryAfter()
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", bucket.maxTokens))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%.0f", bucket.Tokens()))
			w.Header().Set("X-RateLimit-Tier", tier)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate_limited","message":"too many requests","retry_after":` + strconv.Itoa(retryAfter) + `}`))
			return
		}

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", bucket.maxTokens))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%.0f", bucket.Tokens()))
		next.ServeHTTP(w, r)
	})
}

// ClientIP extracts the real client IP from common proxy headers.
func ClientIP(r *http.Request) string {
	// Check X-Forwarded-For first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Use the first IP in the list
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr (strip port)
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// Cleanup removes expired idle buckets (called periodically).
func (tbl *TenantBucketLimiter) Cleanup(maxAge time.Duration) {
	tbl.mu.Lock()
	defer tbl.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	for k, b := range tbl.buckets {
		if b.lastRefill.Before(cutoff) {
			delete(tbl.buckets, k)
		}
	}
}

// BucketCount returns the number of active buckets (for metrics/testing).
func (tbl *TenantBucketLimiter) BucketCount() int {
	tbl.mu.RLock()
	defer tbl.mu.RUnlock()
	return len(tbl.buckets)
}
