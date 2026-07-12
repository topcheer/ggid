package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type CachedIntrospection struct {
	Active    bool       `json:"active"`
	Scope     string     `json:"scope"`
	ClientID  string     `json:"client_id"`
	Expiry    time.Time  `json:"expiry"`
	CachedAt  time.Time  `json:"cached_at"`
}

type CacheStats struct {
	Hits     int `json:"hits"`
	Misses   int `json:"misses"`
	Sets     int `json:"sets"`
	Invalidations int `json:"invalidations"`
}

type IntrospectionCache struct {
	mu          sync.RWMutex
	cache       map[string]*CachedIntrospection
	activeTTL   time.Duration
	inactiveTTL time.Duration
	ttl         time.Duration
	stats       CacheStats
}

func NewIntrospectionCache() *IntrospectionCache {
	return &IntrospectionCache{
		cache:       make(map[string]*CachedIntrospection),
		activeTTL:   60 * time.Second,
		inactiveTTL: 5 * time.Minute,
		ttl:         60 * time.Second,
	}
}

func (c *IntrospectionCache) GetCachedIntrospection(token string) (*CachedIntrospection, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := hashToken(token)
	entry, ok := c.cache[key]
	if !ok {
		c.stats.Misses++
		return nil, false
	}
	if time.Since(entry.CachedAt) > c.ttlFor(entry) {
		c.stats.Misses++
		return nil, false
	}
	c.stats.Hits++
	return entry, true
}

func (c *IntrospectionCache) SetCachedIntrospection(token string, result *CachedIntrospection) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := hashToken(token)
	result.CachedAt = time.Now()
	c.cache[key] = result
	c.stats.Sets++
}

func (c *IntrospectionCache) InvalidateCache(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := hashToken(token)
	delete(c.cache, key)
	c.stats.Invalidations++
}

func (c *IntrospectionCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

func (c *IntrospectionCache) ttlFor(entry *CachedIntrospection) time.Duration {
	if entry.Active {
		return c.activeTTL
	}
	return c.inactiveTTL
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// ctx-compatible methods for existing test compatibility
func (c *IntrospectionCache) Set(ctx context.Context, key string, data map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = &CachedIntrospection{
		Active:   true,
		CachedAt: time.Now(),
	}
	c.cache[key].Scope, _ = data["scope"].(string)
	c.cache[key].ClientID, _ = data["sub"].(string)
	c.stats.Sets++
}

func (c *IntrospectionCache) Get(ctx context.Context, key string) map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.cache[key]
	if !ok {
		c.stats.Misses++
		return nil
	}
	if time.Since(entry.CachedAt) > c.ttl {
		c.stats.Misses++
		return nil
	}
	c.stats.Hits++
	return map[string]any{
		"sub":    entry.ClientID,
		"scope":  entry.Scope,
		"active": entry.Active,
	}
}

func (c *IntrospectionCache) Invalidate(ctx context.Context, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
	c.stats.Invalidations++
}

func (c *IntrospectionCache) InvalidateAll(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*CachedIntrospection)
}