package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// auditMemoryMapRepo provides PostgreSQL persistence for previously in-memory stores.
// Covers: dashboard_widgets, audit_retention_policies, compliance_evidence,
// evidence_auto_tags, evidence_versions, evidence_attachments.
type auditMemoryMapRepo struct {
	pool *pgxpool.Pool
}

func newAuditMemoryMapRepo(pool *pgxpool.Pool) *auditMemoryMapRepo {
	return &auditMemoryMapRepo{pool: pool}
}

func (r *auditMemoryMapRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS dashboard_widgets (
			id TEXT PRIMARY KEY, tenant_id UUID, title TEXT, type TEXT,
			config JSONB DEFAULT '{}', position INT DEFAULT 0,
			enabled BOOLEAN DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT now()
		);
		-- Task-C: the widget handler uses the generic JSONB helpers which
		-- require a "data" column. Add it to both new and legacy schemas.
		ALTER TABLE dashboard_widgets ADD COLUMN IF NOT EXISTS data JSONB DEFAULT '{}';
		CREATE TABLE IF NOT EXISTS audit_retention_policies (
			id TEXT PRIMARY KEY, tenant_id UUID, name TEXT, description TEXT,
			category TEXT, retention_days INT DEFAULT 90,
			auto_delete BOOLEAN DEFAULT FALSE, enabled BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS compliance_evidence (
			id TEXT PRIMARY KEY, tenant_id UUID, title TEXT, description TEXT,
			category TEXT, source TEXT, collected_at TIMESTAMPTZ,
			status TEXT DEFAULT 'collected', metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS evidence_auto_tags (
			id TEXT PRIMARY KEY, evidence_id TEXT NOT NULL, tag TEXT,
			rule_id TEXT, confidence FLOAT DEFAULT 1.0,
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS evidence_versions (
			id TEXT PRIMARY KEY, evidence_id TEXT NOT NULL, version INT,
			content_hash TEXT, changed_by TEXT, change_note TEXT,
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS evidence_attachments (
			id TEXT PRIMARY KEY, evidence_id TEXT NOT NULL, filename TEXT,
			content_type TEXT, size_bytes BIGINT, storage_path TEXT,
			uploaded_by TEXT, created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS tenant_branding (
			tenant_id UUID PRIMARY KEY, primary_color TEXT DEFAULT '#6366f1',
			logo_url TEXT, custom_domain TEXT, css TEXT,
			updated_at TIMESTAMPTZ DEFAULT now()
		);
	`)
	return err
}

// --- Generic JSONB store helpers for simple maps ---

// StoreJSON stores a JSON-serializable record into a table by ID.
func (r *auditMemoryMapRepo) storeJSON(ctx context.Context, table, id string, data map[string]any) error {
	if r.pool == nil {
		return nil
	}
	jsonData, _ := json.Marshal(data)
	_, err := r.pool.Exec(ctx, fmt.Sprintf(
		`INSERT INTO %s (id, data, created_at) VALUES ($1, $2, now())
		 ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data`, table), id, jsonData)
	return err
}

// ListJSON retrieves all records from a JSONB table.
func (r *auditMemoryMapRepo) listJSON(ctx context.Context, table string) ([]map[string]any, error) {
	if r.pool == nil {
		return []map[string]any{}, nil
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`SELECT id, data, created_at FROM %s ORDER BY created_at DESC`, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]any
	for rows.Next() {
		var id string
		var data []byte
		var created time.Time
		if err := rows.Scan(&id, &data, &created); err != nil {
			continue
		}
		var m map[string]any
		json.Unmarshal(data, &m)
		m["id"] = id
		m["created_at"] = created
		result = append(result, m)
	}
	return result, nil
}

// DeleteJSON removes a record by ID.
func (r *auditMemoryMapRepo) deleteJSON(ctx context.Context, table, id string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, table), id)
	return err
}

// --- Tenant Branding (identity service helper) ---

func (r *auditMemoryMapRepo) GetBranding(ctx context.Context, tenantID uuid.UUID) (map[string]any, error) {
	if r.pool == nil {
		return map[string]any{"primary_color": "#6366f1", "logo_url": "", "custom_domain": "", "css": ""}, nil
	}
	row := r.pool.QueryRow(ctx, `SELECT primary_color, logo_url, custom_domain, css FROM tenant_branding WHERE tenant_id = $1`, tenantID)
	var colors, logo, domain, css string
	if err := row.Scan(&colors, &logo, &domain, &css); err != nil {
		return map[string]any{"primary_color": "#6366f1", "logo_url": "", "custom_domain": "", "css": ""}, nil
	}
	return map[string]any{"primary_color": colors, "logo_url": logo, "custom_domain": domain, "css": css}, nil
}

func (r *auditMemoryMapRepo) UpsertBranding(ctx context.Context, tenantID uuid.UUID, data map[string]any) error {
	if r.pool == nil {
		return nil
	}
	colors, _ := data["primary_color"].(string)
	logo, _ := data["logo_url"].(string)
	domain, _ := data["custom_domain"].(string)
	css, _ := data["css"].(string)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tenant_branding (tenant_id, primary_color, logo_url, custom_domain, css, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (tenant_id) DO UPDATE SET primary_color=EXCLUDED.primary_color, logo_url=EXCLUDED.logo_url, custom_domain=EXCLUDED.custom_domain, css=EXCLUDED.css, updated_at=now()`,
		tenantID, colors, logo, domain, css)
	return err
}
