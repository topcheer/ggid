package server

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// rebacCache wraps a relationTupleRepo with Redis caching for Check results.
// Cache key: ggid:rebac:{tenant}:{ns}:{obj}#{rel}@{subject}
// TTL: 60s (short — balances performance with permission change propagation).
// Invalidate on WriteTuple/DeleteTuple.
type rebacCache struct {
	repo *relationTupleRepo
	rdb  *redis.Client
	ttl  time.Duration
}

func newRebacCache(repo *relationTupleRepo, rdb *redis.Client) *rebacCache {
	return &rebacCache{
		repo: repo,
		rdb:  rdb,
		ttl:  60 * time.Second,
	}
}

// cacheKey builds the Redis key for a check result.
func rebacCacheKey(tenantID, ns, obj, rel, subject string) string {
	return fmt.Sprintf("ggid:rebac:%s:%s:%s#%s@%s", tenantID, ns, obj, rel, subject)
}

// CheckWithCache first checks Redis, falls back to repo.Check, caches result.
func (c *rebacCache) CheckWithCache(ctx context.Context, req CheckRequest) CheckResponse {
	if c.rdb == nil {
		// No Redis — direct DB check.
		return c.repo.Check(ctx, req)
	}

	key := rebacCacheKey(req.TenantID.String(), req.Namespace, req.Object, req.Relation, req.Subject)

	// Try cache.
	val, err := c.rdb.Get(ctx, key).Result()
	if err == nil {
		allowed := val == "1"
		return CheckResponse{Allowed: allowed, Reason: "cached"}
	}

	// Cache miss — query DB.
	resp := c.repo.Check(ctx, req)

	// Cache result (best-effort, don't block on errors).
	cachedVal := "0"
	if resp.Allowed {
		cachedVal = "1"
	}
	_ = c.rdb.Set(ctx, key, cachedVal, c.ttl).Err()

	return resp
}

// InvalidateOnWrite clears cached check results when tuples change.
// Called by WriteTuple/DeleteTuple.
func (c *rebacCache) InvalidateOnWrite(ctx context.Context, tenantID, ns, obj string) {
	if c.rdb == nil {
		return
	}
	// Delete all cached checks for this object (wildcard pattern).
	pattern := fmt.Sprintf("ggid:rebac:%s:%s:%s#*", tenantID, ns, obj)
	iter := c.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if len(keys) > 0 {
		c.rdb.Del(ctx, keys...)
	}
}

// CachedDirectSubjects wraps DirectSubjects with Redis caching.
func (c *rebacCache) CachedDirectSubjects(ctx context.Context, tenantID uuid.UUID, ns, obj, rel string) ([]string, error) {
	if c.rdb == nil {
		return c.repo.DirectSubjects(ctx, tenantID, ns, obj, rel)
	}

	key := fmt.Sprintf("ggid:rebac:ds:%s:%s:%s:%s", tenantID, ns, obj, rel)

	// Try cache.
	vals, err := c.rdb.SMembers(ctx, key).Result()
	if err == nil && len(vals) > 0 {
		return vals, nil
	}

	// Cache miss — query DB.
	subjects, err := c.repo.DirectSubjects(ctx, tenantID, ns, obj, rel)
	if err != nil || len(subjects) == 0 {
		return subjects, err
	}

	// Cache result as a Redis set.
	members := make([]any, len(subjects))
	for i, s := range subjects {
		members[i] = s
	}
	pipe := c.rdb.Pipeline()
	pipe.Del(ctx, key) // Clear old set
	pipe.SAdd(ctx, key, members...)
	pipe.Expire(ctx, key, c.ttl)
	pipe.Exec(ctx)

	return subjects, nil
}
