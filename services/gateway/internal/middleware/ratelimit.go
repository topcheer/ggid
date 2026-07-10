package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimitConfig defines per-endpoint rate limits.
type RateLimitConfig struct {
	LoginLimit    int           // requests per minute for login
	RegisterLimit int           // requests per minute for register
	APILimit      int           // requests per minute for general API
	Window        time.Duration // sliding window
}

// DefaultRateLimitConfig returns production-ready defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		LoginLimit:    5,
		RegisterLimit: 3,
		APILimit:      100,
		Window:        time.Minute,
	}
}

// RateLimiter provides in-memory fixed-window rate limiting.
// For production with multiple gateway instances, replace with Redis-backed limiter.
type rateBucket struct {
	count    int
	expireAt time.Time
}

type RateLimiter struct {
	cfg     RateLimitConfig
	mu      sync.Mutex
	buckets map[string]*rateBucket
}

// NewRateLimiter creates a new rate limiter with the given config.
func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		cfg:     cfg,
		buckets: make(map[string]*rateBucket),
	}
	// Background cleanup of expired buckets
	go rl.cleanup()
	return rl
}

// Middleware returns an HTTP middleware that enforces rate limits.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for non-API paths
		if r.URL.Path == "/healthz" || r.URL.Path == "/docs" ||
			r.URL.Path == "/api-docs" || r.URL.Path == "/login" ||
			r.URL.Path == "/register" {
			next.ServeHTTP(w, r)
			return
		}

		limit := rl.getLimit(r.URL.Path)
		if limit == 0 {
			next.ServeHTTP(w, r)
			return
		}

		key := rl.bucketKey(r)
		allowed, remaining, resetAt := rl.allow(key, limit)

		// Set standard rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		if !allowed {
			secs := int(time.Until(resetAt).Seconds())
			if secs < 1 {
				secs = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "rate limit exceeded",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getLimit(path string) int {
	switch {
	case path == "/api/v1/auth/login":
		return rl.cfg.LoginLimit
	case path == "/api/v1/auth/register":
		return rl.cfg.RegisterLimit
	case len(path) > 8 && path[:8] == "/api/v1/":
		return rl.cfg.APILimit
	default:
		return 0 // no limit
	}
}

func (rl *RateLimiter) bucketKey(r *http.Request) string {
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = fwd
	}
	return fmt.Sprintf("%s:%s", r.URL.Path, ip)
}

func (rl *RateLimiter) allow(key string, limit int) (allowed bool, remaining int, resetAt time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.buckets[key]

	if !exists || now.After(bucket.expireAt) {
		// New window
		resetAt = now.Add(rl.cfg.Window)
		rl.buckets[key] = &rateBucket{count: 1, expireAt: resetAt}
		return true, limit - 1, resetAt
	}

	if bucket.count >= limit {
		return false, 0, bucket.expireAt
	}

	bucket.count++
	return true, limit - bucket.count, bucket.expireAt
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for k, b := range rl.buckets {
			if now.After(b.expireAt) {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}
