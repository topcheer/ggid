# Multi-Tenant Data Isolation Verification for IAM Systems

**Document Type:** Security Research & Code Audit
**Scope:** GGID IAM Suite — tenant context propagation, PostgreSQL RLS, cross-tenant leakage, cache/event isolation
**Classification:** Internal Security Audit
**Date:** 2025

---

## Table of Contents

1. [Multi-Tenant Isolation Threat Model](#1-multi-tenant-isolation-threat-model)
2. [GGID Tenant Context Propagation](#2-ggid-tenant-context-propagation)
3. [PostgreSQL RLS Verification](#3-postgresql-rls-verification)
4. [Cross-Tenant Leakage Test Suite](#4-cross-tenant-leakage-test-suite)
5. [Tenant Context Injection Prevention](#5-tenant-context-injection-prevention)
6. [Shared Resource Isolation](#6-shared-resource-isolation)
7. [Cache Isolation](#7-cache-isolation)
8. [NATS Event Isolation](#8-nats-event-isolation)
9. [GGID RLS Audit](#9-ggid-rls-audit)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. Multi-Tenant Isolation Threat Model

Multi-tenant Identity and Access Management (IAM) systems store user data, credentials, roles,
and audit logs for multiple organizations in a shared database. A failure in tenant isolation
results in **cross-tenant data leakage** — one tenant accessing another tenant's users, roles,
audit logs, or configuration. In an IAM system, this is catastrophic: leaked credentials or role
assignments can lead to complete account takeover.

### 1.1 Cross-Tenant Data Leakage Vectors

**Vector 1: Missing `tenant_id` in WHERE Clause**

The most common isolation failure occurs when a query omits the `tenant_id` filter. Without RLS
enforcement, any query that forgets `WHERE tenant_id = $1` returns data from ALL tenants.

```sql
-- VULNERABLE: returns users from all tenants
SELECT * FROM users WHERE email = $1;

-- SAFE: tenant-scoped query
SELECT * FROM users WHERE tenant_id = $1 AND email = $2;
```

**Attack scenario:** An attacker in tenant A requests `GET /api/v1/users/{id}` where `{id}`
belongs to tenant B. If the handler queries by ID only without tenant scoping, the response
leaks tenant B's user profile including email, phone, and potentially credential metadata.

**Vector 2: Tenant Context Not Propagated**

When tenant context is set on the HTTP request but not forwarded to the database layer. This
occurs when:
- gRPC service handlers extract tenant_id from metadata but forget to pass it to the repository.
- Background workers (cron, NATS consumers) process events without establishing tenant context.
- Internal service-to-service calls drop the `X-Tenant-ID` header.

**Vector 3: RLS Policy Bypass via `SET LOCAL` Injection**

PostgreSQL RLS relies on a session variable (`app.tenant_id`) set via `SET LOCAL`. If the value
is interpolated unsafely, SQL injection can override the RLS context:

```go
// VULNERABLE: string interpolation in SET LOCAL
tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID))

// If tenantID is ever a non-UUID string (e.g., "x'; RESET app.tenant_id; --"),
// the RLS context is cleared, and subsequent queries return all tenants' data.
```

**Vector 4: Shared Cache Poisoning**

When a cache key omits the tenant ID, one tenant's cached data is served to another:

```go
// VULNERABLE: cache key has no tenant prefix
cacheKey := "user:" + userID.String()

// SAFE: tenant-scoped cache key
cacheKey := "tenant:" + tenantID.String() + ":user:" + userID.String()
```

**Vector 5: Cross-Tenant Search/Filter Leakage**

Search endpoints that filter by attributes (email, username) without tenant scoping leak
existence information. Even if the response returns "not found," timing differences or error
messages can reveal whether a resource exists in another tenant.

**Vector 6: IDOR (Insecure Direct Object Reference)**

Endpoints that accept a resource UUID without verifying it belongs to the requesting tenant.
The attacker enumerates UUIDs or uses predictable ID patterns to access other tenants' data.

### 1.2 Attack Surface in IAM Systems

| Resource | Isolation Risk | Impact |
|---|---|---|
| Users | Email/phone/profile leakage | PII breach, phishing |
| Credentials | Password hash leakage | Account takeover |
| Roles | Cross-tenant privilege escalation | Unauthorized access |
| Organizations | Org structure leakage | Competitive intelligence |
| Audit Logs | Activity monitoring | Privacy violation, attack recon |
| OAuth Providers | Client secret leakage | Token theft, impersonation |
| Sessions | Session hijacking | Account takeover |

---

## 2. GGID Tenant Context Propagation

### 2.1 Current Architecture

GGID resolves tenant identity through a middleware chain in the API Gateway:

```
Client Request
    │
    ▼
┌─────────────────────────────────────────────────────┐
│  Gateway Middleware Chain (Handler())               │
│                                                     │
│  PanicRecovery                                      │
│    └─ SecurityHeaders                               │
│       └─ CORS                                       │
│          └─ RequestID                               │
│             └─ RequestLogger                        │
│                └─ RateLimiter                       │
│                   └─ TenantResolver ◄── THIS STEP   │
│                      └─ JWTAuth                     │
│                         └─ Gateway ServeHTTP        │
│                            └─ ReverseProxy          │
│                               └─ Backend Service    │
└─────────────────────────────────────────────────────┘
```

### 2.2 TenantResolver Middleware

The `TenantResolver` middleware in `middleware.go` resolves the tenant ID from three sources
in priority order:

```go
// Resolution priority:
// 1. X-Tenant-ID header (HIGHEST PRIORITY — client-controllable)
// 2. JWT claim "tenant_id" (parsed without signature verification)
// 3. Subdomain prefix (e.g., acme.iam.example.com)
```

The resolved tenant ID is injected into:
1. The Go `context.Context` via `tenant.WithContext()` as a `*tenant.Context` struct.
2. A string context value `TenantIDKey` for downstream access.
3. The `X-Tenant-ID` HTTP header forwarded to backend services.
4. The `tenant_id` query parameter on the proxied URL.
5. The `tenant_id` field in JSON request bodies (for POST/PUT/PATCH).

```go
// pkg/tenant/tenant.go — tenant context propagation via context.Context
package tenant

type contextKey struct{}

type Context struct {
    TenantID       uuid.UUID
    IsolationLevel IsolationLevel
    SchemaName     string
    Settings       map[string]any
}

func FromContext(ctx context.Context) (*Context, error) {
    tc, ok := ctx.Value(contextKey{}).(*Context)
    if !ok || tc == nil {
        return nil, fmt.Errorf("no tenant context found")
    }
    return tc, nil
}

func WithContext(ctx context.Context, tc *Context) context.Context {
    return context.WithValue(ctx, contextKey{}, tc)
}
```

### 2.3 Gateway Proxy Tenant Injection

The reverse proxy director injects tenant context before forwarding:

```go
proxy.Director = func(req *http.Request) {
    originalDirector(req)
    if tenantID, ok := middleware.TenantIDFromRequest(req); ok {
        // 1. Set header for backend services
        req.Header.Set("X-Tenant-ID", tenantID)
        // 2. Inject as query param (for services that read tenant_id from URL)
        q := req.URL.Query()
        if q.Get("tenant_id") == "" {
            q.Set("tenant_id", tenantID)
            req.URL.RawQuery = q.Encode()
        }
        // 3. Inject into JSON body for write methods
        injectTenantIntoBody(req, tenantID)
    }
}
```

### 2.4 Identified Gaps

**Gap A: Header overrides JWT claim (CRITICAL)**

The `TenantResolver` gives the `X-Tenant-ID` header the **highest priority**. Since this header
is client-controllable, an authenticated user from tenant A can send `X-Tenant-ID: <tenant_B_id>`
and have their requests processed in tenant B's context. The JWT `tenant_id` claim should be the
authoritative source for authenticated requests.

**Gap B: TenantResolver runs before JWTAuth**

The middleware order is `TenantResolver → JWTAuth`. This means:
- The tenant context is resolved **before** the JWT is verified.
- On public endpoints (login, register), this is expected — there is no JWT yet.
- On protected endpoints, the JWT claim is never used for tenant resolution because
  `TenantResolver` already set it from the header.
- The `JWTAuth` middleware does extract `tenant_id` into context, but `TenantResolver`
  has already set the value.

**Gap C: Body injection skips when `tenant_id` already present**

```go
// injectTenantIntoBody skips if the body already has tenant_id
if _, exists := bodyMap["tenant_id"]; exists {
    restore()
    return
}
```

A malicious client can include `"tenant_id": "<target_tenant>"` in their JSON body. The gateway
will NOT override it. This allows cross-tenant resource creation.

**Gap D: gRPC proxy does not propagate tenant metadata**

The gRPC proxy (`grpc.go`) tunnels raw TCP connections. It does not extract or propagate
`tenant_id` via gRPC metadata. Any gRPC backend service must independently extract tenant
context from headers or metadata, and the TCP tunnel provides no tenant isolation guarantees.

**Gap E: No tenant validation on public endpoints**

Public endpoints (login, register) accept any `X-Tenant-ID` header value without validation.
An attacker can register users in arbitrary tenants or attempt logins against other tenants'
user bases.

---

## 3. PostgreSQL RLS Verification

### 3.1 RLS Architecture in GGID

GGID uses PostgreSQL Row Level Security (RLS) as the primary data isolation mechanism for the
`shared` isolation model. RLS policies use a session variable `app.tenant_id` that is set via
`SET LOCAL` at the beginning of each transaction.

```go
// services/identity/internal/repository/pg_repo.go
func setTenantRLS(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
    _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID.String()))
    if err != nil {
        return fmt.Errorf("set tenant RLS: %w", err)
    }
    return nil
}
```

The RLS policy (defined in migration SQL) would look like:

```sql
-- Enable RLS on all tenant-scoped tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

-- Policy: only rows matching the session's app.tenant_id are visible
CREATE POLICY tenant_isolation ON users
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### 3.2 Two Implementation Patterns in GGID

GGID has **inconsistent RLS patterns** across services:

| Service | RLS Pattern | Parameterization | Risk |
|---|---|---|---|
| Identity | `fmt.Sprintf("SET LOCAL...")` | String interpolation | Low (UUID-parsed) |
| Auth (MFA) | `tx.Exec(ctx, "SET LOCAL app.tenant_id = $1", ...)` | Parameterized | Safe |
| Auth (creds) | No SET LOCAL — uses `WHERE tenant_id = $1` | Query parameterized | Medium |
| Audit | **No RLS at all** — direct pool queries | WHERE filter only | **HIGH** |

The identity service pattern using `fmt.Sprintf` is technically safe because `uuid.UUID.String()`
produces a fixed-format hex string that cannot contain SQL metacharacters. However, it is a **bad
practice** that could become dangerous if the tenant ID source changes.

### 3.3 RLS Verification Test

The following Go test verifies that RLS enforcement prevents cross-tenant queries:

```go
package integration

import (
    "context"
    "fmt"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgxpool"
)

func TestRLS_CrossTenantIsolation(t *testing.T) {
    if testing.Short() {
        t.Skip("requires PostgreSQL")
    }
    dbURL := "postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable"
    pool, err := pgxpool.New(context.Background(), dbURL)
    if err != nil {
        t.Fatalf("connect to DB: %v", err)
    }
    defer pool.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    tenantA := uuid.MustParse("00000000-0000-0000-0000-000000000001")
    tenantB := uuid.MustParse("00000000-0000-0000-0000-000000000002")

    // Insert user in tenant A
    tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        t.Fatalf("begin tx: %v", err)
    }
    _, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantA))
    if err != nil {
        t.Fatalf("set RLS: %v", err)
    }
    userAID := uuid.New()
    _, err = tx.Exec(ctx, `
        INSERT INTO users (id, tenant_id, username, email, status, password_hash, locale, timezone)
        VALUES ($1, $2, $3, $4, 'active', 'hash', 'en', 'UTC')`,
        userAID, tenantA, "tenant_a_user", "a@tenant-a.test")
    if err != nil {
        t.Fatalf("create user: %v", err)
    }
    if err := tx.Commit(ctx); err != nil {
        t.Fatalf("commit: %v", err)
    }

    // Attempt to read tenant A's user as tenant B — should return 0 rows
    tx2, err := pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        t.Fatalf("begin tx2: %v", err)
    }
    defer tx2.Rollback(ctx)

    _, err = tx2.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantB))
    if err != nil {
        t.Fatalf("set RLS for tenant B: %v", err)
    }

    var count int
    err = tx2.QueryRow(ctx,
        `SELECT count(*) FROM users WHERE id = $1`,
        userAID,
    ).Scan(&count)

    if err != nil {
        t.Fatalf("query as tenant B: %v", err)
    }
    // RLS should hide tenant A's row from tenant B
    if count != 0 {
        t.Errorf("RLS VIOLATION: tenant B can see tenant A's user (count=%d)", count)
    }

    // Verify tenant A can see its own user
    tx3, err := pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        t.Fatalf("begin tx3: %v", err)
    }
    defer tx3.Rollback(ctx)

    _, err = tx3.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantA))
    if err != nil {
        t.Fatalf("set RLS for tenant A: %v", err)
    }

    var email string
    err = tx3.QueryRow(ctx,
        `SELECT email FROM users WHERE id = $1`,
        userAID,
    ).Scan(&email)
    if err != nil {
        t.Errorf("tenant A should see its own user: %v", err)
    }
    if email != "a@tenant-a.test" {
        t.Errorf("email mismatch: got %s", email)
    }
}

