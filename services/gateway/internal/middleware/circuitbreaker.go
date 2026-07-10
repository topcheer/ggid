// Package middleware implements HTTP middleware for the API Gateway.
package middleware

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // normal operation, requests flow
	CircuitOpen                          // tripped, requests fail-fast
	CircuitHalfOpen                      // testing if backend recovered
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitConfig holds circuit breaker configuration.
type CircuitConfig struct {
	MaxFailures    int           // failures before opening (default: 5)
	Timeout        time.Duration // open→half-open cooldown (default: 30s)
	HalfOpenMax    int           // max trial requests in half-open (default: 3)
	HalfOpenSuccess int          // successes to close circuit (default: 2)
}

// DefaultCircuitConfig returns sensible defaults.
func DefaultCircuitConfig() CircuitConfig {
	return CircuitConfig{
		MaxFailures:     5,
		Timeout:         30 * time.Second,
		HalfOpenMax:     3,
		HalfOpenSuccess: 2,
	}
}

// CircuitBreaker implements the circuit breaker pattern for backend services.
// When a backend consistently fails, the circuit opens and requests fail-fast
// instead of waiting for timeouts. After a cooldown, it enters half-open state
// to test if the backend has recovered.
type CircuitBreaker struct {
	mu          sync.Mutex
	config      CircuitConfig
	state       CircuitState
	failures    int
	successes   int
	halfOpenReq int
	lastFailure time.Time
}

// NewCircuitBreaker creates a circuit breaker with the given config.
func NewCircuitBreaker(cfg CircuitConfig) *CircuitBreaker {
	if cfg.MaxFailures <= 0 {
		cfg.MaxFailures = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.HalfOpenMax <= 0 {
		cfg.HalfOpenMax = 3
	}
	if cfg.HalfOpenSuccess <= 0 {
		cfg.HalfOpenSuccess = 2
	}
	return &CircuitBreaker{
		config: cfg,
		state:  CircuitClosed,
	}
}

// Allow checks if a request should be allowed through.
// Returns true if the request should proceed, false if the circuit is open.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if cooldown has elapsed
		if time.Since(cb.lastFailure) >= cb.config.Timeout {
			cb.state = CircuitHalfOpen
			cb.halfOpenReq = 0
			cb.successes = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		if cb.halfOpenReq < cb.config.HalfOpenMax {
			cb.halfOpenReq++
			return true
		}
		return false
	default:
		return true
	}
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitHalfOpen {
		cb.successes++
		if cb.successes >= cb.config.HalfOpenSuccess {
			cb.state = CircuitClosed
			cb.failures = 0
			cb.successes = 0
		}
	} else if cb.state == CircuitClosed {
		// Reset failure count on success (sliding window)
		cb.failures = 0
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailure = time.Now()

	if cb.state == CircuitHalfOpen {
		// Failure during half-open → re-open
		cb.state = CircuitOpen
		cb.failures = 0
	} else if cb.state == CircuitClosed {
		cb.failures++
		if cb.failures >= cb.config.MaxFailures {
			cb.state = CircuitOpen
		}
	}
}

// State returns the current circuit state (thread-safe).
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Stats holds a snapshot of circuit breaker statistics.
type CircuitStats struct {
	State       CircuitState `json:"state"`
	Failures    int          `json:"failures"`
	Successes   int          `json:"successes"`
	LastFailure time.Time    `json:"last_failure,omitempty"`
}

// Stats returns a snapshot of circuit breaker stats (thread-safe).
func (cb *CircuitBreaker) Stats() CircuitStats {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return CircuitStats{
		State:       cb.state,
		Failures:    cb.failures,
		Successes:   cb.successes,
		LastFailure: cb.lastFailure,
	}
}

// --- Registry for per-backend circuit breakers ---

// CircuitRegistry manages circuit breakers per backend prefix.
type CircuitRegistry struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	config   CircuitConfig
}

// NewCircuitRegistry creates a registry with the given default config.
func NewCircuitRegistry(cfg CircuitConfig) *CircuitRegistry {
	return &CircuitRegistry{
		breakers: make(map[string]*CircuitBreaker),
		config:   cfg,
	}
}

// Get returns the circuit breaker for the given prefix, creating one if needed.
func (r *CircuitRegistry) Get(prefix string) *CircuitBreaker {
	r.mu.RLock()
	cb, ok := r.breakers[prefix]
	r.mu.RUnlock()
	if ok {
		return cb
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	// Double-check after acquiring write lock
	if cb, ok := r.breakers[prefix]; ok {
		return cb
	}
	cb = NewCircuitBreaker(r.config)
	r.breakers[prefix] = cb
	return cb
}

// AllStats returns stats for all circuit breakers.
func (r *CircuitRegistry) AllStats() map[string]CircuitStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]CircuitStats, len(r.breakers))
	for prefix, cb := range r.breakers {
		result[prefix] = cb.Stats()
	}
	return result
}

// --- Atomic counter for stats ---

// requestCounter tracks total allowed/blocked requests per backend.
type requestCounter struct {
	allowed  atomic.Uint64
	blocked  atomic.Uint64
}

// CircuitMiddleware wraps an http.Handler with circuit breaker protection.
// If the circuit is open, it returns 503 Service Unavailable immediately.
// Backend errors (5xx) and network errors trip the circuit.
func CircuitMiddleware(prefix string, registry *CircuitRegistry, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cb := registry.Get(prefix)
		if !cb.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Circuit-State", "open")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"circuit breaker open","backend":"` + prefix + `"}`))
			return
		}

		// Wrap the ResponseWriter to detect 5xx errors
		sw := &circuitResponseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)

		if sw.status >= 500 {
			cb.RecordFailure()
		} else {
			cb.RecordSuccess()
		}
	})
}

type circuitResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *circuitResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
