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

		-- Batch 5: OTP, passkey, biometric, credential vault, trusted devices, session limits, session risks,
		-- device fingerprints, encryption keys, breach notifs, passwordless sessions, impersonation
		CREATE TABLE IF NOT EXISTS auth_otp_entries (
			code TEXT PRIMARY KEY, email TEXT NOT NULL, tenant_id TEXT,
			hashed_code TEXT, attempts INT DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT now(), expires_at TIMESTAMPTZ NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_otp_email ON auth_otp_entries(email);
		CREATE TABLE IF NOT EXISTS auth_otp_sendlog (
			id SERIAL PRIMARY KEY, email TEXT NOT NULL, sent_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_otp_sendlog_email ON auth_otp_sendlog(email, sent_at DESC);

		CREATE TABLE IF NOT EXISTS auth_passkey_sessions (
			id TEXT PRIMARY KEY, session_type TEXT NOT NULL,
			user_id TEXT, challenge TEXT, tenant_id TEXT,
			created_at TIMESTAMPTZ DEFAULT now(), expires_at TIMESTAMPTZ NOT NULL
		);
		CREATE TABLE IF NOT EXISTS auth_passkey_credentials (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, credential_id TEXT,
			public_key TEXT, device_type TEXT, tenant_id TEXT,
			created_at TIMESTAMPTZ DEFAULT now(), last_used TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS idx_passkey_cred_user ON auth_passkey_credentials(user_id);

		CREATE TABLE IF NOT EXISTS auth_biometrics (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL,
			template_hash TEXT, device_id TEXT, algorithm TEXT,
			enabled BOOLEAN DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_biometrics_user ON auth_biometrics(user_id);

		CREATE TABLE IF NOT EXISTS auth_credential_vault (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, cred_key TEXT NOT NULL,
			encrypted_value TEXT, cred_type TEXT,
			created_at TIMESTAMPTZ DEFAULT now(), updated_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_credvault_user ON auth_credential_vault(user_id, cred_key);

		CREATE TABLE IF NOT EXISTS auth_trusted_devices (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, device_id TEXT,
			trust_level TEXT DEFAULT 'trusted', fingerprint TEXT,
			created_at TIMESTAMPTZ DEFAULT now(), last_seen TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS idx_trusted_dev_user ON auth_trusted_devices(user_id);

		CREATE TABLE IF NOT EXISTS auth_session_limits (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, tenant_id TEXT,
			max_sessions INT DEFAULT 5, current_sessions INT DEFAULT 0,
			updated_at TIMESTAMPTZ DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS auth_session_risks (
			id TEXT PRIMARY KEY, session_id TEXT NOT NULL, user_id TEXT,
			risk_score REAL DEFAULT 0, decision TEXT,
			evaluated_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_session_risk_user ON auth_session_risks(user_id, evaluated_at DESC);

		CREATE TABLE IF NOT EXISTS auth_device_fingerprints (
			id TEXT PRIMARY KEY, fingerprint TEXT NOT NULL,
			user_id TEXT, device_info JSONB DEFAULT '{}',
			first_seen TIMESTAMPTZ DEFAULT now(), last_seen TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS idx_devfp_hash ON auth_device_fingerprints(fingerprint);

		CREATE TABLE IF NOT EXISTS auth_encryption_keys (
			id TEXT PRIMARY KEY, key_name TEXT NOT NULL,
			encrypted_key BYTEA, algorithm TEXT DEFAULT 'AES-256-GCM',
			rotation_count INT DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT now(), rotated_at TIMESTAMPTZ
		);

		CREATE TABLE IF NOT EXISTS auth_breach_notifications (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL,
			breach_name TEXT, severity TEXT DEFAULT 'medium',
			notified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_breach_notif_user ON auth_breach_notifications(user_id);

		CREATE TABLE IF NOT EXISTS auth_passwordless_sessions (
			id TEXT PRIMARY KEY, user_id TEXT,
			challenge TEXT, status TEXT DEFAULT 'pending',
			created_at TIMESTAMPTZ DEFAULT now(), expires_at TIMESTAMPTZ NOT NULL
		);

		CREATE TABLE IF NOT EXISTS auth_impersonation_jti (
			jti TEXT PRIMARY KEY, impersonator TEXT NOT NULL,
			target TEXT NOT NULL, expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS auth_breach_warnings (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL,
			source TEXT, severity TEXT DEFAULT 'medium',
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_breach_warn_user ON auth_breach_warnings(user_id);

		CREATE TABLE IF NOT EXISTS auth_session_anomalies (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL,
			anomaly_type TEXT, detail JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_session_anom_user ON auth_session_anomalies(user_id);

		CREATE TABLE IF NOT EXISTS auth_credential_stuffing (
			ip TEXT PRIMARY KEY, attempt_count INT DEFAULT 0,
			blocked BOOLEAN DEFAULT FALSE,
			first_seen TIMESTAMPTZ DEFAULT now(), last_seen TIMESTAMPTZ
		);

		CREATE TABLE IF NOT EXISTS auth_throttle_states (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL,
			state TEXT DEFAULT 'normal', retry_after TIMESTAMPTZ,
			updated_at TIMESTAMPTZ DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS auth_velocity_rules (
			id TEXT PRIMARY KEY, tenant_id TEXT,
			rule_name TEXT, max_events INT DEFAULT 100,
			window_seconds INT DEFAULT 3600, action TEXT DEFAULT 'block',
			enabled BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS auth_dlp_policies (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS auth_pwd_reset_tokens (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS auth_login_notify_configs (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS auth_adaptive_mfa_decisions (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT now()
		);

		-- Generic JSON tables for write-through handlers (StoreJSON/ListJSON/GetJSON)
		CREATE TABLE IF NOT EXISTS auth_otp_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_passkey_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_biometric_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_credvault_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_trusted_devices_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_session_limits_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_session_risks_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_device_fingerprints_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_encryption_keys_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_breach_notifs_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_passwordless_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_breach_warnings_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_session_anomalies_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_cred_stuffing_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS auth_throttle_states_json (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
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

// GetJSON retrieves a single row by ID from a generic JSON table.
func (r *authMemoryMapRepo) GetJSON(ctx context.Context, table, id string) (map[string]any, error) {
	if r.pool == nil {
		return nil, nil
	}
	var data []byte
	var created time.Time
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT data, created_at FROM %s WHERE id = $1`, table), id).Scan(&data, &created)
	if err != nil {
		return nil, nil
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	m["id"] = id
	m["created_at"] = created
	return m, nil
}

// CleanupExpired deletes rows past their expires_at timestamp.
func (r *authMemoryMapRepo) CleanupExpired(ctx context.Context) (int64, error) {
	if r.pool == nil {
		return 0, nil
	}
	var total int64
	for _, table := range []string{"auth_otp_entries", "auth_passkey_sessions", "auth_passwordless_sessions", "auth_impersonation_jti"} {
		ct, err := r.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE expires_at IS NOT NULL AND expires_at < now()`, table))
		if err == nil {
			total += ct.RowsAffected()
		}
	}
	return total, nil
}
