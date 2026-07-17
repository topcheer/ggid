package server

import (
	"context"
	"fmt"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProtectedApp represents a ZTNA-protected application.
type ProtectedApp struct {
	ID                  uuid.UUID      `json:"id"`
	TenantID            uuid.UUID      `json:"tenant_id"`
	Name                string         `json:"name"`
	Slug                string         `json:"slug"`
	UpstreamURL         string         `json:"upstream_url"`
	Icon                string         `json:"icon,omitempty"`
	Description         string         `json:"description,omitempty"`
	AuthMode            string         `json:"auth_mode"`
	AccessPolicy        map[string]any `json:"access_policy"`
	InjectHeaders       []map[string]any `json:"inject_headers"`
	HealthCheckPath     string         `json:"health_check_path"`
	HealthCheckInterval int            `json:"health_check_interval"`
	HealthStatus        string         `json:"health_status"`
	RateLimitPerMin     int            `json:"rate_limit_per_min"`
	Enabled             bool           `json:"enabled"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// AppAccessLog records a single access to a protected app.
type AppAccessLog struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	AppID          uuid.UUID `json:"app_id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	UserName       string    `json:"user_name,omitempty"`
	Method         string    `json:"method"`
	Path           string    `json:"path"`
	StatusCode     int       `json:"status_code"`
	ResponseTimeMs int       `json:"response_time_ms,omitempty"`
	IPAddress      string    `json:"ip_address,omitempty"`
	UserAgent      string    `json:"user_agent,omitempty"`
	PDPDecision    string    `json:"pdp_decision,omitempty"`
	PDPReason      string    `json:"pdp_reason,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// accessBrokerRepo manages protected apps and access logs.
type accessBrokerRepo struct {
	pool *pgxpool.Pool
}

func newAccessBrokerRepo(pool *pgxpool.Pool) *accessBrokerRepo {
	return &accessBrokerRepo{pool: pool}
}

func (r *accessBrokerRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS protected_apps (
			id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id           UUID NOT NULL,
			name                TEXT NOT NULL,
			slug                TEXT NOT NULL,
			upstream_url        TEXT NOT NULL,
			icon                TEXT,
			description         TEXT,
			auth_mode           TEXT NOT NULL DEFAULT 'jwt',
			access_policy       JSONB NOT NULL DEFAULT '{}',
			inject_headers      JSONB NOT NULL DEFAULT '[]',
			health_check_path   TEXT DEFAULT '/health',
			health_check_interval INT DEFAULT 30,
			health_status       TEXT DEFAULT 'unknown',
			rate_limit_per_min  INT DEFAULT 100,
			enabled             BOOLEAN NOT NULL DEFAULT TRUE,
			created_by          UUID,
			created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE(tenant_id, slug)
		);
		CREATE INDEX IF NOT EXISTS idx_protected_apps_slug ON protected_apps(tenant_id, slug) WHERE enabled = TRUE;
		CREATE TABLE IF NOT EXISTS app_access_logs (
			id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id         UUID NOT NULL,
			app_id            UUID NOT NULL,
			user_id           UUID,
			user_name         TEXT,
			method            TEXT NOT NULL,
			path              TEXT NOT NULL,
			status_code       INT NOT NULL,
			response_time_ms  INT,
			ip_address        TEXT,
			user_agent        TEXT,
			pdp_decision      TEXT,
			pdp_reason        TEXT,
			created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_app_logs_app_time ON app_access_logs(tenant_id, app_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_app_logs_user ON app_access_logs(tenant_id, user_id, created_at DESC);
	`)
	return err
}

// CreateApp stores a protected application.
func (r *accessBrokerRepo) CreateApp(ctx context.Context, app *ProtectedApp) error {
	if r.pool == nil {
		return nil
	}
	if app.ID == uuid.Nil {
		app.ID = uuid.New()
	}
	policyJSON, _ := json.Marshal(app.AccessPolicy)
	headersJSON, _ := json.Marshal(app.InjectHeaders)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO protected_apps (id, tenant_id, name, slug, upstream_url, icon, description, auth_mode, access_policy, inject_headers, health_check_path, rate_limit_per_min, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		app.ID, app.TenantID, app.Name, app.Slug, app.UpstreamURL, app.Icon, app.Description,
		app.AuthMode, policyJSON, headersJSON, app.HealthCheckPath, app.RateLimitPerMin, app.Enabled)
	return err
}

