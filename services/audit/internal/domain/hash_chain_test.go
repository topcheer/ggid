package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func setupHashChainForTest() {
	SetHashChainSecret([]byte("test-secret-key-for-hash-chain"))
}

func makeTestEvent(action string, createdAt time.Time) *AuditEvent {
	tenantID := uuid.New()
	actorID := uuid.New()
	resourceID := uuid.New()
	return &AuditEvent{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ActorType:    ActorUser,
		ActorID:      &actorID,
		Action:       action,
		ResourceType: "document",
		ResourceID:   &resourceID,
		Result:       "success",
		IPAddress:    "192.168.1.100",
		CreatedAt:    createdAt,
	}
}

// TestHashChain_ValidChain verifies that a correctly built chain passes verification.
func TestHashChain_ValidChain(t *testing.T) {
	setupHashChainForTest()

	now := time.Now()
	events := make([]*AuditEvent, 5)
	prevHash := ""
	for i := range events {
		events[i] = makeTestEvent("create", now.Add(time.Duration(i)*time.Second))
		events[i].PrevHash = prevHash
		events[i].Hash = events[i].ComputeHash(prevHash)
		prevHash = events[i].Hash
	}

	brokenIdx := VerifyChain(events)
	if brokenIdx != -1 {
		t.Fatalf("valid chain should return -1, got broken at index %d", brokenIdx)
	}
}

// TestHashChain_TamperedEvent verifies that modifying an event's action after hashing is detected.
func TestHashChain_TamperedEvent(t *testing.T) {
	setupHashChainForTest()

	now := time.Now()
	events := make([]*AuditEvent, 3)
	prevHash := ""
	for i := range events {
		events[i] = makeTestEvent("create", now.Add(time.Duration(i)*time.Second))
		events[i].PrevHash = prevHash
		events[i].Hash = events[i].ComputeHash(prevHash)
		prevHash = events[i].Hash
	}

	// Tamper with event 1's action (don't recompute hash)
	events[1].Action = "delete"

	brokenIdx := VerifyChain(events)
	if brokenIdx != 1 {
		t.Fatalf("tampered chain should detect break at index 1, got %d", brokenIdx)
	}
}

// TestHashChain_TamperedHash verifies that swapping an event's hash is detected.
func TestHashChain_TamperedHash(t *testing.T) {
	setupHashChainForTest()

	now := time.Now()
	events := make([]*AuditEvent, 3)
	prevHash := ""
	for i := range events {
		events[i] = makeTestEvent("read", now.Add(time.Duration(i)*time.Second))
		events[i].PrevHash = prevHash
		events[i].Hash = events[i].ComputeHash(prevHash)
		prevHash = events[i].Hash
	}

	// Swap hashes between events 0 and 1
	events[0].Hash, events[1].Hash = events[1].Hash, events[0].Hash

	brokenIdx := VerifyChain(events)
	if brokenIdx == -1 {
		t.Fatal("swapped hashes should be detected")
	}
}

// TestHashChain_EmptyHashOnEvent verifies that an event with empty hash fails verification.
func TestHashChain_EmptyHashOnEvent(t *testing.T) {
	setupHashChainForTest()

	now := time.Now()
	events := make([]*AuditEvent, 3)
	prevHash := ""
	for i := range events {
		events[i] = makeTestEvent("update", now.Add(time.Duration(i)*time.Second))
		events[i].PrevHash = prevHash
		events[i].Hash = events[i].ComputeHash(prevHash)
		prevHash = events[i].Hash
	}

	// Remove hash from event 2
	events[2].Hash = ""

	brokenIdx := VerifyChain(events)
	if brokenIdx != 2 {
		t.Fatalf("empty hash should break at index 2, got %d", brokenIdx)
	}
}

// TestHashChain_EmptyChain verifies empty chain returns -1 (valid).
func TestHashChain_EmptyChain(t *testing.T) {
	setupHashChainForTest()
	brokenIdx := VerifyChain(nil)
	if brokenIdx != -1 {
		t.Fatalf("empty chain should return -1, got %d", brokenIdx)
	}
}

// TestHashChain_SingleEvent verifies single-event chain works correctly.
func TestHashChain_SingleEvent(t *testing.T) {
	setupHashChainForTest()

	e := makeTestEvent("create", time.Now())
	e.PrevHash = ""
	e.Hash = e.ComputeHash("")

	brokenIdx := VerifyChain([]*AuditEvent{e})
	if brokenIdx != -1 {
		t.Fatalf("single valid event should return -1, got %d", brokenIdx)
	}
}

