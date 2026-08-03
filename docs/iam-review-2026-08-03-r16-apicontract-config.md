# R316 - API Contract (11th) + Configuration Management (11th) Deep Review

**Review Mode**: Read-only code audit, first exposure to ggid platform
**Scope**: `services/*/` | `pkg/` | `api/`
**Date**: 2026-08-03

---

## Part 1: API Contract Review

### P0 - Critical

#### P0-1: OpenAPI specs are empty shells - 6/6 swagger.json have `"paths": {}`

**Files**: `api/openapi/v1/{audit,auth,identity,oauth,org,policy}/v1/*.swagger.json`
**Status**: Persistent (R300 P2 -> upgraded to P0 this round)

All 6 swagger.json files are empty protoc-gen-openapiv2 outputs:
- `"paths": {}` - no HTTP path mappings
- `"version": "version not set"` - no semantic version
- `definitions` contains only `protobufAny` and `rpcStatus`, zero business schemas

**Root Cause**: `api/proto/buf.gen.yaml` has no openapi plugin. Proto files have zero `google.api.http` annotations. Without annotations, protoc-gen-openapiv2 cannot map RPCs to HTTP paths.

**Risk**: External integrators have no usable API documentation. Client SDK generation fails. API contract cannot be CI-enforced.

**Fix**: Add `google.api.http` annotations to all proto RPCs; add openapi plugin to `buf.gen.yaml`.

---

#### P0-2: gRPC vs REST severe asymmetry - REST far exceeds gRPC

**Files**: All `grpc_handler.go` vs `server/*.go`

**gRPC Services Actually Registered** (only 3 services have gRPC handlers):
| Service | gRPC Registered | RPCs in Proto |
|---------|----------------|---------------|
| auth | AuthServiceServer | 9 |
| identity | IdentityServiceServer | 17 |
| oauth | OAuthServiceServer | 5 |
| org | Has proto but NO RegisterServer evidence | 22 |
| policy | Has proto but NO RegisterServer evidence | 17 |
| audit | Has proto but NO RegisterServer evidence | 2 |

**REST features with NO gRPC equivalent**:
- **MFA/TOTP/WebAuthn** - auth has 49 handler files with MFA; proto has only `bool mfa_required` field
- **SCIM 2.0** - identity has 23 SCIM files; proto has zero SCIM definitions
- **Audit aggregation** - audit has dashboard/stats/detection/SOAR/UEBA handlers; proto has only 2 RPCs
- **API Keys** - auth/identity manage API keys; proto has none
- **Social/SAML login** - REST only

**Risk**: Microservices cannot call MFA/SCIM/audit features via gRPC. External gRPC clients access only a tiny subset. Proto as "API contract source" is unfulfilled.

**Fix**: Add MFA, SCIM, Audit aggregation RPCs to proto; register org/policy/audit gRPC servers.

---

#### P0-3: Error response format 4-way inconsistency

**Quantified** (services/ directory):

| Error Function | Calls | Response Format |
|----------------|-------|-----------------|
| `writeJSONError()` | **1411** (320 files) | `{"error": "string"}` (flat, no code) |
| `WriteSimpleAPIError()` | **113** (16 files) | `{"error": {"code": "...", "message": "..."}}` |
| `WriteAPIError()` | **4** (4 files) | `{"error": {"code", "message", "request_id"}}` |
| `http.Error()` | **10** (7 files) | `text/plain` (non-JSON) |

**Key Files**:
- `pkg/httputil/response.go:19-26` - `WriteError` and `WriteJSONError` are aliases, both output flat `{"error": "string"}` with NO code field
- `pkg/errors/api_error.go:50-82` - `WriteAPIError` outputs structured `{"error": {"code", "message", "request_id"}}`
- 1411 flat-format calls vs 117 structured calls = 92% use the wrong format

**Risk**: Clients cannot reliably parse errors. Two different JSON structures coexist. `http.Error()` returns text/plain breaking JSON API consistency.

**Fix**: Replace all `writeJSONError` with `WriteAPIError`/`WriteSimpleAPIError`. Delete `WriteJSONError` alias. Eliminate all `http.Error()` in REST handlers.

---

### P1 - High Priority

#### P1-1: Pagination strategy inconsistency - LIMIT/OFFSET 58 vs cursor 75

- `LIMIT ... OFFSET`: 25 files, 58 matches
- cursor/token: 10 files, 75 matches
- Proto definitions use `page_token`/`next_page_token` (cursor)
- REST implementations use `limit`+`offset` (OFFSET pagination)
- CLI uses `?page=X&page_size=Y` (page pagination)

**Files**: `policy/internal/service/role_service.go:135,402`, `org/internal/service/services.go:166,270,325`

**Risk**: Poor performance on large datasets with OFFSET. Three pagination parameter styles coexist.

**Fix**: Unify to cursor pagination (proto already defines correct pattern).

---

#### P1-2: Idempotency is memory-only - no persistence

- `Idempotency-Key` header appears only in test files
- `user_roles_handler.go:164` uses `ON CONFLICT DO NOTHING` (DB-level, good but isolated)
- No global Idempotency-Key persistence middleware

