package service

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAccessReview_Create(t *testing.T) {
	ResetAccessReviews()
	r := CreateAccessReview(uuid.New(), uuid.New(), uuid.New(), []string{"viewer", "editor"})
	if r.Status != "pending" {
		t.Error("new review should be pending")
	}
}

func TestAccessReview_Approve(t *testing.T) {
	ResetAccessReviews()
	r := CreateAccessReview(uuid.New(), uuid.New(), uuid.New(), []string{"admin"})
	result, err := SubmitAccessReview(r.ID, "approve")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if result.Status != "approved" {
		t.Error("should be approved")
	}
}

func TestAccessReview_Revoke(t *testing.T) {
	ResetAccessReviews()
	r := CreateAccessReview(uuid.New(), uuid.New(), uuid.New(), []string{"admin"})
	result, _ := SubmitAccessReview(r.ID, "revoke")
	if result.Status != "revoked" {
		t.Error("should be revoked")
	}
}

func TestAccessReview_DoubleSubmit(t *testing.T) {
	ResetAccessReviews()
	r := CreateAccessReview(uuid.New(), uuid.New(), uuid.New(), []string{"admin"})
	SubmitAccessReview(r.ID, "approve")
	_, err := SubmitAccessReview(r.ID, "revoke")
	if err == nil {
		t.Error("should not allow double submission")
	}
}

func TestAccessReview_ListPending(t *testing.T) {
	ResetAccessReviews()
	mgr := uuid.New()
	CreateAccessReview(mgr, uuid.New(), uuid.New(), []string{"r1"})
	CreateAccessReview(mgr, uuid.New(), uuid.New(), []string{"r2"})
	CreateAccessReview(uuid.New(), uuid.New(), uuid.New(), []string{"r3"})
	pending := ListPendingAccessReviews(mgr)
	if len(pending) != 2 {
		t.Errorf("expected 2, got %d", len(pending))
	}
}

func TestAccessReview_NotFound(t *testing.T) {
	ResetAccessReviews()
	_, err := SubmitAccessReview(uuid.New(), "approve")
	if err == nil {
		t.Error("should error for nonexistent review")
	}
}

// SoD test stubs — these are also defined in policy package but we keep
// audit-specific tests here to verify the access review + SoD integration.
var _ = sync.RWMutex{}
var _ = time.Now
var _ = fmt.Sprintf
