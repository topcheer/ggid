package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func init() {
	// Set a test secret for hash chain tests
	SetHashChainSecret([]byte("test-secret-key-for-hash-chain"))
}

func TestComputeHash_Deterministic(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	now := time.Now()

	e1 := &AuditEvent{
		ID:        id,
		TenantID:  tenantID,
		ActorType: ActorUser,
		Action:    "user.login",
		Result:    ResultSuccess,
		CreatedAt: now,
	}
	e2 := &AuditEvent{
		ID:        id,
		TenantID:  tenantID,
		ActorType: ActorUser,
		Action:    "user.login",
		Result:    ResultSuccess,
		CreatedAt: now,
	}

	h1 := e1.ComputeHash("")
	h2 := e2.ComputeHash("")

	if h1 != h2 {
		t.Error("same events should produce same hash")
	}
	if h1 == "" {
		t.Error("hash should not be empty")
	}
}

func TestComputeHash_DifferentEvents(t *testing.T) {
	e1 := &AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Action:    "user.login",
		CreatedAt: time.Now(),
	}
	e2 := &AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Action:    "user.logout",
		CreatedAt: time.Now(),
	}

	h1 := e1.ComputeHash("")
	h2 := e2.ComputeHash("")

	if h1 == h2 {
		t.Error("different events should produce different hashes")
	}
}

func TestComputeHash_ChainDependent(t *testing.T) {
	e := &AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Action:    "test",
		CreatedAt: time.Now(),
	}

	h1 := e.ComputeHash("prev1")
	h2 := e.ComputeHash("prev2")

	if h1 == h2 {
		t.Error("same event with different prev_hash should produce different hashes")
	}
}

func TestVerifyHash_ValidChain(t *testing.T) {
	prevHash := ""
	e := &AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Action:    "test",
		CreatedAt: time.Now(),
	}
	e.Hash = e.ComputeHash(prevHash)

	if !e.VerifyHash(prevHash) {
		t.Error("verify should return true for correctly computed hash")
	}
}

func TestVerifyHash_InvalidChain(t *testing.T) {
	e := &AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Action:    "test",
		CreatedAt: time.Now(),
	}
	e.Hash = "invalid-hash-value"

	if e.VerifyHash("") {
		t.Error("verify should return false for invalid hash")
	}
}

func TestVerifyHash_EmptyHash(t *testing.T) {
	e := &AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Action:    "test",
		CreatedAt: time.Now(),
	}
	// Hash is empty by default

	if e.VerifyHash("") {
		t.Error("verify should return false for empty hash")
	}
}

func TestVerifyChain_ValidChain(t *testing.T) {
	prevHash := ""
	events := make([]*AuditEvent, 5)
	for i := range events {
		e := &AuditEvent{
			ID:        uuid.New(),
			TenantID:  uuid.New(),
			Action:    "test.action",
			Result:    ResultSuccess,
			CreatedAt: time.Now(),
		}
		e.Hash = e.ComputeHash(prevHash)
		prevHash = e.Hash
		events[i] = e
	}

	brokenAt := VerifyChain(events)
	if brokenAt != -1 {
		t.Errorf("valid chain should return -1, got %d", brokenAt)
	}
}

func TestVerifyChain_BrokenLink(t *testing.T) {
	prevHash := ""
	events := make([]*AuditEvent, 5)
	for i := range events {
		e := &AuditEvent{
			ID:        uuid.New(),
			TenantID:  uuid.New(),
			Action:    "test.action",
			CreatedAt: time.Now(),
		}
		e.Hash = e.ComputeHash(prevHash)
		prevHash = e.Hash
		events[i] = e
	}

	// Tamper with event 2
	events[2].Action = "tampered.action"

	brokenAt := VerifyChain(events)
	// Should detect break at event 2 (the tampered one)
	if brokenAt != 2 {
		t.Errorf("broken chain should return 2, got %d", brokenAt)
	}
}

func TestVerifyChain_EmptyChain(t *testing.T) {
	brokenAt := VerifyChain(nil)
	if brokenAt != -1 {
		t.Errorf("empty chain should return -1, got %d", brokenAt)
	}
}

func TestVerifyChain_SingleEvent(t *testing.T) {
	e := &AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Action:    "single",
		CreatedAt: time.Now(),
	}
	e.Hash = e.ComputeHash("")

	brokenAt := VerifyChain([]*AuditEvent{e})
	if brokenAt != -1 {
		t.Errorf("single valid event should return -1, got %d", brokenAt)
	}
}

func TestCanonicalJSON(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	e := &AuditEvent{
		ID:        id,
		TenantID:  tenantID,
		Action:    "test",
		CreatedAt: time.Now(),
	}

	data, err := e.CanonicalJSON()
	if err != nil {
		t.Fatalf("CanonicalJSON error: %v", err)
	}
	if len(data) == 0 {
		t.Error("CanonicalJSON should return non-empty data")
	}

	// Should be deterministic
	data2, _ := e.CanonicalJSON()
	if string(data) != string(data2) {
		t.Error("CanonicalJSON should be deterministic")
	}
}

func TestSetHashChainSecret_ChangesHash(t *testing.T) {
	e := &AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Action:    "test",
		CreatedAt: time.Now(),
	}

	SetHashChainSecret([]byte("secret1"))
	h1 := e.ComputeHash("")

	SetHashChainSecret([]byte("secret2"))
	h2 := e.ComputeHash("")

	if h1 == h2 {
		t.Error("different secrets should produce different hashes")
	}

	// Restore test secret
	SetHashChainSecret([]byte("test-secret-key-for-hash-chain"))
}
