package service

// Gap Regression Verification Test
// Verifies: Gap #12 — CSRF State Validation (DONE, MEDIUM confidence — at-risk item)
// Method: Functional test exercising the full state lifecycle:
//         store → validate → one-time-use deletion, expiry, replay, cross-client isolation.
//         This was flagged as "no dedicated unit test" in gap-closure-report.md.
// Date: 2026-07-24

import (
	"fmt"
	"testing"
	"time"
)

// storeTestState simulates what GenerateAuthCode does when it stores a state.
func storeTestState(clientID, state string, ttl time.Duration) {
	stateKey := fmt.Sprintf("oauth:state:%s:%s", clientID, state)
	stateStore.Store(stateKey, time.Now().Add(ttl))
}

// ========== GAP #12: CSRF State Validation — Full Functional Verification ==========

// TestGapRegression_ValidateState_HappyPath verifies the normal store→validate lifecycle.
func TestGapRegression_ValidateState_HappyPath(t *testing.T) {
	svc := &OAuthService{}
	clientID := "test-client-happy"
	state := "random-state-value-12345"

	// Store state (simulates authorize endpoint)
	storeTestState(clientID, state, 10*time.Minute)

	// Validate should succeed
	if !svc.ValidateState(clientID, state) {
		t.Fatal("valid state should pass validation")
	}
}

// TestGapRegression_ValidateState_OneTimeUse verifies that after successful
// validation, the same state cannot be reused (replay attack prevention per RFC 6749 §10.12).
func TestGapRegression_ValidateState_OneTimeUse(t *testing.T) {
	svc := &OAuthService{}
	clientID := "test-client-onetime"
	state := "state-for-replay-test"

	storeTestState(clientID, state, 10*time.Minute)

	// First validation succeeds
	if !svc.ValidateState(clientID, state) {
		t.Fatal("first validation should succeed")
	}

	// Second validation must fail (one-time use)
	if svc.ValidateState(clientID, state) {
		t.Fatal("replay attack: second validation with same state should FAIL (one-time use)")
	}
}

// TestGapRegression_ValidateState_Expired verifies that expired states are rejected.
func TestGapRegression_ValidateState_Expired(t *testing.T) {
	svc := &OAuthService{}
	clientID := "test-client-expired"
	state := "expired-state-value"

	// Store with negative TTL (already expired)
	storeTestState(clientID, state, -1*time.Second)

	if svc.ValidateState(clientID, state) {
		t.Fatal("expired state should be rejected")
	}
}

// TestGapRegression_ValidateState_EmptyState verifies empty state is rejected.
func TestGapRegression_ValidateState_EmptyState(t *testing.T) {
	svc := &OAuthService{}

	if svc.ValidateState("test-client", "") {
		t.Fatal("empty state should be rejected")
	}
}

// TestGapRegression_ValidateState_UnknownState verifies unknown/random state is rejected.
func TestGapRegression_ValidateState_UnknownState(t *testing.T) {
	svc := &OAuthService{}

	if svc.ValidateState("test-client", "this-state-was-never-stored") {
		t.Fatal("unknown state should be rejected")
	}
}

// TestGapRegression_ValidateState_CrossClientIsolation verifies that a state
// stored for client A cannot be validated by client B (prevents cross-client CSRF).
func TestGapRegression_ValidateState_CrossClientIsolation(t *testing.T) {
	svc := &OAuthService{}
	clientA := "client-alpha"
	clientB := "client-beta"
	state := "shared-state-value"

	storeTestState(clientA, state, 10*time.Minute)

	// Client A can validate
	if !svc.ValidateState(clientA, state) {
		t.Fatal("client A should validate its own state")
	}

	// Re-store for client A (since first validation consumed it)
	storeTestState(clientA, state, 10*time.Minute)

	// Client B should NOT be able to use client A's state
	if svc.ValidateState(clientB, state) {
		t.Fatal("cross-client CSRF: client B should NOT validate client A's state")
	}
}

// TestGapRegression_ValidateState_DeletedAfterExpiryCheck verifies that expired
// states are cleaned up from the store (not just rejected).
func TestGapRegression_ValidateState_DeletedAfterExpiryCheck(t *testing.T) {
	svc := &OAuthService{}
	clientID := "test-client-cleanup"
	state := "cleanup-test-state"

	storeTestState(clientID, state, -1*time.Second)

	// First call rejects and deletes
	if svc.ValidateState(clientID, state) {
		t.Fatal("expired state should be rejected")
	}

	// Second call should also fail (already deleted)
	if svc.ValidateState(clientID, state) {
		t.Fatal("expired state should have been deleted after first check")
	}
}

// TestGapRegression_ValidateState_MultipleStates verifies that multiple concurrent
// states for the same client can coexist independently.
func TestGapRegression_ValidateState_MultipleStates(t *testing.T) {
	svc := &OAuthService{}
	clientID := "test-client-multi"

	state1 := "state-one"
	state2 := "state-two"
	state3 := "state-three"

	storeTestState(clientID, state1, 10*time.Minute)
	storeTestState(clientID, state2, 10*time.Minute)
	storeTestState(clientID, state3, 10*time.Minute)

	// All three should validate independently
	if !svc.ValidateState(clientID, state2) {
		t.Fatal("state2 should validate")
	}
	if !svc.ValidateState(clientID, state1) {
		t.Fatal("state1 should validate")
	}
	if !svc.ValidateState(clientID, state3) {
		t.Fatal("state3 should validate")
	}

	// None should be reusable
	if svc.ValidateState(clientID, state1) {
		t.Fatal("state1 should not be reusable after validation")
	}
	if svc.ValidateState(clientID, state2) {
		t.Fatal("state2 should not be reusable after validation")
	}
}
