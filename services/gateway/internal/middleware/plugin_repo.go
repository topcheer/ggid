package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PluginRecord represents a WASM plugin stored in PostgreSQL.
type PluginRecord struct {
	ID          uuid.UUID      `json:"id"`
	TenantID    uuid.UUID      `json:"tenant_id"`
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Author      string         `json:"author"`
	Description string         `json:"description"`
	WasmPath    string         `json:"wasm_path"`
	WasmHash    string         `json:"wasm_hash"`
	Signature   string         `json:"signature"`
	Config      map[string]any `json:"config"`
	Hooks       []string       `json:"hooks"`
	Enabled     bool           `json:"enabled"`
	MaxMemoryMB int            `json:"max_memory_mb"`
	TimeoutMs   int            `json:"timeout_ms"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// HookBinding represents a plugin-to-hook binding.
type HookBinding struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	PluginID  uuid.UUID `json:"plugin_id"`
	HookName  string    `json:"hook_name"`
	Priority  int       `json:"priority"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// pluginRepo manages WASM plugin records in PostgreSQL.
type pluginRepo struct {
	pool *pgxpool.Pool
}

// NewPluginRepo creates a new plugin repository.
func NewPluginRepo(pool *pgxpool.Pool) *pluginRepo {
	return &pluginRepo{pool: pool}
}

func (r *pluginRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS wasm_plugins (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL, name TEXT NOT NULL UNIQUE,
			version TEXT DEFAULT '1.0.0', author TEXT DEFAULT '', description TEXT DEFAULT '',
			wasm_path TEXT NOT NULL, wasm_hash TEXT DEFAULT '', signature TEXT DEFAULT '',
			config JSONB DEFAULT '{}', hooks TEXT[] DEFAULT '{}',
			enabled BOOLEAN DEFAULT FALSE, max_memory_mb INT DEFAULT 16,
			timeout_ms INT DEFAULT 100,
			created_at TIMESTAMPTZ DEFAULT now(), updated_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_wasm_plugins_tenant ON wasm_plugins(tenant_id, enabled);
		CREATE INDEX IF NOT EXISTS idx_wasm_plugins_name ON wasm_plugins(name);
		CREATE TABLE IF NOT EXISTS wasm_plugin_hook_bindings (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			plugin_id UUID NOT NULL REFERENCES wasm_plugins(id) ON DELETE CASCADE,
			hook_name TEXT NOT NULL, priority INT DEFAULT 100,
			enabled BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_hook_bindings_tenant ON wasm_plugin_hook_bindings(tenant_id, hook_name, enabled);
		CREATE INDEX IF NOT EXISTS idx_hook_bindings_plugin ON wasm_plugin_hook_bindings(plugin_id);
	`)
	return err
}

func (r *pluginRepo) Create(ctx context.Context, p *PluginRecord) error {
	if r.pool == nil {
		return nil
	}
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	cfgJSON, _ := json.Marshal(p.Config)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO wasm_plugins (id,tenant_id,name,version,author,description,wasm_path,wasm_hash,signature,config,hooks,enabled,max_memory_mb,timeout_ms)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		p.ID, p.TenantID, p.Name, p.Version, p.Author, p.Description,
		p.WasmPath, p.WasmHash, p.Signature, cfgJSON, p.Hooks, p.Enabled,
		p.MaxMemoryMB, p.TimeoutMs)
	return err
}

func (r *pluginRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*PluginRecord, error) {
	if r.pool == nil {
		return []*PluginRecord{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id,tenant_id,name,version,author,description,wasm_path,wasm_hash,signature,config,hooks,enabled,max_memory_mb,timeout_ms,created_at,updated_at
		FROM wasm_plugins WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*PluginRecord
	for rows.Next() {
		p, err := scanPluginRow(rows)
		if err != nil {
			continue
		}
		result = append(result, p)
	}
	return result, nil
}

func (r *pluginRepo) GetByName(ctx context.Context, name string) (*PluginRecord, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("no pool")
	}
	row := r.pool.QueryRow(ctx, `
		SELECT id,tenant_id,name,version,author,description,wasm_path,wasm_hash,signature,config,hooks,enabled,max_memory_mb,timeout_ms,created_at,updated_at
		FROM wasm_plugins WHERE name=$1`, name)
	return scanPluginRowSingle(row)
}

func (r *pluginRepo) Update(ctx context.Context, p *PluginRecord) error {
	if r.pool == nil {
		return nil
	}
	cfgJSON, _ := json.Marshal(p.Config)
	_, err := r.pool.Exec(ctx, `
		UPDATE wasm_plugins SET version=$2,author=$3,description=$4,config=$5,hooks=$6,enabled=$7,max_memory_mb=$8,timeout_ms=$9,updated_at=now()
		WHERE id=$1`,
		p.ID, p.Version, p.Author, p.Description, cfgJSON, p.Hooks, p.Enabled, p.MaxMemoryMB, p.TimeoutMs)
	return err
}

func (r *pluginRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM wasm_plugins WHERE id=$1`, id)
	return err
}

func (r *pluginRepo) SetEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `UPDATE wasm_plugins SET enabled=$2,updated_at=now() WHERE id=$1`, id, enabled)
	return err
}