// TestRLS_NoTenantContext verifies that queries without SET LOCAL return zero rows
// (assuming FORCE ROW LEVEL SECURITY is enabled).
func TestRLS_NoTenantContext(t *testing.T) {
    if testing.Short() {
        t.Skip("requires PostgreSQL")
    }
    dbURL := "postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable"
    pool, err := pgxpool.New(context.Background(), dbURL)
    if err != nil {
        t.Fatalf("connect to DB: %v", err)
    }
    defer pool.Close()

    ctx := context.Background()

    // Query WITHOUT setting app.tenant_id — RLS should deny all rows
    tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        t.Fatalf("begin tx: %v", err)
    }
    defer tx.Rollback(ctx)

    var count int
    err = tx.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&count)
    if err != nil {
        t.Fatalf("query without RLS context: %v", err)
    }
    // With FORCE ROW LEVEL SECURITY and no app.tenant_id set,
    // current_setting('app.tenant_id') returns empty string,
    // which won't match any UUID, so count should be 0.
    if count > 0 {
        t.Errorf("RLS GAP: query without tenant context returned %d rows", count)
    }
}
```

### 3.4 RLS Audit Checklist

For every database-backed service, verify:
1. Every tenant-scoped table has `ENABLE ROW LEVEL SECURITY` and `FORCE ROW LEVEL SECURITY`.
2. Every query path calls `SET LOCAL app.tenant_id` before executing tenant-scoped queries.
3. `SET LOCAL` is called within a transaction (`SET LOCAL` only applies to the current transaction).
4. The database user is NOT a superuser (superusers bypass RLS).
5. No raw pool queries (without transaction + SET LOCAL) access tenant-scoped tables.

---

## 4. Cross-Tenant Leakage Test Suite

### 4.1 Integration Test Patterns

The following test suite verifies end-to-end tenant isolation through the Gateway:

```go
package integration

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "testing"
)

