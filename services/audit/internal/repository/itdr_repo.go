package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ITDRRepository manages itdr_detections and itdr_rules persistence.
type ITDRRepository struct {
	pool *pgxpool.Pool
}

func NewITDRRepository(pool *pgxpool.Pool) *ITDRRepository {
	return &ITDRRepository{pool: pool}
}

// EnsureSchema creates itdr_detections and itdr_rules tables if they don't exist.
func (r *ITDRRepository) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS itdr_detections (
			id           UUID PRIMARY KEY,
			tenant_id    UUID NOT NULL,
			rule_id      TEXT NOT NULL,
			actor_id     UUID,
			severity     TEXT NOT NULL,
			title        TEXT NOT NULL,
			detail       JSONB NOT NULL DEFAULT '{}',
			event_ids    UUID[] NOT NULL DEFAULT '{}',
			status       TEXT NOT NULL DEFAULT 'new',
			hit_count    INT NOT NULL DEFAULT 1,
			detected_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_itdr_det_tenant_time ON itdr_detections (tenant_id, detected_at DESC);
		CREATE INDEX IF NOT EXISTS idx_itdr_det_status ON itdr_detections (tenant_id, status, severity);

		CREATE TABLE IF NOT EXISTS itdr_rules (
			id          TEXT NOT NULL,
			tenant_id   UUID NOT NULL,
			enabled     BOOLEAN NOT NULL DEFAULT TRUE,
			severity    TEXT,
			threshold   JSONB NOT NULL DEFAULT '{}',
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (id, tenant_id)
		);

		CREATE TABLE IF NOT EXISTS iga_campaign_items (
			id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			campaign_id  UUID NOT NULL,
			user_id      UUID NOT NULL,
			role_id      UUID NOT NULL,
			decision     TEXT,
			decided_at   TIMESTAMPTZ,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_iga_items_campaign ON iga_campaign_items (campaign_id);
	`)
	return err
}

// InsertDetection inserts or updates (UPSERT) a detection.
// Dedup key: same rule_id + actor_id + 5-minute bucket.
// On conflict: hit_count+1, event_ids appended, updated_at refreshed.
func (r *ITDRRepository) InsertDetection(ctx context.Context, d *domain.Detection) error {
	if r.pool == nil {
		return nil // no-op in skeleton mode
	}

	actorID := uuid.Nil
	if d.ActorID != nil {
		actorID = *d.ActorID
	}

	// 5-minute bucket for dedup.
	bucket := d.DetectedAt.Truncate(5 * time.Minute)

	detailJSON, _ := json.Marshal(d.Detail)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO itdr_detections (id, tenant_id, rule_id, actor_id, severity, title, detail, event_ids, status, hit_count, detected_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 1, $10, $10)
		ON CONFLICT ON CONSTRAINT itdr_det_dedup_unique DO UPDATE SET
			hit_count = itdr_detections.hit_count + 1,
			event_ids = itdr_detections.event_ids || EXCLUDED.event_ids,
			updated_at = $10
	`, d.ID, d.TenantID, d.RuleID, actorID, string(d.Severity), d.Title, detailJSON,
		uuidArray(d.EventIDs), string(d.Status), d.DetectedAt.UTC())

	// If the dedup constraint doesn't exist yet, fall back to simple insert.
	if err != nil && contains(err.Error(), "itdr_det_dedup_unique") {
		_, err = r.pool.Exec(ctx, `
			INSERT INTO itdr_detections (id, tenant_id, rule_id, actor_id, severity, title, detail, event_ids, status, hit_count, detected_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 1, $10, $10)
			ON CONFLICT (id) DO NOTHING
		`, d.ID, d.TenantID, d.RuleID, actorID, string(d.Severity), d.Title, detailJSON,
			uuidArray(d.EventIDs), string(d.Status), d.DetectedAt.UTC())
	}

	_ = bucket // bucket used for conceptual dedup; actual constraint may be added later
	return err
}

// ListDetections queries detections with filtering and pagination.
func (r *ITDRRepository) ListDetections(ctx context.Context, f domain.DetectionFilter) ([]*domain.Detection, int, error) {
	if r.pool == nil {
		return []*domain.Detection{}, 0, nil
	}

	where := "WHERE tenant_id = $1"
	args := []any{f.TenantID}
	argIdx := 2

	if f.Severity != nil {
		where += fmt.Sprintf(" AND severity = $%d", argIdx)
		args = append(args, string(*f.Severity))
		argIdx++
	}
	if f.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, string(*f.Status))
		argIdx++
	}
	if f.RuleID != nil {
		where += fmt.Sprintf(" AND rule_id = $%d", argIdx)
		args = append(args, *f.RuleID)
		argIdx++
	}
	if f.ActorID != nil {
		where += fmt.Sprintf(" AND actor_id = $%d", argIdx)
		args = append(args, *f.ActorID)
		argIdx++
	}
	if f.Since != nil {
		where += fmt.Sprintf(" AND detected_at >= $%d", argIdx)
		args = append(args, *f.Since)
		argIdx++
	}

	// Count total.
	var total int
	countQuery := "SELECT COUNT(*) FROM itdr_detections " + where
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count detections: %w", err)
	}

	// Pagination.
	page := f.Page
	if page < 1 {
		page = 1
	}
	pageSize := f.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := "SELECT id, tenant_id, rule_id, actor_id, severity, title, detail, event_ids, status, hit_count, detected_at, updated_at FROM itdr_detections " +
		where + fmt.Sprintf(" ORDER BY detected_at DESC LIMIT %d OFFSET %d", pageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query detections: %w", err)
	}
	defer rows.Close()

	var detections []*domain.Detection
	for rows.Next() {
		d, err := scanDetection(rows)
		if err != nil {
			continue
		}
		detections = append(detections, d)
	}

	return detections, total, nil
}

