package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantQuota defines per-tenant resource limits.
type TenantQuota struct {
	TenantID        string `json:"tenant_id"`
	Plan            string `json:"plan"` // free, pro, enterprise
	MaxUsers        int    `json:"max_users"`
	MaxAPIKeys      int    `json:"max_api_keys"`
	MaxSessions     int    `json:"max_sessions"`
	MaxStorageMB    int    `json:"max_storage_mb"`
	MaxAPICallsDay  int    `json:"max_api_calls_per_day"`
}

// TenantUsage tracks current resource consumption.
type TenantUsage struct {
	UserCount    int `json:"user_count"`
	APIKeyCount  int `json:"api_key_count"`
	SessionCount int `json:"session_count"`
	StorageMB    int `json:"storage_mb"`
	APICallsToday int `json:"api_calls_today"`
}

// quotaRepo manages tenant quotas + usage in PostgreSQL.
type quotaRepo struct {
	pool *pgxpool.Pool
}

func newQuotaRepo(pool *pgxpool.Pool) *quotaRepo {
	return &quotaRepo{pool: pool}
}

func (r *quotaRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil { return nil }
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tenant_quotas (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id TEXT NOT NULL UNIQUE, plan TEXT DEFAULT 'free',
			max_users INT DEFAULT 100, max_api_keys INT DEFAULT 5,
			max_sessions INT DEFAULT 50, max_storage_mb INT DEFAULT 1024,
			max_api_calls_per_day INT DEFAULT 10000,
			created_at TIMESTAMPTZ DEFAULT now(), updated_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS tenant_usage (
			tenant_id TEXT NOT NULL, metric TEXT NOT NULL,
			value INT NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ DEFAULT now(),
			PRIMARY KEY (tenant_id, metric)
		);
	`)
	return err
}

func (r *quotaRepo) GetQuota(ctx context.Context, tenantID string) (*TenantQuota, error) {
	if r.pool == nil { return defaultQuota(tenantID), nil }
	q := &TenantQuota{TenantID: tenantID}
	err := r.pool.QueryRow(ctx,
		`SELECT plan,max_users,max_api_keys,max_sessions,max_storage_mb,max_api_calls_per_day FROM tenant_quotas WHERE tenant_id=$1`,
		tenantID,
	).Scan(&q.Plan, &q.MaxUsers, &q.MaxAPIKeys, &q.MaxSessions, &q.MaxStorageMB, &q.MaxAPICallsDay)
	if err != nil { return defaultQuota(tenantID), nil }
	return q, nil
}

func (r *quotaRepo) UpsertQuota(ctx context.Context, q *TenantQuota) error {
	if r.pool == nil { return nil }
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tenant_quotas (tenant_id,plan,max_users,max_api_keys,max_sessions,max_storage_mb,max_api_calls_per_day) VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (tenant_id) DO UPDATE SET plan=EXCLUDED.plan,max_users=EXCLUDED.max_users,max_api_keys=EXCLUDED.max_api_keys,max_sessions=EXCLUDED.max_sessions,max_storage_mb=EXCLUDED.max_storage_mb,max_api_calls_per_day=EXCLUDED.max_api_calls_per_day,updated_at=now()`,
		q.TenantID, q.Plan, q.MaxUsers, q.MaxAPIKeys, q.MaxSessions, q.MaxStorageMB, q.MaxAPICallsDay)
	return err
}

func (r *quotaRepo) GetUsage(ctx context.Context, tenantID string) (*TenantUsage, error) {
	if r.pool == nil { return &TenantUsage{}, nil }
	usage := &TenantUsage{}
	metrics := map[string]*int{"user_count": &usage.UserCount, "api_key_count": &usage.APIKeyCount, "session_count": &usage.SessionCount, "storage_mb": &usage.StorageMB, "api_calls_today": &usage.APICallsToday}
	for metric, ptr := range metrics {
		var val int
		r.pool.QueryRow(ctx, `SELECT value FROM tenant_usage WHERE tenant_id=$1 AND metric=$2`, tenantID, metric).Scan(&val)
		*ptr = val
	}
	return usage, nil
}

