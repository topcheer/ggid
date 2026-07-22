package middleware

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// rbac_dynamic.go implements ADR-dynamic-rbac: DB-driven route permissions
// (role_route_permissions table) with Redis cache (60s TTL), in-memory
// warm-start fallback, and hardcoded-prefix fallback when neither is available.

const (
	rbacCacheKey     = "rbac:routes:v2:all"
	rbacCacheTTL     = 60 * time.Second
	rbacMemCacheTTL  = 60 * time.Second
	rbacQueryTimeout = 3 * time.Second
)

// routePermRow is one row of the flattened role_route_permissions + roles join.
type routePermRow struct {
	RoleName string `json:"role_name"`
	RoleKey  string `json:"role_key"`
	Prefix   string `json:"route_prefix"`
	Level    string `json:"permission_level"`
	// TenantID scopes the rule to the tenant that owns the role. Grant
	// decisions only match rows of the caller's own tenant — otherwise a
	// role named "Administrator" created in tenant B would inherit the
	// platform tenant's Administrator grants (privilege escalation).
	TenantID string `json:"tenant_id"`
}

// permLevelRank orders permission levels: read < write < admin.
func permLevelRank(level string) int {
	switch strings.ToLower(level) {
	case "admin":
		return 3
	case "write":
		return 2
	case "read":
		return 1
	default:
		return 0
	}
}

// RBACResolver resolves route access decisions from role_route_permissions,
// cached in Redis and memory. It is safe for concurrent use.
type RBACResolver struct {
	rdb  *redis.Client
	dbURL string

	mu        sync.RWMutex
	pool      *pgxpool.Pool
	poolTried bool
	snapshot  []routePermRow
	loadedAt  time.Time
	everLoaded bool
}

// NewRBACResolver creates a resolver. Either rdb or databaseURL may be empty;
// with both empty the resolver is unavailable and callers fall back to the
// hardcoded prefix list.
func NewRBACResolver(rdb *redis.Client, databaseURL string) *RBACResolver {
	return &RBACResolver{rdb: rdb, dbURL: databaseURL}
}

// --- public-path exemptions ---------------------------------------------
//
// RequireAdminScope historically ran for every request — including public
// paths — but could only block the hardcoded admin prefixes, so public
// endpoints were never affected. Dynamic rules can match arbitrary
// prefixes, so public paths must be exempted explicitly (P0 incident:
// /oauth/token was blocked by a broad DB prefix row).

var (
	exemptMu       sync.RWMutex
	exemptPrefixes []string
)

// SetRBACExemptPrefixes installs path prefixes that bypass dynamic RBAC
// handling entirely (the router's publicPaths list).
func SetRBACExemptPrefixes(prefixes []string) {
	exemptMu.Lock()
	defer exemptMu.Unlock()
	exemptPrefixes = append([]string(nil), prefixes...)
}

// isRBACExempt reports whether the path bypasses dynamic RBAC checks.
func isRBACExempt(path string) bool {
	exemptMu.RLock()
	defer exemptMu.RUnlock()
	for _, p := range exemptPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// --- package-level wiring (keeps RequireAdminScope signature unchanged) ---

var (
	resolverMu sync.RWMutex
	resolver   *RBACResolver
)

// SetRBACResolver installs the process-wide resolver used by RequireAdminScope.
func SetRBACResolver(r *RBACResolver) {
	resolverMu.Lock()
	defer resolverMu.Unlock()
	resolver = r
}

// getRBACResolver returns the installed resolver (may be nil).
func getRBACResolver() *RBACResolver {
	resolverMu.RLock()
	defer resolverMu.RUnlock()
	return resolver
}

// WarmStart pre-loads the route-permission snapshot so the gateway can decide
// even if Redis/DB become unavailable later (in-memory fallback).
func (r *RBACResolver) WarmStart(ctx context.Context) {
	if _, err := r.load(ctx, true); err != nil {
		log.Printf("RBAC resolver warm-start: %v (will retry on first request)", err)
	}
}

// Available reports whether the resolver has ever loaded a snapshot.
// When false, RequireAdminScope falls back to the hardcoded prefix logic.
func (r *RBACResolver) Available() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.everLoaded
}

// load returns the current snapshot, refreshing from Redis/DB when stale.
// Order: fresh memory → Redis → DB (re-cache) → stale memory.
func (r *RBACResolver) load(ctx context.Context, force bool) ([]routePermRow, error) {
	r.mu.RLock()
	fresh := r.everLoaded && time.Since(r.loadedAt) < rbacMemCacheTTL
	snap := r.snapshot
	ever := r.everLoaded
	r.mu.RUnlock()

	if fresh && !force {
		return snap, nil
	}

	// Redis cache.
	if r.rdb != nil {
		if data, err := r.rdb.Get(ctx, rbacCacheKey).Bytes(); err == nil {
			var rows []routePermRow
			if json.Unmarshal(data, &rows) == nil {
				r.storeSnapshot(rows)
				return rows, nil
			}
		}
	}

	// PostgreSQL.
	if rows, err := r.loadFromDB(ctx); err == nil {
		r.storeSnapshot(rows)
		if r.rdb != nil {
			if data, merr := json.Marshal(rows); merr == nil {
				_ = r.rdb.Set(ctx, rbacCacheKey, data, rbacCacheTTL).Err()
			}
		}
		return rows, nil
	}

	// Stale memory fallback.
	if ever {
		return snap, nil
	}
	return nil, errNoRBACData
}

