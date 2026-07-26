package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AccessReview represents a periodic access certification entry.
type AccessReview struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	ManagerID  uuid.UUID `json:"manager_id"`
	UserID     uuid.UUID `json:"user_id"`
	Roles      []string  `json:"roles"`
	Status     string    `json:"status"`
	Decision   string    `json:"decision,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	ReviewedAt time.Time `json:"reviewed_at,omitempty"`
}

// AccessReviewRepo provides DB-backed persistence for access reviews.
type AccessReviewRepo struct {
	pool *pgxpool.Pool
}

// NewAccessReviewRepo creates a repository using the given pgxpool.
func NewAccessReviewRepo(pool *pgxpool.Pool) *AccessReviewRepo {
	return &AccessReviewRepo{pool: pool}
}

// Create inserts a new pending access certification.
func (repo *AccessReviewRepo) Create(ctx context.Context, tenantID, managerID, userID uuid.UUID, roles []string) (*AccessReview, error) {
	if repo.pool == nil {
		return nil, fmt.Errorf("database pool not available")
	}
	r := &AccessReview{
		ID:        uuid.New(),
		TenantID:  tenantID,
		ManagerID: managerID,
		UserID:    userID,
		Roles:     roles,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}
	_, err := repo.pool.Exec(ctx,
		`INSERT INTO access_reviews (id, tenant_id, manager_id, user_id, roles, status, decision, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		r.ID, r.TenantID, r.ManagerID, r.UserID, r.Roles, r.Status, r.Decision, r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create access review: %w", err)
	}
	return r, nil
}

// SubmitDecision records an approve/revoke decision for a review.
func (repo *AccessReviewRepo) SubmitDecision(ctx context.Context, tenantID, reviewID uuid.UUID, decision string) (*AccessReview, error) {
	if repo.pool == nil {
		return nil, fmt.Errorf("database pool not available")
	}
	r := &AccessReview{}
	err := repo.pool.QueryRow(ctx,
		`UPDATE access_reviews
		 SET status = CASE WHEN $1 = 'approve' THEN 'approved' ELSE 'revoked' END,
		     decision = $1,
		     reviewed_at = now()
		 WHERE id = $2 AND tenant_id = $3 AND status = 'pending'
		 RETURNING id, tenant_id, manager_id, user_id, roles, status, decision, created_at, reviewed_at`,
		decision, reviewID, tenantID).Scan(
		&r.ID, &r.TenantID, &r.ManagerID, &r.UserID, &r.Roles, &r.Status, &r.Decision, &r.CreatedAt, &r.ReviewedAt)
	if err != nil {
		return nil, fmt.Errorf("submit access review decision: %w", err)
	}
	return r, nil
}

// ListPending returns pending reviews filtered by tenant.
// When managerID is non-zero, further filters by manager.
// When status is non-empty, filters by that status (default: "pending").
func (repo *AccessReviewRepo) ListPending(ctx context.Context, tenantID uuid.UUID, managerID uuid.UUID, status string) ([]*AccessReview, error) {
	if repo.pool == nil {
		return []*AccessReview{}, nil
	}
	if status == "" {
		status = "pending"
	}

	// Build query dynamically based on whether managerID is provided.
	rows, err := repo.pool.Query(ctx,
		`SELECT id, tenant_id, manager_id, user_id, roles, status, decision, created_at, reviewed_at
		 FROM access_reviews
		 WHERE tenant_id = $1 AND status = $2 AND ($3::uuid = '00000000-0000-0000-0000-000000000000' OR manager_id = $3)
		 ORDER BY created_at DESC`,
		tenantID, status, managerID)
	if err != nil {
		return nil, fmt.Errorf("list access reviews: %w", err)
	}
	defer rows.Close()

	var out []*AccessReview
	for rows.Next() {
		r := &AccessReview{}
		if err := rows.Scan(&r.ID, &r.TenantID, &r.ManagerID, &r.UserID, &r.Roles, &r.Status, &r.Decision, &r.CreatedAt, &r.ReviewedAt); err != nil {
			continue
		}
		out = append(out, r)
	}
	if out == nil {
		out = []*AccessReview{}
	}
	return out, nil
}