// ListApps returns all enabled protected apps for a tenant.
func (r *accessBrokerRepo) ListApps(ctx context.Context, tenantID uuid.UUID) ([]*ProtectedApp, error) {
	if r.pool == nil {
		return []*ProtectedApp{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, name, slug, upstream_url, COALESCE(icon,''), COALESCE(description,''), auth_mode, access_policy, inject_headers, COALESCE(health_check_path,'/health'), COALESCE(health_check_interval,30), COALESCE(health_status,'unknown'), COALESCE(rate_limit_per_min,100), enabled, created_at, updated_at
		FROM protected_apps WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []*ProtectedApp
	for rows.Next() {
		app, err := scanProtectedApp(rows)
		if err != nil {
			continue
		}
		apps = append(apps, app)
	}
	return apps, nil
}

// GetAppBySlug looks up a protected app by its URL slug.
func (r *accessBrokerRepo) GetAppBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*ProtectedApp, error) {
	if r.pool == nil {
		return nil, nil
	}
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, slug, upstream_url, COALESCE(icon,''), COALESCE(description,''), auth_mode, access_policy, inject_headers, COALESCE(health_check_path,'/health'), COALESCE(health_check_interval,30), COALESCE(health_status,'unknown'), COALESCE(rate_limit_per_min,100), enabled, created_at, updated_at
		FROM protected_apps WHERE tenant_id = $1 AND slug = $2 AND enabled = TRUE`, tenantID, slug)
	return scanProtectedApp(row)
}

// UpdateApp updates a protected application.
func (r *accessBrokerRepo) UpdateApp(ctx context.Context, app *ProtectedApp) error {
	if r.pool == nil {
		return nil
	}
	policyJSON, _ := json.Marshal(app.AccessPolicy)
	headersJSON, _ := json.Marshal(app.InjectHeaders)
	_, err := r.pool.Exec(ctx, `
		UPDATE protected_apps SET name=$3, upstream_url=$4, icon=$5, description=$6, auth_mode=$7, access_policy=$8, inject_headers=$9, rate_limit_per_min=$10, enabled=$11, updated_at=now()
		WHERE id=$1 AND tenant_id=$2`,
		app.ID, app.TenantID, app.Name, app.UpstreamURL, app.Icon, app.Description,
		app.AuthMode, policyJSON, headersJSON, app.RateLimitPerMin, app.Enabled)
	return err
}

// DeleteApp soft-deletes a protected application.
func (r *accessBrokerRepo) DeleteApp(ctx context.Context, id, tenantID uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM protected_apps WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	return err
}

// LogAccess records an access log entry.
func (r *accessBrokerRepo) LogAccess(ctx context.Context, log *AppAccessLog) {
	if r.pool == nil {
		return
	}
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	r.pool.Exec(ctx, `
		INSERT INTO app_access_logs (id, tenant_id, app_id, user_id, user_name, method, path, status_code, response_time_ms, ip_address, user_agent, pdp_decision, pdp_reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		log.ID, log.TenantID, log.AppID, log.UserID, log.UserName, log.Method, log.Path,
		log.StatusCode, log.ResponseTimeMs, log.IPAddress, log.UserAgent, log.PDPDecision, log.PDPReason)
}

// ListAccessLogs returns recent access logs for a tenant.
func (r *accessBrokerRepo) ListAccessLogs(ctx context.Context, tenantID uuid.UUID, appID *uuid.UUID, limit int) ([]*AppAccessLog, error) {
	if r.pool == nil {
		return []*AppAccessLog{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `SELECT id, tenant_id, app_id, user_id, COALESCE(user_name,''), method, path, status_code, COALESCE(response_time_ms,0), COALESCE(ip_address,''), COALESCE(user_agent,''), COALESCE(pdp_decision,''), COALESCE(pdp_reason,''), created_at
		FROM app_access_logs WHERE tenant_id = $1`
	args := []any{tenantID}
	if appID != nil {
		query += ` AND app_id = $2`
		args = append(args, *appID)
	}
	query += ` ORDER BY created_at DESC LIMIT` + fmt.Sprintf(` %d`, limit)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []*AppAccessLog
	for rows.Next() {
		var l AppAccessLog
		if err := rows.Scan(&l.ID, &l.TenantID, &l.AppID, &l.UserID, &l.UserName, &l.Method, &l.Path, &l.StatusCode, &l.ResponseTimeMs, &l.IPAddress, &l.UserAgent, &l.PDPDecision, &l.PDPReason, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, &l)
	}
	return logs, nil
}

type appScanner interface {
	Scan(dest ...any) error
}

func scanProtectedApp(s appScanner) (*ProtectedApp, error) {
	var app ProtectedApp
	var policyJSON, headersJSON []byte
	if err := s.Scan(&app.ID, &app.TenantID, &app.Name, &app.Slug, &app.UpstreamURL, &app.Icon, &app.Description,
		&app.AuthMode, &policyJSON, &headersJSON, &app.HealthCheckPath, &app.HealthCheckInterval, &app.HealthStatus,
		&app.RateLimitPerMin, &app.Enabled, &app.CreatedAt, &app.UpdatedAt); err != nil {
		return nil, err
	}
	if len(policyJSON) > 0 {
		json.Unmarshal(policyJSON, &app.AccessPolicy)
	}
	if len(headersJSON) > 0 {
		json.Unmarshal(headersJSON, &app.InjectHeaders)
	}
	if app.AccessPolicy == nil {
		app.AccessPolicy = map[string]any{}
	}
	if app.InjectHeaders == nil {
		app.InjectHeaders = []map[string]any{}
	}
	return &app, nil
}
