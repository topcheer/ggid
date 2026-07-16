package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
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

// CampaignItem represents a single user's review within a campaign.
type CampaignItem struct {
	ID         string     `json:"id"`
	CampaignID string     `json:"campaign_id"`
	UserID     string     `json:"user_id"`
	RoleID     string     `json:"role_id"`
	Decision   string     `json:"decision"`
	DecidedAt  *time.Time `json:"decided_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ListRevokeItems returns campaign items with decision=revoke for a campaign.
func (r *CampaignRepo) ListRevokeItems(ctx context.Context, campaignID string) ([]*CampaignItem, error) {
	if r.pool == nil {
		return []*CampaignItem{}, nil
	}
	cid, err := uuid.Parse(campaignID)
	if err != nil {
		return nil, fmt.Errorf("invalid campaign_id: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, campaign_id::text, user_id::text, role_id::text, decision, decided_at, created_at
		FROM iga_campaign_items WHERE campaign_id = $1 AND decision = 'revoke'
	`, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*CampaignItem
	for rows.Next() {
		var item CampaignItem
		var decidedAt *time.Time
		if err := rows.Scan(&item.ID, &item.CampaignID, &item.UserID, &item.RoleID, &item.Decision, &decidedAt, &item.CreatedAt); err != nil {
			continue
		}
		item.DecidedAt = decidedAt
		items = append(items, &item)
	}
	return items, nil
}

// AddItem adds a review item to a campaign.
func (r *CampaignRepo) AddItem(ctx context.Context, item *CampaignItem) error {
	if r.pool == nil {
		return nil
	}
	cid, err := uuid.Parse(item.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign_id: %w", err)
	}
	uid, err := uuid.Parse(item.UserID)
	if err != nil {
		return fmt.Errorf("invalid user_id: %w", err)
	}
	rid, err := uuid.Parse(item.RoleID)
	if err != nil {
		return fmt.Errorf("invalid role_id: %w", err)
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO iga_campaign_items (campaign_id, user_id, role_id, decision)
		VALUES ($1, $2, $3, COALESCE($4, 'pending'))
		RETURNING id, created_at
	`, cid, uid, rid, item.Decision).Scan(&item.ID, &item.CreatedAt)
}
