package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ThreatIntelSource represents an external threat intelligence feed source.
type ThreatIntelSource struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Name         string     `json:"name"`
	SourceType   string     `json:"source_type"` // ip | credential | domain | url
	APIEndpoint  string     `json:"api_endpoint"`
	APIKeyRef    string     `json:"api_key_ref,omitempty"`
	PollInterval string     `json:"poll_interval"`
	LastPoll     *time.Time `json:"last_poll,omitempty"`
	Enabled      bool       `json:"enabled"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ThreatIndicator represents a single indicator of compromise.
type ThreatIndicator struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	SourceID       uuid.UUID  `json:"source_id"`
	IndicatorType  string     `json:"indicator_type"`  // ip | email | credential_hash | domain | url
	IndicatorValue string     `json:"indicator_value"`
	Severity       string     `json:"severity"`       // low | medium | high | critical
	Confidence     int        `json:"confidence"`     // 0-100
	FirstSeen      time.Time  `json:"first_seen"`
	LastSeen       time.Time  `json:"last_seen"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// ThreatIntelRepository manages threat_intel_sources and threat_indicators tables.
type ThreatIntelRepository struct {
	pool *pgxpool.Pool
}

func NewThreatIntelRepository(pool *pgxpool.Pool) *ThreatIntelRepository {
	return &ThreatIntelRepository{pool: pool}
}

// EnsureSchema creates threat_intel_sources and threat_indicators tables.
func (r *ThreatIntelRepository) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS threat_intel_sources (
			id            UUID PRIMARY KEY,
			tenant_id     UUID NOT NULL,
			name          TEXT NOT NULL,
			source_type   TEXT NOT NULL,
			api_endpoint  TEXT NOT NULL,
			api_key_ref   TEXT,
			poll_interval TEXT NOT NULL DEFAULT '1 hour',
			last_poll     TIMESTAMPTZ,
			enabled       BOOLEAN NOT NULL DEFAULT TRUE,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE(tenant_id, name)
		);

		CREATE TABLE IF NOT EXISTS threat_indicators (
			id              UUID PRIMARY KEY,
			tenant_id       UUID NOT NULL,
			source_id       UUID NOT NULL REFERENCES threat_intel_sources(id) ON DELETE CASCADE,
			indicator_type  TEXT NOT NULL,
			indicator_value TEXT NOT NULL,
			severity        TEXT NOT NULL DEFAULT 'medium',
			confidence      INT NOT NULL DEFAULT 50,
			first_seen      TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_seen       TIMESTAMPTZ NOT NULL DEFAULT now(),
			expires_at      TIMESTAMPTZ,
			metadata        JSONB DEFAULT '{}',
			UNIQUE(tenant_id, indicator_type, indicator_value)
		);

		CREATE INDEX IF NOT EXISTS idx_threat_indicators_lookup
			ON threat_indicators(tenant_id, indicator_type, indicator_value);
		CREATE INDEX IF NOT EXISTS idx_threat_indicators_expiry
			ON threat_indicators(expires_at) WHERE expires_at IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_threat_sources_tenant
			ON threat_intel_sources(tenant_id, enabled);
	`)
	return err
}

// CreateSource inserts a new threat intel source.
func (r *ThreatIntelRepository) CreateSource(ctx context.Context, s *ThreatIntelSource) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO threat_intel_sources (id, tenant_id, name, source_type, api_endpoint, api_key_ref, poll_interval, enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())`,
		s.ID, s.TenantID, s.Name, s.SourceType, s.APIEndpoint, s.APIKeyRef, s.PollInterval, s.Enabled)
	return err
}

// ListSources returns enabled (or all) sources for a tenant.
func (r *ThreatIntelRepository) ListSources(ctx context.Context, tenantID uuid.UUID, enabledOnly bool) ([]ThreatIntelSource, error) {
	if r.pool == nil {
		return nil, nil
	}
	q := `SELECT id, tenant_id, name, source_type, api_endpoint, api_key_ref, poll_interval, last_poll, enabled, created_at
		FROM threat_intel_sources WHERE tenant_id = $1`
	if enabledOnly {
		q += ` AND enabled = TRUE`
	}
	q += ` ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []ThreatIntelSource
	for rows.Next() {
		var s ThreatIntelSource
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.SourceType, &s.APIEndpoint, &s.APIKeyRef, &s.PollInterval, &s.LastPoll, &s.Enabled, &s.CreatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, nil
}

// DeleteSource removes a source by ID (cascades to indicators).
func (r *ThreatIntelRepository) DeleteSource(ctx context.Context, tenantID, id uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM threat_intel_sources WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	return err
}

// UpsertIndicator inserts or updates an indicator (refreshes last_seen).
func (r *ThreatIntelRepository) UpsertIndicator(ctx context.Context, ind *ThreatIndicator) error {
	if r.pool == nil {
		return nil
	}
	metaJSON, _ := json.Marshal(ind.Metadata)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO threat_indicators (id, tenant_id, source_id, indicator_type, indicator_value, severity, confidence, first_seen, last_seen, expires_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now(), $8, $9)
		ON CONFLICT (tenant_id, indicator_type, indicator_value)
		DO UPDATE SET severity = EXCLUDED.severity, confidence = EXCLUDED.confidence, last_seen = now(), expires_at = EXCLUDED.expires_at, metadata = EXCLUDED.metadata`,
		ind.ID, ind.TenantID, ind.SourceID, ind.IndicatorType, ind.IndicatorValue, ind.Severity, ind.Confidence, ind.ExpiresAt, metaJSON)
	return err
}

