package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// authMemoryMapRepo provides PG persistence for auth in-memory stores.
// Covers: device_bindings, device_trusts, geofence_rules, travel_events, login_flows
type authMemoryMapRepo struct {
	pool *pgxpool.Pool
}

func NewAuthMemoryMapRepo(pool *pgxpool.Pool) *authMemoryMapRepo {
	return &authMemoryMapRepo{pool: pool}
}

func (r *authMemoryMapRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS auth_device_bindings (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, device_id TEXT, device_name TEXT,
			platform TEXT, trusted BOOLEAN DEFAULT FALSE, last_used TIMESTAMPTZ,
			metadata JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_devbind_user ON auth_device_bindings(user_id);
		CREATE TABLE IF NOT EXISTS auth_device_trusts (
			id TEXT PRIMARY KEY, device_id TEXT UNIQUE, user_id TEXT,
			trust_score INT DEFAULT 0, managed BOOLEAN DEFAULT FALSE,
			encrypted BOOLEAN DEFAULT FALSE, compliant_os BOOLEAN DEFAULT FALSE,
			jailbreak BOOLEAN DEFAULT FALSE, metadata JSONB DEFAULT '{}',
			updated_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_devtrust_device ON auth_device_trusts(device_id);
		CREATE TABLE IF NOT EXISTS auth_geofence_rules (
			id TEXT PRIMARY KEY, tenant_id UUID, name TEXT, lat FLOAT, lng FLOAT,
			radius_meters INT DEFAULT 500, action TEXT DEFAULT 'warn',
			enabled BOOLEAN DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_travel_events (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, login_time TIMESTAMPTZ,
			ip TEXT, country TEXT, city TEXT, latitude FLOAT, longitude FLOAT,
			flagged BOOLEAN DEFAULT FALSE, created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_travel_user ON auth_travel_events(user_id, created_at DESC);
		CREATE TABLE IF NOT EXISTS auth_login_flows (
			id TEXT PRIMARY KEY, tenant_id UUID, user_id TEXT, flow_type TEXT,
			step TEXT, status TEXT DEFAULT 'in_progress', ip TEXT, user_agent TEXT,
			metadata JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_loginflow_tenant ON auth_login_flows(tenant_id, created_at DESC);
	`)
	return err
}

func (r *authMemoryMapRepo) StoreJSON(ctx context.Context, table, id string, data map[string]any) error {
	if r.pool == nil {
		return nil
	}
	jsonData, _ := json.Marshal(data)
	_, err := r.pool.Exec(ctx, fmt.Sprintf(
		`INSERT INTO %s (id, data, created_at) VALUES ($1, $2, now())
		 ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data`, table), id, jsonData)
	return err
}

func (r *authMemoryMapRepo) ListJSON(ctx context.Context, table string) ([]map[string]any, error) {
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

func (r *authMemoryMapRepo) DeleteJSON(ctx context.Context, table, id string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, table), id)
	return err
}

// --- Device Binding ---

func (r *authMemoryMapRepo) StoreDeviceBinding(ctx context.Context, d map[string]any) error {
	id, _ := d["id"].(string)
	if id == "" {
		id = uuid.New().String()
		d["id"] = id
	}
	return r.StoreJSON(ctx, "auth_device_bindings", id, d)
}

func (r *authMemoryMapRepo) ListDeviceBindings(ctx context.Context) ([]map[string]any, error) {
	return r.ListJSON(ctx, "auth_device_bindings")
}

func (r *authMemoryMapRepo) DeleteDeviceBinding(ctx context.Context, id string) error {
	return r.DeleteJSON(ctx, "auth_device_bindings", id)
}

// --- Device Trust ---

func (r *authMemoryMapRepo) StoreDeviceTrust(ctx context.Context, d map[string]any) error {
	id, _ := d["id"].(string)
	if id == "" {
		id = uuid.New().String()
		d["id"] = id
	}
	return r.StoreJSON(ctx, "auth_device_trusts", id, d)
}

func (r *authMemoryMapRepo) ListDeviceTrusts(ctx context.Context) ([]map[string]any, error) {
	return r.ListJSON(ctx, "auth_device_trusts")
}

// --- Geofence Rules ---

func (r *authMemoryMapRepo) StoreGeofenceRule(ctx context.Context, d map[string]any) error {
	id, _ := d["id"].(string)
	if id == "" {
		id = uuid.New().String()
		d["id"] = id
	}
	return r.StoreJSON(ctx, "auth_geofence_rules", id, d)
}

func (r *authMemoryMapRepo) ListGeofenceRules(ctx context.Context) ([]map[string]any, error) {
	return r.ListJSON(ctx, "auth_geofence_rules")
}

func (r *authMemoryMapRepo) DeleteGeofenceRule(ctx context.Context, id string) error {
	return r.DeleteJSON(ctx, "auth_geofence_rules", id)
}

// --- Travel Events ---

func (r *authMemoryMapRepo) StoreTravelEvent(ctx context.Context, d map[string]any) error {
	id, _ := d["id"].(string)
	if id == "" {
		id = uuid.New().String()
		d["id"] = id
	}
	return r.StoreJSON(ctx, "auth_travel_events", id, d)
}

func (r *authMemoryMapRepo) ListTravelEvents(ctx context.Context, limit int) ([]map[string]any, error) {
	if r.pool == nil {
		return []map[string]any{}, nil
	}
	if limit <= 0 {
		limit = 500
	}
	rows, err := r.pool.Query(ctx, `SELECT id, user_id, login_time, ip, country, city, latitude, longitude, flagged, created_at FROM auth_travel_events ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]any
	for rows.Next() {
		m := map[string]any{}
		var id, uid, ip, country, city string
		var loginTime, created time.Time
		var lat, lng float64
		var flagged bool
		if err := rows.Scan(&id, &uid, &loginTime, &ip, &country, &city, &lat, &lng, &flagged, &created); err != nil {
			continue
		}
		m["id"] = id
		m["user_id"] = uid
		m["login_time"] = loginTime
		m["ip"] = ip
		m["country"] = country
		m["city"] = city
		m["latitude"] = lat
		m["longitude"] = lng
		m["flagged"] = flagged
		m["created_at"] = created
		result = append(result, m)
	}
	return result, nil
}

// --- Login Flows ---

func (r *authMemoryMapRepo) StoreLoginFlow(ctx context.Context, d map[string]any) error {
	id, _ := d["id"].(string)
	if id == "" {
		id = uuid.New().String()
		d["id"] = id
	}
	return r.StoreJSON(ctx, "auth_login_flows", id, d)
}

func (r *authMemoryMapRepo) ListLoginFlows(ctx context.Context, limit int) ([]map[string]any, error) {
	if r.pool == nil {
		return []map[string]any{}, nil
	}
	if limit <= 0 {
		limit = 1000
	}
	rows, err := r.pool.Query(ctx, `SELECT id, data, created_at FROM auth_login_flows ORDER BY created_at DESC LIMIT $1`, limit)
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
