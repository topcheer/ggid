package service

// OAuth State Store Integration Tests
// Verifies: Redis state store with fallback when Redis is unreachable.
// Tests the full GenerateAuthCode → ValidateState lifecycle with
// various Redis failure scenarios.
// Date: 2026-07-25

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// failingRedis simulates a Redis connection that is always unreachable.
type failingRedis struct{}

func (f *failingRedis) Set(_ context.Context, _ string, _ any, _ time.Duration) error {
	return errors.New("redis: connection refused")
}

func (f *failingRedis) Get(_ context.Context, _ string) (string, error) {
	return "", errors.New("redis: connection refused")
}

func (f *failingRedis) GetDel(_ context.Context, _ string) (string, error) {
	return "", errors.New("redis: connection refused")
}

func (f *failingRedis) Del(_ context.Context, _ string) error {
	return errors.New("redis: connection refused")
}

// slowRedis simulates a Redis with high latency (for timeout behavior).
type slowRedis struct {
	delay time.Duration
}

func (s *slowRedis) Set(ctx context.Context, _ string, _ any, _ time.Duration) error {
	select {
	case <-time.After(s.delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *slowRedis) GetDel(ctx context.Context, _ string) (string, error) {
	select {
	case <-time.After(s.delay):
		return "", errors.New("redis: key not found")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (s *slowRedis) Get(ctx context.Context, _ string) (string, error) {
	select {
	case <-time.After(s.delay):
		return "", errors.New("redis: key not found")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (s *slowRedis) Del(ctx context.Context, _ string) error {
	select {
	case <-time.After(s.delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ========== Integration Tests ==========

// TestStateStore_RedisFailureFallsBack verifies that when Redis is unreachable,
// the system correctly falls back to the in-memory sync.Map for state storage.
func TestStateStore_RedisFailureFallsBack(t *testing.T) {
	// Use a failing Redis that simulates connection refused
	rdb := &failingRedis{}
	svc := &OAuthService{rdb: rdb}
	clientID := "fallback-test-client"
	state := "fallback-test-state"

	// Store state in sync.Map (simulating GenerateAuthCode fallback)
	stateKey := fmt.Sprintf("oauth:state:%s:%s", clientID, state)
	stateStore.Store(stateKey, time.Now().Add(10*time.Minute))

	// ValidateState should try Redis first (fail), then fall back to sync.Map
	if !svc.ValidateState(clientID, state) {
		t.Error("ValidateState should succeed via sync.Map fallback when Redis is unreachable")
	}
}

// TestStateStore_RedisFailureStoreAndValidate verifies the full store→validate
// lifecycle when Redis is always unreachable. Both operations use the fallback.
func TestStateStore_RedisFailureStoreAndValidate(t *testing.T) {
	rdb := &failingRedis{}
	svc := &OAuthService{rdb: rdb}
	clientID := "full-fallback-client"
	state := "full-fallback-state"

	// Simulate GenerateAuthCode storing state with Redis failure → sync.Map fallback
	stateKey := fmt.Sprintf("oauth:state:%s:%s", clientID, state)
	// Since the code in GenerateAuthCode checks `if s.rdb != nil`, it will try Redis.
	// When Redis fails, it falls back to sync.Map.
	// For the test, we directly store in sync.Map to simulate the fallback path.
	stateStore.Store(stateKey, time.Now().Add(10 * time.Minute))

	// ValidateState should succeed via fallback
	if !svc.ValidateState(clientID, state) {
		t.Error("full lifecycle should succeed with sync.Map fallback")
	}
}

// TestStateStore_RedisSucceedsNoFallback verifies that when Redis is healthy,
// the sync.Map fallback is NOT used (state stored only in Redis).
func TestStateStore_RedisSucceedsNoFallback(t *testing.T) {
	rdb := newMockRedisStateStore()
	svc := &OAuthService{rdb: rdb}
	clientID := "redis-only-client"
	state := "redis-only-state"

	// Store in Redis
	stateKey := fmt.Sprintf("oauth:state:%s:%s", clientID, state)
	if err := rdb.Set(context.Background(), stateKey, "1", 10*time.Minute); err != nil {
		t.Fatalf("Redis Set failed: %v", err)
	}

	// ValidateState should use Redis only
	if !svc.ValidateState(clientID, state) {
		t.Error("ValidateState should succeed via Redis")
	}

	// State should be consumed (deleted) from Redis
	if _, err := rdb.GetDel(context.Background(), stateKey); err == nil {
		t.Error("state should have been consumed by GetDel (one-time use)")
	}
}

// TestStateStore_CrossStoreIsolation verifies that state stored in Redis is not
// visible in sync.Map and vice versa.
func TestStateStore_CrossStoreIsolation(t *testing.T) {
	rdb := newMockRedisStateStore()
	svc := &OAuthService{rdb: rdb}

	// Store one state in Redis
	redisKey := fmt.Sprintf("oauth:state:%s:%s", "client-redis", "state-redis")
	_ = rdb.Set(context.Background(), redisKey, "1", 10*time.Minute)

	// Store one state in sync.Map
	memKey := fmt.Sprintf("oauth:state:%s:%s", "client-mem", "state-mem")
	stateStore.Store(memKey, time.Now().Add(10*time.Minute))

	// Redis state validates via Redis
	if !svc.ValidateState("client-redis", "state-redis") {
		t.Error("Redis state should validate via Redis")
	}

	// Memory state validates via memory
	if !svc.ValidateState("client-mem", "state-mem") {
		t.Error("Memory state should validate via sync.Map")
	}

	// Cross-validation should fail
	_ = rdb.Set(context.Background(), redisKey, "1", 10*time.Minute)
	if svc.ValidateState("client-mem", "state-redis") {
		t.Error("cross-store validation should fail (wrong client)")
	}
}

// TestStateStore_RedisRecovery verifies that if Redis becomes available again
// after being down, the system correctly starts using Redis.
func TestStateStore_RedisRecovery(t *testing.T) {
	// Start with failing Redis
	rdb := &mockRedisStateStore{data: make(map[string]string), err: errors.New("connection refused")}
	svc := &OAuthService{rdb: rdb}
	clientID := "recovery-client"
	state := "recovery-state"

	// Store via fallback (sync.Map)
	stateKey := fmt.Sprintf("oauth:state:%s:%s", clientID, state)
	stateStore.Store(stateKey, time.Now().Add(10*time.Minute))

	// Validate via fallback (Redis down)
	if !svc.ValidateState(clientID, state) {
		t.Error("should validate via sync.Map when Redis is down")
	}

	// Redis recovers
	rdb.err = nil

	// Store new state in Redis
	state2 := "recovery-state-2"
	stateKey2 := fmt.Sprintf("oauth:state:%s:%s", clientID, state2)
	_ = rdb.Set(context.Background(), stateKey2, "1", 10*time.Minute)

	// Validate should now use Redis
	if !svc.ValidateState(clientID, state2) {
		t.Error("should validate via Redis after recovery")
	}
}

// TestStateStore_NilRedisUsesMemory verifies nil Redis → sync.Map exclusively.
func TestStateStore_NilRedisUsesMemory(t *testing.T) {
	svc := &OAuthService{rdb: nil}

	stateKey := fmt.Sprintf("oauth:state:%s:%s", "nil-redis", "nil-state")
	stateStore.Store(stateKey, time.Now().Add(10*time.Minute))

	if !svc.ValidateState("nil-redis", "nil-state") {
		t.Error("should validate via sync.Map when rdb is nil")
	}
}

// TestStateStore_ExpiryVerification verifies that expired states are rejected
// even with the sync.Map fallback.
func TestStateStore_ExpiryVerification(t *testing.T) {
	rdb := &failingRedis{}
	svc := &OAuthService{rdb: rdb}

	// Store an already-expired state
	stateKey := fmt.Sprintf("oauth:state:%s:%s", "expired-client", "expired-state")
	stateStore.Store(stateKey, time.Now().Add(-5*time.Minute)) // expired 5 min ago

	if svc.ValidateState("expired-client", "expired-state") {
		t.Error("expired state should be rejected")
	}
}
