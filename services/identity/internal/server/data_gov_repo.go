package server

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DataClassification represents a classification label on a data resource.
type DataClassification struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ResourceType   string    `json:"resource_type"`
	ResourceID     string    `json:"resource_id"`
	Classification string    `json:"classification"` // general, important, core
	Category       string    `json:"category,omitempty"`
	LawfulBasis    string    `json:"lawful_basis,omitempty"`
	RetentionDays  *int      `json:"retention_days,omitempty"`
	CrossBorder    string    `json:"cross_border"` // allowed, restricted, prohibited
	MaskRule       string    `json:"mask_rule"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// DSRRequest represents a Data Subject Rights request.
type DSRRequest struct {
	ID          uuid.UUID      `json:"id"`
	TenantID    uuid.UUID      `json:"tenant_id"`
	UserID      uuid.UUID      `json:"user_id"`
	RequestType string         `json:"request_type"`
	Status      string         `json:"status"`
	Details     map[string]any `json:"details,omitempty"`
	HandledBy   *uuid.UUID     `json:"handled_by,omitempty"`
	HandledAt   *time.Time     `json:"handled_at,omitempty"`
	ResultData  map[string]any `json:"result_data,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// dataGovRepo manages data classification and DSR persistence.
type dataGovRepo struct {
	pool *pgxpool.Pool
}

func newDataGovRepo(pool *pgxpool.Pool) *dataGovRepo {
	return &dataGovRepo{pool: pool}
}

func (r *dataGovRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS data_classifications (
			id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id       UUID NOT NULL,
			resource_type   TEXT NOT NULL,
			resource_id     TEXT NOT NULL,
			classification  TEXT NOT NULL DEFAULT 'general',
			category        TEXT,
			lawful_basis    TEXT,
			retention_days  INT,
			cross_border    TEXT NOT NULL DEFAULT 'allowed',
			mask_rule       TEXT DEFAULT 'none',
			created_by      UUID,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE(tenant_id, resource_type, resource_id)
		);
		CREATE INDEX IF NOT EXISTS idx_data_class_lookup ON data_classifications(tenant_id, resource_type, resource_id);
		CREATE TABLE IF NOT EXISTS dsr_requests (
			id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id     UUID NOT NULL,
			user_id       UUID NOT NULL,
			request_type  TEXT NOT NULL,
			status        TEXT NOT NULL DEFAULT 'pending',
			details       JSONB DEFAULT '{}',
			handled_by    UUID,
			handled_at    TIMESTAMPTZ,
			result_data   JSONB,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_dsr_tenant ON dsr_requests(tenant_id, status, created_at DESC);
	`)
	return err
}

// CreateClassification stores or updates a classification label.
func (r *dataGovRepo) CreateClassification(ctx context.Context, dc *DataClassification) error {
	if r.pool == nil {
		return nil
	}
	if dc.ID == uuid.Nil {
		dc.ID = uuid.New()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO data_classifications (id, tenant_id, resource_type, resource_id, classification, category, lawful_basis, retention_days, cross_border, mask_rule)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tenant_id, resource_type, resource_id) DO UPDATE SET
			classification = EXCLUDED.classification, category = EXCLUDED.category,
			lawful_basis = EXCLUDED.lawful_basis, retention_days = EXCLUDED.retention_days,
			cross_border = EXCLUDED.cross_border, mask_rule = EXCLUDED.mask_rule, updated_at = now()`,
		dc.ID, dc.TenantID, dc.ResourceType, dc.ResourceID, dc.Classification,
		dc.Category, dc.LawfulBasis, dc.RetentionDays, dc.CrossBorder, dc.MaskRule,
	)
	return err
}

