package server

import (
	"testing"

	"github.com/google/uuid"
)

func TestMatchPattern_Exact(t *testing.T) {
	if !matchPattern("/api/v1/auth/login", "/api/v1/auth/login") {
		t.Error("exact match should work")
	}
}

func TestMatchPattern_Wildcard(t *testing.T) {
	if !matchPattern("/api/v1/auth/*", "/api/v1/auth/login") {
		t.Error("wildcard should match prefix")
	}
	if matchPattern("/api/v1/auth/*", "/api/v1/users/list") {
		t.Error("non-matching prefix should fail")
	}
}

func TestMatchPattern_Global(t *testing.T) {
	if !matchPattern("*", "/anything") {
		t.Error("global wildcard should match all")
	}
}

func TestMatchPattern_NoWildcard(t *testing.T) {
	if matchPattern("/api/v1/auth", "/api/v1/auth/login") {
		t.Error("exact pattern without * should not match different path")
	}
}

func TestRateLimitRepo_NilPool(t *testing.T) {
	repo := newRateLimitRepo(nil, nil)
	limits, err := repo.List(nil, uuid.New())
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if len(limits) != 0 {
		t.Error("nil pool should return empty")
	}
	// CheckRateLimit with nil Redis should allow.
	allowed, _ := repo.CheckRateLimit(nil, uuid.New(), "/api/v1/test")
	if !allowed {
		t.Error("nil Redis should allow (fail open)")
	}
}

func TestTenantRateLimit_Defaults(t *testing.T) {
	rl := &TenantRateLimit{EndpointPattern: "/api/v1/*"}
	if rl.RPSLimit <= 0 {
		rl.RPSLimit = 100
	}
	if rl.BurstLimit <= 0 {
		rl.BurstLimit = rl.RPSLimit * 2
	}
	if rl.Strategy == "" {
		rl.Strategy = "token_bucket"
	}
	if rl.RPSLimit != 100 || rl.BurstLimit != 200 || rl.Strategy != "token_bucket" {
		t.Error("defaults not applied correctly")
	}
}
