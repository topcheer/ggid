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
	_ = repo.CreateTarget(nil, &t1)
	// With nil pool, no error expected
	if t1.ID == uuid.Nil {
		t.Error("ID should be set even with nil pool")
	}
}
