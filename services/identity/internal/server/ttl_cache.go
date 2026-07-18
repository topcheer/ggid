package server

import (
	"sync"
	"time"
)

// ttlCache is a simple in-memory TTL cache for hot endpoint responses.
// In production, replace with Redis-backed cache.
type ttlCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
}

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

var globalTTLCache = &ttlCache{entries: make(map[string]*cacheEntry)}

// cacheGet returns cached data if not expired.
func (c *ttlCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.data, true
}

// cacheSet stores data with a TTL.
func (c *ttlCache) Set(key string, data []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
}

// cacheInvalidate removes a key.
func (c *ttlCache) Invalidate(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.entries {
		if len(prefix) > 0 && len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.entries, k)
		}
	}
}

// cacheCleanup removes expired entries (call periodically).
func (c *ttlCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, k)
		}
	}
}
