package server

import (
	"testing"

	"github.com/google/uuid"
)

func TestConsentRepo_NilPool(t *testing.T) {
	repo := newConsentRepo(nil)

	records, err := repo.List(nil, uuid.Nil, "user-1", "")
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if len(records) != 0 {
		t.Error("nil pool should return empty")
	}

	ok, err := repo.HasValidConsent(nil, uuid.Nil, "user-1", "marketing")
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if ok {
		t.Error("nil pool should return false")
	}
}

func TestConsentRepo_GrantNilPool(t *testing.T) {
	repo := newConsentRepo(nil)
	c := &ConsentRecord{UserID: "u-1", Purpose: "analytics", Scopes: []string{"behavior"}}
	if err := repo.Grant(nil, c); err != nil {
		t.Errorf("nil pool Grant should not error: %v", err)
	}
	// With nil pool, Grant is a no-op (no DB to write to)
	records, _ := repo.List(nil, uuid.Nil, "u-1", "")
	if len(records) != 0 {
		t.Error("nil pool should have no records")
	}
}

func TestConsentRepo_PurgeNilPool(t *testing.T) {
	repo := newConsentRepo(nil)
	deleted, err := repo.PurgeUser(nil, uuid.Nil, "u-1")
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if deleted != 0 {
		t.Error("nil pool should delete 0")
	}
}

func TestConsentRepo_WithdrawNilPool(t *testing.T) {
	repo := newConsentRepo(nil)
	if err := repo.Withdraw(nil, uuid.New(), "user request"); err != nil {
		t.Errorf("nil pool Withdraw should not error: %v", err)
	}
}
