package server

import (
	"testing"
	"time"
)

func TestTTLCache_SetGet(t *testing.T) {
	c := &ttlCache{entries: make(map[string]*cacheEntry)}
	c.Set("key1", []byte("value1"), 5*time.Second)

	data, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(data) != "value1" {
		t.Errorf("got %s, want value1", string(data))
	}
}

func TestTTLCache_Expiry(t *testing.T) {
	c := &ttlCache{entries: make(map[string]*cacheEntry)}
	c.Set("key2", []byte("value2"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("key2")
	if ok {
		t.Error("expected cache miss after expiry")
	}
}

func TestTTLCache_Invalidate(t *testing.T) {
	c := &ttlCache{entries: make(map[string]*cacheEntry)}
	c.Set("users:default:tenant1", []byte("data1"), 10*time.Second)
	c.Set("users:default:tenant2", []byte("data2"), 10*time.Second)
	c.Set("roles:tenant1", []byte("data3"), 10*time.Second)

	c.Invalidate("users:default:")

	if _, ok := c.Get("users:default:tenant1"); ok {
		t.Error("users:default:tenant1 should be invalidated")
	}
	if _, ok := c.Get("users:default:tenant2"); ok {
		t.Error("users:default:tenant2 should be invalidated")
	}
	if _, ok := c.Get("roles:tenant1"); !ok {
		t.Error("roles:tenant1 should still exist")
	}
}

func TestTTLCache_Cleanup(t *testing.T) {
	c := &ttlCache{entries: make(map[string]*cacheEntry)}
	c.Set("expired", []byte("x"), 1*time.Millisecond)
	c.Set("valid", []byte("y"), 10*time.Second)
	time.Sleep(5 * time.Millisecond)

	c.Cleanup()

	if _, ok := c.Get("expired"); ok {
		t.Error("expired entry should be cleaned up")
	}
	if _, ok := c.Get("valid"); !ok {
		t.Error("valid entry should still exist")
	}
}
