package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ComplianceMapping represents a single control-to-feature mapping.
type ComplianceMapping struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	Framework     string    `json:"framework"`
	TrustCategory string    `json:"trust_category"`
	ControlID     string    `json:"control_id"`
	ControlName   string    `json:"control_name"`
	GGIDFeature   string    `json:"ggid_feature"`
	Status        string    `json:"status"`
	EvidenceQuery string    `json:"evidence_query"`
	CCMControlID  string    `json:"ccm_control_id"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ComplianceMappingRepository manages compliance_mappings in PostgreSQL.
type ComplianceMappingRepository struct {
	pool *pgxpool.Pool
}

func NewComplianceMappingRepository(pool *pgxpool.Pool) *ComplianceMappingRepository {
	return &ComplianceMappingRepository{pool: pool}
}

// EnsureSchema creates the compliance_mappings table.
func (r *ComplianceMappingRepository) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS compliance_mappings (
			id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id      UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
			framework      TEXT NOT NULL,
			trust_category TEXT NOT NULL DEFAULT '',
			control_id     TEXT NOT NULL,
			control_name   TEXT NOT NULL,
			ggid_feature   TEXT NOT NULL,
			status         TEXT NOT NULL DEFAULT 'covered',
			evidence_query TEXT NOT NULL DEFAULT '',
			ccm_control_id TEXT NOT NULL DEFAULT '',
			description    TEXT NOT NULL DEFAULT '',
			created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (framework, control_id)
		);
		CREATE INDEX IF NOT EXISTS idx_compliance_framework
			ON compliance_mappings (framework, control_id);
		CREATE INDEX IF NOT EXISTS idx_compliance_tenant
			ON compliance_mappings (tenant_id, framework);
	`)
	return err
}

// ListByFramework returns all mappings for a given framework.
func (r *ComplianceMappingRepository) ListByFramework(ctx context.Context, tenantID uuid.UUID, framework string) ([]ComplianceMapping, error) {
	if r.pool == nil {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, framework, trust_category, control_id, control_name,
		       ggid_feature, status, evidence_query, ccm_control_id, description,
		       created_at, updated_at
		FROM compliance_mappings
		WHERE framework = $1 AND (tenant_id = $2 OR tenant_id = '00000000-0000-0000-0000-000000000001')
		ORDER BY control_id
	`, framework, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ComplianceMapping
	for rows.Next() {
		var m ComplianceMapping
		if err := rows.Scan(&m.ID, &m.TenantID, &m.Framework, &m.TrustCategory,
			&m.ControlID, &m.ControlName, &m.GGIDFeature, &m.Status,
			&m.EvidenceQuery, &m.CCMControlID, &m.Description,
			&m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, nil
}

// ListAll returns mappings grouped by framework.
func (r *ComplianceMappingRepository) ListAll(ctx context.Context, tenantID uuid.UUID) ([]ComplianceMapping, error) {
	if r.pool == nil {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, framework, trust_category, control_id, control_name,
		       ggid_feature, status, evidence_query, ccm_control_id, description,
		       created_at, updated_at
		FROM compliance_mappings
		WHERE tenant_id = $1 OR tenant_id = '00000000-0000-0000-0000-000000000001'
		ORDER BY framework, control_id
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ComplianceMapping
	for rows.Next() {
		var m ComplianceMapping
		if err := rows.Scan(&m.ID, &m.TenantID, &m.Framework, &m.TrustCategory,
			&m.ControlID, &m.ControlName, &m.GGIDFeature, &m.Status,
			&m.EvidenceQuery, &m.CCMControlID, &m.Description,
			&m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, nil
}