func (r *quotaRepo) IncrementUsage(ctx context.Context, tenantID, metric string, delta int) error {
	if r.pool == nil { return nil }
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tenant_usage (tenant_id,metric,value) VALUES ($1,$2,$3) ON CONFLICT (tenant_id,metric) DO UPDATE SET value=tenant_usage.value+EXCLUDED.value,updated_at=now()`,
		tenantID, metric, delta)
	return err
}

// CheckQuota verifies if a resource creation is within limits.
// Returns (allowed, remaining, error).
func (r *quotaRepo) CheckQuota(ctx context.Context, tenantID, metric string) (bool, int, error) {
	quota, _ := r.GetQuota(ctx, tenantID)
	usage, _ := r.GetUsage(ctx, tenantID)
	var limit, current int
	switch metric {
	case "user_count":
		limit = quota.MaxUsers; current = usage.UserCount
	case "api_key_count":
		limit = quota.MaxAPIKeys; current = usage.APIKeyCount
	case "session_count":
		limit = quota.MaxSessions; current = usage.SessionCount
	default:
		return true, -1, nil // untracked metric
	}
	remaining := limit - current
	return remaining > 0, remaining, nil
}

func defaultQuota(tenantID string) *TenantQuota {
	return &TenantQuota{TenantID: tenantID, Plan: "free", MaxUsers: 100, MaxAPIKeys: 5, MaxSessions: 50, MaxStorageMB: 1024, MaxAPICallsDay: 10000}
}

// --- Plan Tiers ---

func QuotaForPlan(plan string) *TenantQuota {
	switch strings.ToLower(plan) {
	case "pro":
		return &TenantQuota{Plan: "pro", MaxUsers: 1000, MaxAPIKeys: 50, MaxSessions: 500, MaxStorageMB: 10240, MaxAPICallsDay: 100000}
	case "enterprise":
		return &TenantQuota{Plan: "enterprise", MaxUsers: 999999, MaxAPIKeys: 999999, MaxSessions: 999999, MaxStorageMB: 999999, MaxAPICallsDay: 9999999}
	default:
		return &TenantQuota{Plan: "free", MaxUsers: 100, MaxAPIKeys: 5, MaxSessions: 50, MaxStorageMB: 1024, MaxAPICallsDay: 10000}
	}
}

// --- HTTP Handlers ---

func (h *HTTPHandler) handleTenantQuota(w http.ResponseWriter, r *http.Request) {
	tenantID := strings.TrimPrefix(r.URL.Path, "/api/v1/quotas/")
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		var quota *TenantQuota
		var usage *TenantUsage
		if h.quotaRepo != nil {
			quota, _ = h.quotaRepo.GetQuota(r.Context(), tenantID)
			usage, _ = h.quotaRepo.GetUsage(r.Context(), tenantID)
		} else {
			quota = defaultQuota(tenantID)
			usage = &TenantUsage{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"quota": quota, "usage": usage})
	case http.MethodPut, http.MethodPost:
		var req struct {
			Plan           string `json:"plan"`
			MaxUsers       int    `json:"max_users"`
			MaxAPIKeys     int    `json:"max_api_keys"`
			MaxSessions    int    `json:"max_sessions"`
			MaxStorageMB   int    `json:"max_storage_mb"`
			MaxAPICallsDay int    `json:"max_api_calls_per_day"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		q := &TenantQuota{TenantID: tenantID}
		if req.Plan != "" {
			planDefaults := QuotaForPlan(req.Plan)
			q = planDefaults
			q.TenantID = tenantID
		} else {
			q.MaxUsers = req.MaxUsers; q.MaxAPIKeys = req.MaxAPIKeys
			q.MaxSessions = req.MaxSessions; q.MaxStorageMB = req.MaxStorageMB
			q.MaxAPICallsDay = req.MaxAPICallsDay
		}
		if h.quotaRepo != nil {
			h.quotaRepo.UpsertQuota(r.Context(), q)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "quota": q})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) SetQuotaRepo(repo *quotaRepo) {
	h.quotaRepo = repo
}

var _ = fmt.Sprintf
var _ = uuid.New
var _ = time.Now
