package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// --- RateLimitStore interface ---

// RateLimitStore is the storage backend for sliding window rate limiting.
// Implementations must be safe for concurrent use.
type RateLimitStore interface {
	// CountAndAdd returns the number of entries in the half-open interval
	// [windowStart, now) and atomically adds a new entry at timestamp 'now'.
	CountAndAdd(ctx context.Context, key string, windowStart, now time.Time) (int64, error)
}

// --- In-memory store (fallback when Redis is unavailable) ---

type memBucket struct {
	timestamps []int64 // sorted nanosecond timestamps
}

type MemoryRateLimitStore struct {
	mu      sync.Mutex
	buckets map[string]*memBucket
}

// NewMemoryRateLimitStore creates an in-memory rate limit store suitable for
// single-instance deployments or testing.
func NewMemoryRateLimitStore() *MemoryRateLimitStore {
	return &MemoryRateLimitStore{buckets: make(map[string]*memBucket)}
}

func (s *MemoryRateLimitStore) CountAndAdd(_ context.Context, key string, windowStart, now time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.buckets[key]
	if !ok {
		b = &memBucket{}
		s.buckets[key] = b
	}

	cutoff := windowStart.UnixNano()
	nowNs := now.UnixNano()

	// In-place compaction: keep only entries within the window.
	var idx int
	for _, ts := range b.timestamps {
		if ts > cutoff {
			b.timestamps[idx] = ts
			idx++
		}
	}
	b.timestamps = b.timestamps[:idx]

	count := int64(len(b.timestamps))
	b.timestamps = append(b.timestamps, nowNs)
	return count, nil
}

// --- Redis-backed store ---

// slidingWindowLua atomically removes expired entries, counts remaining ones,
// adds the new request, and sets a TTL on the key.
const slidingWindowLua = `
local key = KEYS[1]
local window_start = tonumber(ARGV[1])
local now_ns = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3])

redis.call('ZREMRANGEBYSCORE', key, '-inf', '(' .. window_start)
local count = redis.call('ZCARD', key)
redis.call('ZADD', key, now_ns, now_ns .. ':' .. redis.call('INCR', key .. ':seq'))
redis.call('EXPIRE', key, ttl)

return count
`

// RedisRateLimitStore implements RateLimitStore using Redis sorted sets.
type RedisRateLimitStore struct {
	client redis.Cmdable
}

// NewRedisRateLimitStore creates a Redis-backed rate limit store.
func NewRedisRateLimitStore(client redis.Cmdable) *RedisRateLimitStore {
	return &RedisRateLimitStore{client: client}
}

func (s *RedisRateLimitStore) CountAndAdd(ctx context.Context, key string, windowStart, now time.Time) (int64, error) {
	ttl := int64(120) // seconds; generous to cover the window + margin
	result, err := s.client.Eval(ctx, slidingWindowLua, []string{key},
		windowStart.UnixNano(), now.UnixNano(), ttl).Int64()
	if err != nil {
		return 0, fmt.Errorf("redis sliding window eval: %w", err)
	}
	return result, nil
}

// --- Configuration ---

// SlidingWindowTierLimit defines per-tier rate limit parameters.
type SlidingWindowTierLimit struct {
	Requests int           // max requests in the window
	Window   time.Duration // sliding window duration
}

// DefaultSlidingWindowTiers returns production-ready tier limits.
func DefaultSlidingWindowTiers() map[Tier]SlidingWindowTierLimit {
	return map[Tier]SlidingWindowTierLimit{
		TierFree:       {Requests: 100, Window: time.Minute},
		TierStarter:    {Requests: 500, Window: time.Minute},
		TierPro:        {Requests: 5000, Window: time.Minute},
		TierEnterprise: {Requests: 0, Window: time.Minute}, // unlimited
	}
}

// Extra tier constant for the "starter" tier requested by the arch spec.
const TierStarter Tier = "starter"

// SlidingWindowConfig configures the sliding window rate limiter.
type SlidingWindowConfig struct {
	Tiers     map[Tier]SlidingWindowTierLimit
	KeyPrefix string // prepended to Redis keys (e.g. "ratelimit:")
}

// DefaultSlidingWindowConfig returns a config with standard tier limits.
func DefaultSlidingWindowConfig() SlidingWindowConfig {
	return SlidingWindowConfig{
		Tiers:     DefaultSlidingWindowTiers(),
		KeyPrefix: "ratelimit:",
	}
}

// --- SlidingWindowLimiter ---

// SlidingWindowLimiter enforces per-tenant sliding window rate limits.
// It uses a RateLimitStore (Redis for distributed, in-memory for standalone)
// and derives the tier from the request context (set by JWT middleware).
type SlidingWindowLimiter struct {
	store   RateLimitStore
	cfg     SlidingWindowConfig
	clock   func() time.Time
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter.
func NewSlidingWindowLimiter(store RateLimitStore, cfg SlidingWindowConfig) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		store: store,
		cfg:   cfg,
		clock: time.Now,
	}
}

// Middleware returns HTTP middleware that enforces sliding window rate limits.
func (sw *SlidingWindowLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip health checks and documentation
		if r.URL.Path == "/healthz" || r.URL.Path == "/healthz/live" ||
			r.URL.Path == "/healthz/ready" || r.URL.Path == "/docs" {
			next.ServeHTTP(w, r)
			return
		}

		// Determine tenant
		tenantID, _ := TenantIDFromRequest(r)
		if tenantID == "" {
			tenantID = "anonymous"
		}

		// Determine tier from context (set by JWT or tier middleware)
		tier := TierFromContext(r.Context())
		if tier == "" {
			tier = TierFree
		}

		limit, ok := sw.cfg.Tiers[tier]
		if !ok {
			// Unknown tier → default to free
			limit = sw.cfg.Tiers[TierFree]
		}

		// Unlimited (enterprise with 0 requests)
		if limit.Requests == 0 {
			next.ServeHTTP(w, r)
			return
		}

		now := sw.clock()
		windowStart := now.Add(-limit.Window)
		key := fmt.Sprintf("%s%s:%s", sw.cfg.KeyPrefix, tenantID, tier)

		count, err := sw.store.CountAndAdd(r.Context(), key, windowStart, now)
		if err != nil {
			// On store errors, fail open (allow the request) but log
			// In production, you might want to fail closed depending on requirements
			next.ServeHTTP(w, r)
			return
		}

		remaining := limit.Requests - int(count) - 1
		if remaining < 0 {
			remaining = 0
		}
		resetAt := now.Add(limit.Window).Truncate(time.Second)

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit.Requests))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
		w.Header().Set("X-RateLimit-Tier", string(tier))

		if int(count) >= limit.Requests {
			secs := int(time.Until(resetAt).Seconds())
			if secs < 1 {
				secs = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error":"rate limit exceeded","tier":"%s","retry_after":%d}`, tier, secs)
			return
		}

		next.ServeHTTP(w, r)
	})
}
