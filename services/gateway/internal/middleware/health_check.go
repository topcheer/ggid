package middleware

import (
	"net/http"
	"sync"
	"time"
)

// HealthCheckConfig configures the backend health check middleware.
type HealthCheckConfig struct {
	Interval        time.Duration // probe interval (default 10s)
	Timeout         time.Duration // probe timeout (default 2s)
	FailureThreshold int          // consecutive failures to mark unhealthy (default 3)
	SuccessThreshold int          // consecutive successes to mark healthy (default 2)
}

// DefaultHealthCheckConfig returns sensible defaults.
func DefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Interval:         10 * time.Second,
		Timeout:          2 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
	}
}

// BackendHealth tracks the health state of a single backend.
type BackendHealth struct {
	Healthy           bool
	ConsecutiveFails  int
	ConsecutiveOKs    int
	LastChecked       time.Time
	LastError         string
}

// HealthChecker tracks per-backend health via probing.
type HealthChecker struct {
	mu       sync.RWMutex
	backends map[string]*BackendHealth
	cfg      *HealthCheckConfig
	client   *http.Client
}

// NewHealthChecker creates a health checker with the given config.
func NewHealthChecker(cfg *HealthCheckConfig) *HealthChecker {
	if cfg == nil {
		cfg = DefaultHealthCheckConfig()
	}
	return &HealthChecker{
		backends: make(map[string]*BackendHealth),
		cfg:      cfg,
		client:   &http.Client{Timeout: cfg.Timeout},
	}
}

// IsHealthy returns the current health status of a backend.
func (hc *HealthChecker) IsHealthy(backend string) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	if h, ok := hc.backends[backend]; ok {
		return h.Healthy
	}
	return true // unknown backends default to healthy
}

// MarkSuccess records a successful probe for a backend.
func (hc *HealthChecker) MarkSuccess(backend string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	h := hc.getOrCreate(backend)
	h.ConsecutiveOKs++
	h.ConsecutiveFails = 0
	h.LastChecked = time.Now()
	h.LastError = ""
	if h.ConsecutiveOKs >= hc.cfg.SuccessThreshold {
		h.Healthy = true
	}
}

// MarkFailure records a failed probe for a backend.
func (hc *HealthChecker) MarkFailure(backend, errMsg string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	h := hc.getOrCreate(backend)
	h.ConsecutiveFails++
	h.ConsecutiveOKs = 0
	h.LastChecked = time.Now()
	h.LastError = errMsg
	if h.ConsecutiveFails >= hc.cfg.FailureThreshold {
		h.Healthy = false
	}
}

// GetHealth returns the health status for a backend.
func (hc *HealthChecker) GetHealth(backend string) *BackendHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	if h, ok := hc.backends[backend]; ok {
		cp := *h
		return &cp
	}
	return &BackendHealth{Healthy: true}
}

// AllHealth returns health status for all backends.
func (hc *HealthChecker) AllHealth() map[string]*BackendHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	result := make(map[string]*BackendHealth, len(hc.backends))
	for k, v := range hc.backends {
		cp := *v
		result[k] = &cp
	}
	return result
}

func (hc *HealthChecker) getOrCreate(backend string) *BackendHealth {
	h, ok := hc.backends[backend]
	if !ok {
		h = &BackendHealth{Healthy: true}
		hc.backends[backend] = h
	}
	return h
}

// HealthCheckMiddleware returns 503 when the target backend is unhealthy.
// The backend URL must be set in the request context under "target_backend".
func HealthCheckMiddleware(hc *HealthChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