type tenantTestEnv struct {
    gatewayURL string
    tenantA    string
    tenantB    string
    tokenA     string
    tokenB     string
}

func setupCrossTenantEnv(t *testing.T) *tenantTestEnv {
    env := &tenantTestEnv{
        gatewayURL: "http://localhost:8080",
        tenantA:    "00000000-0000-0000-0000-000000000001",
        tenantB:    "00000000-0000-0000-0000-000000000002",
    }
    // Register + login in tenant A
    env.tokenA = registerAndLogin(t, env.gatewayURL, env.tenantA, "isolation_a")
    // Register + login in tenant B
    env.tokenB = registerAndLogin(t, env.gatewayURL, env.tenantB, "isolation_b")
    return env
}

// TestCrossTenant_ListUsers verifies that listing users as tenant A
// does not return tenant B's users.
func TestCrossTenant_ListUsers(t *testing.T) {
    env := setupCrossTenantEnv(t)

    // List users as tenant A
    req, _ := http.NewRequest("GET", env.gatewayURL+"/api/v1/users", nil)
    req.Header.Set("Authorization", "Bearer "+env.tokenA)
    req.Header.Set("X-Tenant-ID", env.tenantA)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        t.Fatalf("list users: %v", err)
    }
    defer resp.Body.Close()

    var result struct {
        Users []map[string]any `json:"users"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    // Verify no user belongs to tenant B
    for _, u := range result.Users {
        if tid, ok := u["tenant_id"].(string); ok && tid == env.tenantB {
            t.Errorf("LEAK: tenant A token returned user from tenant B: %v", u)
        }
    }
}

// TestCrossTenant_GetUserByID verifies that tenant B cannot fetch
// a user created by tenant A by guessing the user ID.
func TestCrossTenant_GetUserByID(t *testing.T) {
    env := setupCrossTenantEnv(t)

    // Create a user as tenant A
    createBody := `{"username":"secret_a_user","email":"secret@tenant-a.test","password":"Pass123!"}`
    req, _ := http.NewRequest("POST", env.gatewayURL+"/api/v1/users", bytes.NewBufferString(createBody))
    req.Header.Set("Authorization", "Bearer "+env.tokenA)
    req.Header.Set("X-Tenant-ID", env.tenantA)
    req.Header.Set("Content-Type", "application/json")
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        t.Fatalf("create user: %v", err)
    }
    var created map[string]any
    json.NewDecoder(resp.Body).Decode(&created)
    resp.Body.Close()

    userAID, _ := created["id"].(string)
    if userAID == "" {
        t.Fatal("missing user ID")
    }

    // Attempt to fetch as tenant B
    req2, _ := http.NewRequest("GET", env.gatewayURL+"/api/v1/users/"+userAID, nil)
    req2.Header.Set("Authorization", "Bearer "+env.tokenB)
    req2.Header.Set("X-Tenant-ID", env.tenantB)
    resp2, err := http.DefaultClient.Do(req2)
    if err != nil {
        t.Fatalf("get user as tenant B: %v", err)
    }
    defer resp2.Body.Close()

    if resp2.StatusCode == http.StatusOK {
        t.Errorf("LEAK: tenant B accessed tenant A's user (status=%d)", resp2.StatusCode)
    }
    // Expect 403 or 404
    if resp2.StatusCode != http.StatusNotFound && resp2.StatusCode != http.StatusForbidden {
        t.Errorf("unexpected status: %d (want 403 or 404)", resp2.StatusCode)
    }
}

// TestCrossTenant_HeaderSpoofing verifies that a tenant A user cannot
// access tenant B's data by changing the X-Tenant-ID header.
func TestCrossTenant_HeaderSpoofing(t *testing.T) {
    env := setupCrossTenantEnv(t)

    // Use tenant A's token but set X-Tenant-ID to tenant B
    req, _ := http.NewRequest("GET", env.gatewayURL+"/api/v1/users", nil)
    req.Header.Set("Authorization", "Bearer "+env.tokenA)
    req.Header.Set("X-Tenant-ID", env.tenantB) // SPOOF
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        t.Fatalf("spoofed request: %v", err)
    }
    defer resp.Body.Close()

    // If the gateway forwards this, tenant A's JWT auth runs in tenant B's context.
    // This is the CRITICAL vulnerability — the request should be rejected because
    // the JWT tenant_id (A) doesn't match the header tenant_id (B).
    // Currently this test WILL FAIL (request succeeds) — demonstrating the gap.
    if resp.StatusCode == http.StatusOK {
        t.Error("VULNERABILITY: tenant A token accepted with tenant B X-Tenant-ID header")
    }
}
```

### 4.2 Test Coverage Requirements

| Test Case | Expected Behavior |
|---|---|
| List users with tenant A token | Only tenant A users returned |
| Get tenant A user with tenant B token | 404 Not Found |
| Create resource as A, query as B | 404 or empty result |
| Delete tenant A resource with tenant B token | 403 Forbidden |
| Audit log query cross-tenant | Only tenant-scoped events returned |
| Header spoof (JWT=A, header=B) | 403 Forbidden (JWT tenant must win) |

---

## 5. Tenant Context Injection Prevention

### 5.1 The Header-Based Tenant Problem

GGID's current architecture allows clients to set `X-Tenant-ID` directly. This is dangerous
because:

1. **Client-controlled values are untrusted.** Any HTTP client can send any header value.
2. **The header takes priority over the JWT claim** in `TenantResolver`.
3. **No validation** that the header matches the authenticated user's tenant.

The secure approach is to **derive tenant_id exclusively from the verified JWT claim** for
authenticated requests. The `X-Tenant-ID` header should only be used on public endpoints
(login, register) where no JWT exists yet, or for initial tenant routing before authentication.

### 5.2 Secure Tenant Binding Middleware

```go
// secure_tenant.go — Tenant context bound to JWT claim, not header

package middleware

import (
    "context"
    "net/http"

    "github.com/ggid/ggid/pkg/tenant"
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

// SecureTenantResolver resolves tenant_id from the VERIFIED JWT claim,
// overriding any client-supplied X-Tenant-ID header for authenticated requests.
// For unauthenticated requests, it falls back to the header (needed for login/register).
func SecureTenantResolver(jwks *JWKSClient) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Try to extract tenant_id from the VERIFIED JWT in context
            // (JWTAuth middleware has already validated the token)
            jwtTenantID, ok := r.Context().Value(TenantIDKey).(string)
            if ok && jwtTenantID != "" {
                // JWT is authoritative — override any client header
                id, err := uuid.Parse(jwtTenantID)
                if err == nil {
                    tc := &tenant.Context{
                        TenantID:       id,
                        IsolationLevel: tenant.IsolationShared,
                    }
                    ctx := tenant.WithContext(r.Context(), tc)
                    ctx = context.WithValue(ctx, TenantIDKey, jwtTenantID)
                    // Overwrite header so downstream services receive the JWT tenant
                    r.Header.Set("X-Tenant-ID", jwtTenantID)
                    next.ServeHTTP(w, r.WithContext(ctx))
                    return
                }
            }

            // No valid JWT — use header for public endpoints only
            headerTID := r.Header.Get("X-Tenant-ID")
            if headerTID != "" {
                if id, err := uuid.Parse(headerTID); err == nil {
                    tc := &tenant.Context{
                        TenantID:       id,
                        IsolationLevel: tenant.IsolationShared,
                    }
                    ctx := tenant.WithContext(r.Context(), tc)
                    ctx = context.WithValue(ctx, TenantIDKey, headerTID)
                    next.ServeHTTP(w, r.WithContext(ctx))
                    return
                }
            }

            next.ServeHTTP(w, r)
        })
    }
}

// TenantMatchValidator rejects requests where the JWT tenant_id
// differs from the X-Tenant-ID header. This is a defense-in-depth check.
func TenantMatchValidator() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            jwtTID, ok := r.Context().Value(TenantIDKey).(string)
            if ok && jwtTID != "" {
                headerTID := r.Header.Get("X-Tenant-ID")
                if headerTID != "" && headerTID != jwtTID {
                    // JWT tenant and header tenant mismatch — reject
                    w.Header().Set("Content-Type", "application/json")
                    w.WriteHeader(http.StatusForbidden)
                    json.NewEncoder(w).Encode(map[string]string{
                        "error": "tenant_id in JWT does not match X-Tenant-ID header",
                    })
                    return
                }
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### 5.3 Recommended Middleware Order

```
PanicRecovery
  └─ SecurityHeaders
     └─ CORS
        └─ RequestID
           └─ RequestLogger
              └─ RateLimiter
                 └─ JWTAuth (verifies token, extracts tenant_id claim)
                    └─ SecureTenantResolver (JWT tenant wins over header)
                       └─ TenantMatchValidator (reject header/JWT mismatch)
                          └─ Gateway
```

---

## 6. Shared Resource Isolation

### 6.1 Global vs Tenant-Scoped Resources

Not all resources are tenant-scoped. IAM systems have shared/global resources that cross tenant
boundaries:

| Resource Type | Scope | Isolation Strategy |
|---|---|---|
| Users, Credentials | Tenant-scoped | RLS by `tenant_id` |
| Roles | Tenant-scoped + system roles | RLS + `is_system` flag |
| Organizations | Tenant-scoped | RLS by `tenant_id` |
| Audit Events | Tenant-scoped | RLS or WHERE filter |
| OAuth Provider configs | Global or tenant-scoped | Depends on deployment |
| JWKS keys | Global | Shared infrastructure |
| NATS streams | Infrastructure | Subject namespacing |

### 6.2 System Roles Pattern

System roles (like "everyone", "authenticated") are shared across tenants but must not leak
tenant-specific assignments:

```go
// When listing roles, system roles are visible to all tenants,
// but role assignments are tenant-scoped.
func (r *pgRepo) ListRoles(ctx context.Context, tenantID uuid.UUID) ([]*Role, error) {
    tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return nil, err
    }
    defer tx.Rollback(ctx)

    if err := setTenantRLS(ctx, tx, tenantID); err != nil {
        return nil, err
    }

    // RLS shows tenant-scoped roles. System roles need a UNION
    // with a query that bypasses RLS for is_system = true rows.
    query := `
        SELECT id, tenant_id, key, name, description, is_system
        FROM roles
        WHERE is_system = true
        UNION ALL
        SELECT id, tenant_id, key, name, description, is_system
        FROM roles
        WHERE tenant_id = $1 AND is_system = false
        ORDER BY name
    `
    rows, err := tx.Query(ctx, query, tenantID)
    // ...
}
```

### 6.3 Audit Log Infrastructure

Audit events are tenant-scoped in GGID — each event carries a `tenant_id` field. However, the
audit infrastructure (NATS stream, database table) is shared. The isolation must be enforced at
the query level (WHERE `tenant_id = $1`).

The critical concern: **the audit service itself must not become a cross-tenant oracle.** An
attacker who can query audit events for another tenant can observe login patterns, admin
actions, and security events.

---

## 7. Cache Isolation

### 7.1 Redis Cache Key Namespacing

GGID uses Redis for caching. Cache keys MUST include the tenant ID to prevent cross-tenant
cache collision. Without tenant namespacing, one tenant's cached data could be served to
another tenant.

```go
// pkg/cache/tenant_cache.go — tenant-scoped cache key generation

