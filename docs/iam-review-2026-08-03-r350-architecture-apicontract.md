# IAM Architecture Consistency (R15) + API Contract (R13) Review

**Date**: 2026-08-03  
**Reviewer**: Independent audit (first contact with ggid)  
**Scope**: services/gateway, services/auth, services/identity, services/oauth, services/audit  
**            api/openapi, api/proto, pkg/  

---

## Methodology & Code Paths Examined

### Architecture Consistency (10 dimensions)
- Gateway direct DB: grep'd all `pgxpool`/`pgx.Connect`/`pgx.Query`/`pgx.Exec` in services/gateway — found 10 non-test files with pgx imports, 28 direct SQL statements
- Config split: read all 5 `conf.go`/`config.go` files in each service
- Dependency direction: grep'd pkg/ for `services/` imports (zero found)
- DB driver: checked go.mod for pgx versions
- gRPC: grep'd all `grpc.NewServer`/`net.Listen`/`grpc.Serve`/`GRPCAddr` across services (non-test), plus `grpc.Dial`/`grpc.NewClient` consumers
- OpenAPI/proto: listed all files in api/openapi/v1/* and api/proto/*, read all 4 proto files, read auth.swagger.json
- Startup: read identity/cmd/main.go, audit/cmd/main.go (60% of 423 lines), identity/internal/server/server.go (full)
- Service communication: read gateway config routes map (all 60+ routes)
- DB config duplication: grep'd all `type DatabaseConfig struct`/`type DBConfig struct`

### API Contract (10 dimensions)
- OpenAPI: read all 6 swagger.json files (all 49 lines each)
- Proto sync: read all 4 proto files, cross-referenced with HTTP handlers
- Pagination: grep'd `page_size`/`page_token` (40 files, 149 matches) vs `limit/offset` (21 files, 47 matches)
- Error format: counted writeJSON (394 files, 1835 matches), http.Error (7 files, 10 matches), httputil.WriteError (1 file)
- Content-Type: grep'd `Content-Type.*application/json` across services
- Idempotency: grep'd `Idempotency`/`idempotency` across services
- HTTP status codes: grep'd identity server (123 files, 868 matches)
- API versioning: grep'd `/api/v1/` patterns in identity (101 files, 628 matches)
- Field naming: examined proto snake_case vs Go struct tags

---

## FINDINGS

### ============================================================
### ANGLE 1: Architecture Consistency (15th Round)
### ============================================================

---

### ARCH-01 [P0] — Gateway Direct PostgreSQL Access: 10 Files, 28 SQL Statements Bypassing Identity Service

**Files** (non-test):
- `services/gateway/internal/router/quickstart_handler.go:11` — `pgx.Connect` direct
- `services/gateway/internal/router/integration_handlers.go:15` — `pgx.Connect` direct
- `services/gateway/internal/middleware/apikey_db.go:17` — `pgxpool.Pool` direct
- `services/gateway/internal/middleware/plugin_repo.go:50` — `pgxpool.Pool` direct
- `services/gateway/internal/middleware/wasm_lifecycle.go:12` — `pgxpool.Pool` direct
- `services/gateway/internal/middleware/graphql_enhanced.go:15` — `pgxpool.Pool` direct
- `services/gateway/internal/middleware/consent.go` — direct SQL
- + 3 more files (10 total non-test)

**SQL statements include**: `SELECT id FROM tenants`, `SELECT count(*) FROM tenants`, `SELECT count(*) FROM tenant_access_consents`, `UPDATE api_keys SET last_used_at`, `DELETE FROM plugins`, `SELECT FROM wasm_plugin_hook_bindings`, etc. (28 total)

**Risk**: The API Gateway — which should be a pure HTTP reverse proxy with auth middleware — directly connects to PostgreSQL, bypassing the Identity and Auth services. This violates the microservice boundary contract. The Gateway directly reads/writes `tenants`, `api_keys`, `plugins`, `tenant_access_consents` — tables owned by identity/auth/policy services. Any schema change in those services breaks the Gateway. Cross-tenant data leakage risk if Gateway SQL queries lack tenant_id scoping.

**Fix**: Remove all direct DB access from Gateway. Move API key validation, tenant quickstart, plugin management, and consent queries to the appropriate backend services via HTTP or gRPC calls.

---

### ARCH-02 [P0] — gRPC Servers Declared But Zero Consumers (All gRPC Infrastructure is Dead Code)

**gRPC servers running**:
- `services/identity/internal/server/server.go:295` — listens on `:9090` (default)
- `services/audit/cmd/main.go:149` — listens on `:9072` (AUDIT_GRPC_ADDR)
- `services/auth/cmd/main.go:483` — listens on `AUTH_GRPC_ADDR` (optional)
- `services/oauth/internal/server/grpc_handler.go:229` — listens on `OAUTH_GRPC_ADDR`
- `services/policy/cmd/main.go:39` — gRPC server
- `services/org/internal/config/config.go:21` — `ORG_GRPC_ADDR` default `:9071`

**gRPC consumers**:
- `grpc.Dial` in non-test production code: **0** (only in comments in `pkg/transport/tlsconfig.go` and `gateway/internal/middleware/requestid_propagation.go`)
- `grpc.NewClient` in non-test production code: **0**

**Risk**: 6 services start gRPC servers that nobody calls. All inter-service communication goes through HTTP (the Gateway reverse proxies HTTP to backend services). The gRPC infrastructure — TLS config, interceptors, proto generation, secure opts — is fully wired but produces zero business value. Expands attack surface (6 extra TCP listeners), wastes resources, and creates false confidence that services communicate via typed gRPC contracts.

**Fix**: Either (a) remove all gRPC server code since it's unused, or (b) migrate inter-service calls to actually use gRPC. Current state is the worst of both worlds.

---

### ARCH-03 [P0] — 6/6 OpenAPI Swagger Files Are Empty Shells (paths: {})

**Files** (all identical structure):
- `api/openapi/v1/auth/v1/auth.swagger.json` — `"paths": {}`, only 2 definitions (protobufAny, rpcStatus)
- `api/openapi/v1/identity/v1/identity.swagger.json` — same
- `api/openapi/v1/oauth/v1/oauth.swagger.json` — same
- `api/openapi/v1/audit/v1/audit.swagger.json` — same
- `api/openapi/v1/org/v1/org.swagger.json` — same
- `api/openapi/v1/policy/v1/policy.swagger.json` — same

**All files**: `info.version: "version not set"`, `paths: {}`, definitions only contain generic protobuf error types.

**Risk**: Zero API contract documentation. The OpenAPI specs are auto-generated skeletons from proto compilation but contain no REST endpoint definitions. External integrators, SDK generators, and API gateways cannot use these. The proto files define gRPC services that nobody consumes (ARCH-02), while the actual REST API surface (hundreds of HTTP handlers across identity, auth, audit, oauth) is completely undocumented.

**Fix**: Write proper OpenAPI 3.x specs for the actual REST API surface (the `/api/v1/*` HTTP handlers), or annotate HTTP handlers with swagger:operation directives and generate specs from those.

---

### ARCH-04 [P1] — conf vs config Package Name Split (3:2 Services)

**Inconsistency**:
| Service | Package | File |
|---------|---------|------|
| auth | `conf` | `internal/conf/conf.go` |
| identity | `conf` | `internal/conf/conf.go` |
| oauth | `conf` | `internal/conf/conf.go` |
| gateway | `config` | `internal/config/config.go` |
| audit | `config` | `internal/config/config.go` |

**Risk**: Minor — developers switching between services encounter different import paths for identical concepts. Makes shared tooling harder.

**Fix**: Standardize on one name. `config` is the Go convention; `conf` is non-standard.

---

### ARCH-05 [P1] — 7+ Duplicated Database Configuration Structs

**Definitions found**:
- `services/auth/internal/conf/conf.go:33` — `DatabaseConfig{URL, MaxOpenConns, MaxIdleConns}`
- `services/identity/internal/conf/conf.go:25` — `DBConfig{URL, MaxConns, MinConns, MaxConnLifetime, MaxConnIdleTime}`
- `services/oauth/internal/conf/conf.go:31` — `DBConfig{URL, MaxConns, MinConns, MaxConnLifetime, MaxConnIdleTime}`
- `services/audit/internal/data/db.go` — `data.Config{Host, Port, User, Password, Database, SSLMode, MaxConns, ...}`
- `services/audit/internal/config/config.go:16` — embeds `data.Config`
- `deploy/operator/api/v1alpha1/ggidinstance_types.go:42` — `DatabaseConfig`
- `deploy/operator/internal/api/server.go:198` — `DBConfig`
- `services/identity/internal/data/` — `data.DBConfig` (separate from conf.DBConfig)

**Inconsistencies**:
- auth uses `MaxOpenConns`/`MaxIdleConns`; identity/oauth use `MaxConns`/`MinConns` (different field names)
- audit uses individual fields (Host, Port, User, ...) instead of connection URL
- identity has BOTH `conf.DBConfig` AND `data.DBConfig` — two structs in the same service for the same purpose

**Risk**: Configuration drift. Connection pool tuning differs per service. No shared validation. Adding a new DB config field (e.g. `SSLRootCert`) requires changes in 5+ places.

**Fix**: Extract a shared `pkg/db.Config` struct that all services import.

---

### ARCH-06 [P1] — Identity HTTP Default Port Conflict with Gateway (:8080)

**File**: `services/identity/cmd/main.go:34`  
```go
httpAddr = flag.String("http-addr", ":8080", "HTTP listen address")
```

Gateway also defaults to `:8080` (`services/gateway/internal/config/config.go:56`). In dev mode (all services on one machine), identity and gateway both bind `:8080` — one will fail.

**Risk**: Developer onboarding friction, deployment failures in all-in-one mode.

**Fix**: Change identity default to `:8081` (which is what the Gateway's route map expects: `USERS_SERVICE_URL` defaults to `http://localhost:8081`).

---

### ARCH-07 [P2] — DB Driver Consistency: pgx/v5 Only (No Issue, Verified)

`go.mod`: `github.com/jackc/pgx/v5 v5.10.0` — single version, no pgx/v4.  
All service imports use `github.com/jackc/pgx/v5`. **No inconsistency found.** This is clean.

---

### ARCH-08 [P2] — pkg/ Has No Reverse Dependencies on services/ (No Issue, Verified)

`grep "github.com/ggid/ggid/services/" in pkg/` → 0 matches.  
pkg/ is properly layered below services/. **No issue.**

---

### ARCH-09 [P1] — Identity Service Monolith: 20+ Subsystems in Single Server

**File**: `services/identity/internal/server/server.go:80-159+`

The Identity service `New()` constructor initializes:
- SCIM token management (line 88)
- ReBAC tuple store (line 95)
- JML lifecycle engine (line 101)
- Data governance compliance engine (line 109)
- ZTNA Access Broker (line 116)
- Device posture + compliance (line 123)
- Async import job system (line 130)
- Privilege creep detection (line 137)
- Privileged operations audit log (line 144)
- Automated access review scheduling (line 151)
- NHI PG repos (line 158)
- + more (file is 327 lines, only read to line 159)

Each subsystem calls `EnsureSchema(ctx)` at startup, meaning the Identity service creates 10+ database tables on boot. This is a distributed monolith — one binary doing IAM, SCIM, ReBAC, ZTNA, JML, compliance, posture, import, and access reviews.

**Risk**: A schema bug in any subsystem prevents Identity service from starting. Deployment coupling. Cannot scale subsystems independently.

**Fix**: Extract mature subsystems (ZTNA, compliance, access reviews) into separate services or at minimum separate binaries sharing the identity proto.

---

### ARCH-10 [P1] — Gateway Route Map Has 60+ Routes to 7 Backends via Hardcoded HTTP Proxy

**File**: `services/gateway/internal/config/config.go:63-136`

The Gateway is a pure HTTP reverse proxy with a massive static route table. All 60+ API path prefixes map to 7 backend URLs via `map[string]string`. No service discovery, no dynamic configuration, no health-aware load balancing.

**Risk**: Adding a new API requires a code change + redeploy. No circuit breaker per backend. If identity service is down, `/api/v1/users` returns 502 with no degradation strategy.

**Fix**: Move route table to configuration file (YAML/etcd). Add health-check-based routing.

---

### ============================================================
### ANGLE 2: API Contract (13th Round)
### ============================================================

---

### API-01 [P0] — Proto Defines 39 RPCs But Zero Are Consumed (Proto-HTTP Sync Gap)

**Proto RPCs defined**:
- `auth.proto`: 9 RPCs (Login, Register, Logout, RefreshToken, ForgotPassword, ResetPassword, ChangePassword, ListSessions, RevokeSession)
- `identity.proto`: 15 RPCs (CreateUser, GetUser, ListUsers, UpdateUser, DeleteUser, LockUser, UnlockUser, RegisterUser, VerifyEmail, ListUserEmails, AddUserEmail, RemoveUserEmail, SetPrimaryEmail, ListExternalIdentities, LinkExternalIdentity, UnlinkExternalIdentity)
- `oauth.proto`: 5 RPCs (CreateClient, GetClient, ListClients, UpdateClient, DeleteClient)
- `audit.proto`: 2 RPCs (ListEvents, GetEvent)

**Total**: 31 RPCs in proto files.  
**Actual gRPC consumers**: 0 (per ARCH-02).

**Meanwhile**: Each service has hundreds of HTTP handlers (`/api/v1/*`) that are NOT in the proto files. For example, identity has 628 matches for `/api/v1/` patterns across 101 files — but proto only defines 15 operations.

**Proto coverage**: ~15/394+ handlers = **<4% of actual API surface**.

**Risk**: The proto files are aspirational artifacts, not contracts. Any client attempting to use them will find they don't match reality. Schema evolution is uncontrolled since the real API surface (HTTP handlers) has no IDL.

**Fix**: Either (a) delete proto files since gRPC is unused, and write OpenAPI specs for the REST API; or (b) actually implement gRPC handlers and migrate clients.

---

### API-02 [P0] — Dual Pagination Standards: page_size/page_token vs limit/offset

**Statistics**:
- `page_size`/`page_token` (cursor-based): 40 files, 149 matches
- `limit/offset`: 21 files, 47 matches

**Inconsistency examples**:
- `identity/internal/server/http.go:808` — uses `page_size` query param
- `identity/internal/repository/pg_repo.go:349` — uses `filter.PageSize` (capped at 100)
- `oauth/internal/server/server.go` — uses `limit/offset`
- `policy/internal/service/role_service.go` — uses `limit/offset`
- `org/internal/service/services.go` — uses `limit/offset`

**Proto definition**: `identity.proto:132-145` — `ListUsersRequest` uses `page_size`/`page_token` (cursor-based).  
**Proto definition**: `audit.proto:38-50` — `ListEventsRequest` uses `page_size`/`page_token`.

But actual REST API is mixed — some endpoints return `next_page_token`, others return `total`/`page`/`per_page`.

**Risk**: Clients cannot write generic pagination logic. Each endpoint requires custom handling. Offset-based pagination has performance issues at scale (OFFSET 100000).

**Fix**: Standardize on cursor-based pagination (`page_size` + `next_page_token`) across all list endpoints.

---

### API-03 [P0] — Error Response Format: 3 Competing Patterns

**Statistics** (non-test files):
| Pattern | Files | Matches |
|---------|-------|---------|
| `writeJSON(w, status, map[string]interface{})` | 394 | 1,835 |
| `http.Error(w, msg, code)` | 7 | 10 |
| `httputil.WriteError()` / `httputil.WriteJSONError()` | 1 | 1 |

The `writeJSON` pattern (local helper per service) is dominant but each service defines its own `writeJSON`:
- `gateway/internal/webhooks/webhooks.go` — `writeJSON(w, code, map[string]string{"error": msg})`
- `identity/internal/server/*.go` — similar local helpers
- `oauth/internal/server/*.go` — similar

**Response shapes differ**:
- Some: `{"error": "message"}`
- Some: `{"error": {"code": "NOT_FOUND", "message": "..."}}`
- Some: `{"detail": "message"}`
- `http.Error()` produces plain text, not JSON

**pkg/errors coverage**: `httputil.WriteError` (the standardized error writer in pkg/errors) is used in only **1 file** out of 400+. Despite having a proper error package (`pkg/errors` with `APIError`, `CodeToHTTPStatus`, etc.), 99.7% of error responses bypass it.

**Risk**: API consumers cannot build reliable error handling. Error parsing varies by endpoint. No machine-readable error codes.

**Fix**: Migrate all error responses to `httputil.WriteError()` or `httputil.WriteJSONError()` with consistent `{"error": {"code": "...", "message": "..."}}` shape.

---

### API-04 [P1] — No API Versioning Strategy (v1 Everywhere, No v2 Path)

**Statistics**: 101 files in identity reference `/api/v1/` (628 matches). Zero files reference `/api/v2/`.

**Proto**: `api/proto/*/v1/` — all proto packages are `v1`.  
**OpenAPI**: `api/openapi/v1/` — all swagger files are `v1`.  
**Gateway routes**: All `/api/v1/*` — no `/api/v2/*` routes.

**Risk**: No mechanism for backward-compatible API evolution. Breaking changes must either (a) break all clients, or (b) be avoided indefinitely. No header-based or path-based version negotiation exists.

**Fix**: Define a versioning policy (path-based `/api/v2/` or header-based `Api-Version: 2`). Document breaking change process.

---

### API-05 [P1] — Field Naming: Proto snake_case vs JSON tag inconsistency

**Proto**: All proto fields use `snake_case` (e.g. `access_token`, `refresh_token`, `page_size`).  
**Go HTTP handlers**: Mix of `snake_case` JSON tags and `camelCase`:

From `auth/conf.go:54-65`:
```go
type PasswordPolicy struct {
    MinLength      int           `json:"min_length" yaml:"min_length"`
    RequireUpper   bool          `json:"require_upper" yaml:"require_upper"`
}
```

But some handler responses use camelCase (common in JavaScript-origin APIs).

**Risk**: API consumers must handle mixed casing. Proto-generated JSON uses camelCase by default (proto3 JSON mapping), while hand-written handlers use snake_case. The same field may appear as `accessToken` or `access_token` depending on whether the endpoint is gRPC-Gateway or hand-written HTTP.

**Fix**: Standardize on snake_case for all JSON responses (set `json_names_as_camel_case: false` in proto buf generation config).

---

### API-06 [P1] — Idempotency Support Fragmentary (Only Gateway Coalesce Middleware)

**Findings**:
- `gateway/internal/middleware/adaptive_geo_dedup.go:173` — Request Deduplication (Idempotency-Key) comment
- `gateway/internal/middleware/coalesce_idempotency_test.go` — test only
- `identity/internal/server/user_roles_handler.go:175` — DB-level `ON CONFLICT DO NOTHING` (not HTTP idempotency)

**No service-level Idempotency-Key header processing** exists for POST/PUT operations. No Redis-backed idempotency store. No `Idempotency-Key` header in any OpenAPI spec (which are empty anyway).

**Risk**: Retried requests (network failures, client retries) create duplicate resources. Role assignment, user creation, and OAuth client creation are not idempotent.

**Fix**: Add `Idempotency-Key` header support for all POST/PUT handlers. Store key+response in Redis with TTL (24h).

---

### API-07 [P1] — HTTP Status Code Usage Analysis

**Identity server**: 868 matches for standard status codes across 123 files. Dominant patterns:
- `http.StatusOK` (200) for successful GET/PUT
- `http.StatusCreated` (201) for POST creates
- `http.StatusBadRequest` (400) for validation errors
- `http.StatusNotFound` (404) for missing resources
- `http.StatusInternalServerError` (500) for errors

**Issues**:
- No `http.StatusConflict` (409) for duplicate resource creation — many handlers return 400 or 500 instead
- No `http.StatusTooManyRequests` (429) in handlers — rate limiting is done at middleware level but the status code is not consistently returned
- No `http.StatusGone` (410) for soft-deleted resources
- Audit main.go `/healthz` returns 200 with `"ok"` plain text (line 167-168) — inconsistent with JSON API contract

**Risk**: REST semantics violated. Clients cannot distinguish "already exists" (409) from "bad input" (400).

**Fix**: Audit all error paths and map to correct HTTP status codes per RFC 9110.

---

### API-08 [P2] — Content-Type Handling: Manual Header Setting (470 matches)

**Pattern**: 470 matches for `Content-Type` handling across services (non-test). Each handler manually sets `w.Header().Set("Content-Type", "application/json")`. No middleware-level Content-Type enforcement.

**Issues**:
- No `Accept` header negotiation — all endpoints return JSON regardless
- No Content-Type validation on request bodies (some handlers call `json.NewDecoder(r.Body).Decode()` without checking `Content-Type: application/json`)
- Audit healthz endpoint returns `text/plain` (line 167) while everything else returns JSON

**Risk**: Clients sending XML or form-encoded data get silent misinterpretation. No defense against content-sniffing attacks.

**Fix**: Add a Content-Type validation middleware that rejects non-JSON requests to JSON endpoints. Use `httputil.WriteJSON()` everywhere.

---

### API-09 [P2] — Request/Response Structure Inconsistency: Empty Delete/Optimized Responses

**Proto pattern**:
- `DeleteUserResponse { bool success = 1; }`
- `DeleteClientResponse { bool success = 1; }`
- `LogoutResponse {}` (empty)
- `ResetPasswordResponse {}` (empty)

**HTTP handler pattern**: Some delete endpoints return the deleted entity, some return `{"success": true}`, some return empty 204. No consistency.

**Risk**: Clients must inspect each endpoint individually to know what the response body contains.

**Fix**: Standardize: deletes return 204 No Content (empty body). Operations with side effects return the updated entity.

---

### API-10 [P2] — Audit proto ListEventsResponse includes `total` (Performance Anti-Pattern)

**File**: `api/proto/audit/v1/audit.proto:52-56`
```proto
message ListEventsResponse {
  repeated AuditEvent events = 1;
  string next_page_token = 2;
  int32 total = 3;  // <-- requires COUNT(*) query
}
```

Same in `identity.proto:141-145` (`ListUsersResponse.total`).

**Risk**: Returning `total` count requires a separate `COUNT(*)` query on every list request. For audit events (potentially millions of rows), this is expensive. Combined with cursor-based pagination, `total` is semantically incorrect (it changes between pages).

**Fix**: Remove `total` field from cursor-based paginated responses. If total count is needed, provide a separate `/count` endpoint.

---

## Summary

### Statistics by Severity

| Severity | Count | IDs |
|----------|-------|-----|
| **P0** | 5 | ARCH-01, ARCH-02, ARCH-03, API-01, API-02, API-03 |
| **P1** | 8 | ARCH-04, ARCH-05, ARCH-06, ARCH-09, ARCH-10, API-04, API-05, API-06, API-07 |
| **P2** | 5 | ARCH-07, ARCH-08, API-08, API-09, API-10 |

### Top Priority Actions

1. **P0 ARCH-01**: Remove Gateway direct PostgreSQL access (10 files, 28 SQL statements)
2. **P0 ARCH-02 + API-01**: Decide gRPC fate — either consume or delete (6 servers, 0 consumers, <4% proto coverage)
3. **P0 ARCH-03 + API-01**: Write real OpenAPI specs for REST API surface (6/6 are empty shells)
4. **P0 API-02**: Standardize pagination (page_size/page_token across all endpoints)
5. **P0 API-03**: Migrate all error responses to `httputil.WriteError()` (1/400+ files currently use it)

### Clean Areas (No Issues)
- DB driver: pgx/v5 only, no version split
- Dependency direction: pkg/ has zero reverse dependencies on services/
- gRPC security: TLS + internal-auth interceptors properly wired (even if unused)
