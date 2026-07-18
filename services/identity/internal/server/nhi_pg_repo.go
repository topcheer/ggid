package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NHIRiskPGRepo persists NHI risk data to PostgreSQL.
type NHIRiskPGRepo struct {
	pool *pgxpool.Pool
}

// NewNHIRiskPGRepo creates a new PG-backed NHI risk repo (exported).
func NewNHIRiskPGRepo(pool *pgxpool.Pool) *NHIRiskPGRepo {
	return newNHIRiskPGRepo(pool)
}

func newNHIRiskPGRepo(pool *pgxpool.Pool) *NHIRiskPGRepo {
	return &NHIRiskPGRepo{pool: pool}
}

func (r *NHIRiskPGRepo) EnsureSchema(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS nhi_risk_scores (
			nhi_id       UUID NOT NULL,
			score        INTEGER NOT NULL DEFAULT 0,
			level        TEXT NOT NULL DEFAULT 'low',
			signals      JSONB DEFAULT '{}'::jsonb,
			evaluated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (nhi_id, evaluated_at)
		);
		CREATE INDEX IF NOT EXISTS idx_nhi_risk_nhi ON nhi_risk_scores(nhi_id);
		CREATE TABLE IF NOT EXISTS nhi_behavior_baselines (
			id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			nhi_id            TEXT NOT NULL,
			endpoint          TEXT NOT NULL,
			avg_calls_per_hour DOUBLE PRECISION DEFAULT 0,
			std_calls_per_hour DOUBLE PRECISION DEFAULT 0,
			known_ips         TEXT[] DEFAULT '{}',
			known_hours       INTEGER[] DEFAULT '{}',
			total_calls       BIGINT DEFAULT 0,
			first_seen        TIMESTAMPTZ DEFAULT now(),
			last_seen         TIMESTAMPTZ DEFAULT now(),
			UNIQUE(nhi_id, endpoint)
		);
		CREATE INDEX IF NOT EXISTS idx_nhi_baseline_nhi ON nhi_behavior_baselines(nhi_id);
	`)
	return err
}

// SaveRiskScore persists a risk score evaluation.
func (r *NHIRiskPGRepo) SaveRiskScore(ctx context.Context, score *NHIRiskScore) error {
	signalsJSON, _ := json.Marshal(score.Signals)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO nhi_risk_scores (nhi_id, score, level, signals, evaluated_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (nhi_id, evaluated_at) DO NOTHING`,
		score.NHIID, score.Score, score.Level, signalsJSON, score.EvaluatedAt)
	return err
}

// GetRiskScore returns the latest risk score for an NHI.
func (r *NHIRiskPGRepo) GetRiskScore(ctx context.Context, nhiID uuid.UUID) (*NHIRiskScore, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT nhi_id, score, level, signals, evaluated_at
		 FROM nhi_risk_scores WHERE nhi_id = $1 ORDER BY evaluated_at DESC LIMIT 1`, nhiID)

	s := &NHIRiskScore{}
	var signalsJSON []byte
	err := row.Scan(&s.NHIID, &s.Score, &s.Level, &signalsJSON, &s.EvaluatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(signalsJSON, &s.Signals)
	return s, nil
}

// ListHighRisk returns all NHIs with score >= threshold.
func (r *NHIRiskPGRepo) ListHighRisk(ctx context.Context, threshold int) ([]*NHIRiskScore, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT ON (nhi_id) nhi_id, score, level, signals, evaluated_at
		 FROM nhi_risk_scores WHERE score >= $1 ORDER BY nhi_id, evaluated_at DESC`, threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*NHIRiskScore
	for rows.Next() {
		s := &NHIRiskScore{}
		var signalsJSON []byte
		if err := rows.Scan(&s.NHIID, &s.Score, &s.Level, &signalsJSON, &s.EvaluatedAt); err != nil {
			continue
		}
		_ = json.Unmarshal(signalsJSON, &s.Signals)
		result = append(result, s)
	}
	return result, nil
}

// SaveBaseline upserts a behavior baseline for an NHI endpoint.
func (r *NHIRiskPGRepo) SaveBaseline(ctx context.Context, b *NHIBehaviorBaseline) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO nhi_behavior_baselines (nhi_id, endpoint, avg_calls_per_hour, std_calls_per_hour,
		    known_ips, known_hours, total_calls, first_seen, last_seen)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (nhi_id, endpoint) DO UPDATE SET
		   avg_calls_per_hour = EXCLUDED.avg_calls_per_hour,
		   std_calls_per_hour = EXCLUDED.std_calls_per_hour,
		   known_ips = EXCLUDED.known_ips,
		   known_hours = EXCLUDED.known_hours,
		   total_calls = EXCLUDED.total_calls,
		   last_seen = EXCLUDED.last_seen`,
		b.NHIID, b.Endpoint, b.AvgCallsPerHour, b.StdCallsPerHour,
		b.KnownIPs, b.KnownHours, b.TotalCalls, b.FirstSeen, b.LastSeen)
	return err
}

// GetBaselines returns all baselines for an NHI.
func (r *NHIRiskPGRepo) GetBaselines(ctx context.Context, nhiID string) ([]*NHIBehaviorBaseline, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT nhi_id, endpoint, avg_calls_per_hour, std_calls_per_hour,
		        known_ips, known_hours, total_calls, first_seen, last_seen
		 FROM nhi_behavior_baselines WHERE nhi_id = $1`, nhiID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*NHIBehaviorBaseline
	for rows.Next() {
		b := &NHIBehaviorBaseline{}
		if err := rows.Scan(&b.NHIID, &b.Endpoint, &b.AvgCallsPerHour, &b.StdCallsPerHour,
			&b.KnownIPs, &b.KnownHours, &b.TotalCalls, &b.FirstSeen, &b.LastSeen); err != nil {
			continue
		}
		result = append(result, b)
	}
	return result, nil
}

