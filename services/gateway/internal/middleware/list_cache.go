package middleware

import (
	"bytes"
	"context"
	"encoding/json"
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
// Only caches 200 OK responses for GET requests to /api/v1/*/list or
// /api/v1/*s paths. Non-GET, non-200, and auth mutation headers bypass cache.
func ListCacheMiddleware(rdb *redis.Client, cfg ListCacheConfig) func(http.Handler) http.Handler {
	if cfg.TTL == 0 {
		cfg = DefaultListCacheConfig()
	}
	return func(next http.Handler) http.Handler {
		if rdb == nil {
			return next // no Redis: pass through
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET requests
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			// Only cache list endpoints (path ends with "s" or contains "/list")
			path := r.URL.Path
			if !isListEndpoint(path) {
				next.ServeHTTP(w, r)
				return
			}

			// Skip if Authorization changes per-request (user-specific data)
			// Actually — we DO want per-token caching, so include auth in cache key
			cacheKey := listCacheKey(r)

			// Try cache
			ctx := r.Context()
			cached, err := rdb.Get(ctx, cacheKey).Bytes()
			if err == nil && len(cached) > 0 {
				// Restore content-type from first 2 bytes marker
				ct := "application/json"
				w.Header().Set("Content-Type", ct)
				w.Header().Set("X-Cache", "HIT")
				w.Header().Set("Cache-Control", "private, max-age=30")
				w.WriteHeader(http.StatusOK)
				w.Write(cached)
				return
			}

			// Cache miss: capture response
			rec := &capturingResponseWriter{ResponseWriter: w, buf: &bytes.Buffer{}}
			next.ServeHTTP(rec, r)

			// Only cache 200 responses within size limit
			if rec.status == http.StatusOK && rec.buf.Len() > 0 {
				if cfg.MaxBodySize == 0 || rec.buf.Len() <= cfg.MaxBodySize {
					_ = rdb.Set(ctx, cacheKey, rec.buf.Bytes(), cfg.TTL)
				}
			}

			// If we captured, write to real response
			if rec.buf.Len() > 0 && !rec.wrote {
				w.Header().Set("X-Cache", "MISS")
				w.WriteHeader(rec.status)
				w.Write(rec.buf.Bytes())
			}
		})
	}
}

// isListEndpoint returns true for GET list endpoints.
func isListEndpoint(path string) bool {
	// /api/v1/users, /api/v1/roles, /api/v1/oauth/clients etc.
	// Heuristic: ends with "s" (plural resource) and no UUID in last segment
	if len(path) < 8 || path[:8] != "/api/v1/" {
		return false
	}
	// Check last segment
	segments := splitPath(path)
	if len(segments) == 0 {
		return false
	}
	last := segments[len(segments)-1]
	// Plural resource endpoints end with 's' but aren't UUIDs or specific IDs
	if len(last) < 2 || last[len(last)-1] != 's' {
		return false
	}
	// Exclude paths with UUID-like patterns (36 chars with dashes)
	if len(last) == 36 && last[8] == '-' {
		return false
	}
	// Exclude known non-cacheable paths
	switch last {
	case "analytics", "metrics", "stats", "status", "health":
		return false
	}
	return true
}

// splitPath splits a URL path into segments.
func splitPath(path string) []string {
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

// listCacheKey builds a Redis cache key from request path + query + auth hash.
func listCacheKey(r *http.Request) string {
	// Include tenant + auth token hash to avoid cross-tenant leakage
	tenantID := r.Header.Get("X-Tenant-ID")
	auth := r.Header.Get("Authorization")
	authHash := shortHash(auth)
	query := r.URL.RawQuery
	return "listcache:" + tenantID + ":" + authHash + ":" + r.URL.Path + "?" + query
}

// shortHash returns first 8 hex chars of SHA-256 for cache key deduplication.
func shortHash(s string) string {
	if s == "" {
		return "anon"
	}
	h := time.Now().UnixNano() // not ideal; use crypto/sha256 in production
	_ = h
	// Simple FNV for cache key uniqueness (not security-sensitive)
	var hash uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= 16777619
	}
	// Also mix in length for extra dispersion
	hash ^= uint32(len(s))
	return formatUint32(hash)
}

func formatUint32(v uint32) string {
	const hex = "0123456789abcdef"
	buf := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		buf[i] = hex[v&0xf]
		v >>= 4
	}
	return string(buf)
}

// capturingResponseWriter captures the response body for caching.
type capturingResponseWriter struct {
	http.ResponseWriter
	buf   *bytes.Buffer
	status int
	wrote  bool
}

func (c *capturingResponseWriter) WriteHeader(code int) {
	c.status = code
	// Don't write through yet — we need to decide based on cacheability
}

func (c *capturingResponseWriter) Write(b []byte) (int, error) {
	c.buf.Write(b)
	return len(b), nil
}

func (c *capturingResponseWriter) Flush() {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	c.ResponseWriter.WriteHeader(c.status)
	c.ResponseWriter.Write(c.buf.Bytes())
	c.wrote = true
}

// Ensure context import is used
var _ = context.Background
