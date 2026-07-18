package middleware

import (
	"bytes"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// ListCacheConfig controls GET list endpoint caching.
type ListCacheConfig struct {
	TTL         time.Duration // default: 30s
	MaxBodySize int           // skip caching for responses larger than this (bytes). 0 = unlimited
}

// DefaultListCacheConfig returns production-ready defaults.
func DefaultListCacheConfig() ListCacheConfig {
	return ListCacheConfig{
		TTL:         30 * time.Second,
		MaxBodySize: 256 * 1024, // skip caching responses > 256KB
	}
}

// ListCacheMiddleware caches GET list endpoint responses in Redis.
// Only caches 200 OK responses for GET requests to list-type endpoints.
// Cache key includes tenant ID + auth token hash to prevent cross-tenant leakage.
// POST/PUT/DELETE always bypass cache. Non-200 responses bypass cache.
func ListCacheMiddleware(rdb *redis.Client, cfg ListCacheConfig) func(http.Handler) http.Handler {
	if cfg.TTL == 0 {
		cfg = DefaultListCacheConfig()
	}
	return func(next http.Handler) http.Handler {
		if rdb == nil {
			return next // no Redis: pass through without caching
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET requests to list endpoints
			if r.Method != http.MethodGet || !isListEndpoint(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			cacheKey := listCacheKey(r)

			// Try cache first
			cached, err := rdb.Get(r.Context(), cacheKey).Bytes()
			if err == nil && len(cached) > 0 {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cache", "HIT")
				w.Header().Set("Cache-Control", "private, max-age=30")
				w.WriteHeader(http.StatusOK)
				w.Write(cached)
				return
			}

			// Cache miss: capture response, then decide whether to store
			rec := &captureWriter{ResponseWriter: w, buf: &bytes.Buffer{}}
			next.ServeHTTP(rec, r)

			// Always write the captured response to the client
			if rec.buf.Len() > 0 {
				if rec.status == 0 {
					rec.status = http.StatusOK
				}
				w.Header().Set("X-Cache", "MISS")
				w.WriteHeader(rec.status)
				w.Write(rec.buf.Bytes())

				// Store in Redis if cacheable
				if rec.status == http.StatusOK &&
					(cfg.MaxBodySize == 0 || rec.buf.Len() <= cfg.MaxBodySize) {
					rdb.Set(r.Context(), cacheKey, rec.buf.Bytes(), cfg.TTL)
				}
			}
		})
	}
}

// isListEndpoint returns true for GET list endpoints.
func isListEndpoint(path string) bool {
	if len(path) < 8 || path[:8] != "/api/v1/" {
		return false
	}
	segments := splitURLPath(path)
	if len(segments) == 0 {
		return false
	}
	last := segments[len(segments)-1]
	if len(last) < 2 || last[len(last)-1] != 's' {
		return false
	}
	// Exclude UUID-like patterns (36 chars with dashes at position 8)
	if len(last) == 36 && last[8] == '-' {
		return false
	}
	switch last {
	case "analytics", "metrics", "stats", "status", "health":
		return false
	}
	return true
}

// splitURLPath splits a URL path into segments.
func splitURLPath(path string) []string {
	var segments []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				segments = append(segments, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		segments = append(segments, path[start:])
	}
	return segments
}

// listCacheKey builds a Redis cache key from request path + query + tenant + auth.
func listCacheKey(r *http.Request) string {
	tenantID := r.Header.Get("X-Tenant-ID")
	auth := r.Header.Get("Authorization")
	authHash := fnv1aShort(auth)
	return "listcache:" + tenantID + ":" + authHash + ":" + r.URL.Path + "?" + r.URL.RawQuery
}

// fnv1aShort returns an 8-char hex hash for cache key uniqueness.
func fnv1aShort(s string) string {
	if s == "" {
		return "anon"
	}
	var hash uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= 16777619
	}
	hash ^= uint32(len(s))
	const hex = "0123456789abcdef"
	buf := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		buf[i] = hex[hash&0xf]
		hash >>= 4
	}
	return string(buf)
}

// captureWriter captures response body for caching while passing headers through.
type captureWriter struct {
	http.ResponseWriter
	buf    *bytes.Buffer
	status int
}

func (c *captureWriter) WriteHeader(code int) {
	c.status = code
	// Don't delegate to inner ResponseWriter yet — we control timing
}

func (c *captureWriter) Write(b []byte) (int, error) {
	return c.buf.Write(b)
}