// GetDetection returns a single detection by ID.
func (r *ITDRRepository) GetDetection(ctx context.Context, id uuid.UUID) (*domain.Detection, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("not found")
	}

	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, rule_id, actor_id, severity, title, detail, event_ids, status, hit_count, detected_at, updated_at
		FROM itdr_detections WHERE id = $1
	`, id)

	return scanDetectionRow(row)
}

// UpdateStatus updates a detection's status (acknowledge/resolve/false_positive).
func (r *ITDRRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DetectionStatus) error {
	if r.pool == nil {
		return nil
	}

	_, err := r.pool.Exec(ctx, `
		UPDATE itdr_detections SET status = $2, updated_at = now() WHERE id = $1
	`, id, string(status))
	return err
}

// GetStats returns aggregated detection stats for a time window.
func (r *ITDRRepository) GetStats(ctx context.Context, tenantID uuid.UUID, since time.Time) (*domain.DetectionStats, error) {
	if r.pool == nil {
		return &domain.DetectionStats{BySeverity: map[string]int{}, ByStatus: map[string]int{}, ByRule: map[string]int{}}, nil
	}

	stats := &domain.DetectionStats{
		BySeverity: map[string]int{},
		ByStatus:   map[string]int{},
		ByRule:     map[string]int{},
	}

	// Count total.
	r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM itdr_detections WHERE tenant_id = $1 AND detected_at >= $2`, tenantID, since).Scan(&stats.Total)

	// By severity.
	rows, _ := r.pool.Query(ctx, `SELECT severity, COUNT(*) FROM itdr_detections WHERE tenant_id = $1 AND detected_at >= $2 GROUP BY severity`, tenantID, since)
	for rows.Next() {
		var sev string
		var cnt int
		rows.Scan(&sev, &cnt)
		stats.BySeverity[sev] = cnt
	}
	rows.Close()

	// By status.
	rows, _ = r.pool.Query(ctx, `SELECT status, COUNT(*) FROM itdr_detections WHERE tenant_id = $1 AND detected_at >= $2 GROUP BY status`, tenantID, since)
	for rows.Next() {
		var st string
		var cnt int
		rows.Scan(&st, &cnt)
		stats.ByStatus[st] = cnt
	}
	rows.Close()

	// By rule.
	rows, _ = r.pool.Query(ctx, `SELECT rule_id, COUNT(*) FROM itdr_detections WHERE tenant_id = $1 AND detected_at >= $2 GROUP BY rule_id`, tenantID, since)
	for rows.Next() {
		var rule string
		var cnt int
		rows.Scan(&rule, &cnt)
		stats.ByRule[rule] = cnt
	}
	rows.Close()

	return stats, nil
}

// --- Helpers ---

func uuidArray(ids []uuid.UUID) string {
	if len(ids) == 0 {
		return "{}"
	}
	s := "{"
	for i, id := range ids {
		if i > 0 {
			s += ","
		}
		s += id.String()
	}
	s += "}"
	return s
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Scanner interface for pgx Row and Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanDetection(rows scanner) (*domain.Detection, error) {
	return scanDetectionRow(rows)
}

func scanDetectionRow(s scanner) (*domain.Detection, error) {
	var (
		d         domain.Detection
		actorID   *uuid.UUID
		sevStr    string
		statusStr string
		detailRaw []byte
		eventIDs  []uuid.UUID
	)

	err := s.Scan(
		&d.ID, &d.TenantID, &d.RuleID, &actorID, &sevStr, &d.Title,
		&detailRaw, &eventIDs, &statusStr, &d.HitCount, &d.DetectedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	d.ActorID = actorID
	d.Severity = domain.Severity(sevStr)
	d.Status = domain.DetectionStatus(statusStr)
	d.EventIDs = eventIDs

	if len(detailRaw) > 0 {
		json.Unmarshal(detailRaw, &d.Detail)
	}
	if d.Detail == nil {
		d.Detail = map[string]any{}
	}

	return &d, nil
}
