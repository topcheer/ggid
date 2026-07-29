package service

import (
	"sync"
	"testing"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// tenantWithFirstByte returns a UUID whose first byte is b (shard selection
// uses tenantID[0] & (hashChainShardCount-1)).
func tenantWithFirstByte(b byte) uuid.UUID {
	var id uuid.UUID
	id[0] = b
	return id
}

// TestShardFor_Deterministic verifies the same tenant always maps to the same
// shard (chain state consistency).
func TestShardFor_Deterministic(t *testing.T) {
	s := NewAuditService(nil)
	tenant := uuid.New()
	if s.shardFor(tenant) != s.shardFor(tenant) {
		t.Fatal("same tenant mapped to different shards")
	}
}

// TestShardLock_Independence verifies different shards have independent locks
// (cross-tenant parallelism) while the same shard is mutually exclusive
// (per-tenant chain serialization).
func TestShardLock_Independence(t *testing.T) {
	s := NewAuditService(nil)

	// First byte 0x00 -> shard 0; 0x01 -> shard 1 (different shards).
	tenantA := tenantWithFirstByte(0x00)
	tenantB := tenantWithFirstByte(0x01)
	// 0x40 = 64, 64 & 63 = 0 -> same shard as tenantA.
	tenantA2 := tenantWithFirstByte(0x40)

	shardA := s.shardFor(tenantA)
	shardB := s.shardFor(tenantB)
	shardA2 := s.shardFor(tenantA2)
	if shardA == shardB {
		t.Fatal("test setup: expected different shards")
	}
	if shardA != shardA2 {
		t.Fatal("test setup: expected same shard")
	}

	shardA.mu.Lock()
	defer shardA.mu.Unlock()

	// Different shard: lock is independent — TryLock must succeed.
	if !shardB.mu.TryLock() {
		t.Fatal("different shard was blocked by another shard's lock")
	}
	shardB.mu.Unlock()

	// Same shard (different tenant): must be mutually exclusive.
	if shardA2.mu.TryLock() {
		shardA2.mu.Unlock()
		t.Fatal("same shard lock acquired while held — tenants in one shard are not serialized")
	}
}

// TestComputeHashChain_ConcurrentIntegrity hammers computeHashChain from many
// goroutines across several tenants and verifies every tenant's chain is
// internally consistent: exactly one genesis event and unbroken Hash links.
func TestComputeHashChain_ConcurrentIntegrity(t *testing.T) {
	s := NewAuditService(nil)

	const tenants = 8
	const eventsPerTenant = 50

	tenantIDs := make([]uuid.UUID, tenants)
	for i := range tenantIDs {
		tenantIDs[i] = uuid.New()
	}

	var mu sync.Mutex
	events := make([]*domain.AuditEvent, 0, tenants*eventsPerTenant)

	var wg sync.WaitGroup
	for _, tid := range tenantIDs {
		for i := 0; i < eventsPerTenant; i++ {
			wg.Add(1)
			go func(tid uuid.UUID) {
				defer wg.Done()
				e := &domain.AuditEvent{ID: uuid.New(), TenantID: tid, Action: "test"}
				s.computeHashChain(e)
				mu.Lock()
				events = append(events, e)
				mu.Unlock()
			}(tid)
		}
	}
	wg.Wait()

	byTenant := make(map[uuid.UUID][]*domain.AuditEvent)
	for _, e := range events {
		byTenant[e.TenantID] = append(byTenant[e.TenantID], e)
	}

	for _, tid := range tenantIDs {
		evs := byTenant[tid]
		if len(evs) != eventsPerTenant {
			t.Fatalf("tenant %s: got %d events, want %d", tid, len(evs), eventsPerTenant)
		}
		// Exactly one genesis (empty PrevHash).
		genesis := 0
		byHash := make(map[string]*domain.AuditEvent, len(evs))
		for _, e := range evs {
			if e.PrevHash == "" {
				genesis++
			}
			byHash[e.Hash] = e
			if e.Hash == "" {
				t.Fatalf("tenant %s: event with empty Hash", tid)
			}
		}
		if genesis != 1 {
			t.Fatalf("tenant %s: %d genesis events, want exactly 1 (chain broken by race)", tid, genesis)
		}
		// Walk the chain from genesis: must visit every event exactly once.
		visited := 0
		current := ""
		for {
			var next *domain.AuditEvent
			for _, e := range evs {
				if e.PrevHash == current {
					next = e
					break
				}
			}
			if next == nil {
				break
			}
			visited++
			current = next.Hash
		}
		if visited != eventsPerTenant {
			t.Fatalf("tenant %s: chain walk visited %d/%d events (fork or break in chain)", tid, visited, eventsPerTenant)
		}
	}
}
