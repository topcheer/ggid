package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestIntrospectionCache_SetGet(t *testing.T) {
	c := NewIntrospectionCache()
	ctx := context.Background()
	c.Set(ctx, "hash1", map[string]any{"sub": "user1", "active": true})

	got := c.Get(ctx, "hash1")
	if got == nil {
		t.Fatal("should return cached entry")
	}
	if got["sub"] != "user1" {
		t.Error("wrong subject")
	}
}

func TestIntrospectionCache_Miss(t *testing.T) {
	c := NewIntrospectionCache()
	if c.Get(context.Background(), "nonexistent") != nil {
		t.Error("should return nil for missing key")
	}
}

func TestIntrospectionCache_Invalidate(t *testing.T) {
	c := NewIntrospectionCache()
	ctx := context.Background()
	c.Set(ctx, "hash1", map[string]any{"sub": "user1"})
	c.Invalidate(ctx, "hash1")

	if c.Get(ctx, "hash1") != nil {
		t.Error("should be nil after invalidation")
	}
}

func TestIntrospectionCache_InvalidateAll(t *testing.T) {
	c := NewIntrospectionCache()
	ctx := context.Background()
	c.Set(ctx, "h1", map[string]any{})
	c.Set(ctx, "h2", map[string]any{})
	c.InvalidateAll(ctx)

	if c.Get(ctx, "h1") != nil || c.Get(ctx, "h2") != nil {
		t.Error("all entries should be cleared")
	}
}

func TestIntrospectionCache_TTLExpiry(t *testing.T) {
	c := NewIntrospectionCache()
	c.ttl = 50 * time.Millisecond
	ctx := context.Background()
	c.Set(ctx, "hash1", map[string]any{})

	time.Sleep(80 * time.Millisecond)
	if c.Get(ctx, "hash1") != nil {
		t.Error("should expire after TTL")
	}
}

func TestHashToken(t *testing.T) {
	h := hashToken("short")
	if h == "" {
		t.Error("hash should not be empty")
	}
	h2 := hashToken("short")
	if h != h2 {
		t.Error("same input should produce same hash")
	}
}

var _ = sync.Mutex{}
