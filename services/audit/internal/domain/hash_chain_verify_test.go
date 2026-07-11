package domain

// Audit Hash Chain Verify Endpoint Tests
// Verifies: VerifyChain + VerifyHash functionality for the audit hash chain endpoint
// Date: 2026-07-25

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHashChain_VerifyEndpoint_ValidChain verifies that a properly chained
// sequence of events passes VerifyChain.
func TestHashChain_VerifyEndpoint_ValidChain(t *testing.T) {
	SetHashChainSecret([]byte("test-secret-for-chain"))
	tenantID := uuid.New()

	base := time.Now()
	events := make([]*AuditEvent, 5)
	prevHash := ""
	for i := range events {
		e := &AuditEvent{
			ID:           uuid.New(),
			TenantID:     tenantID,
			ActorType:    "user",
			Action:       "login",
			ResourceType: "auth",
			Result:       "success",
			IPAddress:    "10.0.0.1",
			CreatedAt:    base.Add(time.Duration(i) * time.Second),
		}
		e.Hash = e.ComputeHash(prevHash)
		events[i] = e
		prevHash = e.Hash
	}

	// Full chain should be valid
	brokenAt := VerifyChain(events)
	if brokenAt != -1 {
		t.Errorf("valid chain should return -1, got broken at index %d", brokenAt)
	}

	// Each event should individually verify
	prevHash = ""
	for i, e := range events {
		if !e.VerifyHash(prevHash) {
			t.Errorf("event %d failed individual VerifyHash", i)
		}
		prevHash = e.Hash
	}
}

// TestHashChain_VerifyEndpoint_TamperedEvent verifies that modifying an event
// breaks the chain and VerifyChain returns the tampered index.
func TestHashChain_VerifyEndpoint_TamperedEvent(t *testing.T) {
	SetHashChainSecret([]byte("tamper-detection-secret"))
	tenantID := uuid.New()

	base := time.Now()
	events := make([]*AuditEvent, 4)
	prevHash := ""
	for i := range events {
		e := &AuditEvent{
			ID:           uuid.New(),
			TenantID:     tenantID,
			ActorType:    "user",
			Action:       "create",
			ResourceType: "user",
			Result:       "success",
			CreatedAt:    base.Add(time.Duration(i) * time.Second),
		}
		e.Hash = e.ComputeHash(prevHash)
		events[i] = e
		prevHash = e.Hash
	}

	// Tamper with event #2's action (but keep hash unchanged)
	events[2].Action = "DELETE" // was "create"

	brokenAt := VerifyChain(events)
	if brokenAt != 2 {
		t.Errorf("tampered chain should break at index 2, got %d", brokenAt)
	}
}

// TestHashChain_VerifyEndpoint_EmptyChain verifies empty chain returns -1.
func TestHashChain_VerifyEndpoint_EmptyChain(t *testing.T) {
	brokenAt := VerifyChain([]*AuditEvent{})
	if brokenAt != -1 {
		t.Errorf("empty chain should return -1, got %d", brokenAt)
	}
}

// TestHashChain_VerifyEndpoint_SingleEvent verifies a single event chain works.
func TestHashChain_VerifyEndpoint_SingleEvent(t *testing.T) {
	SetHashChainSecret([]byte("single-event-secret"))
	e := &AuditEvent{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		ActorType:    "system",
		Action:       "boot",
		ResourceType: "system",
		Result:       "success",
		CreatedAt:    time.Now(),
	}
	e.Hash = e.ComputeHash("")

	brokenAt := VerifyChain([]*AuditEvent{e})
	if brokenAt != -1 {
		t.Errorf("single event chain should be valid (-1), got %d", brokenAt)
	}
}

// TestHashChain_VerifyEndpoint_MissingHash verifies event without hash fails.
func TestHashChain_VerifyEndpoint_MissingHash(t *testing.T) {
	SetHashChainSecret([]byte("missing-hash-secret"))

	e := &AuditEvent{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		ActorType:    "user",
		Action:       "login",
		ResourceType: "auth",
		Result:       "success",
		CreatedAt:    time.Now(),
		Hash:         "", // missing hash
	}

	brokenAt := VerifyChain([]*AuditEvent{e})
	if brokenAt != 0 {
		t.Errorf("event without hash should break at index 0, got %d", brokenAt)
	}
}

// TestHashChain_VerifyEndpoint_ChainReplay verifies that reordering events
// breaks the chain (each event's hash depends on the previous).
func TestHashChain_VerifyEndpoint_ChainReplay(t *testing.T) {
	SetHashChainSecret([]byte("replay-detection-secret"))
	tenantID := uuid.New()

	base := time.Now()
	events := make([]*AuditEvent, 3)
	prevHash := ""
	for i := range events {
		e := &AuditEvent{
			ID:           uuid.New(),
			TenantID:     tenantID,
			ActorType:    "user",
			Action:       "read",
			ResourceType: "document",
			Result:       "success",
			CreatedAt:    base.Add(time.Duration(i) * time.Second),
		}
		e.Hash = e.ComputeHash(prevHash)
		events[i] = e
		prevHash = e.Hash
	}

	// Swap events 1 and 2 — chain should break
	events[1], events[2] = events[2], events[1]
	brokenAt := VerifyChain(events)
	if brokenAt == -1 {
		t.Error("reordered chain should NOT be valid")
	}
}

// TestHashChain_VerifyEndpoint_DifferentSecret verifies that verification
// with a different secret fails.
func TestHashChain_VerifyEndpoint_DifferentSecret(t *testing.T) {
	SetHashChainSecret([]byte("secret-A"))
	e := &AuditEvent{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		ActorType:    "user",
		Action:       "login",
		ResourceType: "auth",
		Result:       "success",
		CreatedAt:    time.Now(),
	}
	e.Hash = e.ComputeHash("")

	// Change secret → verification should fail
	SetHashChainSecret([]byte("secret-B"))
	if e.VerifyHash("") {
		t.Error("VerifyHash should fail with different secret")
	}

	// Restore secret → verification should pass
	SetHashChainSecret([]byte("secret-A"))
	if !e.VerifyHash("") {
		t.Error("VerifyHash should pass with original secret")
	}
}
