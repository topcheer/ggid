package detection

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStateStore implements StateStore using Redis sorted sets and counters.
// Safe for multi-replica deployments (unlike MemStateStore).
//
// - AddEvent: ZADD key member with score=ts + EXPIRE key ttl
// - EventsSince: ZRANGEBYSCORE key since +inf
// - Incr: INCR key + EXPIRE key ttl
type RedisStateStore struct {
	rdb *redis.Client
}

// NewRedisStateStore creates a Redis-backed state store for the detection engine.
func NewRedisStateStore(rdb *redis.Client) *RedisStateStore {
	return &RedisStateStore{rdb: rdb}
}

// AddEvent adds a member to a sorted set with score=timestamp.
func (s *RedisStateStore) AddEvent(ctx context.Context, key string, ts int64, member string, windowTTL time.Duration) error {
	if s.rdb == nil {
		return nil
	}
	pipe := s.rdb.Pipeline()
	pipe.ZAdd(ctx, redisKey(key), redis.Z{Score: float64(ts), Member: member})
	pipe.Expire(ctx, redisKey(key), windowTTL)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis state store AddEvent: %w", err)
	}
	return nil
}

// EventsSince returns all members with score >= since.
func (s *RedisStateStore) EventsSince(ctx context.Context, key string, since int64) ([]string, error) {
	if s.rdb == nil {
		return nil, nil
	}
	result, err := s.rdb.ZRangeByScore(ctx, redisKey(key), &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", since),
		Max: "+inf",
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("redis state store EventsSince: %w", err)
	}
	return result, nil
}

// Incr atomically increments a counter and sets TTL.
func (s *RedisStateStore) Incr(ctx context.Context, key string, windowTTL time.Duration) (int64, error) {
	if s.rdb == nil {
		return 0, nil
	}
	rk := redisKey(key)
	pipe := s.rdb.Pipeline()
	incr := pipe.Incr(ctx, rk)
	pipe.Expire(ctx, rk, windowTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("redis state store Incr: %w", err)
	}
	return incr.Val(), nil
}

// redisKey prefixes the key to avoid collisions with other Redis users.
func redisKey(key string) string {
	return "ggid:itdr:" + key
}