**Risk**: POST/PUT retries create duplicate data. Gateway restart loses idempotency state.

**Fix**: Implement Redis-backed Idempotency-Key middleware for critical write operations.

---

#### P1-3: HTTP status codes non-standard - POST/PUT return 200 instead of 201

`StatusCreated` appears only 37 times vs thousands of POST/PUT handlers using `WriteJSON(w, http.StatusOK, ...)`.

**Fix**: Audit all POST handlers; resource creation should return `201 Created`.

---

#### P1-4: publicPaths security - broad prefix matching

**File**: `services/gateway/internal/router/router.go:30-104`

- Line 77: `"/api/v1/auth/saml/"` - prefix wildcard may expose SAML config endpoints
- Line 96: `/.well-known/` - global wildcard may expose unexpected metadata
- `strings.HasPrefix` matching risks matching unintended subpaths

**Fix**: Use exact matching or route-tree matching instead of prefix matching for public paths.

---

### P2 - Medium Priority

#### P2-1: Proto package naming inconsistency

- `api.auth.v1`, `api.identity.v1`, `api.oauth.v1` - with `api.` prefix
- `ggid.audit.v1`, `ggid.org.v1`, `ggid.policy.v1` - with `ggid.` prefix

**Fix**: Unify to `ggid.<service>.v1`.

---

#### P2-2: JSON naming - snake_case vs camelCase coexist

- snake_case json tags: 605 files, 5728 matches (mainstream)
- camelCase json tags: 19 files, 76 matches (minority)
- Proto-generated pb.go auto-uses camelCase, conflicting with hand-written snake_case REST handlers

**Fix**: Unify to snake_case.

---

#### P2-3: API versioning only `/api/v1/` - no version negotiation

All routes hardcoded `/api/v1/` (70+ entries in router.go). No Accept header or path version flexibility.

---

## Part 2: Configuration Management Review

### P0 - Critical

#### P0-4: `conf` vs `config` package naming split - persistent 5+ rounds

- `package conf` (3 services): `auth/internal/conf/`, `oauth/internal/conf/`, `identity/internal/conf/`
- `package config` (6+ services): `gateway/`, `org/`, `policy/`, `audit/`, `ggid-cli/`

**Structural difference**:
- `conf` packages: YAML structured config + `Default()` + `LoadFromEnv()` override (e.g. `auth/conf/conf.go:79-182`)
- `config` packages: `FromEnv()` directly builds from env vars, no YAML (e.g. `org/config/config.go:19-36`)

**Risk**: Configuration source inconsistency. Operators must maintain two config patterns. Developer confusion.

