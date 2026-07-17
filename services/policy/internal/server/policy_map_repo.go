package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// policyMapRepo provides PG persistence for policy-side in-memory stores.
// Covers: conditional_access_store, access_requests_store,
// access_optimization_store, auto_assignments_store
type policyMapRepo struct {
	pool *pgxpool.Pool
}

func NewPolicyMapRepo(pool *pgxpool.Pool) *policyMapRepo {
	return &policyMapRepo{pool: pool}
}

func (r *policyMapRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS conditional_access_store (
			id TEXT PRIMARY KEY, tenant_id UUID,
			data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_cond_access_tenant ON conditional_access_store(tenant_id);
		CREATE TABLE IF NOT EXISTS access_requests_store (
			id TEXT PRIMARY KEY, tenant_id UUID,
			data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_access_req_tenant ON access_requests_store(tenant_id);
		CREATE TABLE IF NOT EXISTS access_optimization_store (
			id TEXT PRIMARY KEY, tenant_id UUID,
			data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auto_assignments_store (
			id TEXT PRIMARY KEY, tenant_id UUID,
			data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
	`)
	return err
}

func (r *policyMapRepo) Store(ctx context.Context, table, id string, data map[string]any) error {
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

func (r *policyMapRepo) List(ctx context.Context, table string) ([]map[string]any, error) {
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

func (r *policyMapRepo) Delete(ctx context.Context, table, id string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, table), id)
	return err
}

func (r *policyMapRepo) Get(ctx context.Context, table, id string) (map[string]any, error) {
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
