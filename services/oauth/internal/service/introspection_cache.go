package service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// IntrospectionCache caches token introspection results with TTL.
type IntrospectionCache struct {
	mu    sync.RWMutex
	store map[string]*cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	result   map[string]any
	expireAt time.Time
}

func NewIntrospectionCache() *IntrospectionCache {
	return &IntrospectionCache{store: make(map[string]*cacheEntry), ttl: 30 * time.Second}
}

func (c *IntrospectionCache) Get(_ context.Context, tokenHash string) map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.store[tokenHash]
	if !ok || time.Now().After(e.expireAt) {
		return nil
	}
	return e.result
}

func (c *IntrospectionCache) Set(_ context.Context, tokenHash string, result map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[tokenHash] = &cacheEntry{result: result, expireAt: time.Now().Add(c.ttl)}
}

func (c *IntrospectionCache) Invalidate(_ context.Context, tokenHash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, tokenHash)
}

func (c *IntrospectionCache) InvalidateAll(_ context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store = make(map[string]*cacheEntry)
}

func hashToken(token string) string {
	n := len(token)
	if n > 32 { n = 32 }
	return fmt.Sprintf("%x", []byte(token[:n]))
}