// ListIndicators returns indicators for a tenant, optionally filtered by type.
func (r *ThreatIntelRepository) ListIndicators(ctx context.Context, tenantID uuid.UUID, indType string, pageSize int) ([]ThreatIndicator, int, error) {
	if r.pool == nil {
		return nil, 0, nil
	}
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 100
	}
	q := `SELECT id, tenant_id, source_id, indicator_type, indicator_value, severity, confidence, first_seen, last_seen, expires_at, metadata
		FROM threat_indicators WHERE tenant_id = $1`
	args := []any{tenantID}
	if indType != "" {
		q += ` AND indicator_type = $2`
		args = append(args, indType)
	}
	q += fmt.Sprintf(` ORDER BY last_seen DESC LIMIT %d`, pageSize)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var indicators []ThreatIndicator
	for rows.Next() {
		var ind ThreatIndicator
		var metaBytes []byte
		if err := rows.Scan(&ind.ID, &ind.TenantID, &ind.SourceID, &ind.IndicatorType, &ind.IndicatorValue, &ind.Severity, &ind.Confidence, &ind.FirstSeen, &ind.LastSeen, &ind.ExpiresAt, &metaBytes); err != nil {
			return nil, 0, err
		}
		if metaBytes != nil {
			json.Unmarshal(metaBytes, &ind.Metadata)
		}
		indicators = append(indicators, ind)
	}

	// Count total
	countQ := `SELECT count(*) FROM threat_indicators WHERE tenant_id = $1`
	countArgs := []any{tenantID}
	if indType != "" {
		countQ += ` AND indicator_type = $2`
		countArgs = append(countArgs, indType)
	}
	var total int
	r.pool.QueryRow(ctx, countQ, countArgs...).Scan(&total)
	return indicators, total, nil
}

// CheckIndicator queries for a specific indicator match.
func (r *ThreatIntelRepository) CheckIndicator(ctx context.Context, tenantID uuid.UUID, indType, value string) (*ThreatIndicator, error) {
	if r.pool == nil {
		return nil, nil
	}
	var ind ThreatIndicator
	var metaBytes []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, source_id, indicator_type, indicator_value, severity, confidence, first_seen, last_seen, expires_at, metadata
		FROM threat_indicators
		WHERE tenant_id = $1 AND indicator_type = $2 AND indicator_value = $3
		AND (expires_at IS NULL OR expires_at > now())
		LIMIT 1`, tenantID, indType, value).Scan(
		&ind.ID, &ind.TenantID, &ind.SourceID, &ind.IndicatorType, &ind.IndicatorValue,
		&ind.Severity, &ind.Confidence, &ind.FirstSeen, &ind.LastSeen, &ind.ExpiresAt, &metaBytes)
	if err != nil {
		return nil, nil // no match
	}
	if metaBytes != nil {
		json.Unmarshal(metaBytes, &ind.Metadata)
	}
	return &ind, nil
}

// Stats returns aggregate threat intel statistics.
type ThreatIntelStats struct {
	SourcesEnabled int            `json:"sources_enabled"`
	IndicatorsTotal int           `json:"indicators_total"`
	Hits24h        int            `json:"hits_24h"`
	ByType         map[string]int `json:"by_type"`
}

func (r *ThreatIntelRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*ThreatIntelStats, error) {
	if r.pool == nil {
		return &ThreatIntelStats{ByType: map[string]int{}}, nil
	}
	stats := &ThreatIntelStats{ByType: map[string]int{}}

	r.pool.QueryRow(ctx, `SELECT count(*) FROM threat_intel_sources WHERE tenant_id = $1 AND enabled = TRUE`, tenantID).Scan(&stats.SourcesEnabled)
	r.pool.QueryRow(ctx, `SELECT count(*) FROM threat_indicators WHERE tenant_id = $1`, tenantID).Scan(&stats.IndicatorsTotal)

	// By type breakdown
	rows, err := r.pool.Query(ctx, `SELECT indicator_type, count(*) FROM threat_indicators WHERE tenant_id = $1 GROUP BY indicator_type`, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			var c int
			rows.Scan(&t, &c)
			stats.ByType[t] = c
		}
	}

	return stats, nil
}

// DeleteExpired removes indicators past their expiry time.
func (r *ThreatIntelRepository) DeleteExpired(ctx context.Context) (int64, error) {
	if r.pool == nil {
		return 0, nil
	}
	ct, err := r.pool.Exec(ctx, `DELETE FROM threat_indicators WHERE expires_at IS NOT NULL AND expires_at < now()`)
	return ct.RowsAffected(), err
}

// UpdateLastPoll records when a source was last polled.
func (r *ThreatIntelRepository) UpdateLastPoll(ctx context.Context, sourceID uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `UPDATE threat_intel_sources SET last_poll = now() WHERE id = $1`, sourceID)
	return err
}

// ListEnabledSources returns all enabled sources across all tenants (for collector).
func (r *ThreatIntelRepository) ListEnabledSources(ctx context.Context) ([]ThreatIntelSource, error) {
	if r.pool == nil {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id, tenant_id, name, source_type, api_endpoint, api_key_ref, poll_interval, last_poll, enabled, created_at
		FROM threat_intel_sources WHERE enabled = TRUE ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []ThreatIntelSource
	for rows.Next() {
		var s ThreatIntelSource
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.SourceType, &s.APIEndpoint, &s.APIKeyRef, &s.PollInterval, &s.LastPoll, &s.Enabled, &s.CreatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, nil
}
