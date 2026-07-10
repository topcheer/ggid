package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// StickySessionConfig controls sticky session routing behavior.
type StickySessionConfig struct {
	// CookieName is the name of the cookie used to track sticky sessions.
	CookieName string

	// HeaderName is an alternative header to read the sticky key from.
	// If empty, only cookie is used.
	HeaderName string

	// TTL is how long a sticky binding persists.
	TTL time.Duration

	// Backends is the list of backend URLs for load balancing.
	Backends []string
}

// DefaultStickyConfig returns sensible defaults.
func DefaultStickyConfig() *StickySessionConfig {
	return &StickySessionConfig{
		CookieName: "ggid_sticky",
		HeaderName: "X-Sticky-Key",
		TTL:        30 * time.Minute,
		Backends:   []string{},
	}
}

// StickyRouter routes requests to backends based on a sticky key.
// The same sticky key always maps to the same backend until the TTL expires.
type StickyRouter struct {
	mu      sync.RWMutex
	config  *StickySessionConfig
	bindings map[string]*stickyBinding
}

type stickyBinding struct {
	backend   string
	createdAt time.Time
}

// NewStickyRouter creates a sticky session router.
func NewStickyRouter(cfg *StickySessionConfig) *StickyRouter {
	if cfg == nil {
		cfg = DefaultStickyConfig()
	}
	if cfg.CookieName == "" {
		cfg.CookieName = "ggid_sticky"
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 30 * time.Minute
	}
	return &StickyRouter{
		config:   cfg,
		bindings: make(map[string]*stickyBinding),
	}
}

// ResolveBackend returns the backend URL for the given request.
// If a sticky key is present and has an active binding, returns that backend.
// Otherwise, picks a backend using consistent hashing and creates a new binding.
func (sr *StickyRouter) ResolveBackend(r *http.Request) string {
	if len(sr.config.Backends) == 0 {
		return ""
	}
	if len(sr.config.Backends) == 1 {
		return sr.config.Backends[0]
	}

	key := sr.extractKey(r)
	if key == "" {
		// No sticky key — pick first backend
		return sr.config.Backends[0]
	}

	// Check existing binding
	sr.mu.RLock()
	binding, ok := sr.bindings[key]
	sr.mu.RUnlock()

	if ok && time.Since(binding.createdAt) < sr.config.TTL {
		return binding.backend
	}

	// Create new binding using consistent hash
	idx := consistentHash(key, len(sr.config.Backends))
	backend := sr.config.Backends[idx]

	sr.mu.Lock()
	sr.bindings[key] = &stickyBinding{
		backend:   backend,
		createdAt: time.Now(),
	}
	sr.mu.Unlock()

	return backend
}

// SetStickyCookie writes the sticky session cookie to the response.
func (sr *StickyRouter) SetStickyCookie(w http.ResponseWriter, r *http.Request) {
	key := sr.extractKey(r)
	if key != "" {
		return // already has cookie
	}
	// Generate a new sticky key from request attributes
	key = generateStickyKey(r)
	http.SetCookie(w, &http.Cookie{
		Name:     sr.config.CookieName,
		Value:    key,
		Path:     "/",
		MaxAge:   int(sr.config.TTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// extractKey reads the sticky key from header or cookie.
func (sr *StickyRouter) extractKey(r *http.Request) string {
	// Try header first
	if sr.config.HeaderName != "" {
		if v := r.Header.Get(sr.config.HeaderName); v != "" {
			return v
		}
	}
	// Then cookie
	if c, err := r.Cookie(sr.config.CookieName); err == nil {
		return c.Value
	}
	return ""
}

// BindingCount returns the number of active sticky bindings.
func (sr *StickyRouter) BindingCount() int {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return len(sr.bindings)
}

// CleanupExpired removes expired bindings.
func (sr *StickyRouter) CleanupExpired() {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	now := time.Now()
	for key, binding := range sr.bindings {
		if now.Sub(binding.createdAt) >= sr.config.TTL {
			delete(sr.bindings, key)
		}
	}
}

// consistentHash maps a string key to an index in [0, max).
func consistentHash(key string, max int) int {
	h := sha256.Sum256([]byte(key))
	var val uint64
	for i := 0; i < 8; i++ {
		val = val<<8 | uint64(h[i])
	}
	return int(val % uint64(max))
}

// generateStickyKey creates a new sticky key from request attributes.
func generateStickyKey(r *http.Request) string {
	raw := r.RemoteAddr + "|" + strconv.FormatInt(time.Now().UnixNano(), 36)
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:8])
}

// StickyMiddleware injects sticky routing into the request pipeline.
// It resolves the backend and sets the sticky cookie on the response.
func StickyMiddleware(router *StickyRouter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backend := router.ResolveBackend(r)
		if backend != "" {
			r.Header.Set("X-Sticky-Backend", backend)
		}
		router.SetStickyCookie(w, r)
		next.ServeHTTP(w, r)
	})
}
