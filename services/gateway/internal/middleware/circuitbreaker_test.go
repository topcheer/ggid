package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestCircuitBreaker_StartsClosed(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitConfig())
	if cb.State() != CircuitClosed {
		t.Errorf("expected closed, got %s", cb.State())
	}
	if !cb.Allow() {
		t.Error("closed circuit should allow requests")
	}
}

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	cfg := CircuitConfig{MaxFailures: 3, Timeout: 100 * time.Millisecond}
	cb := NewCircuitBreaker(cfg)

	// 3 failures should open the circuit
	for i := 0; i < 3; i++ {
		if !cb.Allow() {
			t.Fatal("should allow while closed")
		}
		cb.RecordFailure()
	}
	if cb.State() != CircuitOpen {
		t.Errorf("expected open after %d failures, got %s", 3, cb.State())
	}
	if cb.Allow() {
		t.Error("open circuit should NOT allow requests")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cfg := CircuitConfig{
		MaxFailures:     2,
		Timeout:         50 * time.Millisecond,
		HalfOpenMax:     3,
		HalfOpenSuccess: 2,
	}
	cb := NewCircuitBreaker(cfg)

	// Trip the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open and allow
	if !cb.Allow() {
		t.Error("should allow after timeout (half-open)")
	}
	if cb.State() != CircuitHalfOpen {
		t.Errorf("expected half-open, got %s", cb.State())
	}
}

func TestCircuitBreaker_ClosesAfterHalfOpenSuccess(t *testing.T) {
	cfg := CircuitConfig{
		MaxFailures:     2,
		Timeout:         50 * time.Millisecond,
		HalfOpenMax:     5,
		HalfOpenSuccess: 2,
	}
	cb := NewCircuitBreaker(cfg)

	// Trip the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)

	// Enter half-open
	cb.Allow()
	cb.Allow()

	// 2 successes should close the circuit
	cb.RecordSuccess()
	cb.RecordSuccess()

	if cb.State() != CircuitClosed {
		t.Errorf("expected closed after half-open success, got %s", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	cfg := CircuitConfig{
		MaxFailures:     2,
		Timeout:         50 * time.Millisecond,
		HalfOpenMax:     5,
		HalfOpenSuccess: 2,
	}
	cb := NewCircuitBreaker(cfg)

	// Trip the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)

	// Enter half-open
	cb.Allow()

	// Failure during half-open should re-open
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Errorf("expected open after half-open failure, got %s", cb.State())
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cfg := CircuitConfig{MaxFailures: 3}
	cb := NewCircuitBreaker(cfg)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess() // resets failure count

	if cb.State() != CircuitClosed {
		t.Error("should still be closed")
	}
	if cb.failures != 0 {
		t.Errorf("expected 0 failures after success, got %d", cb.failures)
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitConfig())
	cb.RecordFailure()
	cb.RecordFailure()

	stats := cb.Stats()
	if stats.State != CircuitClosed {
		t.Errorf("expected closed, got %s", stats.State)
	}
	if stats.Failures != 2 {
		t.Errorf("expected 2 failures, got %d", stats.Failures)
	}
}

func TestCircuitRegistry_Get(t *testing.T) {
	reg := NewCircuitRegistry(DefaultCircuitConfig())

	cb1 := reg.Get("/api/v1/auth")
	cb2 := reg.Get("/api/v1/auth")
	if cb1 != cb2 {
		t.Error("Get should return the same breaker for the same prefix")
	}

	cb3 := reg.Get("/api/v1/users")
	if cb3 == cb1 {
		t.Error("Get should return different breakers for different prefixes")
	}
}

func TestCircuitRegistry_AllStats(t *testing.T) {
	reg := NewCircuitRegistry(DefaultCircuitConfig())
	reg.Get("/api/v1/auth").RecordFailure()
	reg.Get("/api/v1/users").RecordFailure()
	reg.Get("/api/v1/users").RecordFailure()

	stats := reg.AllStats()
	if len(stats) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(stats))
	}
	if stats["/api/v1/auth"].Failures != 1 {
		t.Errorf("expected 1 failure for auth, got %d", stats["/api/v1/auth"].Failures)
	}
	if stats["/api/v1/users"].Failures != 2 {
		t.Errorf("expected 2 failures for users, got %d", stats["/api/v1/users"].Failures)
	}
}

