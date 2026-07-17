package httpserver

import (
	"testing"
)

func TestPDPCacheKey_Uniqueness(t *testing.T) {
	req1 := &AuthorizeRequest{Subject: "user:1", Resource: "doc:1", Action: "read"}
	req2 := &AuthorizeRequest{Subject: "user:2", Resource: "doc:1", Action: "read"}
	k1 := pdpCacheKey(req1)
	k2 := pdpCacheKey(req2)
	if k1 == k2 {
		t.Error("different subjects should produce different cache keys")
	}
}

func TestPDPCache_SetGet(t *testing.T) {
	FlushPDPCache()
	req := &AuthorizeRequest{Subject: "user:1", Resource: "doc:1", Action: "read"}
	key := pdpCacheKey(req)
	resp := &AuthorizeResponse{Allowed: true, DecisionID: "test-1"}
	cacheSet(key, resp)

	got, ok := cacheGet(key)
	if !ok {
		t.Fatal("cache miss after set")
	}
	if !got.Allowed {
		t.Error("should be allowed")
	}
	if !got.CacheHit {
		t.Error("should be cache hit")
	}
}

func TestPDPRepo_NilPool(t *testing.T) {
	repo := NewPDPRepo(nil)
	repo.LogDecision(nil, &AuthorizeResponse{DecisionID: "test"}, &AuthorizeRequest{Subject: "s", Resource: "r", Action: "a"})
	decisions, err := repo.ListDecisions(nil, 10, 0)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if len(decisions) != 0 { t.Error("nil pool should return empty") }
}

func TestFlushPDPCache(t *testing.T) {
	cacheSet("test-key", &AuthorizeResponse{Allowed: true})
	FlushPDPCache()
	if _, ok := cacheGet("test-key"); ok {
		t.Error("cache should be empty after flush")
	}
}

func TestBoolToDecision(t *testing.T) {
	if boolToDecision(true) != "allow" { t.Error("true should be allow") }
	if boolToDecision(false) != "deny" { t.Error("false should be deny") }
}
