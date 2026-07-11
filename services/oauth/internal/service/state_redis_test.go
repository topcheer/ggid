package service

// OAuth State Store Redis Migration Tests
// Verifies: Gap #12 — State store Redis migration with sync.Map fallback
// Date: 2026-07-25

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// mockRedisStateStore implements RedisCmdable for testing.
type mockRedisStateStore struct {
	data map[string]string
	err  error // inject error for all operations
}

func newMockRedisStateStore() *mockRedisStateStore {
	return &mockRedisStateStore{data: make(map[string]string)}
}

func (m *mockRedisStateStore) Set(_ context.Context, key string, _ any, _ time.Duration) error {
	if m.err != nil {
		return m.err
	}
	m.data[key] = "1"
	return nil
}

func (m *mockRedisStateStore) GetDel(_ context.Context, key string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	val, ok := m.data[key]
	if !ok {
		return "", errors.New("redis: nil") // simulates key-not-found
	}
	delete(m.data, key) // atomic get-and-delete
	return val, nil
}

// ========== State Store Redis Migration Tests ==========

// TestRedisStateStore_StoreAndValidate verifies the normal store→validate lifecycle
// using Redis.
func TestRedisStateStore_StoreAndValidate(t *testing.T) {
	rdb := newMockRedisStateStore()
	svc := &OAuthService{rdb: rdb}
	clientID := "redis-client-1"
	state := "redis-state-abc"

	// Simulate storing state (via GenerateAuthCode path)
	stateKey := fmt.Sprintf("oauth:state:%s:%s", clientID, state)
	if err := rdb.Set(context.Background(), stateKey, "1", 10*time.Minute); err != nil {
		t.Fatalf("failed to store state: %v", err)
	}

	// Validate should succeed
	if !svc.ValidateState(clientID, state) {
		t.Error("ValidateState should return true for valid Redis-stored state")
	}
}

// TestRedisStateStore_OneTimeUse verifies state is consumed after validation (Redis).
func TestRedisStateStore_OneTimeUse(t *testing.T) {
	rdb := newMockRedisStateStore()
	svc := &OAuthService{rdb: rdb}
	clientID := "redis-client-onetime"
	state := "state-replay-redis"

	stateKey := fmt.Sprintf("oauth:state:%s:%s", clientID, state)
	_ = rdb.Set(context.Background(), stateKey, "1", 10*time.Minute)

	// First validation succeeds
	if !svc.ValidateState(clientID, state) {
		t.Error("first validation should succeed")
	}

	// Second validation must fail (consumed by GetDel)
	if svc.ValidateState(clientID, state) {
		t.Error("replay attack: second validation should FAIL (state already consumed)")
	}
}

// TestRedisStateStore_UnknownRejected verifies unknown state is rejected.
func TestRedisStateStore_UnknownRejected(t *testing.T) {
	rdb := newMockRedisStateStore()
	svc := &OAuthService{rdb: rdb}

	if svc.ValidateState("client-x", "never-stored") {
		t.Error("unknown state should be rejected")
	}
}

// TestRedisStateStore_EmptyState verifies empty state is rejected.
func TestRedisStateStore_EmptyState(t *testing.T) {
	rdb := newMockRedisStateStore()
	svc := &OAuthService{rdb: rdb}

	if svc.ValidateState("client-x", "") {
		t.Error("empty state should be rejected")
	}
}

// TestRedisStateStore_CrossClientIsolation verifies client A's state can't be
// validated by client B.
func TestRedisStateStore_CrossClientIsolation(t *testing.T) {
	rdb := newMockRedisStateStore()
	svc := &OAuthService{rdb: rdb}
	clientA := "redis-client-a"
	clientB := "redis-client-b"
	state := "shared-redis-state"

	stateKey := fmt.Sprintf("oauth:state:%s:%s", clientA, state)
	_ = rdb.Set(context.Background(), stateKey, "1", 10*time.Minute)

	// Client A validates successfully
	if !svc.ValidateState(clientA, state) {
		t.Error("client A should validate its own state")
	}

	// Re-store for client A
	_ = rdb.Set(context.Background(), stateKey, "1", 10*time.Minute)

	// Client B should NOT validate client A's state
	if svc.ValidateState(clientB, state) {
		t.Error("cross-client CSRF: client B should NOT validate client A's state")
	}
}

// TestRedisStateStore_FallbackOnRedisError verifies that when Redis fails,
// the in-memory sync.Map fallback is used.
func TestRedisStateStore_FallbackOnRedisError(t *testing.T) {
	// Redis that always errors
	failingRdb := &mockRedisStateStore{err: errors.New("connection refused")}
	svc := &OAuthService{rdb: failingRdb}
	clientID := "fallback-client"
	state := "fallback-state"

	// Store in sync.Map fallback (simulating Redis failure during GenerateAuthCode)
	stateKey := fmt.Sprintf("oauth:state:%s:%s", clientID, state)
	stateStore.Store(stateKey, time.Now().Add(10*time.Minute))

	// ValidateState should fall through Redis error to sync.Map
	if !svc.ValidateState(clientID, state) {
		t.Error("ValidateState should succeed via in-memory fallback when Redis fails")
	}
}

// TestRedisStateStore_MultipleStates verifies multiple concurrent states.
func TestRedisStateStore_MultipleStates(t *testing.T) {
	rdb := newMockRedisStateStore()
	svc := &OAuthService{rdb: rdb}
	clientID := "redis-client-multi"

	states := []string{"state-r1", "state-r2", "state-r3"}
	for _, st := range states {
		key := fmt.Sprintf("oauth:state:%s:%s", clientID, st)
		_ = rdb.Set(context.Background(), key, "1", 10*time.Minute)
	}

	// Validate in reverse order
	for i := len(states) - 1; i >= 0; i-- {
		if !svc.ValidateState(clientID, states[i]) {
			t.Errorf("state %s should validate", states[i])
		}
	}

	// None should be reusable
	for _, st := range states {
		if svc.ValidateState(clientID, st) {
			t.Errorf("state %s should not be reusable", st)
		}
	}
}

// TestRedisStateStore_NilRedisUsesSyncMap verifies that when rdb is nil,
// the in-memory sync.Map is used exclusively.
func TestRedisStateStore_NilRedisUsesSyncMap(t *testing.T) {
	svc := &OAuthService{rdb: nil} // no Redis
	clientID := "nil-redis-client"
	state := "nil-redis-state"

	// Store via sync.Map
	stateKey := fmt.Sprintf("oauth:state:%s:%s", clientID, state)
	stateStore.Store(stateKey, time.Now().Add(10*time.Minute))

	// Validate should work via sync.Map
	if !svc.ValidateState(clientID, state) {
		t.Error("ValidateState should work with nil Redis (sync.Map only)")
	}
}