package cache

import (
    "fmt"

    "github.com/google/uuid"
)

// TenantCacheKey generates a cache key namespaced by tenant ID.
// Format: tenant:{tenantID}:{resource}:{key}
func TenantCacheKey(tenantID uuid.UUID, resource, key string) string {
    return fmt.Sprintf("tenant:%s:%s:%s", tenantID.String(), resource, key)
}

// TenantCacheKeyPattern generates a glob pattern for invalidating
// all cached entries for a specific tenant and resource.
func TenantCacheKeyPattern(tenantID uuid.UUID, resource string) string {
    return fmt.Sprintf("tenant:%s:%s:*", tenantID.String(), resource)
}

// Example usage in a service handler:
func (h *UserHandler) GetUser(ctx context.Context, tenantID, userID uuid.UUID) (*User, error) {
    cacheKey := cache.TenantCacheKey(tenantID, "user", userID.String())

    // Try cache
    if cached, err := h.redis.Get(ctx, cacheKey).Result(); err == nil {
        var user User
        if json.Unmarshal([]byte(cached), &user) == nil {
            return &user, nil
        }
    }

    // Cache miss — query database with RLS
    user, err := h.repo.GetUserByID(ctx, tenantID, userID)
    if err != nil {
        return nil, err
    }

    // Cache with TTL
    data, _ := json.Marshal(user)
    h.redis.Set(ctx, cacheKey, data, 5*time.Minute)

    return user, nil
}

