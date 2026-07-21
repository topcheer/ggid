package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// GatewayError is a unified JSON error response for 502/503/504.
type GatewayError struct {
	Error     string    `json:"error"`
	Code      int       `json:"code"`
	RequestID string    `json:"request_id"`
	Backend   string    `json:"backend,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

// ErrorHandler wraps the reverse proxy error handler to return unified JSON errors.
func ErrorHandler() func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = "unknown"
		}

		code := http.StatusBadGateway // 502 default
		msg := "upstream connection failed"

		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "context deadline exceeded") {
			code = http.StatusGatewayTimeout // 504
			msg = "upstream timeout"
		} else if strings.Contains(err.Error(), "connection refused") {
			code = http.StatusBadGateway // 502
			msg = "upstream unavailable"
		}

		gwErr := GatewayError{
			Error:     http.StatusText(code),
			Code:      code,
			RequestID: reqID,
			Timestamp: time.Now().UTC(),
			Message:   msg,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(gwErr)
	}
}

// WriteGatewayError writes a custom error page for a given status code.
func WriteGatewayError(w http.ResponseWriter, r *http.Request, code int, backend, msg string) {
	reqID := r.Header.Get("X-Request-ID")
	if reqID == "" {
		reqID = "unknown"
	}
	gwErr := GatewayError{
		Error:     http.StatusText(code),
		Code:      code,
		RequestID: reqID,
		Backend:   backend,
		Timestamp: time.Now().UTC(),
		Message:   msg,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(gwErr)
}

// --- CORS Preflight Cache ---

// PreflightCache caches OPTIONS preflight responses to reduce backend load.
type PreflightCache struct {
	mu      sync.RWMutex
	entries map[string]*preflightEntry
	ttl     time.Duration
}

type preflightEntry struct {
	status  int
	header  http.Header
	expires time.Time
}

// NewPreflightCache creates a cache with the given TTL (default 5 minutes).
func NewPreflightCache(ttl time.Duration) *PreflightCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &PreflightCache{
		entries: make(map[string]*preflightEntry),
		ttl:     ttl,
	}
}

// preflightKey generates a cache key from origin + path + method.
func preflightKey(r *http.Request) string {
	origin := r.Header.Get("Origin")
	method := r.Header.Get("Access-Control-Request-Method")
	return fmt.Sprintf("%s|%s|%s", origin, r.URL.Path, method)
}

// Get returns cached preflight response if still valid.
func (c *PreflightCache) Get(r *http.Request) (*preflightEntry, bool) {
	key := preflightKey(r)
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expires) {
		return nil, false
	}
	return entry, true
}

// Set caches a preflight response.
func (c *PreflightCache) Set(r *http.Request, status int, header http.Header) {
	key := preflightKey(r)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = &preflightEntry{
		status:  status,
		header:  header.Clone(),
		expires: time.Now().Add(c.ttl),
	}
}

// PreflightCacheMiddleware caches OPTIONS preflight responses.
// If a cached response exists, it's returned immediately without calling the backend.
func PreflightCacheMiddleware(cache *PreflightCache, corsHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodOptions {
			corsHandler.ServeHTTP(w, r)
			return
		}

		// Check cache
		if entry, ok := cache.Get(r); ok {
			for k, v := range entry.header {
				w.Header()[k] = v
			}
			w.WriteHeader(entry.status)
			return
		}

		// Capture response and cache it
		rec := &preflightRecorder{ResponseWriter: w, status: http.StatusOK}
		corsHandler.ServeHTTP(rec, r)

		// Only cache successful preflight responses
		if rec.status >= 200 && rec.status < 300 {
			cache.Set(r, rec.status, rec.Header())
		}
	})
}

type preflightRecorder struct {
	http.ResponseWriter
	status int
}

func (p *preflightRecorder) WriteHeader(code int) {
	p.status = code
	p.ResponseWriter.WriteHeader(code)
}

func (p *preflightRecorder) Header() http.Header {
	return p.ResponseWriter.Header()
}

// --- Per-Route Body Size Limit ---

// RouteBodySizeConfig holds per-route body size limits.
type RouteBodySizeConfig struct {
	Limits map[string]int64 // route prefix → max bytes (0 = unlimited)
	Default int64            // default limit if route not found
}

// NewRouteBodySizeConfig creates config with defaults.
func NewRouteBodySizeConfig() *RouteBodySizeConfig {
	return &RouteBodySizeConfig{
		Limits: map[string]int64{
			"/api/v1/auth/verify":    1 * 1024,       // 1KB for login
			"/api/v1/auth/register": 2 * 1024,       // 2KB for register
			"/api/v1/audit":         0,              // unlimited for audit queries
		},
		Default: 10 * 1024 * 1024, // 10MB default
	}
}

// GetLimit returns the body size limit for a given path.
func (c *RouteBodySizeConfig) GetLimit(path string) int64 {
	for prefix, limit := range c.Limits {
		if strings.HasPrefix(path, prefix) {
			return limit
		}
	}
	return c.Default
}

// RouteBodySizeMiddleware enforces per-route body size limits.
func RouteBodySizeMiddleware(cfg *RouteBodySizeConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limit := cfg.GetLimit(r.URL.Path)
			if limit > 0 {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}
