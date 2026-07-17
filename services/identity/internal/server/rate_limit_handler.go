package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// TenantRateLimit defines per-tenant API rate limits.
type TenantRateLimit struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	EndpointPattern string    `json:"endpoint_pattern"` // e.g. "/api/v1/*" or "/api/v1/auth/*"
	RPSLimit        int       `json:"rps_limit"`        // requests per second
	BurstLimit      int       `json:"burst_limit"`      // max burst
	Strategy        string    `json:"strategy"`         // "token_bucket" or "sliding_window"
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// rateLimitRepo manages tenant rate limit configs in PostgreSQL.
type rateLimitRepo struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
}

func newRateLimitRepo(pool *pgxpool.Pool, rdb *redis.Client) *rateLimitRepo {
	return &rateLimitRepo{pool: pool, rdb: rdb}
}

func (r *rateLimitRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tenant_rate_limits (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			endpoint_pattern TEXT NOT NULL,
			rps_limit INT NOT NULL DEFAULT 100,
			burst_limit INT NOT NULL DEFAULT 200,
			strategy TEXT NOT NULL DEFAULT 'token_bucket',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE(tenant_id, endpoint_pattern)
		);
		CREATE INDEX IF NOT EXISTS idx_rate_limits_tenant ON tenant_rate_limits(tenant_id, enabled);
	`)
	return err
}

func (r *rateLimitRepo) Upsert(ctx context.Context, rl *TenantRateLimit) error {
	if r.pool == nil {
		return nil
	}
	if rl.ID == uuid.Nil {
		rl.ID = uuid.New()
	}
	rl.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tenant_rate_limits (id, tenant_id, endpoint_pattern, rps_limit, burst_limit, strategy, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())
		ON CONFLICT (tenant_id, endpoint_pattern) DO UPDATE SET
			rps_limit = EXCLUDED.rps_limit, burst_limit = EXCLUDED.burst_limit,
			strategy = EXCLUDED.strategy, enabled = EXCLUDED.enabled, updated_at = now()`,
		rl.ID, rl.TenantID, rl.EndpointPattern, rl.RPSLimit, rl.BurstLimit, rl.Strategy, rl.Enabled)
	return err
}

func (r *rateLimitRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*TenantRateLimit, error) {
	if r.pool == nil {
		return []*TenantRateLimit{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, endpoint_pattern, rps_limit, burst_limit, strategy, enabled, created_at, updated_at
		FROM tenant_rate_limits WHERE tenant_id = $1 ORDER BY endpoint_pattern`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*TenantRateLimit
	for rows.Next() {
		var rl TenantRateLimit
		if err := rows.Scan(&rl.ID, &rl.TenantID, &rl.EndpointPattern, &rl.RPSLimit, &rl.BurstLimit, &rl.Strategy, &rl.Enabled, &rl.CreatedAt, &rl.UpdatedAt); err != nil {
			continue
		}
		result = append(result, &rl)
	}
	return result, nil
}

func (r *rateLimitRepo) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM tenant_rate_limits WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

// CheckRateLimit evaluates the rate limit using Redis token bucket.
// Returns allowed=true if request is within limit, false if rate exceeded.
func (r *rateLimitRepo) CheckRateLimit(ctx context.Context, tenantID uuid.UUID, endpoint string) (allowed bool, retryAfter int) {
	if r.rdb == nil {
		return true, 0 // no Redis → allow
	}

	// Find matching config.
	limits, err := r.List(ctx, tenantID)
	if err != nil || len(limits) == 0 {
		return true, 0 // no config → allow
	}

	// Match endpoint against patterns.
	var matched *TenantRateLimit
	for _, rl := range limits {
		if !rl.Enabled {
			continue
		}
		if matchPattern(rl.EndpointPattern, endpoint) {
			matched = rl
			break
		}
	}
	if matched == nil {
		return true, 0
	}

	// Token bucket via Redis INCR with TTL.
	key := "ratelimit:" + tenantID.String() + ":" + matched.EndpointPattern
	count, err := r.rdb.Incr(ctx, key).Result()
	if err != nil {
		return true, 0 // Redis error → allow (fail open)
	}
	if count == 1 {
		// First request in window — set TTL to 1 second (sliding 1s window).
		r.rdb.Expire(ctx, key, time.Second)
	}
	if count > int64(matched.BurstLimit) {
		return false, 1 // retry after 1 second
	}
	return true, 0
}

// matchPattern checks if an endpoint matches a glob pattern.
// Supports * wildcard: "/api/v1/auth/*" matches "/api/v1/auth/login"
func matchPattern(pattern, endpoint string) bool {
	if pattern == "*" || pattern == "/*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == endpoint
	}
	// Convert glob to prefix match.
	prefix := strings.TrimSuffix(pattern, "*")
	return strings.HasPrefix(endpoint, prefix)
}

// --- API Handlers ---

func (h *HTTPHandler) handleRateLimits(w http.ResponseWriter, r *http.Request) {
	tc, _ := ggidtenant.FromContext(r.Context())
	path := r.URL.Path

	// DELETE /api/v1/identity/tenants/{id}/rate-limits/{rule_id}
	if r.Method == http.MethodDelete {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			ruleID, err := uuid.Parse(parts[len(parts)-1])
			if err == nil && h.rateLimitRepo != nil && tc != nil {
				h.rateLimitRepo.Delete(r.Context(), ruleID, tc.TenantID)
			}
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
		return
	}

	switch r.Method {
	case http.MethodPost:
		var rl TenantRateLimit
		if err := json.NewDecoder(r.Body).Decode(&rl); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if tc != nil {
			rl.TenantID = tc.TenantID
		}
		if rl.EndpointPattern == "" {
			writeError(w, http.StatusBadRequest, "endpoint_pattern required")
			return
		}
		if rl.RPSLimit <= 0 {
			rl.RPSLimit = 100
		}
		if rl.BurstLimit <= 0 {
			rl.BurstLimit = rl.RPSLimit * 2
		}
		if rl.Strategy == "" {
			rl.Strategy = "token_bucket"
		}
		rl.Enabled = true
		if h.rateLimitRepo != nil {
			if err := h.rateLimitRepo.Upsert(r.Context(), &rl); err != nil {
				writeError(w, http.StatusInternalServerError, "failed")
				return
			}
		}
		writeJSON(w, http.StatusCreated, rl)
	case http.MethodGet:
		var limits []*TenantRateLimit
		if h.rateLimitRepo != nil && tc != nil {
			limits, _ = h.rateLimitRepo.List(r.Context(), tc.TenantID)
		}
		if limits == nil {
			limits = []*TenantRateLimit{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"rate_limits": limits, "total": len(limits)})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// SetRateLimitRepo injects the rate limit repository.
func (h *HTTPHandler) SetRateLimitRepo(repo *rateLimitRepo) {
	h.rateLimitRepo = repo
}

// suppress unused
var _ sync.RWMutex