// InvalidateTenantCache removes all cached entries for a tenant.
// Called when a tenant is deleted or undergoes a bulk data change.
func (h *UserHandler) InvalidateTenantCache(ctx context.Context, tenantID uuid.UUID) error {
    pattern := cache.TenantCacheKeyPattern(tenantID, "user")
    iter := h.redis.Scan(ctx, 0, pattern, 0).Iterator()
    for iter.Next(ctx) {
        h.redis.Del(ctx, iter.Val())
    }
    return iter.Err()
}
```

### 7.2 Cache Collision Risk in Current GGID

GGID's gateway response cache (`response_cache.go`) caches HTTP responses. If the cache key is
based only on the URL path and method (without tenant ID), a response for tenant A could be
served to tenant B for the same path.

**Vulnerable pattern:**
```go
// BAD: cache key omits tenant
cacheKey := r.Method + ":" + r.URL.Path
```

**Correct pattern:**
```go
// GOOD: include tenant in cache key
tenantID, _ := middleware.TenantIDFromRequest(r)
cacheKey := tenantID + ":" + r.Method + ":" + r.URL.Path
```

### 7.3 Cache Invalidation Scope

Cache invalidation must be scoped to the tenant. When tenant A updates a user, only tenant A's
cache should be invalidated. A broad `DEL user:*` command would invalidate all tenants' caches,
causing unnecessary cache misses and database load.

---

## 8. NATS Event Isolation

### 8.1 Current NATS Subject Structure

GGID's audit publisher uses a **flat subject** — all audit events are published to the same
NATS subject `audit.events`:

```go
// pkg/audit/publisher.go — current implementation
const (
    DefaultStreamName  = "AUDIT"
    DefaultSubjectName = "audit.events"  // NOT tenant-scoped
)

