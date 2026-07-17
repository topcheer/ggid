package server

import (
	"testing"

	"github.com/google/uuid"
)

func TestRebacCacheKey_Format(t *testing.T) {
	key := rebacCacheKey("tenant-1", "document", "report", "viewer", "user:alice")
	expected := "ggid:rebac:tenant-1:document:report#viewer@user:alice"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestRebacCache_NilRedis_DirectCheck(t *testing.T) {
	cache := newRebacCache(newRelationTupleRepo(nil), nil)
	// With nil pool AND nil redis, Check returns false (not configured).
	resp := cache.CheckWithCache(nil, CheckRequest{
		TenantID:  uuid.New(),
		Namespace: "doc",
		Object:    "r",
		Relation:  "viewer",
		Subject:   "user:a",
	})
	if resp.Allowed {
		t.Error("nil pool should not allow")
	}
}

func TestRebacCache_InvalidateNoPanic(t *testing.T) {
	cache := newRebacCache(newRelationTupleRepo(nil), nil)
	// With nil redis, Invalidate should be a no-op (no panic).
	cache.InvalidateOnWrite(nil, uuid.New().String(), "doc", "report")
}
