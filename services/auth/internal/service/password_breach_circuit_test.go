package service

// Password Breach Check Circuit Breaker Tests
// Verifies: Gap #15 — HIBP API circuit breaker behavior
// Date: 2026-07-25

import (
	"testing"
)

// TestBreachCircuitBreaker_ClosedState verifies that with 0 failures the circuit is closed.
func TestBreachCircuitBreaker_ClosedState(t *testing.T) {
	resetBreachCircuitForTest()

	if breachCircuitIsOpen() {
		t.Error("circuit should be CLOSED with 0 failures")
	}
}

// TestBreachCircuitBreaker_OpenAfter3Failures verifies that 3 consecutive
// failures open the circuit.
func TestBreachCircuitBreaker_OpenAfter3Failures(t *testing.T) {
	resetBreachCircuitForTest()

	// 2 failures — still closed
	breachCircuitRecordFailure()
	breachCircuitRecordFailure()
	if breachCircuitIsOpen() {
		t.Error("circuit should be CLOSED after 2 failures (threshold is 3)")
	}

	// 3rd failure — circuit opens
	breachCircuitRecordFailure()
	if !breachCircuitIsOpen() {
		t.Error("circuit should be OPEN after 3 consecutive failures")
	}
}

// TestBreachCircuitBreaker_FailOpenWhenOpen verifies that CheckPasswordBreach
// returns nil (fail-open) immediately when the circuit is open, without
// making any HTTP call.
func TestBreachCircuitBreaker_FailOpenWhenOpen(t *testing.T) {
	resetBreachCircuitForTest()

	// Force open by recording 3 failures
	breachCircuitRecordFailure()
	breachCircuitRecordFailure()
	breachCircuitRecordFailure()

	if !breachCircuitIsOpen() {
		t.Fatal("circuit should be open")
	}

	// CheckPasswordBreach should return nil immediately (fail-open)
	// without attempting to reach HIBP (which would fail in test env anyway)
	ps := &PasswordService{}
	err := ps.CheckPasswordBreach(nil, "test-password-123")
	if err != nil {
		t.Errorf("fail-open: should return nil when circuit is open, got %v", err)
	}
}

// TestBreachCircuitBreaker_ResetOnSuccess verifies that a successful API call
// resets the failure counter back to 0.
func TestBreachCircuitBreaker_ResetOnSuccess(t *testing.T) {
	resetBreachCircuitForTest()

	// Accumulate 2 failures
	breachCircuitRecordFailure()
	breachCircuitRecordFailure()

	// Success resets the counter
	breachCircuitRecordSuccess()

	// Now 1 more failure should NOT open the circuit (counter was reset to 0)
	breachCircuitRecordFailure()
	if breachCircuitIsOpen() {
		t.Error("circuit should NOT open after reset + 1 failure")
	}

	// 2 more failures (total 3 from counter=0) should open it
	breachCircuitRecordFailure()
	breachCircuitRecordFailure()
	if !breachCircuitIsOpen() {
		t.Error("circuit should OPEN after 3 failures since last reset")
	}
}

// TestBreachCircuitBreaker_HalfOpenAfterCooldown verifies that after the
// cooldown period, the circuit transitions to half-open (allows a trial request).
func TestBreachCircuitBreaker_HalfOpenAfterCooldown(t *testing.T) {
	resetBreachCircuitForTest()

	// Force open
	breachCircuitRecordFailure()
	breachCircuitRecordFailure()
	breachCircuitRecordFailure()
	if !breachCircuitIsOpen() {
		t.Fatal("circuit should be open")
	}

	// Manually set openedAt to the past (simulating cooldown elapsed)
	breachOpenedAt.Store(0) // 0 means closed — simulate cooldown expiry

	// Circuit should now be half-open (returns false = allows request)
	if breachCircuitIsOpen() {
		t.Error("circuit should be half-open (allowing trial request) after cooldown")
	}
}

// TestBreachCircuitBreaker_SuccessClosesCircuit verifies that recording success
// fully closes an open circuit.
func TestBreachCircuitBreaker_SuccessClosesCircuit(t *testing.T) {
	resetBreachCircuitForTest()

	// Open the circuit
	breachCircuitRecordFailure()
	breachCircuitRecordFailure()
	breachCircuitRecordFailure()
	if !breachCircuitIsOpen() {
		t.Fatal("circuit should be open")
	}

	// Record success
	breachCircuitRecordSuccess()

	// Circuit should now be closed
	if breachCircuitIsOpen() {
		t.Error("circuit should be CLOSED after success recorded")
	}

	// And failure count should be 0 — need 3 more to re-open
	breachCircuitRecordFailure()
	breachCircuitRecordFailure()
	if breachCircuitIsOpen() {
		t.Error("circuit should still be closed with only 2 failures after reset")
	}
}

// TestBreachCircuitBreaker_MoreFailuresAfterOpen verifies that continued
// failures after opening keep the circuit open.
func TestBreachCircuitBreaker_MoreFailuresAfterOpen(t *testing.T) {
	resetBreachCircuitForTest()

	// Open the circuit
	for i := 0; i < 3; i++ {
		breachCircuitRecordFailure()
	}
	if !breachCircuitIsOpen() {
		t.Fatal("circuit should be open")
	}

	// More failures keep updating openedAt (cooldown restarts)
	openedBefore := breachOpenedAt.Load()
	breachCircuitRecordFailure()
	openedAfter := breachOpenedAt.Load()

	if openedAfter < openedBefore {
		t.Error("openedAt should be updated (cooldown restarted) on failure while open")
	}
}