func (p *Publisher) Publish(ctx context.Context, event Event) error {
    data, err := json.Marshal(event)
    // ...
    _, err = p.js.Publish(ctx, p.subject, data)  // flat subject
    return err
}
```

The `tenant_id` is embedded in the JSON payload but NOT in the subject. This means:
- All consumers receive all tenants' events.
- No NATS-level filtering by tenant.
- Consumers must filter by `tenant_id` in the payload.

### 8.2 Recommended Tenant-Scoped Subject Structure

```go
// pkg/audit/tenant_publisher.go — tenant-scoped NATS publishing

package audit

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "github.com/nats-io/nats.go/jetstream"
)

// TenantScopedPublisher publishes audit events with tenant-scoped subjects.
type TenantScopedPublisher struct {
    js     jetstream.JetStream
    stream string
}

const (
    TenantStreamName = "AUDIT"
    // Subject pattern: audit.events.{tenant_id}
    // Wildcard consumer can use: audit.events.>
    // Per-tenant consumer can use: audit.events.{tenant_id}
    TenantSubjectTemplate = "audit.events.%s"
)

func NewTenantScopedPublisher(ctx context.Context, natsURL string) (*TenantScopedPublisher, error) {
    // Stream covers all tenant subjects via wildcard
    _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
        Name:     TenantStreamName,
        Subjects: []string{"audit.events.>"},  // wildcard — covers all tenants
        Retention: jetstream.LimitsPolicy,
        Storage:  jetstream.FileStorage,
        MaxAge:   72 * time.Hour,
    })
    // ...
}

