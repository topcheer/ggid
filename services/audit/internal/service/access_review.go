package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AccessReview represents a periodic access certification entry.
type AccessReview struct {
	ID         uuid.UUID
	ManagerID  uuid.UUID
	UserID     uuid.UUID
	TenantID   uuid.UUID
	Roles      []string
	Status     string // pending, approved, revoked
	CreatedAt  time.Time
	ReviewedAt time.Time
	Decision   string
}

var accessReviews sync.Map // key: uuid.UUID, value: *AccessReview

// CreateAccessReview creates a pending access certification.
func CreateAccessReview(managerID, userID, tenantID uuid.UUID, roles []string) *AccessReview {
	r := &AccessReview{
		ID:        uuid.New(),
		ManagerID: managerID,
		UserID:    userID,
		TenantID:  tenantID,
		Roles:     roles,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}
	accessReviews.Store(r.ID, r)
	return r
}

// SubmitAccessReview records approve/revoke decision.
func SubmitAccessReview(reviewID uuid.UUID, decision string) (*AccessReview, error) {
	v, ok := accessReviews.Load(reviewID)
	if !ok {
		return nil, fmt.Errorf("review not found")
	}
	r := v.(*AccessReview)
	if r.Status != "pending" {
		return nil, fmt.Errorf("already completed")
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

// ListPendingAccessReviews returns pending reviews for a manager.
func ListPendingAccessReviews(managerID uuid.UUID) []*AccessReview {
	var out []*AccessReview
	accessReviews.Range(func(key, value any) bool {
		r := value.(*AccessReview)
		if r.ManagerID == managerID && r.Status == "pending" {
			out = append(out, r)
		}
		return true
	})
	return out
}

// ResetAccessReviews clears all reviews (for testing).
func ResetAccessReviews() {
	accessReviews.Range(func(key, value any) bool {
		accessReviews.Delete(key)
		return true
	})
}
