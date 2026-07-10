package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestDefaultCircuitConfig(t *testing.T) {
	cfg := DefaultCircuitConfig()
	if cfg.MaxFailures != 5 {
		t.Errorf("MaxFailures = %d, want 5", cfg.MaxFailures)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
	if cfg.HalfOpenMax != 3 {
		t.Errorf("HalfOpenMax = %d, want 3", cfg.HalfOpenMax)
	}
	if cfg.HalfOpenSuccess != 2 {
		t.Errorf("HalfOpenSuccess = %d, want 2", cfg.HalfOpenSuccess)
	}
}

func TestNewCircuitBreaker_DefaultsOnZero(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{})
	if cb.config.MaxFailures != 5 {
		t.Errorf("MaxFailures = %d, want 5 (default)", cb.config.MaxFailures)
	}
	if cb.config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s (default)", cb.config.Timeout)
	}
	if cb.config.HalfOpenMax != 3 {
		t.Errorf("HalfOpenMax = %d, want 3 (default)", cb.config.HalfOpenMax)
	}
	if cb.config.HalfOpenSuccess != 2 {
		t.Errorf("HalfOpenSuccess = %d, want 2 (default)", cb.config.HalfOpenSuccess)
	}
}

func TestNewCircuitBreaker_NegativeConfig(t *testing.T) {
	cfg := CircuitConfig{
		MaxFailures:     -1,
		Timeout:         -1,
		HalfOpenMax:     -1,
		HalfOpenSuccess: -1,
	}
	cb := NewCircuitBreaker(cfg)
	// Should fall back to defaults
	if cb.config.MaxFailures <= 0 {
		t.Error("negative MaxFailures should fall back to positive default")
	}
	if cb.config.Timeout <= 0 {
		t.Error("negative Timeout should fall back to positive default")
	}
}

func TestCircuitBreaker_HalfOpenMaxReached(t *testing.T) {
	cfg := CircuitConfig{
		MaxFailures:     1,
		Timeout:         50 * time.Millisecond,
		HalfOpenMax:     2,
		HalfOpenSuccess: 1,
	}
	cb := NewCircuitBreaker(cfg)

	// Trip the circuit
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)

	// Enter half-open, use up both trial requests
	if !cb.Allow() {
		t.Error("first half-open request should be allowed")
	}
	if cb.State() != CircuitHalfOpen {
		t.Errorf("expected half-open, got %s", cb.State())
	}
	if !cb.Allow() {
		t.Error("second half-open request should be allowed")
	}

	// Third request should be rejected (HalfOpenMax=2)
	if cb.Allow() {
		t.Error("third half-open request should be rejected (max reached)")
	}
}

func TestCircuitBreaker_RecordSuccessInOpenState(t *testing.T) {
	cfg := CircuitConfig{MaxFailures: 1, Timeout: 10 * time.Minute}
	cb := NewCircuitBreaker(cfg)

	// Trip the circuit
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open")
	}

	// RecordSuccess in open state should not change state
	cb.RecordSuccess()
	if cb.State() != CircuitOpen {
		t.Errorf("success in open state should not change state, got %s", cb.State())
	}
}

func TestCircuitBreaker_RecordFailureInOpenState(t *testing.T) {
	cfg := CircuitConfig{MaxFailures: 1, Timeout: 10 * time.Minute}
	cb := NewCircuitBreaker(cfg)

	// Trip the circuit
	cb.RecordFailure()
	oldFailure := cb.lastFailure

	// RecordFailure in open state should update lastFailure
	time.Sleep(time.Millisecond)
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Errorf("failure in open state should stay open, got %s", cb.State())
	}
	// lastFailure should have been updated
	if !cb.lastFailure.After(oldFailure) {
		t.Error("lastFailure should be updated after failure in open state")
	}
}