// NHIPGRepo persists NHI identities to PostgreSQL (replaces in-memory map in service layer).
type NHIPGRepo struct {
	pool *pgxpool.Pool
}

// NewNHIPGRepo creates a new PG-backed NHI identity repo (exported).
func NewNHIPGRepo(pool *pgxpool.Pool) *NHIPGRepo {
	return newNHIPGRepo(pool)
}

func newNHIPGRepo(pool *pgxpool.Pool) *NHIPGRepo {
	return &NHIPGRepo{pool: pool}
}

func (r *NHIPGRepo) EnsureSchema(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS nhi_identities (
			id         TEXT PRIMARY KEY,
			tenant_id  UUID,
			type       TEXT NOT NULL DEFAULT 'service_account',
			name       TEXT NOT NULL,
			status     TEXT NOT NULL DEFAULT 'active',
			owner      TEXT DEFAULT '',
			metadata   JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_used  TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS idx_nhi_tenant ON nhi_identities(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_nhi_status ON nhi_identities(status);
	`)
	return err
}

// RegisterNHI creates or updates an NHI identity.
func (r *NHIPGRepo) RegisterNHI(ctx context.Context, id, tenantID, nhiType, name, owner string, metadata map[string]any) error {
	metaJSON, _ := json.Marshal(metadata)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO nhi_identities (id, tenant_id, type, name, status, owner, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'active', $5, $6, now(), now())
		 ON CONFLICT (id) DO UPDATE SET name = $4, owner = $5, metadata = $6, updated_at = now()`,
		id, tenantID, nhiType, name, owner, metaJSON)
	return err
}

// ListNHI returns all NHI identities for a tenant.
func (r *NHIPGRepo) ListNHI(ctx context.Context, tenantID uuid.UUID) ([]map[string]any, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, type, name, status, owner, metadata, created_at, updated_at, last_used
		 FROM nhi_identities WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var id, typ, name, status, owner string
		var metaJSON []byte
		var createdAt, updatedAt time.Time
		var lastUsed *time.Time
		if err := rows.Scan(&id, &typ, &name, &status, &owner, &metaJSON, &createdAt, &updatedAt, &lastUsed); err != nil {
			continue
		}
		entry := map[string]any{
			"id": id, "type": typ, "name": name, "status": status, "owner": owner,
			"created_at": createdAt, "updated_at": updatedAt,
		}
		if lastUsed != nil {
			entry["last_used"] = *lastUsed
		}
		if len(metaJSON) > 0 {
			var meta map[string]any
			_ = json.Unmarshal(metaJSON, &meta)
			entry["metadata"] = meta
		}
		result = append(result, entry)
	}
	return result, nil
}

// GetNHI returns a single NHI by ID.
func (r *NHIPGRepo) GetNHI(ctx context.Context, id string) (map[string]any, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, type, name, status, owner, metadata, created_at, updated_at, last_used
		 FROM nhi_identities WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	var id2, typ, name, status, owner string
	var metaJSON []byte
	var createdAt, updatedAt time.Time
	var lastUsed *time.Time
	if err := rows.Scan(&id2, &typ, &name, &status, &owner, &metaJSON, &createdAt, &updatedAt, &lastUsed); err != nil {
		return nil, err
	}
	entry := map[string]any{
		"id": id2, "type": typ, "name": name, "status": status, "owner": owner,
		"created_at": createdAt, "updated_at": updatedAt,
	}
	if lastUsed != nil {
		entry["last_used"] = *lastUsed
	}
	return entry, nil
}

// DecommissionNHI marks an NHI as decommissioned.
func (r *NHIPGRepo) DecommissionNHI(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE nhi_identities SET status = 'decommissioned', updated_at = now() WHERE id = $1`, id)
	return err
}

// ListOrphans returns NHIs not used since the threshold.
func (r *NHIPGRepo) ListOrphans(ctx context.Context, tenantID uuid.UUID, thresholdDays int) ([]map[string]any, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, type, status, last_used, GREATEST(now() - COALESCE(last_used, created_at), interval '0') as age
		 FROM nhi_identities
		 WHERE tenant_id = $1 AND status = 'active'
		   AND (last_used IS NULL OR last_used < now() - make_interval(days => $2))
		 ORDER BY last_used NULLS FIRST`, tenantID, thresholdDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var id, name, typ, status string
		var lastUsed *time.Time
		var ageStr string
		if err := rows.Scan(&id, &name, &typ, &status, &lastUsed, &ageStr); err != nil {
			continue
		}
		entry := map[string]any{
			"id": id, "name": name, "type": typ, "status": status, "age": ageStr,
		}
		if lastUsed != nil {
			entry["last_used"] = *lastUsed
		}
		result = append(result, entry)
	}
	return result, nil
}
