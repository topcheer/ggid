package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// oauthMapRepo provides PG persistence for OAuth in-memory stores.
// Covers: branding, client_scopes, dpop_bindings, resource_allow,
// custom_scopes, delegation_chains.
type oauthMapRepo struct {
	pool *pgxpool.Pool
}

func newOAuthMapRepo(pool *pgxpool.Pool) *oauthMapRepo {
	return &oauthMapRepo{pool: pool}
}

// mapRepoVar is the package-level instance set during buildHandler init.
var mapRepoVar *oauthMapRepo

func (r *oauthMapRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS oauth_branding (
			id TEXT PRIMARY KEY, client_id TEXT NOT NULL,
			data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_oauth_branding_client ON oauth_branding(client_id);
		CREATE TABLE IF NOT EXISTS oauth_client_scopes (
			id TEXT PRIMARY KEY, client_id TEXT NOT NULL,
			data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS oauth_dpop_bindings (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS oauth_resource_allow (
			id TEXT PRIMARY KEY, client_id TEXT NOT NULL,
			data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS oauth_custom_scopes (
			id TEXT PRIMARY KEY, scope_name TEXT NOT NULL,
			data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS oauth_delegation_chains (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
	`)
	return err
}

func (r *oauthMapRepo) Store(ctx context.Context, table, id string, data map[string]any) error {
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

func (r *oauthMapRepo) List(ctx context.Context, table string) ([]map[string]any, error) {
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

func (r *oauthMapRepo) Get(ctx context.Context, table, id string) (map[string]any, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("not found")
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

func (r *oauthMapRepo) Delete(ctx context.Context, table, id string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, table), id)
	return err
}

// --- typed helpers ---

func (r *oauthMapRepo) StoreBranding(ctx context.Context, clientID string, data map[string]any) error {
	return r.Store(ctx, "oauth_branding", clientID, data)
}

func (r *oauthMapRepo) GetBranding(ctx context.Context, clientID string) (map[string]any, error) {
	return r.Get(ctx, "oauth_branding", clientID)
}

func (r *oauthMapRepo) StoreCustomScope(ctx context.Context, scopeName string, data map[string]any) error {
	return r.Store(ctx, "oauth_custom_scopes", scopeName, data)
}

func (r *oauthMapRepo) ListCustomScopes(ctx context.Context) ([]map[string]any, error) {
	return r.List(ctx, "oauth_custom_scopes")
}