func TestCircuitMiddleware_AllowsWhenClosed(t *testing.T) {
	reg := NewCircuitRegistry(DefaultCircuitConfig())
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	CircuitMiddleware("/api/v1/test", reg, next).ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called when circuit is closed")
	}
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCircuitMiddleware_BlocksWhenOpen(t *testing.T) {
	cfg := CircuitConfig{MaxFailures: 2, Timeout: 10 * time.Minute}
	reg := NewCircuitRegistry(cfg)

	// Trip the circuit
	cb := reg.Get("/api/v1/test")
	cb.RecordFailure()
	cb.RecordFailure()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	CircuitMiddleware("/api/v1/test", reg, next).ServeHTTP(w, req)

	if called {
		t.Error("handler should NOT be called when circuit is open")
	}
	if w.Code != 503 {
		t.Errorf("expected 503, got %d", w.Code)
	}
	if w.Header().Get("X-Circuit-State") != "open" {
		t.Error("expected X-Circuit-State: open header")
	}
}

func TestCircuitMiddleware_TripsOn5xx(t *testing.T) {
	cfg := CircuitConfig{MaxFailures: 2, Timeout: 10 * time.Minute}
	reg := NewCircuitRegistry(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})

	// Send 2 failing requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		CircuitMiddleware("/api/v1/test", reg, next).ServeHTTP(w, req)
	}

	cb := reg.Get("/api/v1/test")
	if cb.State() != CircuitOpen {
		t.Errorf("expected open after 2 5xx errors, got %s", cb.State())
	}
}

func TestCircuitMiddleware_DoesNotTripOn4xx(t *testing.T) {
	reg := NewCircuitRegistry(CircuitConfig{MaxFailures: 2})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		CircuitMiddleware("/api/v1/test", reg, next).ServeHTTP(w, req)
	}

	cb := reg.Get("/api/v1/test")
	if cb.State() != CircuitClosed {
		t.Errorf("expected closed after 4xx errors, got %s", cb.State())
	}
}

func TestCircuitMiddleware_SuccessRecovers(t *testing.T) {
	cfg := CircuitConfig{
		MaxFailures:     2,
		Timeout:         50 * time.Millisecond,
		HalfOpenMax:     5,
		HalfOpenSuccess: 2,
	}
	reg := NewCircuitRegistry(cfg)

	// Trip with 2 failures
	failing := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		CircuitMiddleware("/api/v1/test", reg, failing).ServeHTTP(w, req)
	}

	cb := reg.Get("/api/v1/test")
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)

	// Send successful requests in half-open
	success := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		CircuitMiddleware("/api/v1/test", reg, success).ServeHTTP(w, req)
	}

	cb = reg.Get("/api/v1/test")
	if cb.State() != CircuitClosed {
		t.Errorf("expected closed after recovery, got %s", cb.State())
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state CircuitState
		want  string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("%d.String() = %s, want %s", tt.state, got, tt.want)
		}
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{MaxFailures: 100, Timeout: time.Minute})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.Allow()
			cb.RecordSuccess()
			cb.State()
			cb.Stats()
		}()
	}
	wg.Wait()
	// Should not panic or race
}

func TestCircuitMiddleware_ResponseBody(t *testing.T) {
	cfg := CircuitConfig{MaxFailures: 1, Timeout: 10 * time.Minute}
	reg := NewCircuitRegistry(cfg)

	// Trip immediately
	reg.Get("/api/v1/test").RecordFailure()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	CircuitMiddleware("/api/v1/test", reg, next).ServeHTTP(w, req)

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON body, got: %s", w.Body.String())
	}
	if body["error"] != "circuit breaker open" {
		t.Errorf("unexpected error message: %s", body["error"])
	}
}
