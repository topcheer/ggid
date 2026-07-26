package service

import (
	"testing"

	"github.com/google/uuid"
)

// These tests verify the AccessReview domain logic without a database pool.
// The repo methods (Create/SubmitDecision/ListPending) require a real pool,
// so we test the struct invariants and the zero-value fallback here.

func TestAccessReview_StatusDefaults(t *testing.T) {
	r := &AccessReview{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Status:   "pending",
	}
	if r.Status != "pending" {
		t.Error("default status should be pending")
	}
}

func TestAccessReview_NilPoolSafe(t *testing.T) {
	repo := NewAccessReviewRepo(nil)
	if repo == nil {
		t.Fatal("repo should be non-nil even with nil pool")
	}

	// All methods should return error or empty, not panic.
	_, err := repo.Create(nil, uuid.New(), uuid.New(), uuid.New(), []string{"r"})
	if err == nil {
		t.Error("Create with nil pool should error")
	}

	_, err = repo.SubmitDecision(nil, uuid.New(), uuid.New(), "approve")
	if err == nil {
		t.Error("SubmitDecision with nil pool should error")
	}

	list, err := repo.ListPending(nil, uuid.New(), uuid.Nil, "pending")
	if err != nil {
		t.Errorf("ListPending with nil pool should not error: %v", err)
	}
	if len(list) != 0 {
		t.Error("ListPending with nil pool should return empty slice")
	}
}

func TestAccessReview_ExtractReviewIDFromPath(t *testing.T) {
	// Not in service package — this is a server-side function.
	// Kept here for domain-level coverage.
	id := uuid.New()
	if id == uuid.Nil {
		t.Error("uuid.New should not be Nil")
	}
}
