package middleware

import (
	"net/http"
	"sync"
	"time"
)

// SlowRequestConfig controls slow request detection.
type SlowRequestConfig struct {
	// Threshold is the duration after which a request is considered slow.
	Threshold time.Duration

	// LogSlowRequests enables logging of slow requests.
	LogSlowRequests bool

	// OnSlow is called when a slow request is detected (e.g. publish to NATS).
	OnSlow func(info *SlowRequestInfo)
}

// SlowRequestInfo contains metadata about a slow request.
type SlowRequestInfo struct {
	Method    string        `json:"method"`
	Path      string        `json:"path"`
	Duration  time.Duration `json:"duration_ms"`
	TenantID  string        `json:"tenant_id,omitempty"`
	Backend   string        `json:"backend,omitempty"`
	RequestID string        `json:"request_id,omitempty"`
}

// DefaultSlowRequestConfig returns sensible defaults (5s threshold).
func DefaultSlowRequestConfig() *SlowRequestConfig {
	return &SlowRequestConfig{
		Threshold:       5 * time.Second,
		LogSlowRequests: true,
	}
}

// SlowRequestMiddleware detects requests that exceed the configured threshold.
// Slow requests are logged and optionally reported via OnSlow callback.
func SlowRequestMiddleware(cfg *SlowRequestConfig) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = DefaultSlowRequestConfig()
	}
	if cfg.Threshold <= 0 {
		cfg.Threshold = 5 * time.Second
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &slowRequestRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			elapsed := time.Since(start)
			if elapsed >= cfg.Threshold {
				info := &SlowRequestInfo{
					Method:    r.Method,
					Path:      r.URL.Path,
					Duration:  elapsed,
					TenantID:  r.Header.Get("X-Tenant-ID"),
					RequestID: r.Header.Get("X-Request-ID"),
				}
				if cfg.OnSlow != nil {
					cfg.OnSlow(info)
				}
			}
		})
	}
}

// --- WebSocket Connection Limiter ---

// WSConnLimiter enforces per-tenant WebSocket connection limits.
// When the limit is exceeded, the oldest connection is evicted.
type WSConnLimiter struct {
	mu          sync.Mutex
	maxPerTenant int
	connections  map[string][]string // tenantID → ordered list of session IDs
	sessions    map[string]string   // sessionID → tenantID
}

// NewWSConnLimiter creates a limiter with the given per-tenant max.
func NewWSConnLimiter(maxPerTenant int) *WSConnLimiter {
	if maxPerTenant <= 0 {
		maxPerTenant = 100
	}
	return &WSConnLimiter{
		maxPerTenant: maxPerTenant,
		connections:  make(map[string][]string),
		sessions:     make(map[string]string),
	}
}

// Allow checks if a new connection is allowed for the given tenant.
// Returns true if allowed. If the tenant is at capacity, returns false
// and the session ID of the oldest connection to evict.
func (l *WSConnLimiter) Allow(tenantID, sessionID string) (allowed bool, evictSessionID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	conns := l.connections[tenantID]
	if len(conns) >= l.maxPerTenant {
		// Evict oldest
		evictID := conns[0]
		l.connections[tenantID] = append(conns[1:], sessionID)
		delete(l.sessions, evictID)
		l.sessions[sessionID] = tenantID
		return true, evictID
	}

	l.connections[tenantID] = append(conns, sessionID)
	l.sessions[sessionID] = tenantID
	return true, ""
}

// Release removes a connection from the limiter.
func (l *WSConnLimiter) Release(sessionID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	tenantID, ok := l.sessions[sessionID]
	if !ok {
		return
	}
	delete(l.sessions, sessionID)

	conns := l.connections[tenantID]
	for i, id := range conns {
		if id == sessionID {
			l.connections[tenantID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	if len(l.connections[tenantID]) == 0 {
		delete(l.connections, tenantID)
	}
}

// Count returns the number of active connections for a tenant.
func (l *WSConnLimiter) Count(tenantID string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.connections[tenantID])
}

// TotalCount returns the total number of active connections across all tenants.
func (l *WSConnLimiter) TotalCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.sessions)
}

// --- Fallback Response ---

// FallbackConfig holds per-route fallback responses.
type FallbackConfig struct {
	// Responses maps route prefix to a cached successful response.
	Responses map[string]*CachedResponse
}

// CachedResponse is a previously successful response used as fallback.
type CachedResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	CachedAt   time.Time
}

// NewFallbackConfig creates an empty fallback config.
func NewFallbackConfig() *FallbackConfig {
	return &FallbackConfig{
		Responses: make(map[string]*CachedResponse),
	}
}

// Set stores a fallback response for a route prefix.
func (f *FallbackConfig) Set(prefix string, resp *CachedResponse) {
	f.Responses[prefix] = resp
}

// Get returns the fallback response for a route, or nil if none configured.
func (f *FallbackConfig) Get(path string) *CachedResponse {
	for prefix, resp := range f.Responses {
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return resp
		}
	}
	return nil
}

// FallbackMiddleware serves cached fallback responses when the backend
// returns 502/503/504, instead of passing the error to the client.
func FallbackMiddleware(cfg *FallbackConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &fallbackRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			// If backend errored and we have a fallback, serve it
			if rec.status >= 502 {
				if fb := cfg.Get(r.URL.Path); fb != nil {
					for k, v := range fb.Headers {
						w.Header()[k] = v
					}
					w.Header().Set("X-Fallback", "true")
					w.Header().Set("Warning", "199 - serving cached fallback response")
					w.WriteHeader(fb.StatusCode)
					w.Write(fb.Body)
					return
				}
			}
		})
	}
}

// --- internal types ---

type slowRequestRecorder struct {
	http.ResponseWriter
	status int
}

func (r *slowRequestRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

type fallbackRecorder struct {
	http.ResponseWriter
	status int
}

func (r *fallbackRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
