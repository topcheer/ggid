package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// TestMain sets a hash chain secret so that IsHashChainEnabled() returns true
// during tests, allowing ComputeHash/VerifyHash to work correctly.
func TestMain(m *testing.M) {
	domain.SetHashChainSecret([]byte("test-hash-chain-secret"))
	os.Exit(m.Run())
}

// hashChainTestRepo is a minimal in-memory repo for hash chain tests.
type hashChainTestRepo struct {
	events []*domain.AuditEvent
}

func (r *hashChainTestRepo) Insert(_ context.Context, e *domain.AuditEvent) error {
	r.events = append(r.events, e)
	return nil
}
func (r *hashChainTestRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.AuditEvent, error) {
	for _, e := range r.events {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, nil
}
func (r *hashChainTestRepo) List(_ context.Context, filter domain.ListFilter, limit, offset int) ([]*domain.AuditEvent, int, error) {
	var result []*domain.AuditEvent
	for _, e := range r.events {
		if filter.TenantID != uuid.Nil && e.TenantID != filter.TenantID {
			continue
		}
		result = append(result, e)
	}
	total := len(result)
	if offset < len(result) {
		end := offset + limit
		if end > len(result) {
			end = len(result)
		}
		result = result[offset:end]
	}
	return result, total, nil
}
func (r *hashChainTestRepo) GetStats(_ context.Context, _ uuid.UUID, _ time.Time) (*domain.Stats, error) {
	return &domain.Stats{}, nil
}
func (r *hashChainTestRepo) DeleteOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

func makeChainEvent(tenantID uuid.UUID, action string) *domain.AuditEvent {
	return &domain.AuditEvent{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Action:    action,
		ActorName: "test-user",
		Result:    domain.ResultSuccess,
		CreatedAt: time.Now().UTC(),
	}
}

func TestHashChain_SingleEvent(t *testing.T) {
	repo := &hashChainTestRepo{}
	svc := NewAuditService(repo)
	tenantID := uuid.New()

	e := makeChainEvent(tenantID, "user.login")
	if err := svc.InsertEvent(context.Background(), e); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if e.Hash == "" {
		t.Error("hash should be set after insert")
	}
	if e.PrevHash != "" {
		t.Error("first event should have empty prev_hash")
	}
}

func TestHashChain_ChainedEvents(t *testing.T) {
	repo := &hashChainTestRepo{}
	svc := NewAuditService(repo)
	tenantID := uuid.New()

	e1 := makeChainEvent(tenantID, "user.login")
	e2 := makeChainEvent(tenantID, "role.assign")
	e3 := makeChainEvent(tenantID, "user.logout")

	_ = svc.InsertEvent(context.Background(), e1)
	_ = svc.InsertEvent(context.Background(), e2)
	_ = svc.InsertEvent(context.Background(), e3)

	// First event: no prev hash
	if e1.PrevHash != "" {
		t.Error("first event prev_hash should be empty")
	}
	// Second event's prev_hash should equal first event's hash
	if e2.PrevHash != e1.Hash {
		t.Error("second event prev_hash should equal first event hash")
	}
	// Third event's prev_hash should equal second event's hash
	if e3.PrevHash != e2.Hash {
		t.Error("third event prev_hash should equal second event hash")
	}
	// All three hashes should be different
	if e1.Hash == e2.Hash || e2.Hash == e3.Hash || e1.Hash == e3.Hash {
		t.Error("all hashes should be different")
	}
}

func TestHashChain_PerTenantIsolation(t *testing.T) {
	repo := &hashChainTestRepo{}
	svc := NewAuditService(repo)
	tenantA := uuid.New()
	tenantB := uuid.New()

	eA := makeChainEvent(tenantA, "user.login")
	eB := makeChainEvent(tenantB, "user.login")

	_ = svc.InsertEvent(context.Background(), eA)
	_ = svc.InsertEvent(context.Background(), eB)

	// Both first events for different tenants should have empty prev_hash
	if eA.PrevHash != "" {
		t.Error("first event for tenantA should have empty prev_hash")
	}
	if eB.PrevHash != "" {
		t.Error("first event for tenantB should have empty prev_hash")
	}
}

func TestHashChain_VerifyIntegrity_Valid(t *testing.T) {
	repo := &hashChainTestRepo{}
	svc := NewAuditService(repo)
	tenantID := uuid.New()

	_ = svc.InsertEvent(context.Background(), makeChainEvent(tenantID, "user.login"))
	_ = svc.InsertEvent(context.Background(), makeChainEvent(tenantID, "role.assign"))
	_ = svc.InsertEvent(context.Background(), makeChainEvent(tenantID, "user.logout"))

	err := svc.VerifyIntegrity(context.Background(), tenantID)
	if err != nil {
		t.Errorf("integrity check should pass: %v", err)
	}
}

func TestHashChain_VerifyIntegrity_Empty(t *testing.T) {
	repo := &hashChainTestRepo{}
	svc := NewAuditService(repo)

	err := svc.VerifyIntegrity(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("integrity check on empty chain should return nil: %v", err)
	}
}

func TestHashChain_VerifyIntegrity_NilTenant(t *testing.T) {
	repo := &hashChainTestRepo{}
	svc := NewAuditService(repo)

	err := svc.VerifyIntegrity(context.Background(), uuid.Nil)
	if err == nil {
		t.Error("should error on nil tenant_id")
	}
}

func TestHashChain_VerifyIntegrity_Tampered(t *testing.T) {
	repo := &hashChainTestRepo{}
	svc := NewAuditService(repo)
	tenantID := uuid.New()

	_ = svc.InsertEvent(context.Background(), makeChainEvent(tenantID, "user.login"))
	_ = svc.InsertEvent(context.Background(), makeChainEvent(tenantID, "role.assign"))

	// Tamper: modify a stored event's action after hashing
	repo.events[0].Action = "admin.delete"

	err := svc.VerifyIntegrity(context.Background(), tenantID)
	if err == nil {
		t.Error("integrity check should detect tampering")
	}
}

func TestHashChain_CanonicalData_Deterministic(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	now := time.Now().UTC()

	e1 := &domain.AuditEvent{
		ID: id, TenantID: tenantID, Action: "test", ActorName: "user",
		Result: domain.ResultSuccess, CreatedAt: now,
	}
	e2 := &domain.AuditEvent{
		ID: id, TenantID: tenantID, Action: "test", ActorName: "user",
		Result: domain.ResultSuccess, CreatedAt: now,
	}

	d1 := canonicalEventData(e1)
	d2 := canonicalEventData(e2)

	if string(d1) != string(d2) {
		t.Error("canonical data should be deterministic for identical events")
	}
}

func TestHashChain_MetadataSortedOrder(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	now := time.Now().UTC()

	e1 := &domain.AuditEvent{
		ID: id, TenantID: tenantID, Action: "test", CreatedAt: now,
		Metadata: map[string]any{"z": "last", "a": "first", "m": "middle"},
	}
	e2 := &domain.AuditEvent{
		ID: id, TenantID: tenantID, Action: "test", CreatedAt: now,
		Metadata: map[string]any{"a": "first", "m": "middle", "z": "last"},
	}

	d1 := canonicalEventData(e1)
	d2 := canonicalEventData(e2)

	if string(d1) != string(d2) {
		t.Error("canonical data should be identical regardless of metadata insertion order")
	}
}
