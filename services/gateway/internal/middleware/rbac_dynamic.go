package middleware

import (
	"context"
	"encoding/json"
	"log"
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
	rbacCacheKey     = "rbac:routes:all"
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
		SELECT r.name, r.key, rrp.route_prefix, rrp.permission_level
		FROM role_route_permissions rrp
		JOIN roles r ON r.id = rrp.role_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []routePermRow
	for rows.Next() {
		var row routePermRow
		if err := rows.Scan(&row.RoleName, &row.RoleKey, &row.Prefix, &row.Level); err != nil {
			continue
		}
		out = append(out, row)
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
	// 1. Superuser bypass — preserves existing behavior for admins even if
	// the seed migration has not run yet.
	if hasAdminScope(claims.Scopes) || hasAdminScope(claims.Roles) {
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