// Publish publishes an audit event to a tenant-scoped subject.
func (p *TenantScopedPublisher) Publish(ctx context.Context, event Event) error {
    if event.TenantID == uuid.Nil {
        return fmt.Errorf("audit event requires tenant_id")
    }
    subject := fmt.Sprintf(TenantSubjectTemplate, event.TenantID.String())

    data, err := json.Marshal(event)
    if err != nil {
        return err
    }

    _, err = p.js.Publish(ctx, subject, data)
    return err
}
```

### 8.3 Consumer Permissions Per Tenant

NATS supports per-subject authorization. A tenant-specific consumer should only be allowed to
subscribe to its own tenant subject:

```go
// NATS account/user permissions for tenant-scoped consumption
permissions := &nats.SubjectPermission{
    Allow: []string{
        fmt.Sprintf("audit.events.%s", tenantID),
    },
    Deny: []string{
        "audit.events.>",  // deny wildcard — must be explicit
    },
}
```

### 8.4 Preventing Cross-Tenant Event Subscription

With the flat subject (`audit.events`), any consumer subscribing to the subject receives ALL
events. This is acceptable for the central audit service (which processes all events) but
dangerous for tenant-specific consumers. The tenant-scoped subject pattern ensures that:

- Central audit consumer subscribes to `audit.events.>` (wildcard).
- Tenant-specific consumers subscribe to `audit.events.{their_tenant_id}` only.
- NATS server authorization enforces that a tenant consumer cannot subscribe to another
  tenant's subject.

---

## 9. GGID RLS Audit

### 9.1 Tenant Context Management (`pkg/tenant/tenant.go`)

The tenant context package provides clean context propagation:

```go
type Context struct {
    TenantID       uuid.UUID
    IsolationLevel IsolationLevel  // shared | schema | database
    SchemaName     string
    Settings       map[string]any
}
```

**Finding:** The package is well-designed. It supports multiple isolation levels (shared,
schema, database). However, all services currently use `IsolationShared` hardcoded in the
middleware. There is no mechanism to switch a tenant to schema-level or database-level isolation
at runtime.

### 9.2 Service-by-Service RLS Audit

**Identity Service (`services/identity/internal/repository/pg_repo.go`):**
- **Pattern:** Every query method calls `setTenantRLS(ctx, tx, tenantID)` within a transaction.
- **SQL injection risk:** Uses `fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID.String())`.
  UUID format is safe, but the pattern should use parameterized queries.
- **Gap:** `ConsumeEmailVerificationToken` explicitly skips RLS ("No RLS needed here — token is
  globally unique"). The token query is by hash, not tenant-scoped. **Risk:** an attacker who
  obtains a verification token hash from any tenant can consume it.
- **Assessment:** Generally safe, but the SQL interpolation pattern is a latent vulnerability.

**Auth Service (`services/auth/internal/repository/`):**
- **MFA repo:** Uses parameterized `SET LOCAL app.tenant_id = $1` — correct and safe.
- **Credential repo:** Does NOT use `SET LOCAL`. Instead, uses explicit `WHERE tenant_id = $1`
  in queries. This is defense-in-depth but relies entirely on application-level filtering.
- **Gap:** If a credential query omits the tenant_id WHERE clause, there is no RLS backstop.

**Audit Service (`services/audit/internal/repository/audit_repo.go`):**
- **NO RLS at all.** Uses direct `r.db.QueryRow()` and `r.db.Query()` without transactions
  or `SET LOCAL`.
- **Critical gap:** `GetByID` queries `WHERE id = $1` with NO tenant filter. Any caller who
  knows an audit event UUID can read it regardless of tenant.
- **Critical gap:** `DeleteOlderThan` deletes `WHERE created_at < $1` with NO tenant filter.
  A scheduled cleanup job affects all tenants' data indiscriminately.
- **Assessment:** **HIGH RISK.** The audit service relies entirely on the gRPC handler to
  enforce tenant scoping. There is no database-level isolation.

**Policy Service:**
- Uses `WHERE tenant_id = $1` in queries (application-level filtering).
- No RLS enforcement at the database level.

**Org Service:**
- Uses `WHERE tenant_id = $1` in queries (application-level filtering).
- No RLS enforcement at the database level.

### 9.3 Queries That Bypass RLS

| Location | Query | Issue |
|---|---|---|
| `audit_repo.go:54` | `GetByID` — `WHERE id = $1` | No tenant filter |
| `audit_repo.go:278` | `DeleteOlderThan` — `WHERE created_at < $1` | No tenant filter |
| `pg_repo.go:669` | `ConsumeEmailVerificationToken` — `WHERE token_hash = $1` | No tenant context |
| Gateway `injectTenantIntoBody` | Body injection skipped if client sends `tenant_id` | Client can set arbitrary tenant |

---

## 10. Gap Analysis & Recommendations

### Summary of Findings

| # | Finding | Severity | Service |
|---|---|---|---|
| 1 | `X-Tenant-ID` header overrides JWT claim | **CRITICAL** | Gateway |
| 2 | Audit service has no RLS — `GetByID` and `DeleteOlderThan` are not tenant-scoped | **HIGH** | Audit |
| 3 | Gateway body injection skips when client sends `tenant_id` | **HIGH** | Gateway |
| 4 | NATS subject is flat — no tenant namespacing | **MEDIUM** | Audit/Infra |
| 5 | Identity service uses `fmt.Sprintf` for SET LOCAL | **LOW** | Identity |
| 6 | No tenant match validation between JWT and header | **HIGH** | Gateway |
| 7 | Cache keys may omit tenant ID | **MEDIUM** | Gateway |

### Action Items

**Action 1: Bind tenant_id to JWT claim (CRITICAL, 2-3 days)**

Replace the `TenantResolver` middleware priority so that the verified JWT `tenant_id` claim
takes precedence over the `X-Tenant-ID` header for authenticated requests. Add a
`TenantMatchValidator` middleware that rejects requests where the JWT tenant_id differs from
the header. See Section 5 for implementation.

**Action 2: Add RLS to audit service (HIGH, 3-5 days)**

Wrap all audit repository queries in transactions with `SET LOCAL app.tenant_id`. Add explicit
`tenant_id` filters to `GetByID` and `DeleteOlderThan`. The `DeleteOlderThan` function should
either accept a tenant_id parameter or be restricted to a maintenance role that operates
outside RLS (with explicit documentation).

**Action 3: Fix body injection bypass (HIGH, 0.5 days)**

Change `injectTenantIntoBody` to **always overwrite** the `tenant_id` field in the JSON body,
regardless of whether the client sent one. The gateway is the trust boundary — it must enforce
the authenticated tenant_id.

```go
// FIX: always overwrite, never skip
bodyMap["tenant_id"] = tenantID  // remove the existence check
```

**Action 4: Tenant-scope NATS subjects (MEDIUM, 2-3 days)**

Change the audit publisher to use `audit.events.{tenant_id}` subject format. Update the stream
configuration to accept `audit.events.>` wildcard. Update consumers to filter by tenant subject.
See Section 8 for implementation.

**Action 5: Parameterize SET LOCAL in identity service (LOW, 0.5 days)**

Replace `fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID)` with the parameterized
version already used in the auth service:
```go
tx.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID.String())
```

**Action 6: Add cross-tenant isolation test suite (MEDIUM, 2-3 days)**

Implement the test suite described in Section 4 as integration tests that run against a real
PostgreSQL database with RLS enabled. These tests should run in CI on every PR to prevent
regressions.

### Effort Summary

| Action | Effort | Priority |
|---|---|---|
| Bind tenant to JWT claim | 2-3 days | P0 |
| Add RLS to audit service | 3-5 days | P0 |
| Fix body injection bypass | 0.5 days | P0 |
| Tenant-scope NATS subjects | 2-3 days | P1 |
| Parameterize SET LOCAL | 0.5 days | P2 |
| Cross-tenant test suite | 2-3 days | P1 |
| **Total** | **10-15 days** | |

---

## Appendix A: References

- PostgreSQL Row Level Security: https://www.postgresql.org/docs/current/ddl-rls.html
- OWASP Multi-Tenant Security: https://owasp.org/www-project-web-security-testing-guide/
- GGID source: `/Users/zhanju/ggai/ggid`
  - Tenant context: `pkg/tenant/tenant.go`
  - Gateway middleware: `services/gateway/internal/middleware/middleware.go`
  - Tenant resolver (enhanced): `services/gateway/internal/middleware/tenant_enhanced.go`
  - Identity repo: `services/identity/internal/repository/pg_repo.go`
  - Auth MFA repo: `services/auth/internal/repository/mfa_pg_repo.go`
  - Audit repo: `services/audit/internal/repository/audit_repo.go`
  - NATS publisher: `pkg/audit/publisher.go`
  - Integration tests: `test/integration/e2e_test.go`

---

*This document is based on a direct source code audit of the GGID codebase. All code examples
are derived from actual GGID source files or are proposed improvements.*
