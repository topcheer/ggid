# IAM Code Audit Report — Performance (R20) + Code Quality (R14)

**Date:** 2026-08-03  
**Auditor:** Independent (first contact with ggid)  
**Scope:** services/gateway, services/auth, services/identity, services/oauth, services/audit, pkg/, console/src/  
**Method:** Read-only static analysis, no modifications  

---

## Summary

| Severity | Count |
|----------|-------|
| P0       | 4     |
| P1       | 10    |
| P2       | 8     |

**Code paths examined:**
- `pkg/db/db.go` (full 250 lines), `pkg/httputil/response.go`, `pkg/rbac/permissions.go` (306 lines), `pkg/rbac/pgx_adapter.go`, `pkg/rbac/route_permissions.go`
- `services/identity/internal/data/db.go`, `repository/pg_repo.go` (926 lines), `server/bulk_import.go`, `server/quota_handler.go`, `server/ttl_cache.go`, `server/rebac_cache.go`
- `services/audit/internal/data/db.go`, `repository/audit_repo.go` (480 lines)
- `services/oauth/internal/service/introspection_cache.go`
- `services/gateway/internal/router/router.go` (1523 lines, 40 functions), `healthcheck/healthcheck.go`, `middleware/metering.go`, `middleware/timeout.go`, `middleware/rbac_dynamic.go`
- Cross-cutting grep scans: N+1 patterns (89+176+121 files), goroutine launches (46 files), mutex usage (49+214 files), `fmt.Errorf` (1061 matches in 199 files), `pkg/errors` (167 matches in 44 files), `os.Getenv` (213 matches in 59 files), `json.NewEncoder` (338 matches), `json.Marshal` (242 matches)

---

## Performance Findings (Round 20)

### PERF-P0-1: Audit hash chain FOR UPDATE serializes ALL writes per tenant

**File:** `services/audit/internal/repository/audit_repo.go:58-93`  
**Severity:** P0

**Problem:** Every `Insert()` call begins a transaction, then issues:
```sql
SELECT COALESCE(hash, '') FROM audit_events
  WHERE tenant_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1 FOR UPDATE
```
This `FOR UPDATE` on the latest row serializes ALL audit inserts for a tenant. Since audit events are written on virtually every API call (logins, role changes, token issuance), this creates a hard serialization point.

**Risk:** Under concurrent load (>50 req/s per tenant), the audit insert becomes the system bottleneck. Each insert waits for the previous transaction to commit, creating a sequential chain. Connection pool exhaustion is likely as requests queue up waiting for the row lock.