var errNoRBACData = errRBAC("rbac: no route-permission data available")

type errRBAC string

func (e errRBAC) Error() string { return string(e) }

func (r *RBACResolver) storeSnapshot(rows []routePermRow) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot = rows
	r.loadedAt = time.Now()
	r.everLoaded = true
}

// Invalidate drops the in-memory snapshot (Redis expires on its own TTL).
func (r *RBACResolver) Invalidate() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot = nil
	r.everLoaded = false
	r.loadedAt = time.Time{}
}

func (r *RBACResolver) getPool(ctx context.Context) (*pgxpool.Pool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pool != nil {
		return r.pool, nil
	}
	if r.poolTried || r.dbURL == "" {
		return nil, errRBAC("rbac: database not configured")
	}
	r.poolTried = true
	pool, err := pgxpool.New(ctx, r.dbURL)
	if err != nil {
		return nil, err
	}
	r.pool = pool
	return pool, nil
}

func (r *RBACResolver) loadFromDB(ctx context.Context) ([]routePermRow, error) {
	pool, err := r.getPool(ctx)
	if err != nil {
		return nil, err
	}
	qctx, cancel := context.WithTimeout(ctx, rbacQueryTimeout)
	defer cancel()
	rows, err := pool.Query(qctx, `
		SELECT r.name, r.key, rrp.route_prefix, rrp.permission_level, r.tenant_id::text
		FROM role_route_permissions rrp
		JOIN roles r ON r.id = rrp.role_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []routePermRow
	skipped := 0
	for rows.Next() {
		var row routePermRow
		if err := rows.Scan(&row.RoleName, &row.RoleKey, &row.Prefix, &row.Level, &row.TenantID); err != nil {
			continue
		}
		if row.Prefix == "" || !strings.HasPrefix(row.Prefix, "/api/") {
			skipped++
		}
		out = append(out, row)
	}
	if skipped > 0 {
		slog.Warn("rbac: ignoring non-/api/ route permission rules (they cannot gate API traffic)", "count", skipped)
	}
	return out, rows.Err()
}

// CheckAccess decides whether the request may proceed based on dynamic
// route permissions. handled=false means no dynamic rule matched (or the
// resolver has no data) and the caller should fall back to static logic.
//
// Decision rules (ADR-dynamic-rbac):
//  1. Platform/tenant admin scope bypass (superuser, backwards compatible).
//  2. Longest-prefix match against role_route_permissions wins.
//  3. Required level: GET → read, all other methods → write.
//  4. Access granted if any of the user's roles (JWT roles claim, matched on
//     role name or key) holds a level >= required for the matched prefix.
func (r *RBACResolver) CheckAccess(ctx context.Context, path, method string, claims JWTCClaims) (allow, handled bool) {
	// 0. Self-service endpoints bypass dynamic RBAC — every authenticated
	// user can view/edit their own profile. Only the whitelisted exact
	// paths are exempt (see SelfServicePaths). Deep sub-resources like
	// /users/me/settings are NOT exempt and must match a dynamic rule.
	if SelfServicePaths[path] ||
		strings.HasPrefix(path, "/api/v1/tenants/resolve") {
		return false, false // not handled → static fallback
	}

	// 1. Superuser bypass — scopes claim ONLY. Role display names must never
	// grant superuser access: a tenant admin can create a role named
	// "Administrator" and would otherwise escalate to platform admin.
	// Role-based access is handled by rule 2 (role_route_permissions).
	if hasAdminScope(claims.Scopes) {
		return true, true
	}

	rows, err := r.load(ctx, false)
	if err != nil {
		return false, false
	}

	// 2. Longest-prefix match.
	bestLen := -1
	required := 1 // read
	if method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
		required = 2 // write
	}
	userRoles := make(map[string]bool, len(claims.Roles)+len(claims.Scopes))
	for _, rn := range claims.Roles {
		userRoles[strings.ToLower(rn)] = true
	}

	grant := 0
	for _, row := range rows {
		// Tenant isolation: rules only apply to the caller's own tenant.
		// Rows without a tenant (legacy cache entries) match no one except
		// token-less tenant claims, which cannot occur for authenticated
		// requests.
		if row.TenantID != claims.TenantID {
			continue
		}
		// Ignore non-API prefixes (console navigation routes like /dashboard)
		// and empty prefixes — they must never gate API traffic.
		if row.Prefix == "" || !strings.HasPrefix(row.Prefix, "/api/") {
			continue
		}
		if !strings.HasPrefix(path, row.Prefix) {
			continue
		}
		if len(row.Prefix) < bestLen {
			continue
		}
		if len(row.Prefix) > bestLen {
			bestLen = len(row.Prefix)
			grant = 0
		}
		// Same longest prefix: keep the highest grant among the user's roles.
		if userRoles[strings.ToLower(row.RoleName)] || userRoles[strings.ToLower(row.RoleKey)] {
			if lvl := permLevelRank(row.Level); lvl > grant {
				grant = lvl
			}
		}
	}

	if bestLen < 0 {
		// No dynamic rule for this path — caller applies static logic.
		return false, false
	}
	return grant >= required, true
}
