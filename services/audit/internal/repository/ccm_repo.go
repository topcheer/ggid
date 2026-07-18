package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CCMResultRecord is the persisted form of a CCM control evaluation result.
type CCMResultRecord struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	ControlID    string          `json:"control_id"`
	ControlName  string          `json:"control_name"`
	Category     string          `json:"category"`
	Status       string          `json:"status"`
	MetricValue  float64         `json:"metric_value"`
	Threshold    float64         `json:"threshold"`
	ThresholdDir string          `json:"threshold_dir"`
	Details      json.RawMessage `json:"details"`
	CheckedAt    time.Time       `json:"checked_at"`
}

// CCMRepository manages ccm_results persistence in PostgreSQL.
type CCMRepository struct {
	pool *pgxpool.Pool
}

func NewCCMRepository(pool *pgxpool.Pool) *CCMRepository {
	return &CCMRepository{pool: pool}
}

// EnsureSchema creates the ccm_results table.
func (r *CCMRepository) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS ccm_results (
			id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id     UUID NOT NULL,
			control_id    TEXT NOT NULL,
			control_name  TEXT NOT NULL,
			category      TEXT NOT NULL,
			status        TEXT NOT NULL,
			metric_value  DOUBLE PRECISION NOT NULL DEFAULT 0,
			threshold     DOUBLE PRECISION NOT NULL DEFAULT 0,
			threshold_dir TEXT NOT NULL DEFAULT 'lt',
			details       JSONB NOT NULL DEFAULT '{}',
			checked_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_ccm_tenant_time ON ccm_results (tenant_id, checked_at DESC);
		CREATE INDEX IF NOT EXISTS idx_ccm_control ON ccm_results (tenant_id, control_id, checked_at DESC);
	`)
	return err
}

// Store persists a CCM result to the database.
func (r *CCMRepository) Store(ctx context.Context, rec *CCMResultRecord) error {
	if r.pool == nil {
		return nil
	}
	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	if rec.CheckedAt.IsZero() {
		rec.CheckedAt = time.Now()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ccm_results (id, tenant_id, control_id, control_name, category, status, metric_value, threshold, threshold_dir, details, checked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		rec.ID, rec.TenantID, rec.ControlID, rec.ControlName, rec.Category,
		rec.Status, rec.MetricValue, rec.Threshold, rec.ThresholdDir, rec.Details, rec.CheckedAt,
	)
	return err
}

// StoreBatch persists multiple CCM results in one transaction.
func (r *CCMRepository) StoreBatch(ctx context.Context, records []*CCMResultRecord) error {
	if r.pool == nil {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, rec := range records {
		if rec.ID == uuid.Nil {
			rec.ID = uuid.New()
		}
		if rec.CheckedAt.IsZero() {
			rec.CheckedAt = time.Now()
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO ccm_results (id, tenant_id, control_id, control_name, category, status, metric_value, threshold, threshold_dir, details, checked_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			rec.ID, rec.TenantID, rec.ControlID, rec.ControlName, rec.Category,
			rec.Status, rec.MetricValue, rec.Threshold, rec.ThresholdDir, rec.Details, rec.CheckedAt,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ListLatest returns the most recent result for each control in a tenant.
func (r *CCMRepository) ListLatest(ctx context.Context, tenantID uuid.UUID) ([]*CCMResultRecord, error) {
	if r.pool == nil {
		return []*CCMResultRecord{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (control_id) id, tenant_id, control_id, control_name, category, status, metric_value, threshold, threshold_dir, details, checked_at
		FROM ccm_results
		WHERE tenant_id = $1
		ORDER BY control_id, checked_at DESC`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*CCMResultRecord
	for rows.Next() {
		rec, err := scanCCMRow(rows)
		if err != nil {
			continue
		}
		records = append(records, rec)
	}
	return records, nil
}

// ListHistory returns historical results for a tenant, optionally filtered by control_id.
func (r *CCMRepository) ListHistory(ctx context.Context, tenantID uuid.UUID, controlID string, limit int) ([]*CCMResultRecord, error) {
	if r.pool == nil {
		return []*CCMResultRecord{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var rows interface {
		Next() bool
		Close()
		Scan(dest ...any) error
	}
	var err error

	if controlID != "" {
		rows, err = r.pool.Query(ctx, `
			SELECT id, tenant_id, control_id, control_name, category, status, metric_value, threshold, threshold_dir, details, checked_at
			FROM ccm_results
			WHERE tenant_id = $1 AND control_id = $2
			ORDER BY checked_at DESC LIMIT $3`, tenantID, controlID, limit)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, tenant_id, control_id, control_name, category, status, metric_value, threshold, threshold_dir, details, checked_at
			FROM ccm_results
			WHERE tenant_id = $1
			ORDER BY checked_at DESC LIMIT $2`, tenantID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*CCMResultRecord
	for rows.Next() {
		rec, err := scanCCMRow(rows)
		if err != nil {
			continue
		}
		records = append(records, rec)
	}
	return records, nil
}

// scanner is a minimal interface matching pgx.Row and pgx.Rows scanning.
func scanCCMRow(row interface{ Scan(dest ...any) error }) (*CCMResultRecord, error) {
	var rec CCMResultRecord
	err := row.Scan(
		&rec.ID, &rec.TenantID, &rec.ControlID, &rec.ControlName, &rec.Category,
		&rec.Status, &rec.MetricValue, &rec.Threshold, &rec.ThresholdDir,
		&rec.Details, &rec.CheckedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}
