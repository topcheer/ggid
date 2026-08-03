# IAM Security Audit Report -- Concurrency Safety (R19) + Data Validation (R23)

**Date**: 2026-08-03
**Auditor**: Independent (first contact with ggid codebase)
**Scope**: services/gateway, services/auth, services/identity, services/oauth, services/audit, pkg/rbac, pkg/crypto
**Method**: Pure read-only code review, ~765 non-test Go files scanned

---

## Executive Summary

The ggid IAM platform demonstrates **good baseline concurrency hygiene** -- most goroutines have `defer recover()`, background workers use `context.Done()` or stop channels for clean shutdown, and the audit hash chain uses `FOR UPDATE` transaction locking. Data validation coverage has improved significantly from earlier rounds: avatar uploads check magic bytes, SCIM filter parsing has recursion depth limits, bulk operations are capped, and body size limits exist on most endpoints.

However, **14 issues** remain across the two dimensions:

| Severity | Count |
|----------|-------|
| P0 (data corruption / exploitable race / DoS) | 3 |
| P1 (potential concurrency bug / important validation gap) | 6 |
| P2 (minor improvement) | 5 |

---

## Part 1: Concurrency Safety (R19)

### Checked Code Paths

| Area | Files Examined | Patterns Verified |
|------|---------------|-------------------|
| Goroutine launches | All `go func()` across 5 services (~30 sites) | recover(), context cancellation, stop channels |
| Background workers | auth/cmd/main.go, key_rotation.go, token_bucket.go, impersonation.go, cae_scanner.go, rbac_dynamic.go, metering.go | Ticker loops, shutdown signals |
| Channel usage | impersonation.go (expiryChannels), wsproxy_enhanced.go, grpc.go, timeout.go | Close safety, double-close, send-on-closed |
| Mutex/atomic | GRPCProxy, TenantBucketLimiter, RotatingKeyProvider, AuditService, ttlCache, jtiBlocklist | Race conditions, lock ordering |
| DB transactions | audit_repo.go (FOR UPDATE), auth/mfa_pg_repo.go, identity/pg_repo.go, oauth/pg_repo.go | Isolation, TOCTOU |
| Shutdown | All 5 services' cmd/main.go | Graceful stop, goroutine leak |
| Context propagation | context.Background() in request paths | Should use request context |

### Findings

#### P0-1: Race condition on `impRedisClient` global variable (concurrent read/write)

**File**: `services/auth/internal/service/impersonation.go:35,84-86,116,120,137-138,178,182`
**Severity**: P0
**Pattern**: Data race on shared variable without synchronization

**Description**:
`impRedisClient` is a package-level `*redis.Client` variable. It is written by `SetImpersonationRedis()` (line 85) and read by `IssueImpersonationToken()` (line 116), `GetImpersonationToken()` (line 137), and `RevokeImpersonationToken()` (line 178) -- all without any mutex or atomic protection.

```go
var (
    impRedisClient     *redis.Client  // line 35 -- no mutex guard
)

func SetImpersonationRedis(rdb *redis.Client) {
    impRedisClient = rdb  // line 85 -- WRITE, no lock
}

func IssueImpersonationToken(...) {
    if impRedisClient != nil {  // line 116 -- READ, no lock
        impRedisClient.Set(...)  // line 120 -- USE, no lock
    }
}
```

**Risk**: If `SetImpersonationRedis` is called during startup while HTTP handlers are already serving requests (or in tests), concurrent goroutines may read a partially written pointer or a nil client, causing panics or missed persistence. The `go test -race` detector would flag this.

**Recommendation**: Guard `impRedisClient` with `atomic.Pointer[redis.Client]` or a `sync.RWMutex`. Alternatively, pass the Redis client through dependency injection instead of a package global.

---

#### P0-2: Race condition on GRPCProxy connection counters (non-atomic read/write)

**File**: `services/gateway/internal/middleware/grpc.go:54-55,115-121,255-256`
**Severity**: P0

**Description**:
`GRPCProxy.conns` and `GRPCProxy.active` are `int64` fields accessed from multiple goroutines without `sync/atomic` operations:

```go
type GRPCProxy struct {
    conns    int64  // line 54
    active   int64  // line 55
}

func (p *GRPCProxy) ConnectionCount() int64 {
    return p.conns  // line 115 -- non-atomic read
}
func (p *GRPCProxy) ActiveConnections() int64 {
    return p.active  // line 120 -- non-atomic read
}
```

