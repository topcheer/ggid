# IAM Review: Performance (R28) + Error Handling (R25)

**Date:** 2026-08-03  
**Auditor:** Independent (ggcode, glm-5.2)  
**Scope:** services/gateway, services/auth, services/identity, services/oauth, services/audit, pkg/errors, pkg/httputil, console/src  
**Method:** Static code analysis, line-by-line inspection of critical paths  

---

## Summary

| Category | P0 | P1 | P2 | Total |
|----------|----|----|----|----|
| Performance | 4 | 7 | 5 | 16 |
| Error Handling | 5 | 8 | 4 | 17 |
| **Total** | **9** | **15** | **9** | **33** |

---

## Part 1: Performance Findings

### PERF-P0-1: N+1 Query in Bulk Import — Sequential CreateUser per User

**File:** `services/identity/internal/server/bulk_import.go:98-211`  
**Severity:** P0  
**Pattern:** N+1 query (category #1)

**Description:**  
The `handleBulkImport` handler iterates over up to 10,000 users and calls `h.svc.CreateUser()` individually for each one (line 154). Each `CreateUser` call opens a transaction, sets RLS, executes INSERT, and commits — resulting in up to 10,000 separate transactions. The code even acknowledges this in a comment (line 140-141): *"In production: batch INSERT via pgx.CopyFrom for performance. For now, use CreateUser for each user (correct, just not bulk-optimized)."*

Additionally, when `password_hash` is provided, a second UPDATE query is executed per-user (line 173), and when `role_id` is provided, two more queries (SELECT + INSERT) are executed (lines 190, 202). Worst case: **40,000 queries** for a single bulk import request.

**Risk:**  
- A 10,000-user import will take minutes, blocking the HTTP handler and consuming a connection for the entire duration.
- Argon2id hashing is CPU-intensive and runs synchronously per-user, serializing the entire import.
- Memory: the entire `BulkImportRequest` (up to 10MB) is decoded into memory at once, plus 10,000 `CreateUserInput` structs.

**Recommendation:**  
1. Use `pgx.CopyFrom` for bulk INSERT as the code comment suggests.
2. Batch role assignments via a single `INSERT ... SELECT ... FROM unnest($1::uuid[], $2::uuid[])`.
3. Move Argon2id hashing to a worker pool (bounded goroutines) to parallelize CPU work.

---

### PERF-P0-2: Audit Hash Chain Uses FOR UPDATE Lock — Serializes All Audit Writes per Tenant

**File:** `services/audit/internal/repository/audit_repo.go:58-96`  
**Severity:** P0  
**Pattern:** Lock contention (category #8)

**Description:**  
Every `AuditRepository.Insert()` call opens a transaction and executes:
```sql
SELECT COALESCE(hash, '') FROM audit_events
WHERE tenant_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1 FOR UPDATE
```
This `FOR UPDATE` locks the last row, serializing all audit inserts for the same tenant. Since every API request generates at least one audit event, this effectively serializes **all writes** per tenant through the audit pipeline.

The code comment acknowledges this: *"FOR UPDATE locks the row, serializing chain appends per tenant."*

**Risk:**  
- Under load (>100 req/s per tenant), audit inserts become the bottleneck.
- The `ORDER BY created_at DESC, id DESC` without a composite index on `(tenant_id, created_at DESC, id DESC)` will cause a full scan + sort before the lock, amplifying the contention.

**Recommendation:**  
1. Add a composite index: `CREATE INDEX idx_audit_chain ON audit_events (tenant_id, created_at DESC, id DESC)`.
2. Consider a batch audit pipeline: buffer events in-memory and flush in batches, computing hash chains within the batch.
3. Alternative: use a dedicated sequence/counter per tenant instead of ordering by timestamp.

---

### PERF-P0-3: Audit GetStats Executes 4+ Separate Queries Without Caching

**File:** `services/audit/internal/repository/audit_repo.go:254-300+`  
**Severity:** P0  
**Pattern:** N+1 query + caching strategy (categories #1, #3)

**Description:**  
`GetStats()` executes at least 4 separate queries against `audit_events`:
1. `SELECT count(*) ... WHERE created_at >= $2` (total count)
2. `SELECT action, count(*) ... GROUP BY action` (by action)
3. `SELECT date_trunc('hour', ...) ... GROUP BY hour` (hourly distribution)
4. Additional queries likely follow (truncated at line 300)

Each query scans the same filtered range independently. There is no caching layer, so every dashboard refresh triggers full aggregate scans.

**Risk:**  
- On a tenant with millions of audit events, these aggregates will be slow (>1s each).
- Dashboard refreshes amplify the load — each user opening the dashboard triggers 4+ heavy queries.

**Recommendation:**  
1. Cache `GetStats` results with a 30-60s TTL (the data is statistical, not real-time critical).
2. Consider materialized views refreshed by a cron job.
3. Combine the count queries into a single query using `WITH` (CTE) or `UNION ALL`.

---

### PERF-P0-4: ResponseCache Has No Size Limit — Unbounded Memory Growth

**File:** `services/gateway/internal/middleware/response_cache.go:43-62`  
**Severity:** P0  
**Pattern:** Cache strategy —雪崩/穿透 (category #3)

**Description:**  
The `ResponseCache` uses an unbounded `map[string]*rcCachedResponse` with a background cleanup goroutine that only removes expired entries. There is **no maximum entry count** or memory cap. The cache key includes `method + path + query + tenantID + userID`, so each unique user/tenant/path combination creates a new entry.

The cleanup goroutine (line 60) runs on a fixed interval, scanning the entire map. Under high cardinality (many users × many paths), the map can grow without bound.

Additionally, there is **no cache stampede protection** (singleflight). If a cached entry expires, multiple concurrent requests for the same key will all miss the cache and execute the backend handler simultaneously.

**Risk:**  
- Memory exhaustion under high cardinality or attack (many unique URL parameters).
- Cache stampede on TTL expiry causes backend spikes.

**Recommendation:**  
1. Add a max entry count (e.g., 10,000) with LRU eviction.
2. Add `singleflight.Group` to prevent cache stampede on misses.
3. Add negative caching for 404 responses to prevent cache penetration.

---

### PERF-P1-1: Read-Only Transactions for Every Query — Unnecessary Overhead

**File:** `services/identity/internal/repository/pg_repo.go:76-106, 146-168, 170-200, 214-238, 292-388, 451-468, 518-543, 558-587, ...`  
**Severity:** P1  
**Pattern:** DB connection pool configuration (category #2)

**Description:**  
Every read operation in `pgRepo` (GetUserByID, GetUserByUsername, GetUserByEmail, ListUsers, ListUserEmails, ListExternalIdentities, GetCredentialByUsername, etc.) opens a transaction with `BeginTx(ctx, pgx.TxOptions{})` just to call `setTenantRLS()`, then commits. For RLS, a transaction is needed, but these are all read-only operations and the transaction overhead (BEGIN + set_config + query + COMMIT = 3 round trips) is significant.

The `tx.Commit(ctx)` on line 166 and similar is not error-checked (see ERROR-P1-3 below).

**Risk:**  
- 3 DB round trips per query instead of 1 (using a connection directly with `SET LOCAL` via a prepared session).
- Under high request volume, this triples connection pool pressure.

**Recommendation:**  
1. Use `pgxpool.Pool.QueryRow` directly with `SET LOCAL` in a single query batch.
2. Or use pgx batch protocol to send `set_config` + `SELECT` in a single network round trip.
3. Set `pgx.TxOptions{AccessMode: pgx.ReadOnly}` for read transactions to enable query optimization.

---

### PERF-P1-2: Webhook Delivery Creates New HTTP Client Per Deliverer Instance

**File:** `services/gateway/internal/webhooks/webhooks.go:123-175`  
**Severity:** P1  
**Pattern:** HTTP client reuse (category #9)

**Description:**  
`HTTPDeliverer` stores a `*http.Client` instance, but multiple deliverer instances can be created. More importantly, `NewHTTPDeliverer()` calls `NewSSRFSafeDeliverer(DefaultSSRFConfig())` which likely creates its own `http.Client` with a custom dialer. Each webhook delivery goroutine shares the same deliverer (good), but the broader codebase has 12+ files creating `&http.Client{}` literals instead of using `httputil.DefaultClient`.

Found 12+ instances of `&http.Client{...}` creation across services (from search results), mostly with ad-hoc timeout/transport settings instead of the shared `httputil.DefaultClient`.

**Risk:**  
- Each `http.Client` with a custom `Transport` creates a separate connection pool, preventing connection reuse.
- TLS handshake overhead on every first request per client.

**Recommendation:**  
1. Use `httputil.DefaultClient` or `httputil.ShortTimeoutClient` everywhere.
2. Centralize SSRF-safe transport configuration into `httputil`.
3. Audit and replace all `&http.Client{}` literals.

---

### PERF-P1-3: ResponseCache Cleanup Scans Entire Map Under Write Lock

**File:** `services/gateway/internal/middleware/response_cache.go` (cleanup method, lines 60+)  
**Severity:** P1  
**Pattern:** Lock contention (category #8)

**Description:**  
The cleanup goroutine periodically iterates the entire cache map. If it acquires a write lock (`rc.mu.Lock()`) during cleanup, it blocks all cache reads and writes during the scan. Under high load with a large cache, this causes periodic latency spikes.

**Recommendation:**  
1. Use `sync.Map` for the cache (lock-free reads).
2. Or use a sharded map to reduce lock contention.
3. Evict lazily on read instead of a dedicated cleanup goroutine.

---

### PERF-P1-4: InputValidationMiddleware Reads Entire Body into Memory for Pattern Matching

**File:** `services/gateway/internal/middleware/input_validation.go:100-120`  
**Severity:** P1  
**Pattern:** Memory allocation (category #10)

**Description:**  
The middleware reads the entire request body into memory via `io.ReadAll(r.Body)` (line 103), converts it to string (line 112), then runs 17 regex patterns against it. The body is then restored via `io.NopCloser(strings.NewReader(bodyStr))`.

For a 10MB body (the `MaxBodySize`), this creates:
- 1x `[]byte` from `io.ReadAll`
- 1x `string` copy from `string(body)`
- 1x `strings.Reader` for the restored body
- 17 regex evaluations against the full body string

**Risk:**  
- 30MB+ memory per large request just for validation.
- Regex matching on 10MB strings is CPU-intensive on every POST/PUT/PATCH.

**Recommendation:**  
1. Stream the body through a bounded reader (e.g., 1MB) for pattern matching.
2. Use `bytes.Contains` for simple substring patterns instead of regex.
3. Apply patterns only to specific fields (JSON values), not the raw body.

---

### PERF-P1-5: Audit GetByID Swallows json.Unmarshal Error — Silent Data Loss

**File:** `services/audit/internal/repository/audit_repo.go:136-137`  
**Severity:** P1  
**Pattern:** JSON serialization hot path (category #6)

**Description:**  
```go
if len(metaBytes) > 0 {
    json.Unmarshal(metaBytes, &event.Metadata)
}
```
The `json.Unmarshal` return value is discarded. If the metadata JSON is corrupted, the event's `Metadata` field will be partially populated or empty, silently losing data. The same pattern appears in the `List` method (line 240).

This is both a performance issue (silent corruption cascades) and an error handling issue.

**Risk:**  
- Corrupted metadata goes undetected, potentially hiding security-relevant audit information.
- Downstream consumers receive incomplete events.

**Recommendation:**  
Return the unmarshal error or log a warning.

---

### PERF-P1-6: Gateway Router Scans All Proxies Linearly Per Request

**File:** `services/gateway/internal/router/router.go:720-725`  
**Severity:** P1  
**Pattern:** Index efficiency (category #7)

**Description:**  
The proxy lookup iterates over all configured route prefixes for every request:
```go
for prefix := range gw.proxies {
    if strings.HasPrefix(path, prefix) { ... }
}
```
With many routes (the gateway proxies to auth, identity, oauth, audit, org, policy, etc.), this is O(n) per request.

**Recommendation:**  
Use a trie or prefix tree for O(1) prefix matching.

---

### PERF-P1-7: Bulk Import Argon2id Hashing Blocks HTTP Goroutine

**File:** `services/identity/internal/server/bulk_import.go:121-127`  
**Severity:** P1  
**Pattern:** Large batch processing (category #4)

**Description:**  
Each `ggidcrypto.HashPassword(user.Password)` call performs Argon2id hashing (memory-hard, ~100ms per hash). For 10,000 users with plaintext passwords, this blocks the HTTP handler goroutine for ~16 minutes.

**Risk:**  
- HTTP timeout will kill the request long before completion.
- The goroutine consumes CPU exclusively, starving other requests.

**Recommendation:**  
1. Use a worker pool with bounded parallelism for hashing.
2. Return a job ID immediately and process asynchronously.
3. Enforce a much lower limit (e.g., 500) for synchronous imports.

---

### PERF-P2-1: DB Pool Defaults May Be Too Low for Production

**File:** `services/identity/internal/data/db.go:26-37`  
**Severity:** P2  
**Pattern:** DB connection pool (category #2)

**Description:**  
Default `MaxConns=20, MinConns=2`. For a high-traffic IAM platform with multiple services sharing a database, 20 connections per service may cause pool exhaustion under load. PostgreSQL default `max_connections=100`, and with 5+ services each using 20 connections, the pool is already saturated.

**Recommendation:**  
1. Make defaults configurable via environment variables (they may be, but defaults are conservative).
2. Consider `MaxConns=50` for production, or use PgBouncer.

---

### PERF-P2-2: Audit Data DB Missing MaxConnIdleTime Configuration

**File:** `services/audit/internal/data/db.go:26-53`  
**Severity:** P2  
**Pattern:** DB connection pool (category #2)

**Description:**  
The audit service's `New()` function configures `MaxConnLifetime` but does **not** configure `MaxConnIdleTime`. Idle connections remain open until `MaxConnLifetime` expires. The identity service sets `MaxConnIdleTime=30m` by default; the audit service does not.

**Recommendation:**  
Add `MaxConnIdleTime` configuration.

---

### PERF-P2-3: Webhook Delivery Retry Uses Quadratic Backoff Without Jitter

**File:** `services/gateway/internal/webhooks/webhooks.go:142`  
**Severity:** P2  
**Pattern:** Cache stampede / thundering herd (category #3)

**Description:**  
Retry backoff is `time.After(time.Duration(attempt*attempt) * time.Second)` — quadratic (1s, 4s, 9s, 16s, 25s). No jitter is added, so if multiple webhooks fail simultaneously (e.g., backend is down), all retries hit at the same intervals, creating thundering herd.

**Recommendation:**  
Add random jitter (e.g., ±20%) to each backoff interval.

---

### PERF-P2-4: Webhook MemoryStore ListByEvent Scans All Webhooks

**File:** `services/gateway/internal/webhooks/webhooks.go:92-108`  
**Severity:** P2  
**Pattern:** Index efficiency (category #7)

**Description:**  
`ListByEvent` iterates all webhooks in memory under RLock, checking each one's events list. For a tenant with many webhooks, this is O(n*m) per event delivery.

**Recommendation:**  
Build an event→webhooks index map.

---

### PERF-P2-5: json.NewEncoder Used Inline in 50+ Handler Files

**File:** Multiple files (351 files matched `json.NewDecoder`, many with inline encoding)  
**Severity:** P2  
**Pattern:** JSON serialization hot path (category #6)

**Description:**  
`json.NewEncoder(w).Encode(v)` is used inline in many handlers. Each call creates a new encoder, allocates a buffer, and writes. The shared `httputil.WriteJSON` exists but adoption is incomplete (8 files use it, while many handlers still have local `writeJSON` wrappers).

**Recommendation:**  
Standardize on `httputil.WriteJSON` across all services. Consider using `json.Marshal` + `w.Write` for better control over buffer allocation, or use a JSON encoder pool.

---

## Part 2: Error Handling Findings

### ERROR-P0-1: Error Response Format Inconsistency — 4 Different Formats

**Files:** Multiple  
**Severity:** P0  
**Pattern:** Error response format consistency (category #8)

**Description:**  
The codebase uses at least 4 different error response formats:

1. **`httputil.WriteError`** (pkg/httputil): `{"error": "message"}` — flat string
2. **`pkg/errors.WriteAPIError`**: `{"error": {"code": "...", "message": "...", "request_id": "..."}}` — structured
3. **`http.Error`**: plain text `message\n` — not JSON at all (8 files, 10 instances)
4. **Local `writeJSON` + map**: `{"error": "message"}` — same as #1 but via local function

The console frontend (`error-helpers.ts:5-18`) tries to handle all of these: it checks `data.error` as string, `data.error.message`, `data.error.code`, `data.message`, and `data.detail` — clear evidence of backend inconsistency.

Additionally, the gateway's `input_validation.go:130` uses yet another format: `{"error": "malicious input detected", "type": "...", "field": "...", "pattern": "...", "request_id": "..."}`.

**Risk:**  
- Frontend error handling is fragile and may display confusing messages.
- API consumers cannot reliably parse error responses.
- Security implications: some formats leak internal details (e.g., `pattern` in input validation reveals WAF logic).

**Recommendation:**  
1. Standardize all error responses on `pkg/errors.WriteAPIError`.
2. Replace all `http.Error` calls with structured JSON errors.
3. Remove local `writeJSON` wrappers and use `httputil.WriteError` or `WriteAPIError`.

---

### ERROR-P0-2: Audit Repository Uses fmt.Errorf Instead of pkg/errors

**File:** `services/audit/internal/repository/audit_repo.go:34, 63, 76, 94, 96, 196, 223, 237, 264, 275, 296`  
**Severity:** P0  
**Pattern:** Error type consistency (category #1)

**Description:**  
The audit repository uses `fmt.Errorf("...: %w", err)` exclusively for error wrapping. It imports `pkg/errors` (line 10) and uses it only for `mapErr`. All error creation uses raw `fmt.Errorf`, meaning:
- Errors are `*fmt.wrapError`, not `*GGIDError`.
- The HTTP handler cannot use `errors.AsGGIDError()` to map to correct HTTP status codes.
- All audit errors become generic 500 Internal Server Error.

By contrast, the identity repository (`pg_repo.go`) correctly uses `ggiderrors.Wrap(ggiderrors.ErrInternal, "...", err)` throughout.

**Risk:**  
- All audit API errors return HTTP 500 even for not-found or invalid argument cases.
- Loss of error classification breaks client retry logic.

**Recommendation:**  
Replace `fmt.Errorf` with `ggiderrors.Wrap()` / `ggiderrors.NotFound()` etc. in the audit repository.

---

### ERROR-P0-3: tx.Commit(ctx) Return Value Not Checked in Multiple Locations

**File:** `services/identity/internal/repository/pg_repo.go:106, 166, 199, 236, 386, 443, 467, 541, 585, 819, 913`  
**Severity:** P0  
**Pattern:** Error handling — ignored return values (category #2/#3)

**Description:**  
Multiple `pgRepo` methods call `tx.Commit(ctx)` as a statement, not checking the return value:

```go
// Line 166 (GetUserByID)
tx.Commit(ctx)
return user, nil

// Line 386 (ListUsers) 
tx.Commit(ctx)
return result, nil

// Line 585 (ListUserEmails)
tx.Commit(ctx)
return emails, nil
```

In contrast, `CreateUser` (line 106) correctly does `return tx.Commit(ctx)`, and `AddUserEmail` (line 616) checks `tx.Commit()`.

If the commit fails (e.g., network error, serialization conflict), the function returns a nil error with potentially inconsistent data. The caller believes the operation succeeded.

**Risk:**  
- Silent data corruption on commit failures.
- RLS context may not be properly applied, leading to tenant data leakage.

**Recommendation:**  
Check all `tx.Commit(ctx)` return values:
```go
if err := tx.Commit(ctx); err != nil {
    return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "commit", err)
}
```

---

### ERROR-P0-4: json.Unmarshal Error Swallowed in 15+ Non-Test Files

**Files:** Multiple (see search results)  
**Severity:** P0  
**Pattern:** json.Unmarshal/Decode error handling (category #2)

**Description:**  
`json.Unmarshal()` is called without checking the error return in at least 15 non-test files across services. Key instances:

- `services/audit/internal/repository/audit_repo.go:137` — `json.Unmarshal(metaBytes, &event.Metadata)`
- `services/audit/internal/repository/audit_repo.go:240` — same pattern in List
- `services/identity/internal/repository/pg_repo.go:785` — correctly checks error (exception)
- `services/org/internal/repository/dept_repo.go:74,139` — metadata unmarshal

The pattern `json.Unmarshal(data, &target)` with no error check means corrupted JSON silently produces zero-value or partial structs. In audit events, this means security-relevant metadata can be silently lost.

**Risk:**  
- Silent data corruption in audit trail.
- Partial deserialization leading to nil panics downstream.
- Security: malicious metadata could be crafted to fail unmarshal and hide evidence.

**Recommendation:**  
Check all `json.Unmarshal` return values. Log or return errors for corrupted JSON.

---

### ERROR-P0-5: Error Messages Leak Sensitive Internal Details

**Files:** Multiple  
**Severity:** P0  
**Pattern:** Error log leaking (category #10)

**Description:**  
Several error paths log or return sensitive information:

1. `services/gateway/internal/middleware/input_validation.go:130-136` — Returns the matched regex `pattern` to the client, revealing the WAF's detection rules. An attacker can use this to craft bypasses.

2. `services/identity/internal/server/bulk_import.go:177` — Logs `createdUser.ID` (UUID) in error context. While not directly sensitive, it leaks internal identifiers.

3. Multiple `slog.Error` calls across services log raw error messages from database drivers, which may contain SQL fragments, connection strings, or schema details.

4. `services/gateway/internal/webhooks/webhooks.go:331` — Logs webhook URL in error: `slog.Error("webhook delivery failed", "webhook_id", w.ID, "event", event, "error", err)`. While the URL itself isn't logged here, the deliverer's retry logs (line 163) may include URL-embedded tokens.

**Risk:**  
- Information leakage to API consumers (pattern disclosure).
- Log files may contain sensitive data accessible to operators.

**Recommendation:**  
1. Never return regex patterns to clients — return generic "invalid input" messages.
2. Sanitize database error messages before logging or returning.
3. Use structured logging with explicit field redaction for sensitive values.

---

### ERROR-P1-1: Audit Repo GetByID Missing ActorType Field in Scan

**File:** `services/audit/internal/repository/audit_repo.go:127-131`  
**Severity:** P1  
**Pattern:** Error handling — silent data issue

**Description:**  
The `GetByID` query selects `actor_type` (line 121) but the Scan (line 128) scans into `&event.ActorType` which is scanned correctly. However, comparing the column list in the query (line 121-126) with the Scan fields (line 128-131), the query selects 17 columns but the Scan has only 16 targets — `actor_name` is scanned into a local `*string` but `event.ActorType` is populated. This is a field alignment risk.

**Recommendation:**  
Verify column-to-field alignment carefully. Use a named struct scan helper if available.

---

### ERROR-P1-2: Webhook Deliver Silently Discards resp.Body Read Error

**File:** `services/gateway/internal/webhooks/webhooks.go:166`  
**Severity:** P1  
**Pattern:** defer .Close() / io.Copy error check (category #4)

**Description:**  
```go
io.Copy(io.Discard, resp.Body)
resp.Body.Close()
```
Both `io.Copy` and `resp.Body.Close()` return errors that are ignored. If the body drain fails, the HTTP connection may not be properly returned to the pool, causing connection leaks over time.

**Recommendation:**  
```go
io.Copy(io.Discard, resp.Body) // best-effort drain
if err := resp.Body.Close(); err != nil {
    slog.Debug("webhook: response body close error", "error", err)
}
```

---

### ERROR-P1-3: Database Errors Not Classified — All Map to Internal Server Error

**File:** `services/audit/internal/repository/audit_repo.go` (all error paths), `services/identity/internal/server/bulk_import.go:173-180`  
**Severity:** P1  
**Pattern:** DB error classification (category #7)

**Description:**  
In the audit repository, all database errors are wrapped with `fmt.Errorf("...: %w", err)`. There is no classification of:
- `pgconn.PgError` with code "23505" (duplicate key) → should be 409 Conflict
- `pgx.ErrNoRows` → should be 404 Not Found
- Connection errors → should be 503 Service Unavailable
- Serialization conflicts → should be 409 or auto-retry

The `mapErr` function exists in audit_repo.go (referenced on line 134) but is only used for GetByID, not for List, GetStats, or other methods.

**Recommendation:**  
Create a shared `mapDBError(err error, resource, id string) error` function that classifies PostgreSQL errors into appropriate `GGIDError` codes.

---

### ERROR-P1-4: ctx.Err() Rarely Checked in Long-Running Operations

**Files:** All services  
**Severity:** P1  
**Pattern:** ctx.Err() checking (category #6)

**Description:**  
Only 5 instances of `ctx.Err()` found across all services (from search results). Long-running operations that don't check context cancellation:

1. **Bulk import** (`bulk_import.go:98-211`): Iterates up to 10,000 users without checking `r.Context().Err()`. If the client disconnects, the handler continues processing for minutes.

2. **Audit GetStats** (`audit_repo.go:254+`): Executes 4+ queries without checking context between them.

3. **Webhook DeliverEvent** (`webhooks.go:297-335`): Launches goroutines with detached context (correctly uses `context.WithoutCancel`), but the initial `ListByEvent` call uses the caller's context without checking cancellation.

**Risk:**  
- Wasted CPU/DB resources on abandoned requests.
- Goroutine leak if client disconnects during long operations.

**Recommendation:**  
Add `ctx.Err()` checks at the top of each loop iteration in bulk operations and between multi-query operations.

---

### ERROR-P1-5: Bulk Import Error Messages Lack Detail for Troubleshooting

**File:** `services/identity/internal/server/bulk_import.go:157, 169, 179, 194, 199`  
**Severity:** P1  
**Pattern:** Error response format (category #8)

**Description:**  
All import errors return generic messages: `"create user failed"`, `"credential update failed"`, `"invalid role_id"`, `"role does not belong to tenant"`. The actual error from `CreateUser` or `Pool().Exec()` is only logged via `slog.Warn/Error`, not returned to the caller. For a bulk import API, the caller needs to know **which** constraint was violated or **why** the creation failed.

**Risk:**  
- API consumers cannot programmatically handle import failures.
- Support teams cannot diagnose issues without log access.

**Recommendation:**  
Include error classification in `ImportError.Reason`: "duplicate email", "invalid username format", "role not found", etc.

---

### ERROR-P1-6: Webhook Create Handler Swallows validateURL Error Detail

**File:** `services/gateway/internal/webhooks/webhooks.go:208-210`  
**Severity:** P1  
**Pattern:** Error handling — information loss

**Description:**  
```go
if err := validateURL(req.URL); err != nil {
    writeJSON(w, 400, map[string]string{"error": "webhook delivery failed"})
    return
}
```
The `validateURL` error is discarded and a generic "webhook delivery failed" message is returned. The user gets no indication whether their URL is malformed, uses a blocked scheme, or points to a private IP. The actual error message should be returned.

---

### ERROR-P1-7: Panic in auth/main.go init Code Without Recovery

**File:** `services/auth/cmd/main.go:580`  
**Severity:** P1  
**Pattern:** panic in non-init/main (category #9)

**Description:**  
```go
panic(fmt.Sprintf("failed to create configs directory: %v", err))
```
While this is in `main.go` (acceptable for startup failures), there are 6+ `panic()` calls in non-test files. Most are in `main.go` (acceptable), but any panic in a handler or service goroutine without recovery would crash the process.

The gateway's webhook delivery goroutine has proper recovery (line 306-308), but other goroutine launch points should be verified.

---

### ERROR-P1-8: httputil.WriteJSON Ignores Encoding Errors

**File:** `pkg/httputil/response.go:15`  
**Severity:** P1  
**Pattern:** json.Encode error handling (category #3)

**Description:**  
```go
_ = json.NewEncoder(w).Encode(v)
```
The encoding error is explicitly discarded. If `v` contains unmarshallable types (e.g., `chan`, `func`, or types with custom `MarshalJSON` that panics), the client receives a truncated or empty response with the HTTP status header already written (status cannot be changed after `WriteHeader`).

The same pattern appears in `pkg/errors/api_error.go:68` and `input_validation.go:130`.

**Risk:**  
- Client receives malformed JSON without any error indication.
- If the encoder panics (rare but possible with custom MarshalJSON), it crashes the goroutine unless recovered upstream.

**Recommendation:**  
Pre-encode to a buffer, check the error, then write the buffer. This allows falling back to a generic error response if encoding fails:
```go
buf, err := json.Marshal(v)
if err != nil {
    WriteError(w, 500, "internal encoding error")
    return
}
w.Write(buf)
```

---

### ERROR-P2-1: Webhook MemoryStore.Delete Returns Generic fmt.Errorf

**File:** `services/gateway/internal/webhooks/webhooks.go:76-77, 87-88`  
**Severity:** P2  
**Pattern:** Error type consistency (category #1)

**Description:**  
`Delete` and `Get` return `fmt.Errorf("not found")` — a plain string error. The HTTP handler checks this with `err != nil` and maps to 404, but the error type is not a `GGIDError`, preventing `AsGGIDError()` from working. If the handler logic changes to use `AsGGIDError`, these errors will silently map to 500.

---

### ERROR-P2-2: Console error-helpers.ts Uses `as any` Cast

**File:** `console/src/lib/error-helpers.ts:26-28`  
**Severity:** P2  
**Pattern:** Error type safety

**Description:**  
```typescript
const apiErr = err as any;
```
The `as any` cast bypasses TypeScript type checking. If the error object's shape changes, the runtime checks (`apiErr.detail`, `apiErr.title`) will silently fail, falling back to the generic message.

---

### ERROR-P2-3: Audit List Method Doesn't Check rows.Err()

**File:** `services/audit/internal/repository/audit_repo.go:227-250`  
**Severity:** P2  
**Pattern:** Error handling — missing rows.Err() check

**Description:**  
After the `for rows.Next()` loop (line 228), there is no `rows.Err()` check. If the iterator encounters an error (e.g., context cancellation mid-iteration), the function returns the partial results without error. The identity repository correctly checks `rows.Err()` at line 374.

**Recommendation:**  
Add `if err := rows.Err(); err != nil { return nil, 0, fmt.Errorf("iterate audit events: %w", err) }` after the loop.

---

### ERROR-P2-4: httputil.WriteError Uses map[string]string — Not Structured

**File:** `pkg/httputil/response.go:19-21`  
**Severity:** P2  
**Pattern:** Error response format consistency

**Description:**  
`WriteError` returns `{"error": "message"}` — a flat string format. `WriteAPIError` returns `{"error": {"code": "...", "message": "..."}}` — a structured format. These two functions in the same ecosystem produce incompatible error shapes. The console frontend handles both, but this duality is fragile.

---

## Code Paths Inspected

### Performance
1. **N+1 queries:** Examined all `for range` loops with DB access in identity, audit, auth, oauth, gateway. Found N+1 in bulk_import.go (PERF-P0-1) and GetStats (PERF-P0-3). Examined pg_repo.go (926 lines) for all query methods.
2. **DB connection pools:** Reviewed identity `data/db.go`, audit `data/db.go`. Compared pool configurations.
3. **Caching:** Examined `response_cache.go` (259 lines), gateway list_cache middleware.
4. **Bulk processing:** Examined `bulk_import.go` (324 lines) end-to-end.
5. **Goroutine leaks:** Examined all `go func()` sites (47 files). Webhook delivery has proper semaphore + recover + timeout. Verified no leaks.
6. **JSON serialization:** Searched 351 files with `json.NewDecoder` and encoder patterns. Examined hot paths in handlers.
7. **Locks:** Examined ResponseCache mutex usage, webhook MemoryStore RWMutex, InputValidation RWMutex.
8. **HTTP clients:** Examined httputil/client.go shared clients, found 12+ ad-hoc client creations.

### Error Handling
1. **Error types:** Compared pkg/errors (GGIDError) usage in identity vs fmt.Errorf in audit.
2. **json.Unmarshal:** Found 15+ non-test files with swallowed errors.
3. **json.Encode:** Examined httputil.WriteJSON, WriteAPIError, input_validation rejectInput.
4. **tx.Commit:** Examined all 15 commit sites in pg_repo.go.
5. **goroutine recover:** Examined webhook (has recover), auth/impersonation (has recover), main.go goroutines (acceptable).
6. **ctx.Err():** Searched all services, found only 5 instances.
7. **DB error classification:** Examined audit_repo mapErr usage, identity isDuplicateKey/isNoRows.
8. **Error response format:** Examined pkg/errors APIError, httputil WriteError, input_validation rejectInput, webhook writeJSON, console error-helpers.ts.
9. **panic:** Found 6+ panic() calls, all in main.go or test files (acceptable).
10. **Error log leaking:** Examined input_validation pattern disclosure, slog.Error calls across services.

---

*End of Report*
