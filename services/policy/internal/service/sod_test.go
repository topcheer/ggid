package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSoD_AdminAndAuditor(t *testing.T) {
	ResetSoDRules()
	violations := CheckSoD(context.Background(), uuid.New(), []string{"admin", "auditor"})
	if len(violations) == 0 {
		t.Error("admin + auditor should violate SoD")
	}
}

func TestSoD_SingleRole(t *testing.T) {
	ResetSoDRules()
	violations := CheckSoD(context.Background(), uuid.New(), []string{"admin"})
	if len(violations) != 0 {
		t.Error("single role should not violate SoD")
	}
}

func TestSoD_NoConflict(t *testing.T) {
	ResetSoDRules()
	violations := CheckSoD(context.Background(), uuid.New(), []string{"viewer", "editor"})
	if len(violations) != 0 {
		t.Error("non-conflicting roles should not violate SoD")
	}
}

func TestSoD_CanAssignRole(t *testing.T) {
	ResetSoDRules()
	err := CanAssignRole(context.Background(), []string{"admin"}, "auditor")
	if err == nil {
		t.Error("should block assigning auditor to admin user")
	}
}

func TestSoD_CanAssignNonConflicting(t *testing.T) {
	ResetSoDRules()
	err := CanAssignRole(context.Background(), []string{"viewer"}, "editor")
	if err != nil {
		t.Errorf("should allow non-conflicting role: %v", err)
	}
}

func TestSoD_CustomRule(t *testing.T) {
	ResetSoDRules()
	AddSoDRule([]string{"devops", "security"}, "devops + security mutually exclusive")

	violations := CheckSoD(context.Background(), uuid.New(), []string{"devops", "security"})
	if len(violations) == 0 {
		t.Error("custom SoD rule should trigger")
	}
}

// --- Access Review ---

// AccessReview represents a periodic access certification.
type AccessReview struct {
	ID         uuid.UUID
	ManagerID  uuid.UUID
	UserID     uuid.UUID
	Roles      []string
	Status     string // "pending", "approved", "revoked"
	CreatedAt  time.Time
	ReviewedAt time.Time
	Decision   string
}

var (
	reviewMu  sync.RWMutex
	reviews   = make(map[uuid.UUID]*AccessReview)
)

// CreateAccessReview creates a pending review for manager to certify user access.
func CreateAccessReview(managerID, userID uuid.UUID, roles []string) *AccessReview {
	reviewMu.Lock()
	defer reviewMu.Unlock()
	r := &AccessReview{
		ID:        uuid.New(),
		ManagerID: managerID,
		UserID:    userID,
		Roles:     roles,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}
	reviews[r.ID] = r
	return r
}

// SubmitReviewDecision records manager's approve/revoke decision.
func SubmitReviewDecision(reviewID uuid.UUID, decision string) (*AccessReview, error) {
	reviewMu.Lock()
	defer reviewMu.Unlock()
	r, ok := reviews[reviewID]
	if !ok {
		return nil, fmt.Errorf("review not found")
	}
	if r.Status != "pending" {
		return nil, fmt.Errorf("review already completed")
	}
	r.Decision = decision
	r.ReviewedAt = time.Now().UTC()
	if decision == "approve" {
		r.Status = "approved"
	} else {
		r.Status = "revoked"
	}
	return r, nil
}

// ListPendingReviews returns pending reviews for a manager.
func ListPendingReviews(managerID uuid.UUID) []*AccessReview {
	reviewMu.RLock()
	defer reviewMu.RUnlock()
	var out []*AccessReview
	for _, r := range reviews {
		if r.ManagerID == managerID && r.Status == "pending" {
			out = append(out, r)
		}
	}
	return out
}

func ResetAccessReviews() {
	reviewMu.Lock()
	defer reviewMu.Unlock()
	reviews = make(map[uuid.UUID]*AccessReview)
}

func TestAccessReview_Create(t *testing.T) {
	ResetAccessReviews()
	r := CreateAccessReview(uuid.New(), uuid.New(), []string{"viewer", "editor"})
	if r.Status != "pending" {
		t.Error("new review should be pending")
	}
}

func TestAccessReview_Approve(t *testing.T) {
	ResetAccessReviews()
	r := CreateAccessReview(uuid.New(), uuid.New(), []string{"admin"})

	result, err := SubmitReviewDecision(r.ID, "approve")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if result.Status != "approved" {
		t.Error("should be approved")
	}
}

func TestAccessReview_Revoke(t *testing.T) {
	ResetAccessReviews()
	r := CreateAccessReview(uuid.New(), uuid.New(), []string{"admin"})

	result, _ := SubmitReviewDecision(r.ID, "revoke")
	if result.Status != "revoked" {
		t.Error("should be revoked")
	}
}

func TestAccessReview_DoubleSubmit(t *testing.T) {
	ResetAccessReviews()
	r := CreateAccessReview(uuid.New(), uuid.New(), []string{"admin"})
	SubmitReviewDecision(r.ID, "approve")

	_, err := SubmitReviewDecision(r.ID, "revoke")
	if err == nil {
		t.Error("should not allow double submission")
	}
}

func TestAccessReview_ListPending(t *testing.T) {
	ResetAccessReviews()
	mgr := uuid.New()
	CreateAccessReview(mgr, uuid.New(), []string{"r1"})
	CreateAccessReview(mgr, uuid.New(), []string{"r2"})
	CreateAccessReview(uuid.New(), uuid.New(), []string{"r3"}) // different manager

	pending := ListPendingReviews(mgr)
	if len(pending) != 2 {
		t.Errorf("expected 2 pending for manager, got %d", len(pending))
	}
}
