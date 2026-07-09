package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a fixed-window rate limiter backed by Redis.
type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

// CheckAndIncrement checks the rate limit for a key and increments the counter.
// Returns ErrRateLimited if the limit is exceeded within the current window.
// The window is 1 minute (60 seconds).
func (rl *RateLimiter) CheckAndIncrement(ctx context.Context, key string, limit int) error {
	redisKey := fmt.Sprintf("ggid:rl:%s", key)

	pipe := rl.rdb.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, time.Minute)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("rate limit check: %w", err)
	}

	count := incr.Val()
	if count > int64(limit) {
		return ErrRateLimited
	}
	return nil
}
