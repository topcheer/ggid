package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// identityPolicyMapRepo provides PG persistence for identity-side policy stores.
// Covers: lifecycle_rules_store, review_campaigns_store
type identityPolicyMapRepo struct {
	pool *pgxpool.Pool
}

func newIdentityPolicyMapRepo(pool *pgxpool.Pool) *identityPolicyMapRepo {
	return &identityPolicyMapRepo{pool: pool}
}

func (r *identityPolicyMapRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS lifecycle_rules_store (
			id TEXT PRIMARY KEY, tenant_id UUID,
			data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_lifecycle_rules_tenant ON lifecycle_rules_store(tenant_id);
		CREATE TABLE IF NOT EXISTS review_campaigns_store (
			id TEXT PRIMARY KEY, tenant_id UUID,
			data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_review_campaigns_tenant ON review_campaigns_store(tenant_id);
	`)
	return err
}

func (r *identityPolicyMapRepo) Store(ctx context.Context, table, id string, data map[string]any) error {
	if r.pool == nil {
		return nil
	}
	if id == "" {
		id = uuid.New().String()
	}
	jsonData, _ := json.Marshal(data)
	_, err := r.pool.Exec(ctx, fmt.Sprintf(
		`INSERT INTO %s (id, data, created_at) VALUES ($1, $2, now())
		 ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data`, table), id, jsonData)
	return err
}

func (r *identityPolicyMapRepo) List(ctx context.Context, table string) ([]map[string]any, error) {
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

func (r *identityPolicyMapRepo) Delete(ctx context.Context, table, id string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, table), id)
	return err
}

func (r *identityPolicyMapRepo) Get(ctx context.Context, table, id string) (map[string]any, error) {
	if r.pool == nil {
		return map[string]any{}, nil
	}
	var data []byte
	var created time.Time
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT data, created_at FROM %s WHERE id = $1`, table), id).Scan(&data, &created)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	m["id"] = id
	m["created_at"] = created
	return m, nil
}
