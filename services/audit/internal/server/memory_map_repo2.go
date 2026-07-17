package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// auditMemoryMapRepo2 provides PG persistence for the remaining audit stores.
// Covers: evidence_integrity, webhook_deliveries, dsr_requests, collect_schedules, event_dedup
// Also provides generic JSONB helpers reused by auth.
type auditMemoryMapRepo2 struct {
	pool *pgxpool.Pool
}

func NewAuditMemoryMapRepo2(pool *pgxpool.Pool) *auditMemoryMapRepo2 {
	return &auditMemoryMapRepo2{pool: pool}
}

func (r *auditMemoryMapRepo2) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS evidence_integrity (
			id TEXT PRIMARY KEY, evidence_id TEXT NOT NULL, hash TEXT,
			algorithm TEXT DEFAULT 'sha256', verified BOOLEAN DEFAULT FALSE,
			verified_by TEXT, created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS webhook_deliveries (
			id TEXT PRIMARY KEY, tenant_id UUID, webhook_id TEXT, event_type TEXT,
			payload JSONB DEFAULT '{}', status TEXT DEFAULT 'pending',
			response_code INT, attempts INT DEFAULT 0, error TEXT,
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS dsr_requests (
			id TEXT PRIMARY KEY, tenant_id UUID, user_id TEXT,
			request_type TEXT DEFAULT 'access', status TEXT DEFAULT 'pending',
			details JSONB DEFAULT '{}', completed_at TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS collect_schedules (
			id TEXT PRIMARY KEY, tenant_id UUID, name TEXT, source TEXT,
			interval_minutes INT DEFAULT 60, enabled BOOLEAN DEFAULT TRUE,
			last_run TIMESTAMPTZ, config JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS event_dedup (
			id TEXT PRIMARY KEY, event_hash TEXT UNIQUE,
			first_seen TIMESTAMPTZ DEFAULT now(), last_seen TIMESTAMPTZ DEFAULT now(), seen_count INT DEFAULT 1
		);
		CREATE TABLE IF NOT EXISTS audit_incidents (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS audit_event_subscriptions (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS audit_reports (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS audit_sig_records (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
	`)
	return err
}

// --- Generic JSONB helpers (also used by auth repo) ---

func (r *auditMemoryMapRepo2) StoreJSON(ctx context.Context, table, id string, data map[string]any) error {
	if r.pool == nil {
		return nil
	}
	jsonData, _ := json.Marshal(data)
	_, err := r.pool.Exec(ctx, fmt.Sprintf(
		`INSERT INTO %s (id, data, created_at) VALUES ($1, $2, now())
		 ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data`, table), id, jsonData)
	return err
}

func (r *auditMemoryMapRepo2) ListJSON(ctx context.Context, table string) ([]map[string]any, error) {
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

func (r *auditMemoryMapRepo2) DeleteJSON(ctx context.Context, table, id string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, table), id)
	return err
}

// --- Typed helpers: Webhook Delivery ---

func (r *auditMemoryMapRepo2) StoreWebhookDelivery(ctx context.Context, d map[string]any) error {
	if r.pool == nil {
		return nil
	}
	id, _ := d["id"].(string)
	if id == "" {
		id = uuid.New().String()
		d["id"] = id
	}
	return r.StoreJSON(ctx, "webhook_deliveries", id, d)
}

func (r *auditMemoryMapRepo2) ListWebhookDeliveries(ctx context.Context) ([]map[string]any, error) {
	return r.ListJSON(ctx, "webhook_deliveries")
}

func (r *auditMemoryMapRepo2) DeleteWebhookDelivery(ctx context.Context, id string) error {
	return r.DeleteJSON(ctx, "webhook_deliveries", id)
}

// --- Typed helpers: DSR Request ---

func (r *auditMemoryMapRepo2) StoreDSR(ctx context.Context, d map[string]any) error {
	if r.pool == nil {
		return nil
	}
	id, _ := d["id"].(string)
	if id == "" {
		id = uuid.New().String()
		d["id"] = id
	}
	return r.StoreJSON(ctx, "dsr_requests", id, d)
}

func (r *auditMemoryMapRepo2) ListDSRs(ctx context.Context) ([]map[string]any, error) {
	return r.ListJSON(ctx, "dsr_requests")
}

// --- Typed helpers: Integrity Record ---

func (r *auditMemoryMapRepo2) StoreIntegrity(ctx context.Context, evidenceID, hash string, verified bool, verifiedBy string) error {
	if r.pool == nil {
		return nil
	}
	id := uuid.New().String()
	_, err := r.pool.Exec(ctx, `INSERT INTO evidence_integrity (id, evidence_id, hash, verified, verified_by) VALUES ($1,$2,$3,$4,$5)`,
		id, evidenceID, hash, verified, verifiedBy)
	return err
}

func (r *auditMemoryMapRepo2) ListIntegrityByEvidence(ctx context.Context, evidenceID string) ([]map[string]any, error) {
	if r.pool == nil {
		return []map[string]any{}, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id, evidence_id, hash, algorithm, verified, verified_by, created_at FROM evidence_integrity WHERE evidence_id=$1 ORDER BY created_at DESC`, evidenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]any
	for rows.Next() {
		m := map[string]any{}
		var id, evID, hash, algo, vb string
		var verified bool
		var created time.Time
		if err := rows.Scan(&id, &evID, &hash, &algo, &verified, &vb, &created); err != nil {
			continue
		}
		m["id"] = id
		m["evidence_id"] = evID
		m["hash"] = hash
		m["algorithm"] = algo
		m["verified"] = verified
		m["verified_by"] = vb
		m["created_at"] = created
		result = append(result, m)
	}
	return result, nil
}

// --- Typed helpers: Collect Schedule ---

func (r *auditMemoryMapRepo2) StoreSchedule(ctx context.Context, d map[string]any) error {
	id, _ := d["id"].(string)
	if id == "" {
		id = uuid.New().String()
		d["id"] = id
	}
	return r.StoreJSON(ctx, "collect_schedules", id, d)
}

func (r *auditMemoryMapRepo2) ListSchedules(ctx context.Context) ([]map[string]any, error) {
	return r.ListJSON(ctx, "collect_schedules")
}

// --- Typed helpers: Event Dedup ---

func (r *auditMemoryMapRepo2) CheckAndStoreDedup(ctx context.Context, eventHash string) (bool, error) {
	if r.pool == nil {
		return false, nil
	}
	id := uuid.New().String()
	_, err := r.pool.Exec(ctx, `INSERT INTO event_dedup (id, event_hash) VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, eventHash)
	if err == nil {
		return true, nil // new event
	}
	return false, nil // duplicate
}
