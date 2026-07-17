// Package auth provides shared authentication utilities.
package auth

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const revokedJTIKey = "ggid:revoked_jti"

// JTIBlocklist manages revoked JWT jti values using a Redis ZSET.
// Score is the JWT expiry timestamp; entries are auto-expired via periodic cleanup.
type JTIBlocklist struct {
	rdb *redis.Client
}

// NewJTIBlocklist creates a Redis-backed JTI blocklist.
func NewJTIBlocklist(rdb *redis.Client) *JTIBlocklist {
	return &JTIBlocklist{rdb: rdb}
}

// Revoke adds a jti to the blocklist with TTL = JWT expiry.
func (b *JTIBlocklist) Revoke(ctx context.Context, jti string, jwtExp time.Time) error {
	if b.rdb == nil {
		return nil // dev mode — no-op
	}
	score := float64(jwtExp.Unix())
	if err := b.rdb.ZAdd(ctx, revokedJTIKey, redis.Z{Score: score, Member: jti}).Err(); err != nil {
		return fmt.Errorf("jti blocklist revoke: %w", err)
	}
	return nil
}

// RevokeAll revokes multiple jtis in a single pipeline.
func (b *JTIBlocklist) RevokeAll(ctx context.Context, jtis []string, jwtExp time.Time) error {
	if b.rdb == nil || len(jtis) == 0 {
		return nil
	}
	pipe := b.rdb.Pipeline()
	score := float64(jwtExp.Unix())
	for _, jti := range jtis {
		pipe.ZAdd(ctx, revokedJTIKey, redis.Z{Score: score, Member: jti})
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("jti blocklist revoke all: %w", err)
	}
	return nil
}

// IsRevoked checks if a jti is in the blocklist. O(1), ~0.3ms.
// Returns false (allow) if Redis is unavailable (graceful degradation).
func (b *JTIBlocklist) IsRevoked(ctx context.Context, jti string) bool {
	if b.rdb == nil || jti == "" {
		return false
	}
	score, err := b.rdb.ZScore(ctx, revokedJTIKey, jti).Result()
	if err != nil {
		// Key doesn't exist or Redis error — allow (degrade gracefully).
		if err != redis.Nil {
			log.Printf("jti blocklist: redis error, allowing: %v", err)
		}
		return false
	}
	return score > 0
}

// CleanupExpired removes entries with score < current time.
// Should be called periodically (e.g., every 5 minutes).
func (b *JTIBlocklist) CleanupExpired(ctx context.Context) error {
	if b.rdb == nil {
		return nil
	}
	now := float64(time.Now().Unix())
	return b.rdb.ZRemRangeByScore(ctx, revokedJTIKey, "-inf", fmt.Sprintf("%.0f", now)).Err()
}
