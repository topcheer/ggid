package middleware

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisJTIReplayTracker uses Redis SETNX for distributed JTI replay detection.
// Falls back to in-memory if Redis is unavailable.
type RedisJTIReplayTracker struct {
	rdb     *redis.Client
	maxAge  time.Duration
	memory  *JTIReplayTracker // fallback when Redis is down
}

// NewRedisJTIReplayTracker creates a Redis-backed tracker with memory fallback.
func NewRedisJTIReplayTracker(rdb *redis.Client, maxAge time.Duration) *RedisJTIReplayTracker {
	return &RedisJTIReplayTracker{
		rdb:    rdb,
		maxAge: maxAge,
		memory: NewJTIReplayTracker(maxAge),
	}
}

// IsReplayed returns true if the jti has already been seen.
// Uses Redis SETNX with TTL for distributed detection.
// Falls back to in-memory tracker if Redis is unavailable.
func (t *RedisJTIReplayTracker) IsReplayed(jti string, expiresAt time.Time) bool {
	if jti == "" {
		return true // empty jti = invalid
	}

	if t.rdb == nil {
		return t.memory.IsReplayed(jti, expiresAt)
	}

	// Calculate TTL — how long until this token expires
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return false // already expired, allow
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// SETNX: returns true if key was set (first use), false if already existed (replay)
	key := "jti:" + jti
	set, err := t.rdb.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		// Redis error — fall back to in-memory (best effort, not ideal)
		return t.memory.IsReplayed(jti, expiresAt)
	}

	// set=true means we just inserted it (first use) → not replayed
	// set=false means it already existed → replayed
	return !set
}
