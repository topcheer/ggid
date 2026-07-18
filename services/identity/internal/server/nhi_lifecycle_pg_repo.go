package server

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NHIPGRepo manages nhi_identities persistence in PostgreSQL.
// Already existed for inventory; this adds lifecycle operations.
type NHILifecyclePGRepo struct {
	pool *pgxpool.Pool
}

func NewNHILifecyclePGRepo(pool *pgxpool.Pool) *NHILifecyclePGRepo {
	return &NHILifecyclePGRepo{pool: pool}
}

// EnsureSchema creates the nhi_identities table (idempotent).
func (r *NHILifecyclePGRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS nhi_identities (
			id          TEXT PRIMARY KEY,
			tenant_id   UUID,
			name        TEXT NOT NULL,
			type        TEXT NOT NULL DEFAULT 'service_account',
			status      TEXT NOT NULL DEFAULT 'active',
			last_used   TIMESTAMPTZ,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			metadata    JSONB NOT NULL DEFAULT '{}'
		);
		CREATE INDEX IF NOT EXISTS idx_nhi_tenant ON nhi_identities(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_nhi_status ON nhi_identities(status);
	`)
	return err
}

// Register inserts or updates an NHI identity.
func (r *NHILifecyclePGRepo) Register(ctx context.Context, id string, tenantID uuid.UUID, name, nhiType string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO nhi_identities (id, tenant_id, name, type, status)
		VALUES ($1, $2, $3, $4, 'active')
		ON CONFLICT (id) DO UPDATE SET name = $3, type = $4`,
		id, tenantID, name, nhiType,
	)
	return err
}

// List returns all NHI identities.
func (r *NHILifecyclePGRepo) List(ctx context.Context, tenantID uuid.UUID) ([]map[string]any, error) {
	if r.pool == nil {
		return []map[string]any{}, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, type, status, last_used, created_at FROM nhi_identities WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var id, name, typ, status string
		var lastUsed, createdAt *string
		if err := rows.Scan(&id, &name, &typ, &status, &lastUsed, &createdAt); err != nil {
			continue
		}
		result = append(result, map[string]any{
			"id": id, "name": name, "type": typ, "status": status,
		})
	}
	return result, nil
}

// Decommission marks an NHI as decommissioned.
func (r *NHILifecyclePGRepo) Decommission(ctx context.Context, id string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `UPDATE nhi_identities SET status = 'decommissioned' WHERE id = $1`, id)
	return err
}