**Fix**: Unify to `package config` with consistent YAML+ENV pattern (recommend conf package's approach).

---

#### P0-5: Default DB password `"ggid"` hardcoded - 5+ services

**Files**:
- `services/auth/internal/conf/conf.go:90` - `"postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable"`
- `services/oauth/internal/conf/conf.go:48` - same pattern
- `services/org/internal/config/config.go:27` - `getEnv("DB_PASSWORD", "ggid")`
- `services/policy/internal/config/config.go:28` - `getEnv("DB_PASSWORD", "ggid")`
- identity, audit (similar patterns expected)

**Risk**: If `DB_PASSWORD` env not set, production uses weak password `"ggid"`. `sslmode=disable` default disables SSL (auth/oauth), MITM risk.

**Mitigation**: oauth has `GGID_DEV_MODE` check (`oauth/cmd/main.go:116`), but default values remain dangerous.

**Fix**: Remove all password defaults; `log.Fatal` if not set in production. Change `sslmode=disable` to `sslmode=require` or no default.

---

#### P0-6: TLS no CipherSuites restriction

**Files**:
- `pkg/transport/tlsconfig.go:26-29` - `LoadServerTLS` sets only `MinVersion: TLS1.3`, no `CipherSuites`
- `pkg/transport/tlsconfig.go:50-53` - `LoadClientTLS` same
- `pkg/transport/grpc_tls.go:71-74` - `TLSServerConfig` same

TLS 1.3 cipher suites are constrained (Go auto-selects), so gRPC/TLS1.3 path is lower risk. But HTTP entry points (per-service `ServerConfig`) have no TLS config visible - services listen on plaintext HTTP (`:9001`, `:8071` etc), relying on gateway TLS termination.

**Fix**: Set explicit `CipherSuites` for all `tls.Config`. Add `PreferServerCipherSuites`.

---

### P1 - High Priority

#### P1-5: `os.Getenv` scattered - 259 occurrences / 73 files

**Env var prefix chaos** (7+ prefixes):

| Prefix | Example | Used By |
|--------|---------|---------|
| `AUTH_` | `AUTH_HTTP_ADDR` | auth service |
| `OAUTH_` | (oauth service) | oauth service |
| `IDENTITY_` | `IDENTITY_URL` | identity service |
| `DB_` | `DB_HOST`, `DB_PASSWORD` | org, policy |
| `REDIS_` | `REDIS_ADDR`, `REDIS_TLS` | multiple |
| `JWT_` | `JWT_ISSUER` | auth |
| `GRPC_` | `GRPC_TLS_ENABLED` | transport |
| `GGID_` | `GGID_ENV`, `GGID_DEV_MODE` | global |
| `NOTIFICATION_` | `NOTIFICATION_WEBHOOK_URL` | notification pkg |
| `NATS_` | `NATS_URL` | org |

**Synonym mismatch**: DB connection uses `DATABASE_URL` (auth/conf) vs `DB_HOST`+`DB_PORT`+`DB_USER`+`DB_PASSWORD` (org/config).

**Fix**: Route all config through `pkg/sysconfig` or structured config. Unify naming convention. Standardize DB connection to single `DATABASE_URL`.

---

#### P1-6: `GGID_ENV` vs `GGID_DEV_MODE` semantic overlap

**Files**:
- `pkg/middleware/internal_auth.go:38-43` - `GGID_ENV == "test" || GGID_ENV == "dev"`
- `services/identity/internal/server/jml_engine.go:73` - `GGID_ENV == "test" || GGID_DEV_MODE == "true"`
- `services/auth/internal/service/hooks.go:162` - same pattern
- `services/oauth/cmd/main.go:116` - `GGID_DEV_MODE != "true"`
- 10+ other locations with mixed checks

**Problem**: Two variables express similar meaning ("non-production") but checks are inconsistent:
- `GGID_ENV` accepts `"test"` and `"dev"`
- `GGID_DEV_MODE` is boolean `"true"`
- Some code checks both (`||`), some only one

**Risk**: Security-critical decisions (internal_auth fail-closed) depend on correct env var setting. Semantic ambiguity risks production misidentifying as dev.

**Fix**: Unify to `GGID_ENV` with values `production|staging|dev|test`.

---

#### P1-7: JWT key rotation only in oauth service

**File**: `services/oauth/internal/service/key_rotation.go:18-80`

oauth has complete `RotatingKeyProvider` (current + previous key + grace period). But:
- **auth service**: `JWTConfig` has only `PrivateKeyPath`/`PublicKeyPath` (single key, no rotation) - `auth/conf/conf.go:45-51`
- **identity service**: same as auth, single key
- Other JWT-issuing services: no rotation mechanism found

**Risk**: auth-issued tokens cannot rotate signing key. Key leak requires file replacement, invalidating all tokens (no grace period).

**Fix**: Promote `RotatingKeyProvider` to `pkg/crypto` or shared layer for all JWT-issuing services.

---

### P2 - Medium Priority

#### P2-4: Redis TLS config inconsistency

**Files**:
- `services/gateway/cmd/main.go:254-268` - Redis TLS uses `MinVersion: TLS1.2` (lower than gRPC TLS1.3)
- `services/oauth/internal/server/server.go:298` - has `redisTLSConfig()` but MinVersion unverified
- R302 fixed Redis TLS skipVerify production protection - needs verification across all services

**Fix**: Unify Redis TLS to `MinVersion: TLS1.2` + explicit CipherSuites. Ensure `InsecureSkipVerify` is guarded by `GGID_ENV != "production"`.

---

#### P2-5: Config defaults lack production safety checks

**File**: `services/oauth/cmd/main.go:116` - only oauth has dev mode check.

auth, identity, org, policy, audit have no "production forbids unsafe defaults" guard.

**Fix**: Add unified `config.Validate()` that rejects unsafe defaults (weak passwords, sslmode=disable, InsecureSkipVerify) in production mode.

---

## Summary

### Quantified Statistics (R316)

| Severity | Count | Fixed |
|----------|-------|-------|
| P0 | 6 | 0 (new/persistent) |
| P1 | 7 | - |
| P2 | 5 | - |

### Persistent P0 Issues (cross-round)

| Issue | Rounds Persistent | Status |
|-------|-------------------|--------|
| conf vs config split | 5+ (R300->R316) | Unfixed |
| os.Getenv scattered | Multiple | Unfixed |
| Default password "ggid" | Multiple (R308 dev default) | Unfixed |
| Error format mixed | Multiple (R300 flagged) | Unfixed |
| OpenAPI empty shell | Multiple (R300 P2->P0) | Unfixed |
| gRPC/REST asymmetry | Multiple | Unfixed |

### New Findings This Round

1. Proto package naming inconsistency (`api.` vs `ggid.` prefix)
2. gRPC registration incomplete - org/policy/audit have proto but no RegisterServer evidence
3. JWT key rotation only in oauth - auth/identity lack rotation
4. `GGID_ENV` vs `GGID_DEV_MODE` semantic overlap with inconsistent checks
5. publicPaths `/saml/` wildcard potential exposure
6. StatusCreated underuse - POST creation returns 200

### Comparison vs Previous Round

- R300 error format 3-way mix -> confirmed **4-way** (+ `http.Error` text/plain)
- R300 os.Getenv 262 -> **259** (slight decrease, essentially unchanged)
- R300 conf vs config -> **completely unchanged** (still 3 vs 6+ split)
- OpenAPI empty shell -> upgraded P2 to **P0** (paths:{} makes API docs completely unusable)