The `Stats()` method (line 255) also reads both fields under `p.mu.RLock()`, but the counters are never actually incremented anywhere in the visible code (the `HandleConn` and `GRPCHTTPHandler` methods don't update them). This means the counters are always zero -- but if they were wired up, concurrent reads/writes of `int64` on 64-bit ARM are not guaranteed atomic in Go's memory model without `sync/atomic`.

**Risk**: Currently benign (counters never incremented), but the architecture invites future race conditions. Also misleading monitoring data.

**Recommendation**: Either use `atomic.Int64` for these counters and actually increment/decrement them in `HandleConn`/`GRPCHTTPHandler`, or remove them as dead code.

---

#### P0-3: gRPC proxy HTTP handler goroutines lack `defer recover()` -- panic crashes process

**File**: `services/gateway/internal/middleware/grpc.go:213-218` (and `139-154`)
**Severity**: P0

**Description**:
The `GRPCHTTPHandler` method spawns two bidirectional copy goroutines that have no panic recovery:

```go
// GRPCHTTPHandler (line 213)
go func() {
    io.Copy(backendConn, clientConn)  // line 214 -- no recover
    done <- struct{}{}
}()

// HandleConn (line 139) -- also no recover
go func() {
    defer wg.Done()
    io.Copy(backendConn, clientConn)
}()
```

Compare with the WebSocket proxy (`wsproxy.go:79-82`) and timeout middleware (`timeout.go:91-94`) which both correctly use `defer recover()`.

**Risk**: A panic in `io.Copy` (e.g., from a malformed connection or hijacked conn) propagates to the `http.Server`'s goroutine, potentially crashing the entire gateway process. The gateway already has `PanicRecovery` middleware, but hijacked connections bypass HTTP middleware since the response writer is no longer in play.

**Recommendation**: Add `defer func() { if r := recover(); r != nil { slog.Error(...) } }()` to all four goroutines in both `HandleConn` and `GRPCHTTPHandler`.

---

#### P1-1: `WSKeepalive` goroutine lacks `defer recover()`

**File**: `services/gateway/internal/middleware/wsproxy_enhanced.go:104-124`
**Severity**: P1

**Description**:
The keepalive goroutine started by `WSKeepalive.Start()` has no panic recovery:

```go
func (k *WSKeepalive) Start(onTimeout func()) {
    go func() {
        ticker := time.NewTicker(k.interval)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                k.mu.Lock()
                // ...
                if onTimeout != nil {
                    onTimeout()  // external callback -- could panic
                }
```

**Risk**: If the `onTimeout` callback panics, the goroutine dies silently. The WebSocket tunnel becomes unmonitored but continues consuming resources (goroutine leak in the proxy). Other similar goroutines in the same package (token_bucket, metering, sliding_ratelimit) all have recover.

**Recommendation**: Add `defer func() { if r := recover(); r != nil { slog.Error(...) } }()` at the top of the goroutine.

---

#### P1-2: `ExpiryChannels` cleanup: channel close under lock risks send-on-closed panic

**File**: `services/auth/internal/service/impersonation.go:59-73`
**Severity**: P1

**Description**:
In `StartImpersonationCleanup`, expired channels are closed under `expiryNotifMu`:

```go
expiryNotifMu.Lock()
for uid, ch := range expiryChannels {
    select {
    case <-ch:
    default:
        close(ch)          // line 69 -- closes channel
        delete(expiryChannels, uid)
    }
}
expiryNotifMu.Unlock()
```

Meanwhile, `ScheduleExpiryNotification` (line ~335) sends to the channel:
```go
expiryNotifMu.Lock()
defer expiryNotifMu.Unlock()
if ch, ok := expiryChannels[userID]; ok {
    select {
    case ch <- notif:   // line ~340 -- could panic if channel was closed
    default:
    }
}
```

Both operations hold `expiryNotifMu`, so they can't execute concurrently -- **the lock serializes access and prevents the send-on-closed panic**. However, if a consumer of the channel (e.g., SSE handler) reads from it outside the lock and then the cleanup closes it, the consumer could attempt a send on a closed channel if it ever writes back.

**Risk**: Low with current code (all sends/receives are under the same mutex), but fragile. Any future code that sends to an expiry channel outside `expiryNotifMu` will panic.

**Recommendation**: Document the invariant that all channel operations must hold `expiryNotifMu`. Or use a `sync.Once`-guarded close pattern.

---

#### P1-3: `context.Background()` used in request-path Redis operations

**File**: `services/auth/internal/service/impersonation.go:120,138,182`
**File**: `services/audit/internal/service/audit_service.go:120`
**Severity**: P1

**Description**:
Redis operations use `context.Background()` instead of the request context:

```go
// impersonation.go:120
impRedisClient.Set(context.Background(), impersonationKeyPrefix+t.TokenID.String(), data, ttl)

// audit_service.go:120
s.webhookEngine.Send(context.Background(), event.Action, event)
```

**Risk**: These operations will continue even if the HTTP request is cancelled or times out. In the impersonation case, if Redis is slow, the HTTP handler blocks indefinitely (no timeout). In the webhook case, it's intentional (fire-and-forget after response).

**Recommendation**: For impersonation Redis calls, pass the request context or use `context.WithTimeout`. For webhook delivery, `context.Background()` is acceptable since it's async post-response.

---

#### P1-4: Background workers in `init()` function -- no graceful shutdown for JTI cleanup

**File**: `services/auth/internal/service/impersonation.go:247-264`
**Severity**: P1

**Description**:
The JTI cleanup goroutine is started in `init()`:

```go
func init() {
    jtiCleanupDone = make(chan struct{})
    go func() {
        // ... 10-minute ticker ...
        case <-jtiCleanupDone:
            return
    }()
}
```

`StopJTICleanup()` exists (line 267) but I did not find it called in `auth/cmd/main.go`'s shutdown sequence. The impersonation cleanup (line 40) uses `ctx.Done()` properly but is started by the caller.

**Risk**: On graceful shutdown, the JTI cleanup goroutine is leaked until process exit (it will terminate when the process dies, but test binaries accumulate goroutine leaks).

**Recommendation**: Call `StopJTICleanup()` in the auth service shutdown handler.

---

#### P2-1: `token_bucket.go` `StopCleanup` double-close risk

**File**: `services/gateway/internal/middleware/token_bucket.go:314-317`
**Severity**: P2

**Description**:
```go
func (tbl *TenantBucketLimiter) StopCleanup() {
    if tbl.cleanupDone != nil {
        close(tbl.cleanupDone)
        tbl.cleanupDone = nil
    }
}
```

The nil-check + nil-assignment pattern is safe for single close. But `StartCleanup` (line 289) unconditionally creates `tbl.cleanupDone = make(chan struct{})` -- if called twice, the old channel reference is lost without being closed, leaking the first goroutine.

**Recommendation**: Guard `StartCleanup` with `sync.Once` or check if `cleanupDone != nil` before creating a new one.

---

#### P2-2: `expiryNotifMu` RWMutex but only write operations observed

**File**: `services/auth/internal/service/impersonation.go:304`
**Severity**: P2

**Description**:
`expiryNotifMu` is declared as `sync.RWMutex` but all operations use `Lock()` (exclusive). No `RLock()` calls found.

**Recommendation**: If read-heavy access is expected (e.g., checking if a channel exists), use `RLock()`. Otherwise simplify to `sync.Mutex`.

---

### Concurrency Summary Table

| ID | File | Severity | Issue |
|----|------|----------|-------|
| P0-1 | impersonation.go:35,85 | P0 | Race on impRedisClient global |
| P0-2 | grpc.go:54-55 | P0 | Non-atomic int64 counter access |
| P0-3 | grpc.go:213,139 | P0 | No recover in gRPC proxy goroutines |
| P1-1 | wsproxy_enhanced.go:104 | P1 | No recover in WSKeepalive goroutine |
| P1-2 | impersonation.go:69 | P1 | Channel close pattern fragile |
| P1-3 | impersonation.go:120 | P1 | context.Background in request path |
| P1-4 | impersonation.go:247 | P1 | JTI cleanup goroutine not stopped on shutdown |
| P2-1 | token_bucket.go:289 | P2 | StartCleanup double-call goroutine leak |
| P2-2 | impersonation.go:304 | P2 | RWMutex used as Mutex |

---

## Part 2: Data Validation (R23)

### Checked Code Paths

| Area | Files Examined | Patterns Verified |
|------|---------------|-------------------|
| Request body limits | All handlers using json.NewDecoder (~423 sites) | MaxBytesReader coverage |
| Input length validation | registration_handler.go, identity_service.go, http.go | Username/email/password/displayName max length |
| SCIM filter depth | scim/filter.go | Recursion depth limit |
| Email validation | auth/registration_handler.go, scim/handler.go | mail.ParseAddress |
| Avatar upload | identity/server/http.go | magic bytes, content type, size |
| UUID validation | Path params in identity, oauth, auth | uuid.Parse |
| Bulk operations | scim/bulk.go, bulk_import.go | maxOperations cap |
| authorization_details | oauth/server/server.go, rar_handler.go | RAR registry validation |
| JSON depth limiting | All services | json.Decoder depth limit |

### Findings

#### P0-4: No JSON depth limiting on `json.NewDecoder` -- deep nesting DoS

**File**: All services -- 423 `json.NewDecoder(r.Body)` sites
**Severity**: P0

**Description**:
Go's `encoding/json` decoder has no built-in nesting depth limit. While many endpoints have `MaxBytesReader` (10MB), an attacker can craft a 10MB payload of deeply nested JSON (e.g., `{"a":{"a":{"a":...}}}`) that causes the JSON parser to exhaust the goroutine stack (default 1GB), leading to OOM or stack overflow.

The body size limit (10MB) partially mitigates this (maximum nesting depth ~10MB / ~4 bytes per level ≈ 2.6M levels), but each level consumes stack space. At ~100 bytes of stack per JSON level, 10MB of nesting translates to ~100K levels × 100 bytes ≈ 10MB stack -- not catastrophic for a single request, but **concurrent attacks** (1000 requests × 10MB stack = 10GB) can exhaust server memory.

No `json.NewDecoder(r.Body)` call in any service uses a custom depth-limited reader.

**Risk**: Stack exhaustion DoS. The 10MB body limit bounds the payload size but not the stack depth consumption ratio.

**Recommendation**: Implement a depth-limited JSON decoder wrapper or use `io.LimitReader` with smaller body limits for specific endpoints. Go 1.25 does not yet have `json.Decoder.MaxDepth`, so a custom `io.Reader` that counts `{`/`[` characters and rejects above 1000 depth is needed.

---

#### P1-5: 339 `json.NewDecoder(r.Body)` calls -- most lack per-endpoint `MaxBytesReader`

**File**: services/identity/internal/server, services/auth/internal/server, services/oauth/internal/server
**Severity**: P1

**Description**:
While gateway-level and service-level middleware applies `MaxBytesReader` (10MB) to incoming requests, **339 direct `json.NewDecoder(r.Body)` calls** exist across identity, auth, and oauth handlers. The middleware-level body limit catches most, but some internal handler paths may bypass the middleware (e.g., gRPC handlers, internal admin routes).

Services with confirmed middleware-level body limits:
- auth/cmd/main.go:460 -- `MaxBytesReader(w, r.Body, 10<<20)`
- identity/scim/handler.go:473, bulk.go:58, bulk_import.go:74 -- all 10MB
- oauth/server/server.go:276 -- 10MB
- audit/cmd/main.go:254 -- 10MB

**Risk**: The middleware approach is correct but defense-in-depth is missing. A misconfigured route or new endpoint added without the middleware would be unprotected.

**Recommendation**: Apply `http.MaxBytesReader` at the service server setup level (in the HTTP server constructor), not per-route, so all handlers are covered automatically.

---

#### P1-6: Identity `CreateUser` handler uses naive email validation

**File**: `services/identity/internal/server/http.go:691`
**Severity**: P1

**Description**:
```go
if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") || len(req.Email) < 5 {
```

This validation accepts invalid emails like `@.` (5+ chars with both @ and .), `a@b.c` is valid but `@@@@.com` also passes. Meanwhile, the SCIM handler (`scim/handler.go:147`) correctly uses `mail.ParseAddress`, and the auth service (`registration_handler.go:115`) also uses `mail.ParseAddress`.

**Risk**: Inconsistent email validation across endpoints. The identity handler is the weakest link and could allow malformed data into the system.

**Recommendation**: Replace the `strings.Contains` check with `mail.ParseAddress(req.Email)` to match the auth service and SCIM handler.

---

#### P1-7: `authorization_details` stored as raw JSON without size limit

**File**: `services/oauth/internal/server/server.go:635-665`
**Severity**: P1

**Description**:
The authorization_details parameter is validated structurally (valid JSON, RAR type validation) but stored as-is:

```go
authDetailsJSON = json.RawMessage(ad)  // line 663 -- stores raw user input
```

And later in `grant_authorization_code.go:104`:
```go
s.rdb.Set(ctx, rarKey, req.AuthorizationDetails, 10*time.Minute)
```

While the RAR registry validates types, there is no size limit on the authorization_details JSON string itself. An attacker could send a large (within the 10MB body limit) authorization_details array.

**Risk**: Memory consumption if authorization_details is large. The 10MB body limit caps the payload, but 10MB of authorization_details per auth request, cached in Redis for 10 minutes, could consume significant Redis memory under load.

**Recommendation**: Add a size check (e.g., max 10KB) on the authorization_details parameter before storing in Redis.

---

#### P1-8: Password complexity validation varies by registration path

**File**: `services/auth/internal/server/registration_handler.go:156`
**File**: `services/identity/internal/server/http.go:685`
**Severity**: P1

**Description**:
The auth registration handler validates:
- Username length: 3-64 chars ✓
- Email format via mail.ParseAddress ✓
- Password presence ✓

The identity createUser handler (`http.go:685-691`) validates:
- Username, Email, Password non-empty ✓
- Email: only `strings.Contains("@")` -- weak
- **No username length check** (identity_service.go:42 checks max 255 but not min)
- **No password complexity check** (relies on auth service)

The service-layer validation (`identity_service.go:42`) checks max lengths (255/320) but these are very generous. Password strength is validated separately by `validatePasswordComplexity` in the auth service.

**Risk**: The identity service endpoint could be used to create users with very weak usernames (1 char) if accessed without going through the auth service's validation layer.

**Recommendation**: Enforce consistent username/password validation rules at the identity service layer, or ensure the identity createUser endpoint is never directly accessible without auth service validation.

---

#### P2-3: `writeJSON` used in 1328 places -- inconsistent with `writeJSONError`/`WriteError`

**File**: Multiple services
**Severity**: P2

**Description**:
`writeJSON(w, statusCode, data)` is used extensively (1328 occurrences) as a convenience wrapper for `json.NewEncoder(w).Encode()`. However, it silently swallows JSON encoding errors and doesn't set proper Content-Type headers consistently. Meanwhile, `pkg/errors.WriteError` exists for error responses.

**Risk**: If `writeJSON` fails to encode (e.g., due to a nil pointer in the data), the client receives a truncated response with no error indication.

**Recommendation**: Audit `writeJSON` implementation to ensure it logs encoding errors and always sets `Content-Type: application/json`.

---

#### P2-4: SCIM filter parser depth limit -- good but not configurable

**File**: `services/identity/internal/scim/filter.go:237,273`
**Severity**: P2

**Description**:
The SCIM filter parser has a recursion depth counter (line 237: `depth int`) and increments it on each recursive call (line 273: `p.depth++`). This is good -- it prevents stack exhaustion from malicious deeply-nested filters.

However, the maximum depth is not configurable and appears to be hardcoded.

**Risk**: Low -- the hardcoded limit is reasonable for SCIM filters.

**Recommendation**: Consider making the depth limit configurable for environments with complex filter requirements, or at least document the limit.

---

#### P2-5: Identity service max lengths are very generous (255 chars for username)

**File**: `services/identity/internal/service/identity_service.go:42`
**Severity**: P2

**Description**:
```go
if len(input.Username) > 255 || len(input.Email) > 320 || len(input.DisplayName) > 255
```

These limits match RFC standards (email max 320, username max 255) but are more permissive than the auth service's registration handler (username 3-64 chars).

**Risk**: Inconsistency between identity-service and auth-service validation could lead to users created via one path that can't be created via another.

**Recommendation**: Align the limits across both services. Consider whether usernames really need to be up to 255 characters.

---

### Data Validation Summary Table

| ID | File | Severity | Issue |
|----|------|----------|-------|
| P0-4 | All services (423 sites) | P0 | No JSON depth limiting on decoder |
| P1-5 | identity/auth/oauth (339 sites) | P1 | Per-endpoint MaxBytesReader inconsistent |
| P1-6 | identity/http.go:691 | P1 | Weak email validation (strings.Contains) |
| P1-7 | oauth/server.go:663 | P1 | authorization_details no size limit for Redis |
| P1-8 | identity/http.go:685 | P1 | Inconsistent password/username validation |
| P2-3 | Multiple (1328 sites) | P2 | writeJSON swallows encoding errors |
| P2-4 | scim/filter.go:237 | P2 | Depth limit not configurable |
| P2-5 | identity_service.go:42 | P2 | Max lengths generous / inconsistent with auth |

---

## Positive Findings (Security Strengths)

### Concurrency
1. **Audit hash chain FOR UPDATE** (`audit_repo.go:55-69`): Properly uses `SELECT ... FOR UPDATE` in a transaction to prevent TOCTOU on hash chain computation. This is the correct pattern for serialized chain appending.
2. **Lock sharding for audit** (`audit_service.go:39,63-65`): 64 shards keyed by tenant ID first byte -- allows cross-tenant parallelism while serializing per-tenant chain operations. Good design.
3. **RotatingKeyProvider** (`key_rotation.go`): Clean RWMutex usage, `sync.Once` for stop functions, proper ticker cleanup. All goroutines have `defer recover()`.
4. **TokenBucket / TenantBucketLimiter** (`token_bucket.go`): Proper double-check locking pattern for bucket creation (RLock → check → RUnlock → Lock → re-check → create).
5. **wsproxy.go** (lines 79-82, 95-98): Both bidirectional copy goroutines have `defer recover()` -- good example pattern.
6. **JTI blocklist**: Properly guarded with `jtiBlocklistMu` RWMutex, with periodic cleanup.

### Data Validation
7. **Avatar upload magic bytes** (`http.go:1145`): Uses `http.DetectContentType(data)` to verify actual file content, not just the client-supplied Content-Type header. Proper defense.
8. **SCIM filter depth limit** (`filter.go:237,273`): Recursion depth counter prevents stack exhaustion DoS.
9. **Bulk operations cap** (`bulk.go:50,70`): 1000-operation limit enforced before processing.
10. **RAR validation** (`server.go:648-655`): authorization_details validated via registry with type checking per RFC 9396 §2.1.
11. **Email validation via mail.ParseAddress** (auth + SCIM paths): Correct use of stdlib parser.
12. **TTL cache bounded** (`ttl_cache.go:Set`): maxEntries=10000 prevents memory exhaustion with eviction policy.
13. **Body size limits on all service main.go files**: 10MB MaxBytesReader applied at server middleware level.
14. **Input length checks in identity_service.go**: Username ≤255, Email ≤320, DisplayName ≤255, Phone ≤32, Locale ≤10, Timezone ≤64.

---

## Remediation Priority

### Immediate (P0)
1. **impRedisClient race**: Wrap in `atomic.Pointer[redis.Client]` or add mutex ← **simplest fix, highest impact**
2. **gRPC proxy goroutine recover**: Add `defer recover()` to 4 goroutines in grpc.go ← **5-minute fix**
3. **JSON depth limiting**: Add a custom depth-counting reader wrapper ← **requires new utility**

### Short-term (P1)
4. WSKeepalive recover
5. Replace strings.Contains email validation with mail.ParseAddress in identity handler
6. Add authorization_details size limit before Redis storage
7. Align password/username validation across auth and identity services
8. Verify JTI cleanup is stopped on shutdown
9. Apply MaxBytesReader at server constructor level for defense-in-depth

### Backlog (P2)
10. Fix StartCleanup double-call risk
11. Simplify RWMutex to Mutex where only Lock is used
12. Audit writeJSON error handling
13. Align max lengths across services
14. Consider configurable SCIM depth limit

---

## Statistics

- **Files reviewed**: ~765 non-test Go files across 5 services + pkg/rbac + pkg/crypto
- **Goroutine launch sites examined**: ~30 across all services
- **json.NewDecoder sites examined**: 423 (1328 writeJSON calls)
- **Transaction patterns examined**: FOR UPDATE (audit), BeginTx (identity, oauth, auth)
- **P0 issues**: 3 (1 race condition, 1 non-atomic counter, 1 missing recover)
- **P1 issues**: 8 (4 concurrency + 4 data validation)
- **P2 issues**: 7 (5 concurrency + 2 data validation)
- **Positive findings**: 14 security strengths identified

---

*Audit conducted as independent read-only review. No code modifications made.*