// --- Hook Bindings ---

func (r *pluginRepo) CreateHookBinding(ctx context.Context, hb *HookBinding) error {
	if r.pool == nil {
		return nil
	}
	if hb.ID == uuid.Nil {
		hb.ID = uuid.New()
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO wasm_plugin_hook_bindings (id,tenant_id,plugin_id,hook_name,priority,enabled) VALUES ($1,$2,$3,$4,$5,$6)`,
		hb.ID, hb.TenantID, hb.PluginID, hb.HookName, hb.Priority, hb.Enabled)
	return err
}

func (r *pluginRepo) ListHookBindings(ctx context.Context, tenantID uuid.UUID, hookName string) ([]*HookBinding, error) {
	if r.pool == nil {
		return []*HookBinding{}, nil
	}
	q := `SELECT id,tenant_id,plugin_id,hook_name,priority,enabled,created_at FROM wasm_plugin_hook_bindings WHERE tenant_id=$1 AND enabled=TRUE`
	args := []any{tenantID}
	if hookName != "" {
		q += ` AND hook_name=$2 ORDER BY priority ASC`
		args = append(args, hookName)
	} else {
		q += ` ORDER BY priority ASC`
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*HookBinding
	for rows.Next() {
		var hb HookBinding
		if err := rows.Scan(&hb.ID, &hb.TenantID, &hb.PluginID, &hb.HookName, &hb.Priority, &hb.Enabled, &hb.CreatedAt); err != nil {
			continue
		}
		result = append(result, &hb)
	}
	return result, nil
}

// --- scan helpers ---

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPluginRow(rows interface{ Scan(...any) error; Next() bool }) (*PluginRecord, error) {
	p := &PluginRecord{}
	var cfgJSON []byte
	if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Version, &p.Author, &p.Description,
		&p.WasmPath, &p.WasmHash, &p.Signature, &cfgJSON, &p.Hooks, &p.Enabled,
		&p.MaxMemoryMB, &p.TimeoutMs, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal(cfgJSON, &p.Config)
	return p, nil
}

func scanPluginRowSingle(row rowScanner) (*PluginRecord, error) {
	p := &PluginRecord{}
	var cfgJSON []byte
	if err := row.Scan(&p.ID, &p.TenantID, &p.Name, &p.Version, &p.Author, &p.Description,
		&p.WasmPath, &p.WasmHash, &p.Signature, &cfgJSON, &p.Hooks, &p.Enabled,
		&p.MaxMemoryMB, &p.TimeoutMs, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal(cfgJSON, &p.Config)
	return p, nil
}
