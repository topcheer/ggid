package httpserver

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CampaignRepo is a DB-backed campaign store replacing the in-memory map.
type CampaignRepo struct {
	pool *pgxpool.Pool
}

func NewCampaignRepo(pool *pgxpool.Pool) *CampaignRepo {
	return &CampaignRepo{pool: pool}
}

func (r *CampaignRepo) Create(ctx context.Context, c *ReviewCampaign) error {
	if r.pool == nil {
		return nil
	}
	reviewerID, _ := uuid.Parse(c.ReviewerID)
	var deadline interface{}
	if !c.Deadline.IsZero() {
		deadline = c.Deadline
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO iga_campaigns (tenant_id, scope, scope_id, reviewer_id, deadline, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, c.TenantID, c.Scope, c.ScopeID, reviewerID, deadline, c.Status,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *CampaignRepo) ListActive(ctx context.Context, tenantID string) ([]*ReviewCampaign, error) {
	if r.pool == nil {
		return []*ReviewCampaign{}, nil
	}
	tid, _ := uuid.Parse(tenantID)
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, scope, scope_id, reviewer_id, deadline, status, decision, notes, created_at, submitted_at
		FROM iga_campaigns WHERE tenant_id = $1 AND status = 'active' ORDER BY created_at DESC
	`, tid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCampaigns(rows)
}

func (r *CampaignRepo) GetByID(ctx context.Context, id string) (*ReviewCampaign, error) {
	if r.pool == nil {
		return nil, nil
	}
	cid, _ := uuid.Parse(id)
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, scope, scope_id, reviewer_id, deadline, status, decision, notes, created_at, submitted_at
		FROM iga_campaigns WHERE id = $1
	`, cid)
	return scanCampaignRow(row)
}

func (r *CampaignRepo) Submit(ctx context.Context, id, decision, notes string) error {
	if r.pool == nil {
		return nil
	}
	cid, _ := uuid.Parse(id)
	_, err := r.pool.Exec(ctx, `
		UPDATE iga_campaigns SET status = 'completed', decision = $2, notes = $3, submitted_at = now()
		WHERE id = $1 AND status = 'active'
	`, cid, decision, notes)
	return err
}

// --- Helpers ---

func scanCampaigns(rows pgx.Rows) ([]*ReviewCampaign, error) {
	var result []*ReviewCampaign
	for rows.Next() {
		c, err := scanCampaignRow(rows)
		if err != nil {
			continue
		}
		result = append(result, c)
	}
	return result, nil
}

func scanCampaignRow(row interface {
	Scan(dest ...any) error
}) (*ReviewCampaign, error) {
	var (
		c           ReviewCampaign
		reviewerID  *uuid.UUID
		decision    *string
		notes       *string
		submittedAt *time.Time
		rawDetail   []byte
	)
	_ = rawDetail
	err := row.Scan(
		&c.ID, &c.TenantID, &c.Scope, &c.ScopeID, &reviewerID, &c.Deadline,
		&c.Status, &decision, &notes, &c.CreatedAt, &submittedAt,
	)
	if err != nil {
		return nil, err
	}
	if reviewerID != nil {
		c.ReviewerID = reviewerID.String()
	}
	if decision != nil {
		c.Decision = *decision
	}
	if notes != nil {
		c.Notes = *notes
	}
	c.SubmittedAt = submittedAt
	return &c, nil
}

// Ensure json import is used.
var _ = json.Marshal