// TestHashChain_LargeChain verifies a 100-event chain.
func TestHashChain_LargeChain(t *testing.T) {
	setupHashChainForTest()

	now := time.Now()
	events := make([]*AuditEvent, 100)
	prevHash := ""
	for i := range events {
		events[i] = makeTestEvent("bulk_action", now.Add(time.Duration(i)*time.Millisecond))
		events[i].PrevHash = prevHash
		events[i].Hash = events[i].ComputeHash(prevHash)
		prevHash = events[i].Hash
	}

	brokenIdx := VerifyChain(events)
	if brokenIdx != -1 {
		t.Fatalf("100-event valid chain should return -1, broke at %d", brokenIdx)
	}

	// Tamper event 50
	events[50].Result = ResultFailure
	brokenIdx = VerifyChain(events)
	if brokenIdx != 50 {
		t.Fatalf("tampered event 50 should be detected, got %d", brokenIdx)
	}
}

// TestHashChain_DetectTamperedTimestamp verifies changing CreatedAt after hashing is detected.
func TestHashChain_DetectTamperedTimestamp(t *testing.T) {
	setupHashChainForTest()

	now := time.Now()
	events := make([]*AuditEvent, 3)
	prevHash := ""
	for i := range events {
		events[i] = makeTestEvent("login", now.Add(time.Duration(i)*time.Second))
		events[i].PrevHash = prevHash
		events[i].Hash = events[i].ComputeHash(prevHash)
		prevHash = events[i].Hash
	}

	// Modify timestamp of event 0
	events[0].CreatedAt = events[0].CreatedAt.Add(time.Hour)

	brokenIdx := VerifyChain(events)
	if brokenIdx != 0 {
		t.Fatalf("tampered timestamp should break at index 0, got %d", brokenIdx)
	}
}

// TestHashChain_DetectTamperedIPAddress verifies changing IP after hashing is detected.
func TestHashChain_DetectTamperedIPAddress(t *testing.T) {
	setupHashChainForTest()

	now := time.Now()
	events := make([]*AuditEvent, 3)
	prevHash := ""
	for i := range events {
		events[i] = makeTestEvent("login", now.Add(time.Duration(i)*time.Second))
		events[i].PrevHash = prevHash
		events[i].Hash = events[i].ComputeHash(prevHash)
		prevHash = events[i].Hash
	}

	// Modify IP address of event 2
	events[2].IPAddress = "10.0.0.1"

	brokenIdx := VerifyChain(events)
	if brokenIdx != 2 {
		t.Fatalf("tampered IP should break at index 2, got %d", brokenIdx)
	}
}

// TestHashChain_NotEnabled verifies that without secret, ComputeHash produces empty hashes.
func TestHashChain_NotEnabled(t *testing.T) {
	// Clear the secret
	SetHashChainSecret(nil)

	if IsHashChainEnabled() {
		t.Fatal("hash chain should be disabled without secret")
	}

	// Restore for other tests
	defer setupHashChainForTest()
}

// TestComputeHash_Deterministic verifies same inputs produce same hash.
func TestComputeHash_Deterministic(t *testing.T) {
	setupHashChainForTest()

	e := makeTestEvent("create", time.Now())
	h1 := e.ComputeHash("prev-hash-1")
	h2 := e.ComputeHash("prev-hash-1")
	if h1 != h2 {
		t.Fatal("ComputeHash should be deterministic for same inputs")
	}
}

// TestComputeHash_DifferentPrevHash verifies different prevHash produces different hash.
func TestComputeHash_DifferentPrevHash(t *testing.T) {
	setupHashChainForTest()

	e := makeTestEvent("create", time.Now())
	h1 := e.ComputeHash("prev-hash-1")
	h2 := e.ComputeHash("prev-hash-2")
	if h1 == h2 {
		t.Fatal("different prevHash should produce different hashes")
	}
}

// TestVerifyHash_NoHash verifies VerifyHash returns false when hash is empty.
func TestVerifyHash_NoHash(t *testing.T) {
	setupHashChainForTest()

	e := makeTestEvent("create", time.Now())
	if e.VerifyHash("") {
		t.Fatal("VerifyHash should return false for empty hash")
	}
}

// TestCanonicalJSON verifies canonical serialization.
func TestCanonicalJSON(t *testing.T) {
	setupHashChainForTest()

	e := makeTestEvent("create", time.Now())
	data, err := e.CanonicalJSON()
	if err != nil {
		t.Fatalf("CanonicalJSON: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("CanonicalJSON should return non-empty data")
	}
}
