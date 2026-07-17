package server

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateDynamicCredential_Format(t *testing.T) {
	targetID := uuid.New()
	cred := GenerateDynamicCredential(targetID, "user-123", time.Now().Add(time.Hour))
	if len(cred) < 20 {
		t.Errorf("credential too short: %s", cred)
	}
	if cred[:4] != "ztb_" {
		t.Errorf("credential should start with ztb_: %s", cred)
	}
}

func TestGenerateDynamicCredential_Unique(t *testing.T) {
	targetID := uuid.New()
	c1 := GenerateDynamicCredential(targetID, "user-1", time.Now().Add(time.Hour))
	c2 := GenerateDynamicCredential(targetID, "user-1", time.Now().Add(time.Hour))
	if c1 == c2 {
		t.Error("credentials should be unique even for same params")
	}
}

func TestSecretBrokerRepo_NilPool(t *testing.T) {
	repo := newSecretBrokerRepo(nil)
	targets, err := repo.ListTargets(nil, uuid.Nil)
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if len(targets) != 0 {
		t.Error("nil pool should return empty")
	}
	grants, err := repo.ListActiveGrants(nil, uuid.Nil)
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if len(grants) != 0 {
		t.Error("nil pool should return empty")
	}
}

func TestSecretBrokerRepo_NilGrantsByTarget(t *testing.T) {
	repo := newSecretBrokerRepo(nil)
	grants, err := repo.ListGrantsByTarget(nil, uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if len(grants) != 0 {
		t.Error("nil pool should return empty")
	}
}

func TestSecretBrokerRepo_CleanupExpired(t *testing.T) {
	repo := newSecretBrokerRepo(nil)
	deleted, err := repo.CleanupExpired(nil, uuid.New())
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if deleted != 0 {
		t.Error("nil pool should delete 0")
	}
}

func TestSecretTargetDefaults(t *testing.T) {
	repo := newSecretBrokerRepo(nil)
	t1 := SecretTarget{Name: "prod-db", Type: "db", TTLSeconds: 0}
	// With nil pool, CreateTarget is a no-op (returns nil immediately)
	if err := repo.CreateTarget(nil, &t1); err != nil {
		t.Errorf("nil pool CreateTarget should not error: %v", err)
	}
	// With nil pool, ListTargets returns empty slice
	targets, _ := repo.ListTargets(nil, uuid.Nil)
	if len(targets) != 0 {
		t.Error("nil pool should return empty list")
	}
}