func TestCircuitBreaker_FullCycle(t *testing.T) {
	cfg := CircuitConfig{
		MaxFailures:     2,
		Timeout:         30 * time.Millisecond,
		HalfOpenMax:     3,
		HalfOpenSuccess: 2,
	}
	cb := NewCircuitBreaker(cfg)

	// Cycle 1: Closed → Open → HalfOpen → Closed
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("cycle 1: expected open after 2 failures, got %s", cb.State())
	}

	time.Sleep(40 * time.Millisecond)
	cb.Allow() // enter half-open
	cb.Allow()
	cb.RecordSuccess()
	cb.RecordSuccess()
	if cb.State() != CircuitClosed {
		t.Fatalf("cycle 1: expected closed after recovery, got %s", cb.State())
	}

	// Cycle 2: Closed → Open → HalfOpen → Open (failure during recovery)
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("cycle 2: expected open, got %s", cb.State())
	}

	time.Sleep(40 * time.Millisecond)
	cb.Allow() // enter half-open
	cb.RecordFailure() // re-open
	if cb.State() != CircuitOpen {
		t.Fatalf("cycle 2: expected open after half-open failure, got %s", cb.State())
	}

	// Cycle 3: Open → HalfOpen → Closed
	time.Sleep(40 * time.Millisecond)
	cb.Allow()
	cb.Allow()
	cb.RecordSuccess()
	cb.RecordSuccess()
	if cb.State() != CircuitClosed {
		t.Errorf("cycle 3: expected closed, got %s", cb.State())
	}
}

func TestCircuitBreaker_StatsAfterTransitions(t *testing.T) {
	cfg := CircuitConfig{MaxFailures: 2, Timeout: 50 * time.Millisecond}
	cb := NewCircuitBreaker(cfg)

	cb.RecordFailure()
	stats := cb.Stats()
	if stats.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", stats.Failures)
	}
	if stats.State != CircuitClosed {
		t.Errorf("expected closed, got %s", stats.State)
	}
	if stats.LastFailure.IsZero() {
		t.Error("expected non-zero last failure time")
	}

	cb.RecordFailure()
	stats = cb.Stats()
	if stats.State != CircuitOpen {
		t.Errorf("expected open after 2 failures, got %s", stats.State)
	}
}

func TestCircuitBreaker_RecordSuccessResetsInClosed(t *testing.T) {
	cfg := CircuitConfig{MaxFailures: 5}
	cb := NewCircuitBreaker(cfg)

	// Accumulate failures
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// One success resets failures
	cb.RecordSuccess()
	stats := cb.Stats()
	if stats.Failures != 0 {
		t.Errorf("expected 0 failures after success, got %d", stats.Failures)
	}
}

func TestCircuitRegistry_EmptyAllStats(t *testing.T) {
	reg := NewCircuitRegistry(DefaultCircuitConfig())
	stats := reg.AllStats()
	if len(stats) != 0 {
		t.Errorf("expected empty stats, got %d entries", len(stats))
	}
}

func TestCircuitRegistry_MultipleBackends(t *testing.T) {
	reg := NewCircuitRegistry(DefaultCircuitConfig())

	for _, prefix := range []string{"/api/v1/auth", "/api/v1/users", "/api/v1/orgs"} {
		cb := reg.Get(prefix)
		if cb == nil {
			t.Errorf("Get(%s) returned nil", prefix)
		}
	}

	stats := reg.AllStats()
	if len(stats) != 3 {
		t.Errorf("expected 3 entries, got %d", len(stats))
	}
}

func TestCircuitRegistry_ConcurrentGet(t *testing.T) {
	reg := NewCircuitRegistry(DefaultCircuitConfig())
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb := reg.Get("/api/v1/concurrent")
			cb.Allow()
			cb.RecordSuccess()
		}()
	}
	wg.Wait()
}

func TestCircuitMiddleware_SuccessDoesNotChangeState(t *testing.T) {
	reg := NewCircuitRegistry(CircuitConfig{MaxFailures: 2})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		CircuitMiddleware("/api/v1/test", reg, next).ServeHTTP(w, req)
	}

	cb := reg.Get("/api/v1/test")
	if cb.State() != CircuitClosed {
		t.Errorf("expected closed after 10 successes, got %s", cb.State())
	}
}

func TestCircuitMiddleware_403DoesNotTrip(t *testing.T) {
	reg := NewCircuitRegistry(CircuitConfig{MaxFailures: 2})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		CircuitMiddleware("/api/v1/test", reg, next).ServeHTTP(w, req)
	}

	cb := reg.Get("/api/v1/test")
	if cb.State() != CircuitClosed {
		t.Errorf("403s should not trip circuit, got %s", cb.State())
	}
}

func TestCircuitMiddleware_UnknownStateFallback(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitConfig())
	// Force into an unexpected state
	cb.mu.Lock()
	cb.state = CircuitState(99) // invalid state
	cb.mu.Unlock()

	// Allow should default to true for unknown states
	if !cb.Allow() {
		t.Error("Allow() should default to true for unknown state")
	}
}