**Assessment:** The TOCTOU protection is correct for hash chain integrity. However, the performance impact is severe. Alternative approaches:
1. Advisory locks (`pg_advisory_xact_lock`) scoped per-tenant — same serialization but cheaper than row locks
2. Batch inserts via a queue (NATS consumer exists but isn't used for inserts)
3. Partition the hash chain by time window to reduce lock contention

**Recommendation:** Use `pg_advisory_xact_lock(hashtext(tenant_id))` instead of `FOR UPDATE` on the row. Same correctness guarantee, no row-level I/O contention.

---

### PERF-P0-2: Bulk import processes 10,000 users sequentially with individual INSERTs

**File:** `services/identity/internal/server/bulk_import.go:98-160`  
**Severity:** P0

**Problem:** The handler accepts up to 10,000 users (`len(req.Users) > 10000` check at line 86) but processes each user individually via `h.svc.CreateUser()` inside a `for` loop. Each `CreateUser` call:
1. Begins a new transaction
2. Sets tenant RLS (`SELECT set_config(...)`)
3. Executes an INSERT
4. Commits

This is 10,000 individual transactions. The code comment at line 140 even acknowledges this:
```go
// In production: batch INSERT via pgx.CopyFrom for performance.
// For now, use CreateUser for each user (correct, just not bulk-optimized).
```

Additionally, each iteration may hash a password with argon2id (`ggidcrypto.HashPassword`), which is deliberately slow (~100ms per hash). For 10,000 users, that's ~16 minutes of CPU time.

**Risk:** A single bulk import request can block for 10+ minutes, holding an HTTP connection open and consuming a goroutine. This is a denial-of-service vector — an admin triggering a large import will degrade the entire identity service.

**Recommendation:**
1. Use `pgx.CopyFrom` for the INSERT batch
2. Move password hashing to a background worker pool with bounded concurrency
3. Return a job ID and process asynchronously
4. At minimum, process in chunks (e.g., 100 per batch INSERT)

---

### PERF-P0-3: IntrospectionCache is unbounded — OOM under token flooding

**File:** `services/oauth/internal/service/introspection_cache.go:26-67`  
**Severity:** P0

**Problem:** The `IntrospectionCache` uses a plain `map[string]*CachedIntrospection` with no size limit, no eviction policy, and no cleanup goroutine. Entries are added on every `SetCachedIntrospection` call but never removed unless explicitly invalidated.

```go
type IntrospectionCache struct {
    cache map[string]*CachedIntrospection  // grows forever
    // ...no maxSize, no eviction
}
```

The cache is keyed by `sha256(token)`. An attacker who generates many unique tokens (via repeated authorization grants, token exchanges, or client credentials flows) will cause unbounded memory growth.

**Risk:** Memory exhaustion leading to OOM kill. In a production OAuth server processing thousands of token introspections, this cache will grow indefinitely.

**Assessment:** The TTL check in `GetCachedIntrospection` (line 53) skips expired entries on read but never deletes them. The map keeps growing. There is no `Cleanup()` or background eviction goroutine.

**Recommendation:**
1. Add a max size (e.g., 10,000 entries) with LRU eviction
2. Add a background cleanup goroutine that periodically removes expired entries
3. Or switch to Redis with TTL (like `rebacCache` does)

---

### PERF-P0-4: ttlCache (identity) has no size limit, no cleanup goroutine

**File:** `services/identity/internal/server/ttl_cache.go:10-65`  
**Severity:** P0

**Problem:** `globalTTLCache` is a package-level `var` — a singleton `ttlCache` with an unbounded `map[string]*cacheEntry`. The `Cleanup()` function exists (line 55) but is never called from any goroutine — it requires manual invocation. There is no background cleanup.

```go
var globalTTLCache = &ttlCache{entries: make(map[string]*cacheEntry)}
```

The `Invalidate(prefix)` method (line 44) iterates ALL entries on every call — O(n) scan under a write lock. If the cache has thousands of entries, this blocks all reads.

**Risk:** Same OOM risk as PERF-P0-3. Additionally, the `Invalidate` scan under write lock creates contention spikes when cache invalidation is triggered.

**Recommendation:**
1. Add a background cleanup goroutine (e.g., every 5 minutes)
2. Cap the entry count; use LRU or random eviction when full
3. Replace with Redis-backed cache in production (comment at line 9 acknowledges this)

---

### PERF-P1-1: EnsureSystemPermissions executes 60+ individual UPSERTs on startup

**File:** `pkg/rbac/permissions.go:229-261`  
**Severity:** P1

**Problem:** `EnsureSystemPermissions` iterates over all `SystemPermissions` (currently 60+) and executes one `INSERT ... ON CONFLICT` per permission. Each is a separate round-trip to the database.

**Risk:** Slow startup on every service instance. With 60 permissions and ~2ms per query, that's 120ms of sequential DB calls. Not critical, but unnecessary.

**Recommendation:** Use a single batch INSERT with `unnest()` or a CTE with `VALUES`:
```sql
INSERT INTO permissions (...)
SELECT * FROM unnest($1::text[], $2::text[], ...)
ON CONFLICT (key) DO UPDATE SET ...
```

---

### PERF-P1-2: Quota GetUsage executes 5 separate SELECT queries in a loop

**File:** `services/identity/internal/server/quota_handler.go:85-95`  
**Severity:** P1

**Problem:** `GetUsage` iterates over a map of 5 metrics and executes one `QueryRow` per metric:
```go
for metric, ptr := range metrics {
    r.pool.QueryRow(ctx, `SELECT value FROM tenant_usage WHERE tenant_id=$1 AND metric=$2`, tenantID, metric).Scan(&val)
    *ptr = val
}
```
This is 5 database round-trips for data that could be fetched in a single query.

**Recommendation:** Replace with:
```sql
SELECT metric, value FROM tenant_usage WHERE tenant_id=$1 AND metric IN (...)
```
Single round-trip, single result set.

---

### PERF-P1-3: Audit Insert has no batch insert path — single INSERT per event

**File:** `services/audit/internal/repository/audit_repo.go:31-93`  
**Severity:** P1

**Problem:** The `AuditRepository` has only `Insert()` for a single event. There is no `InsertBatch()` or `CopyFrom` method. Every audit event = one transaction + one INSERT + one SELECT FOR UPDATE (when hash chain is enabled).

The NATS consumer (`services/audit/internal/consumer/nats_consumer.go`) has `BatchSize` configuration, but the actual batch processing likely calls `Insert()` in a loop rather than a true batch insert.

**Risk:** Under high audit volume (login storms, bulk operations), the audit service becomes the bottleneck.

**Recommendation:** Add a `CopyFrom`-based batch insert for non-hash-chain events, or use a ` unnest()` batch INSERT for events where hash chain is disabled.

---

### PERF-P1-4: IntrospectionCache.GetCachedIntrospection uses write lock instead of read lock

**File:** `services/oauth/internal/service/introspection_cache.go:44-58`  
**Severity:** P1

**Problem:** `GetCachedIntrospection` acquires `c.mu.Lock()` (exclusive/write lock) even though it's only reading. This serializes all introspection cache lookups:
```go
func (c *IntrospectionCache) GetCachedIntrospection(token string) (*CachedIntrospection, bool) {
    c.mu.Lock()  // should be RLock with sync.RWMutex
    defer c.mu.Unlock()
```
The mutex is `sync.RWMutex` (line 27), so `RLock()` would allow concurrent reads. The stats updates (`c.stats.Hits++`) require a write lock, but those could use atomic counters instead.

**Risk:** Under high token introspection volume (>100/s), the exclusive lock creates a serialization point on the hottest OAuth path.

**Recommendation:** Use `RLock()` for reads with atomic counters for stats, or split stats into a separate mutex.

---

### PERF-P1-5: rebacCache.InvalidateOnWrite uses KEYS/SCAN under write pressure

**File:** `services/identity/internal/server/rebac_cache.go:66-80`  
**Severity:** P1

**Problem:** `InvalidateOnWrite` calls `c.rdb.Scan(ctx, 0, pattern, 100).Iterator()` to find all cached check results for an object. Redis SCAN is O(n) over the keyspace. While the count parameter (100) limits per-iteration work, a large Redis keyspace will make this slow.

**Risk:** When permissions change frequently (role updates, group membership changes), the SCAN-based invalidation creates Redis CPU spikes.

**Recommendation:** Use Redis Keyspace Notifications or maintain a secondary index (SET of cache keys per object) for O(1) invalidation.

---

### PERF-P2-1: Gateway router.go is 1523 lines with 40 functions

**File:** `services/gateway/internal/router/router.go`  
**Severity:** P2

**Problem:** The router file has grown to 1523 lines with 40 functions. This is a god-object that mixes routing, health checks, middleware setup, proxy configuration, and admin endpoints.

**Recommendation:** Split into separate files: `router_setup.go`, `router_health.go`, `router_admin.go`, `router_proxy.go`.

---

### PERF-P2-2: 51 json.NewEncoder.Encode calls ignore errors in gateway router

**File:** `services/gateway/internal/router/router.go` (51 instances)  
**Severity:** P2

**Problem:** The pattern `_ = json.NewEncoder(w).Encode(...)` appears 51 times in router.go. If encoding fails (client disconnect, write timeout), the error is silently swallowed.

**Assessment:** This is generally acceptable for HTTP response writes (the connection is already being written to, so recovery is limited). But it masks real serialization bugs.

---

### PERF-P2-3: Audit List query does count + data in two separate queries

**File:** `services/audit/internal/repository/audit_repo.go:152-201`  
**Severity:** P2

**Problem:** `List()` executes a `SELECT count(*)` query followed by the data query. Both scan the same rows. For large audit tables, this doubles the I/O cost.

**Recommendation:** Use window functions (`COUNT(*) OVER()`) or return approximate counts for large datasets.

---

## Code Quality Findings (Round 14)

### QUAL-P0-1: fmt.Errorf dominates error handling — pkg/errors coverage only 13.6%

**Files:** All services (cross-cutting)  
**Severity:** P0

**Problem:** Across all services (excluding tests):
- `fmt.Errorf`: **1,061 matches** in 199 files
- `pkg/errors` (ggiderrors): **167 matches** in 44 files

That's a ratio of **6.4:1** favoring `fmt.Errorf`. The custom `pkg/errors` package (which provides typed errors like `AlreadyExists`, `NotFound`, `ErrInternal`) is used in only ~13.6% of error paths.

**Impact:** Clients cannot reliably distinguish error types. A `404 Not Found` and a `500 Internal Server Error` may both surface as generic 500s because the underlying `fmt.Errorf` wrapping loses the error code. The gateway's error translation layer cannot map these consistently.

**Assessment:** The `pg_repo.go` in identity service shows the correct pattern (using `ggiderrors.AlreadyExists`, `ggiderrors.Wrap(ErrInternal, ...)`) but this pattern is not followed elsewhere. The `audit_repo.go` uses raw `fmt.Errorf` throughout, losing all error type information.

**Recommendation:** Systematically replace `fmt.Errorf` with `pkg/errors` wrappers in all repository and handler layers. Target: >80% coverage.

---

### QUAL-P1-1: Duplicated DBConfig / pool configuration across services

**Files:**  
- `services/identity/internal/data/db.go` — `DBConfig` struct  
- `services/audit/internal/data/db.go` — `Config` struct  
- `pkg/db/db.go` — `newPostgresPool()` defaults  

**Severity:** P1

**Problem:** Each service defines its own DB config struct and pool factory with slightly different defaults:

| Setting | identity | audit | pkg/db |
|---------|----------|-------|--------|
| MaxConns | 20 | env-configurable | 20 |
| MinConns | 2 | env-configurable | 5 |
| MaxConnLifetime | 1h | env-configurable | 30m |
| MaxConnIdleTime | 30m | not set | 5m |
| health_check_period | not set | not set | not set |

Three services, three different defaults. The identity service sets `MinConns=2` while `pkg/db` sets `MinConns=5`. The audit service doesn't set `MaxConnIdleTime` at all (uses pgxpool default of 30m).

**Risk:** Inconsistent pool behavior. Some services may exhaust connections faster than others. No `health_check_period` is set anywhere, meaning dead connections aren't proactively detected.

**Recommendation:** Consolidate into a single `pkg/db.NewDB()` factory used by all services, with env-var overrides for all parameters.

---

### QUAL-P1-2: Duplicated writeJSON/writeJSONError/http.Error patterns across services

**Files:** Multiple handler files across all services  
**Severity:** P1

**Problem:** `pkg/httputil` provides `WriteJSON`, `WriteError`, and `WriteJSONError`. Yet across services:
- `json.NewEncoder(w).Encode(...)` — 338 matches (many ignore errors with `_ =`)
- `json.Marshal(...)` — 242 matches
- Many handler files define their own local `writeJSON` / `writeJSONError` functions instead of importing `httputil`

The `httputil.WriteJSON` comment says "Shared implementation to eliminate 18+ duplicate copies across services" but the duplication persists — the shared function was created but adoption is incomplete.

**Recommendation:** Audit all handler files for local `writeJSON` definitions and replace with `httputil.WriteJSON`/`httputil.WriteError`.

---

### QUAL-P1-3: 213 os.Getenv calls scattered across 59 files instead of centralized config

**Files:** 59 non-test files across all services  
**Severity:** P1

**Problem:** Environment variables are read via `os.Getenv()` directly in business logic, handlers, and repositories — 213 times. This includes:
- `os.Getenv("GGID_ENV")` for environment detection (multiple files)
- `os.Getenv("GGID_GATEWAY_URL")` for gateway URL
- Various DB, Redis, and service-specific config vars

Each service has a `conf/conf.go` package, but it's not consistently used. Configuration values are read ad-hoc throughout the codebase.

**Impact:**
1. No validation — missing env vars silently produce empty strings
2. No type safety — everything is string, manual parsing everywhere
3. No central inventory of required configuration
4. Testing difficulty — must set specific env vars rather than inject config

**Recommendation:** All env var reads should go through each service's `conf` package. Add validation on startup.

---

### QUAL-P1-4: MySQL and SQLite pool factories skip all pool tuning

**File:** `pkg/db/db.go:212-231`  
**Severity:** P1

**Problem:** `newMySQLPool` and `newSQLitePool` call `sql.Open()` but never set `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`, or `SetConnMaxIdleTime`. The Go `database/sql` defaults are:
- MaxOpenConns: 0 (unlimited)
- MaxIdleConns: 2

For MySQL, unlimited connections can exhaust the MySQL server's connection pool. For SQLite, unlimited connections will cause "database is locked" errors under concurrency.

Only the PostgreSQL path gets proper pool tuning. This is a correctness issue for MySQL/SQLite deployments.

**Recommendation:**
```go
db.SetMaxOpenConns(20)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(30 * time.Minute)
db.SetConnMaxIdleTime(5 * time.Minute)
```

---

### QUAL-P1-5: Hashing functions duplicated across packages

**Files:**
- `services/identity/internal/repository/pg_repo.go:63-66` — `hashToken()`
- `services/oauth/internal/service/introspection_cache.go:91-94` — `hashToken()`

**Severity:** P1

**Problem:** Both files define identical `hashToken` functions:
```go
func hashToken(token string) string {
    h := sha256.Sum256([]byte(token))
    return hex.EncodeToString(h[:])
}
```
Same logic, same name, different packages. If one needs to change (e.g., add salt, switch algorithm), the other will be missed.

**Recommendation:** Move to a shared `pkg/crypto` or `pkg/util` package.

---

### QUAL-P2-1: CheckRoutePermission uses linear scan over route table

**File:** `pkg/rbac/route_permissions.go:132-143`  
**Severity:** P2

**Problem:** `CheckRoutePermission` iterates through the entire `RoutePermissions` slice (87 entries) for every API request. Each iteration calls `matchRoute()` which splits both pattern and path by `/`.

**Impact:** On a system handling 1000 req/s, this is 87,000 string-split-and-compare operations per second. Not severe, but wasteful.

**Recommendation:** Pre-compute a map keyed by `(method, first-path-segment)` to narrow the search space, or use a trie/radix tree.

---

### QUAL-P2-2: HasPermission uses fmt.Sscanf for every permission check

**File:** `pkg/rbac/permissions.go:58-82`  
**Severity:** P2

**Problem:** `HasPermission` calls `fmt.Sscanf(p, "%[^:]:%[^:]:%s", &pr, &pa, &ps)` inside a loop over all user permissions. `fmt.Sscanf` is surprisingly slow (format string parsing + reflection).

For a user with 50 permissions, each RBAC check does 50 `Sscanf` calls. On a 1000 req/s system with 50-permission users, that's 50,000 `Sscanf` calls per second.

**Recommendation:** Use `strings.SplitN(p, ":", 3)` — 10x faster than `Sscanf`.

---

### QUAL-P2-3: Magic numbers throughout quota defaults

**File:** `services/identity/internal/server/quota_handler.go:50-53, 126`  
**Severity:** P2

**Problem:** Quota defaults are hardcoded as SQL string literals AND in Go code:
```go
max_users INT DEFAULT 100, max_api_keys INT DEFAULT 5, ...
```
```go
return &TenantQuota{..., MaxUsers: 100, MaxAPIKeys: 5, ...}
```
The same numbers appear twice. If one changes, the other becomes inconsistent.

**Recommendation:** Define constants:
```go
const DefaultMaxUsers = 100
const DefaultMaxAPIKeys = 5
```
And use them in both SQL generation and Go defaults.

---

### QUAL-P2-4: 17 console.log/debug calls in production frontend code

**Files:**  
- `console/src/lib/webauthn-conditional.ts` — 15 instances  
- `console/src/lib/performance.ts` — 1 instance  
- `console/src/components/PWARegister.tsx` — 1 instance  

**Severity:** P2

**Problem:** `console.log`/`console.debug` calls in production code will leak debugging information to the browser console. The `webauthn-conditional.ts` file has 15 instances — likely detailed passkey/WebAuthn flow logging.

**Recommendation:** Use a logging library with level control (e.g., `pino`, `winston`) or wrap in `if (import.meta.env.DEV)` guards.

---

### QUAL-P2-5: IntrospectionCache has redundant TTL fields

**File:** `services/oauth/internal/service/introspection_cache.go:26-33`  
**Severity:** P2

**Problem:** The struct has three TTL-related fields:
```go
activeTTL   time.Duration  // 60s
inactiveTTL time.Duration  // 5m
ttl         time.Duration  // 60s (same as activeTTL)
```
The `ttl` field appears unused (the `ttlFor()` method uses `activeTTL`/`inactiveTTL`). This is dead code / confusing duplication.

**Recommendation:** Remove the `ttl` field if unused, or consolidate the TTL logic.

---

## Goroutine Safety Assessment

**Positive findings:**
- Gateway webhook delivery (`webhooks.go:304`) has `recover()` 
- Metering flush goroutine (`middleware/metering.go:124`) has `recover()`
- RBAC invalidation listener (`middleware/rbac_dynamic.go:145`) has `recover()`
- Timeout middleware goroutine (`middleware/timeout.go:91`) has `recover()` with explicit comment about panic recovery
- JTI blocklist cleanup goroutine (`auth/cmd/main.go:301`) runs as a `for{}` loop — no leak

**Concerns:**
- The JTI blocklist goroutine (`auth/cmd/main.go:301-304`) has no shutdown signal — it runs forever and cannot be stopped gracefully. On shutdown, the goroutine leaks until process exit.
- Various `go func()` calls in main.go files (gateway, auth, policy) for HTTP server listeners are standard Go patterns and acceptable.

## Cache Strategy Assessment

| Cache | Backend | TTL | Size Limit | Eviction | Risk |
|-------|---------|-----|------------|----------|------|
| `globalTTLCache` (identity) | In-memory | Variable | None | None | OOM |
| `IntrospectionCache` (oauth) | In-memory | 60s/5m | None | None | OOM |
| `rebacCache` (identity) | Redis | 60s | N/A (Redis) | TTL | Safe |
| `IntrospectionCache` stats | In-memory | N/A | N/A | N/A | Safe |

Two of four cache implementations are unbounded in-memory maps with no eviction. These are P0 reliability risks.

## Database Pool Assessment

| Service | MaxConns | MinConns | Lifetime | IdleTime | HealthCheck |
|---------|----------|----------|----------|----------|-------------|
| identity | 20 | 2 | 1h | 30m | Not set |
| audit | Env-config | Env-config | Env-config | Not set | Not set |
| pkg/db (postgres) | 20 | 5 | 30m | 5m | Not set |
| pkg/db (mysql) | unlimited | 2 | default | default | Not set |
| pkg/db (sqlite) | unlimited | 2 | default | default | Not set |

**Missing everywhere:** `health_check_period` — pgxpool's proactive dead connection detection is disabled by default. Stale connections will cause request failures until the retry logic kicks in.

---

## Conclusion

The most critical findings are:
1. **PERF-P0-1**: Audit hash chain serialization will bottleneck the entire system under load
2. **PERF-P0-2**: Bulk import can hold the identity service hostage for 10+ minutes  
3. **PERF-P0-3/P0-4**: Two unbounded in-memory caches will cause OOM under sustained load
4. **QUAL-P0-1**: 86% of error paths use `fmt.Errorf` instead of typed errors, breaking client-side error handling

The codebase shows strong architectural patterns (RLS-based tenant isolation, hash chain audit integrity, proper transaction handling) but suffers from performance-blind implementation details in hot paths. The error handling inconsistency is the most impactful quality issue — it affects every API consumer's ability to handle errors correctly.
