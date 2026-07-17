package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BreakGlassRecord represents an emergency access event persisted in DB.
type BreakGlassRecord struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	Requester        uuid.UUID  `json:"requester"`
	RequesterName    string     `json:"requester_name"`
	Reason           string     `json:"reason"`
	Scope            string     `json:"scope"`
	DurationMinutes  int        `json:"duration_minutes"`
	ActivatedAt      time.Time  `json:"activated_at"`
	DeactivatedAt    *time.Time `json:"deactivated_at,omitempty"`
	Status           string     `json:"status"`
}

// BreakGlassRepository manages break-glass record persistence.
type BreakGlassRepository struct {
	pool *pgxpool.Pool
}

func NewBreakGlassRepository(pool *pgxpool.Pool) *BreakGlassRepository {
	return &BreakGlassRepository{pool: pool}
}

// EnsureSchema creates the break_glass_records table if it doesn't exist.
func (r *BreakGlassRepository) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS break_glass_records (
			id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id         UUID NOT NULL,
			requester         UUID NOT NULL,
			requester_name    TEXT,
			reason            TEXT NOT NULL,
			scope             TEXT NOT NULL DEFAULT '',
			duration_minutes  INT NOT NULL DEFAULT 60,
			activated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			deactivated_at    TIMESTAMPTZ,
			status            TEXT NOT NULL DEFAULT 'active'
		);
		CREATE INDEX IF NOT EXISTS idx_break_glass_tenant_time ON break_glass_records (tenant_id, activated_at DESC);
		CREATE INDEX IF NOT EXISTS idx_break_glass_status      ON break_glass_records (tenant_id, status) WHERE status = 'active';
	`)
	return err
}

// Create inserts a new break-glass record.
func (r *BreakGlassRepository) Create(ctx context.Context, rec *BreakGlassRecord) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO break_glass_records (id, tenant_id, requester, requester_name, reason, scope, duration_minutes, activated_at, deactivated_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		rec.ID, rec.TenantID, rec.Requester, rec.RequesterName, rec.Reason, rec.Scope,
		rec.DurationMinutes, rec.ActivatedAt, rec.DeactivatedAt, rec.Status,
	)
	return err
}

// ListByTenant returns break-glass records for a tenant, newest first.
func (r *BreakGlassRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*BreakGlassRecord, error) {
	if r.pool == nil {
		return []*BreakGlassRecord{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, requester, requester_name, reason, scope, duration_minutes, activated_at, deactivated_at, status
		FROM break_glass_records
		WHERE tenant_id = $1
		ORDER BY activated_at DESC
		LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*BreakGlassRecord
	for rows.Next() {
		rec, err := scanBreakGlassRow(rows)
		if err != nil {
			continue
		}
		records = append(records, rec)
	}
	return records, nil
}

// Deactivate marks a break-glass record as expired.
func (r *BreakGlassRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE break_glass_records SET status = 'expired', deactivated_at = now()
		WHERE id = $1 AND status = 'active'`,
		id,
	)
	return err
}

func scanBreakGlassRow(row interface {
	Scan(dest ...any) error
}) (*BreakGlassRecord, error) {
	var rec BreakGlassRecord
	var deactivatedAt *time.Time
	err := row.Scan(
		&rec.ID, &rec.TenantID, &rec.Requester, &rec.RequesterName, &rec.Reason,
		&rec.Scope, &rec.DurationMinutes, &rec.ActivatedAt, &deactivatedAt, &rec.Status,
	)
	if err != nil {
		return nil, err
	}
	rec.DeactivatedAt = deactivatedAt
	return &rec, nil
}
