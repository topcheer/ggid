package middleware

import (
	"testing"
	"time"
)

// TestCovS27_CircuitBreaker_FullLifecycle tests the complete state machine:
// closed → open → half-open → closed (recovery).
func TestCovS27_CircuitBreaker_FullLifecycle(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{
		MaxFailures:     3,
		Timeout:         50 * time.Millisecond,
		HalfOpenSuccess: 2,
	})

	// Phase 1: Closed — requests flow normally
	if cb.State() != CircuitClosed {
		t.Fatalf("expected initial state closed, got %s", cb.State())
	}
	if !cb.Allow() {
		t.Fatal("expected Allow()=true when closed")
	}

	// Phase 2: Open — after MaxFailures consecutive failures
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open after 3 failures, got %s", cb.State())
	}
	if cb.Allow() {
		t.Fatal("expected Allow()=false when open")
	}

	// Verify Stats reflect open state
	stats := cb.Stats()
	if stats.State != CircuitOpen {
		t.Fatalf("expected Stats state open, got %s", stats.State)
	}

	// Phase 3: Half-open — after timeout expires, next Allow() transitions
	time.Sleep(60 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("expected Allow()=true in half-open after timeout")
	}

	// Phase 4: Recovery — record enough successes to close
	cb.RecordSuccess()
	if cb.State() == CircuitClosed {
		t.Fatal("should still be half-open after 1 success (need 2)")
	}
	cb.RecordSuccess()
	if cb.State() != CircuitClosed {
		t.Fatalf("expected closed after 2 successes, got %s", cb.State())
	}
}

// TestCovS27_CircuitBreaker_HalfOpenReopensOnFailure verifies that a failure
// during half-open state immediately re-opens the circuit.
func TestCovS27_CircuitBreaker_HalfOpenReopensOnFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{
		MaxFailures:     2,
		Timeout:         30 * time.Millisecond,
		HalfOpenSuccess: 2,
	})

	// Trip the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	// Wait for timeout → next Allow transitions to half-open
	time.Sleep(40 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("expected Allow() in half-open")
	}

	// Failure during half-open re-opens immediately
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("expected re-opened after half-open failure, got %s", cb.State())
	}
}

// TestCovS27_CircuitBreaker_SuccessResetsCount verifies that successes
// in closed state reset the failure counter.
func TestCovS27_CircuitBreaker_SuccessResetsCount(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{
		MaxFailures:     3,
		Timeout:         30 * time.Second,
		HalfOpenSuccess: 2,
	})

	// Two failures (below threshold)
	cb.RecordFailure()
	cb.RecordFailure()

	// Success should reset the failure count
	cb.RecordSuccess()

	// Two more failures should NOT trip (counter was reset)
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitClosed {
		t.Fatalf("expected still closed after reset+2 failures, got %s", cb.State())
	}
}

// TestCovS27_CircuitBreaker_RegistryGetMultiple verifies the registry
// returns the same instance for the same key and different for different keys.
func TestCovS27_CircuitBreaker_RegistryGetMultiple(t *testing.T) {
	registry := NewCircuitRegistry(CircuitConfig{
		MaxFailures:     5,
		Timeout:         30 * time.Second,
		HalfOpenSuccess: 2,
	})

	cb1 := registry.Get("backend-A")
	cb2 := registry.Get("backend-A")
	cb3 := registry.Get("backend-B")

	if cb1 != cb2 {
		t.Fatal("expected same instance for same key")
	}
	if cb1 == cb3 {
		t.Fatal("expected different instance for different key")
	}
}