// ListClassifications returns all classifications for a tenant.
func (r *dataGovRepo) ListClassifications(ctx context.Context, tenantID uuid.UUID) ([]*DataClassification, error) {
	if r.pool == nil {
		return []*DataClassification{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, resource_type, resource_id, classification, category, lawful_basis, retention_days, cross_border, mask_rule, created_at, updated_at
		FROM data_classifications WHERE tenant_id = $1 ORDER BY resource_type, resource_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*DataClassification
	for rows.Next() {
		var dc DataClassification
		if err := rows.Scan(&dc.ID, &dc.TenantID, &dc.ResourceType, &dc.ResourceID, &dc.Classification, &dc.Category, &dc.LawfulBasis, &dc.RetentionDays, &dc.CrossBorder, &dc.MaskRule, &dc.CreatedAt, &dc.UpdatedAt); err != nil {
			continue
		}
		result = append(result, &dc)
	}
	return result, nil
}

// LookupClassification finds a classification for a specific resource.
func (r *dataGovRepo) LookupClassification(ctx context.Context, tenantID uuid.UUID, resType, resID string) (*DataClassification, error) {
	if r.pool == nil {
		return nil, nil
	}
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, resource_type, resource_id, classification, category, lawful_basis, retention_days, cross_border, mask_rule, created_at, updated_at
		FROM data_classifications WHERE tenant_id = $1 AND resource_type = $2 AND resource_id = $3`, tenantID, resType, resID)
	var dc DataClassification
	if err := row.Scan(&dc.ID, &dc.TenantID, &dc.ResourceType, &dc.ResourceID, &dc.Classification, &dc.Category, &dc.LawfulBasis, &dc.RetentionDays, &dc.CrossBorder, &dc.MaskRule, &dc.CreatedAt, &dc.UpdatedAt); err != nil {
		return nil, nil
	}
	return &dc, nil
}

// DeleteClassification removes a classification.
func (r *dataGovRepo) DeleteClassification(ctx context.Context, id, tenantID uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM data_classifications WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

// CreateDSR creates a new DSR request.
func (r *dataGovRepo) CreateDSR(ctx context.Context, req *DSRRequest) error {
	if r.pool == nil {
		return nil
	}
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	detailsJSON, _ := json.Marshal(req.Details)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO dsr_requests (id, tenant_id, user_id, request_type, status, details)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		req.ID, req.TenantID, req.UserID, req.RequestType, req.Status, detailsJSON)
	return err
}

// ListDSR returns DSR requests for a tenant.
func (r *dataGovRepo) ListDSR(ctx context.Context, tenantID uuid.UUID, status string) ([]*DSRRequest, error) {
	if r.pool == nil {
		return []*DSRRequest{}, nil
	}
	query := `SELECT id, tenant_id, user_id, request_type, status, details, handled_by, handled_at, result_data, created_at
		FROM dsr_requests WHERE tenant_id = $1`
	args := []any{tenantID}
	if status != "" {
		query += ` AND status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC LIMIT 50`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*DSRRequest
	for rows.Next() {
		var req DSRRequest
		var detailsJSON, resultJSON []byte
		if err := rows.Scan(&req.ID, &req.TenantID, &req.UserID, &req.RequestType, &req.Status, &detailsJSON, &req.HandledBy, &req.HandledAt, &resultJSON, &req.CreatedAt); err != nil {
			continue
		}
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &req.Details)
		}
		if len(resultJSON) > 0 {
			json.Unmarshal(resultJSON, &req.ResultData)
		}
		result = append(result, &req)
	}
	return result, nil
}

// UpdateDSRStatus updates a DSR request's status.
func (r *dataGovRepo) UpdateDSRStatus(ctx context.Context, id, handlerID uuid.UUID, status string, result map[string]any) error {
	if r.pool == nil {
		return nil
	}
	resultJSON, _ := json.Marshal(result)
	_, err := r.pool.Exec(ctx, `
		UPDATE dsr_requests SET status = $3, handled_by = $4, handled_at = now(), result_data = $5, updated_at = now()
		WHERE id = $1 AND status = 'pending'`, id, nil, status, handlerID, resultJSON)
	return err
}

// MaskValue applies a masking rule to a string value.
func MaskValue(value string, rule string) string {
	switch rule {
	case "full_mask":
		return strings.Repeat("*", len(value))
	case "partial_mask":
		if len(value) <= 4 {
			return "****"
		}
		return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
	default:
		return value
	}
}
