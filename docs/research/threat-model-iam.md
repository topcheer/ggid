# STRIDE Threat Model for GGID IAM System

> Comprehensive security assessment using the Microsoft STRIDE methodology.
> Grounded in source code analysis of all 7 microservices, shared packages, and
> infrastructure. Cross-references 20+ prior research documents.

**Document version:** 1.0
**Date:** 2025-01-20
**Scope:** GGID IAM Platform — Gateway, Identity, Auth, OAuth, Policy, Org, Audit services
**Classification:** Internal Security Research

---

## Table of Contents

1. [STRIDE Framework Overview](#1-stride-framework-overview)
2. [S — Spoofing Threats](#2-s--spoofing-threats)
3. [T — Tampering Threats](#3-t--tampering-threats)
4. [R — Repudiation Threats](#4-r--repudiation-threats)
5. [I — Information Disclosure](#5-i--information-disclosure)
6. [D — Denial of Service](#6-d--denial-of-service)
7. [E — Elevation of Privilege](#7-e--elevation-of-privilege)
8. [Complete Attack Tree](#8-complete-attack-tree)
9. [Risk Prioritization Matrix](#9-risk-prioritization-matrix)
10. [GGID Security Posture Summary](#10-ggid-security-posture-summary)

---

## 1. STRIDE Framework Overview

STRIDE is a threat categorization model developed by Microsoft that maps six
categories of threats to corresponding security properties:

| STRIDE Category | Threat | Security Property Violated |
|-----------------|--------|---------------------------|
| **S**poofing | Impersonating a user, service, or system | Authentication |
| **T**ampering | Modifying data or code | Integrity |
| **R**epudiation | Denying an action without proof | Non-repudiation |
| **I**nformation Disclosure | Exposing data to unauthorized parties | Confidentiality |
| **D**enial of Service | Disrupting service availability | Availability |
| **E**levation of Privilege | Gaining capabilities beyond authorization | Authorization |

### Why STRIDE for IAM

An Identity and Access Management system is uniquely critical because it IS the
trust boundary for every downstream application. A compromise of the IAM layer
compromises every system it protects. STRIDE is particularly well-suited because:

- **Spoofing** is the primary attack surface — all authentication flows (password,
  LDAP, OAuth, SAML, WebAuthn) must resist identity impersonation.
- **Tampering** of audit logs, JWTs, or SAML assertions directly undermines the
  system's ability to serve as a trusted authority.
- **Repudiation** gaps in an IAM system have legal and compliance consequences
  (SOC 2, GDPR accountability requirements).
- **Information disclosure** of tenant data in a multi-tenant system has
  cross-tenant blast radius.
- **Denial of service** against an IAM platform creates cascading failures in all
  dependent applications.
- **Elevation of privilege** in the authorization engine undermines every access
  control decision.

---

## 2. S — Spoofing Threats

### 2.1 Identity Spoofing: JWT Replay

**Attack scenario:** An attacker captures a valid JWT (via network sniffing, XSS,
or compromised proxy) and replays it before expiry. The JWT's `jti` claim is a
random UUID (`uuid.New().String()` in `token_service.go:85`) but is never tracked
or validated against a denylist.

**Current GGID status:**
- JWT tokens are RS256-signed with 15-minute access token TTL (configurable).
- **No `jti` denylist exists.** The gateway's `JWTAuth` middleware (`middleware.go:500-577`)
  validates signature, expiry, issuer, and audience but does not check if the
  token has been revoked.
- **No token binding** to client TLS certificate or DPoP proof (see `token-replay-defense.md`).
- Refresh tokens DO have replay detection via `RotateRefreshToken()` which revokes
  the entire session chain if a revoked token is reused (`token_service.go:155-161`).

**Risk rating:** HIGH — stolen tokens are fully usable until expiry.

**See:** `token-replay-defense.md`, `dpop-rfc9449.md`

### 2.2 Tenant Spoofing: X-Tenant-ID Header Manipulation

**Attack scenario:** An attacker sends a crafted `X-Tenant-ID` header to access
another tenant's data.

**Current GGID status:** **FIXED.** The `TenantResolver` middleware
(`middleware.go:241-281`) now prioritizes JWT `tenant_id` claims over the
`X-Tenant-ID` header:

```go
// 1. Try JWT claim tenant_id first (highest priority — authenticated source)
// X-Tenant-ID header is unauthenticated and must NOT override JWT claims.
if tidStr := extractTenantFromJWT(r); tidStr != "" { ... }
// 2. Try X-Tenant-ID header (only for public endpoints without JWT)
if tenantID == uuid.Nil {
    if tidStr := r.Header.Get("X-Tenant-ID"); tidStr != "" { ... }
}
```

The header is only used as a fallback for public endpoints (login/register) where
no JWT exists. For authenticated requests, the JWT `tenant_id` claim takes
priority.

**Risk rating:** LOW (mitigated). Remaining risk: public endpoints (login) still
accept arbitrary tenant IDs via header — an attacker could attempt login against
any tenant. This is by design (users must specify their tenant) but should be
rate-limited per tenant+IP.

**See:** `multi-tenant-isolation.md`

### 2.3 Service Spoofing: No mTLS on gRPC

**Attack scenario:** On the internal network, a rogue service connects to any
GGID gRPC server (identity :50051, policy :9070, org :9071, audit :9072). Without
mutual TLS, the server has no way to verify the caller's identity.

**Current GGID status:**
- All 4 gRPC servers use `grpc.NewServer()` with **no TLS credentials**:
  ```
  services/audit/cmd/main.go:72:     grpcServer := grpc.NewServer()
  services/policy/cmd/main.go:78:    grpcServer := grpc.NewServer()
  services/org/cmd/main.go:81:       grpcServer := grpc.NewServer()
  services/identity/.../server.go:46: grpcSrv := grpc.NewServer()
  ```
- No `credentials.NewServerTLSFromFile()` or transport credentials configured.
- The gateway proxy to backends uses plain HTTP (`proxy.Transport` has
  `ForceAttemptHTTP2: true` but no TLS config in the default path).
- In Docker Compose, services communicate over the bridge network without TLS.

**Risk rating:** MEDIUM — mitigated by network segmentation in production
(Kubernetes NetworkPolicies), but defense-in-depth requires mTLS.

**See:** `grpc-security-iam.md`, `zero-trust-iam.md`

### 2.4 OAuth Client Spoofing

**Attack scenario:** An attacker registers a malicious OAuth client or spoofs
an existing `client_id` to obtain authorization codes.

**Current GGID status:**
- `client_id` is looked up from the database per-request
  (`oauth_service.go:203`): `GetClientByID(ctx, req.TenantID, req.ClientID)`.
- Confidential clients require `client_secret` verification
  (`oauth_service.go:306-311`): uses `crypto.VerifyPassword()` with Argon2id.
- `redirect_uri` is validated against registered URIs via
  `client.ValidateRedirectURI()` (`oauth_service.go:212`).
- `state` parameter is required and stored for CSRF validation
  (`oauth_service.go:222-224`, `266-269`).
- PKCE is enforced for public clients (`oauth_service.go:233`).
- Authorization codes are single-use via atomic `ConsumeCode()`
  (`oauth_service.go:314`).

**Remaining gap:** The `state` validation at token exchange is incomplete. The
state is stored in an in-memory `sync.Map` (`stateStore`), not Redis, so it is
lost on restart and doesn't work in multi-instance deployments. The
`ExchangeAuthorizationCode` method does not validate the state parameter against
the stored value — it only checks code, client, redirect_uri, and PKCE.

**Risk rating:** LOW (client spoofing mitigated). MEDIUM for state CSRF (see
`oauth-state-csrf.md`, `cross-site-request-forgery-iam.md`).

---

## 3. T — Tampering Threats

### 3.1 Audit Log Tampering

**Attack scenario:** An attacker with database access modifies or deletes audit
events to cover their tracks.

**Current GGID status:**
- The audit domain model (`audit/internal/domain/models.go:45`) includes a
  `Hash` field described as "HMAC chain hash for tamper detection."
- A chain verification endpoint exists (`audit/internal/server/http.go:799-861`)
  that recomputes HMAC-SHA256 hashes across all events and flags tampered entries.
- The NATS JetStream publisher (`pkg/audit/publisher.go`) uses file storage with
  72-hour retention.
- **Gap:** The HMAC chain uses a shared secret key. If an attacker has DB write
  access AND knows the HMAC key, they can recompute valid hashes. The key should
  be stored in a separate secrets manager, not co-located with the database.
- **Gap:** Events published via `PublishAsync` are fire-and-forget — no delivery
  guarantee if NATS is temporarily unavailable.

**Risk rating:** LOW-MEDIUM. The hash chain exists and detects tampering, but the
HMAC key management and async delivery gaps weaken the guarantee.

**See:** `audit-tampering-detection.md`

### 3.2 JWT Tampering: Algorithm Confusion

**Attack scenario:** An attacker modifies the JWT header to use `alg: none` or
`alg: HS256` with the public key as HMAC secret, bypassing signature verification.

**Current GGID status:** **MITIGATED.** The `JWTAuth` middleware enforces
`jwt.WithValidMethods([]string{"RS256"})` (`middleware.go:527`) and explicitly
checks the signing method type:

```go
if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
    return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
}
```

The `golang-jwt/jwt/v5` library also rejects `none` by default. Both the issuer
and audience are validated via parser options (`middleware.go:530-533`).

**Risk rating:** LOW (mitigated).

**See:** `jwt-algorithm-confusion.md`

### 3.3 Data Tampering via SQL Injection

**Attack scenario:** An attacker injects SQL through API parameters to modify
or exfiltrate data.

**Current GGID status:** **CLEAN.** All database queries use parameterized
queries with `$1, $2, ...` placeholders. The `fmt.Sprintf` calls in repositories
are used only for **column list constants** (e.g., `fmt.Sprintf("SELECT %s FROM
users WHERE ...", userColumns)`), never for user input:

```
services/auth/internal/repository/mfa_repo.go:105:
    query := fmt.Sprintf(`SELECT %s FROM mfa_devices WHERE id = $1`, mfaColumns)
```

The `userColumns`, `mfaColumns`, etc. are package-level string constants, not
derived from user input. No string concatenation of user-supplied values into
SQL was found.

**Risk rating:** LOW (clean).

**See:** `sql-injection-iam-defense.md`

### 3.4 SAML Assertion Tampering: Signature Wrapping

**Attack scenario:** An attacker takes a valid signed SAML assertion, adds a
malicious unsigned assertion inside the same XML document, and the parser
extracts the attacker's assertion while the signature check passes on the
original.

**Current GGID status:**
- The SAML package (`pkg/saml/signed_assertion.go`) parses `SignedInfo`,
  `SignatureValue`, and `Reference` elements from the XML signature.
- `extractSignedInfoBytes()` preserves the exact `SignedInfo` bytes for signature
  verification.
- The code verifies the signature on the `SignedInfo` element and checks the
  `Reference.URI` against the assertion ID.
- **Gap:** The code does not verify that the signed element is the **outermost**
  assertion in the document. Signature wrapping attacks exploit this by nesting
  the signed assertion inside an attacker-controlled wrapper. A full
  countermeasure requires validating that the signature reference points to the
  root element of the parsed assertion, not a nested one.

**Risk rating:** MEDIUM — the SAML module is not fully production-hardened
(coverage at 91.1%, signature wrapping not explicitly tested).

**See:** `multi-tenant-saml.md`, `saml-sp-initiated-sso-design.md`

---

## 4. R — Repudiation Threats

### 4.1 Missing Audit Trails for Critical Operations

**What GGID logs:**
- Login attempts (success and failure) with IP and user-agent
  (`auth_service.go:84-88`, `http.go:278`).
- Audit events are published via `pkg/audit` NATS publisher for any service that
  explicitly calls `Publish()` or `PublishAsync()`.

**What GGID does NOT log:**
- **OAuth consent capture:** When a user grants consent to an OAuth client, no
  audit event is published recording the scopes granted, the client, and the
  timestamp. This breaks GDPR accountability — a user could deny granting consent
  with no evidence to the contrary.
- **Token refresh:** The `RotateRefreshToken` flow does not publish an audit
  event when a token rotation occurs, including when replay detection triggers
  session revocation.
- **Policy/role changes:** The Policy service creates and updates roles but the
  audit event publication is left to the caller. If the handler doesn't
  explicitly call `audit.Publish()`, the change is unlogged.
- **Gateway admin operations:** Route toggling (`handleAdminToggleRoute`),
  route reloading (`handleReloadRoutes`), and admin stats queries are not
  audit-logged.

**Risk rating:** MEDIUM — gaps in OAuth consent and admin action logging create
accountability gaps for compliance.

### 4.2 Log Deletion Without Trace

**Current GGID status:**
- NATS JetStream retention is 72 hours (`publisher.go:75`). Events older than
  72 hours are automatically purged from the stream.
- The Audit service persists events to PostgreSQL. The DELETE endpoint
  (`/api/v1/audit/webhooks?id=X`) only deletes webhook configurations, not
  events.
- **Gap:** There is no soft-delete or append-only enforcement on the audit
  events table. A user with direct DB DELETE access can remove events without
  leaving a trace (though the HMAC chain would detect the gap).
- **Gap:** No WORM (Write Once Read Many) storage or immutable log archival for
  long-term compliance retention.

**Risk rating:** MEDIUM.

### 4.3 No Non-Repudiation for Consent Capture

**Current GGID status:** As noted in 4.1, OAuth consent is not audit-logged.
For compliance with GDPR Article 7(1) ("demonstrate that the data subject has
consented"), the system must record:
- Which user consented
- To which client
- What scopes were granted
- When consent was given
- IP address and user-agent for attribution

None of this is currently captured.

**Risk rating:** HIGH for compliance-critical deployments.

### 4.4 Admin Actions Without Attribution

**Current GGID status:**
- Admin API endpoints (`/api/v1/admin/routes`, `/api/v1/admin/stats`,
  `/api/v1/gateway/routes/reload`) are JWT-protected but do not publish audit
  events recording which admin performed the action.
- User lock/unlock operations in the Identity service have no audit trail.
- Role assignments in the Policy service are not audit-logged at the service
  layer.

**Risk rating:** MEDIUM — admin actions are authenticated but not attributable
after the fact.

---

## 5. I — Information Disclosure

### 5.1 Cross-Tenant Data Leakage

**Current GGID status:** **MITIGATED.**
- All database queries are scoped by `tenant_id` (e.g., `WHERE tenant_id = $1`).
- PostgreSQL Row Level Security (RLS) policies enforce tenant isolation at the
  database level.
- The gateway injects `tenant_id` from authenticated JWT claims into both query
  params and JSON body for backend requests (`router.go:127-135`).
- **Remaining risk:** In Docker Compose, the database uses a superuser that
  bypasses RLS. Production deployments must use a non-superuser role.

**Risk rating:** LOW in production (with RLS). HIGH in Docker dev deployments.

**See:** `multi-tenant-isolation.md`

### 5.2 Error Message Information Leakage

**Attack scenario:** Verbose error messages expose internal system details
(database schema, stack traces, internal hostnames) that aid an attacker in
understanding the system's attack surface.

**Current GGID status:** **PARTIALLY MITIGATED.**
- The Org and Policy services use `writeServiceError()` which maps `GGIDError`
  codes to HTTP status codes with generic messages.
- **Gap:** The Auth service has numerous `writeError(w, http.StatusInternalServerError,
  err.Error())` calls that return raw Go error strings to the client:
  ```
  services/auth/internal/server/http.go:549:  writeError(w, http.StatusInternalServerError, err.Error())
  services/auth/internal/server/http.go:584:  writeError(w, http.StatusInternalServerError, err.Error())
  services/auth/internal/server/http.go:706:  writeError(w, http.StatusInternalServerError, err.Error())
  ```
  These could expose internal details like `"database connection refused"` or
  `"failed to parse RSA key"`.
- The gateway's route reload handler returns `err.Error()` directly:
  `"reload failed: " + err.Error()` (`router.go:480`).
- The healthcheck endpoint returns service error details:
  `healthcheck.go:185: Error: err.Error()`.

**Risk rating:** MEDIUM — internal error details aid reconnaissance.

### 5.3 PII in Logs Without Redaction

**Current GGID status:**
- The structured logger (`middleware.go:73-83`) logs request method, path,
  status code, response size, and request ID. It does NOT log request bodies,
  Authorization headers, or query parameters.
- **Gap:** The Auth service logs login failures with `req.Username` (which is
  typically the user's email) via `RecordLoginAttempt`:
  ```go
  h.authSvc.RecordLoginAttempt(r.Context(), req.Username, ip, userAgent, false, err.Error())
  ```
  This writes email addresses to application logs. Depending on the log
  aggregation system, these could be retained indefinitely.
- **Gap:** No PII redaction filter is applied to structured logs.

**Risk rating:** MEDIUM for GDPR/PII compliance.

### 5.4 API Response Over-Exposure

**Attack scenario:** API endpoints return more fields than necessary, exposing
internal state or sensitive data.

**Current GGID status:**
- User list/detail endpoints return full user objects including `password_hash`
  field name (though the value is hashed). The JSON struct tags determine what
  is serialized.
- The audit query API returns full event objects including `Metadata` which may
  contain sensitive context.
- OAuth client list returns `ClientSecretHash` in the struct (though it should
  be excluded from JSON serialization via `json:"-"` tags — needs verification).

**Risk rating:** LOW-MEDIUM.

---

## 6. D — Denial of Service

### 6.1 Rate Limiting on Auth Endpoints

**Current GGID status:** **PARTIALLY MITIGATED.**
- The gateway has `TenantBucketLimiter` wired into the middleware chain
  (`router.go:342`): `handler = gw.rateLimiter.Middleware(handler)`.
  Default: 100 burst, 10 req/sec sustained. Free tier: 20 burst, 2 req/sec.
- The Auth service has a secondary Redis-based rate limiter for login:
  `5 attempts per minute per IP` (`auth_service.go:85-88`).
- **Gap:** Rate limiting is keyed by `tenantID:ip` via `bucketKey()`. An attacker
  can rotate IPs (botnet, proxies) to bypass per-IP limits. There is no
  per-account or per-username rate limiting.
- **Gap:** The token bucket map grows unbounded. The `Cleanup()` method exists
  but is not called on a timer — memory exhaustion is possible under sustained
  attack with many unique IP+tenant combinations.

**Risk rating:** MEDIUM — basic rate limiting exists but can be circumvented.

**See:** `rate-limiting-iam.md`

### 6.2 No Connection Limits on gRPC

**Current GGID status:**
- gRPC servers are created with `grpc.NewServer()` and no `MaxConcurrentStreams`,
  `MaxRecvMsgSize`, or interceptors for connection limiting.
- No `grpc.KeepaliveParams` are configured to enforce minimum ping intervals or
  maximum connection age.
- An attacker can open thousands of gRPC streams, exhausting server goroutines
  and file descriptors.

**Risk rating:** MEDIUM.

**See:** `grpc-security-iam.md`

### 6.3 No Payload Size Limits

**Current GGID status:**
- The gateway does not enforce a global request body size limit. The proxy
  transport has connection pooling but no `MaxBytesReader` wrapper.
- gRPC servers have the default `MaxRecvMsgSize` of 4MB, but this is not
  explicitly configured and could be overridden by default behaviors.
- The `injectTenantIntoBody` function reads the entire request body into memory
  (`router.go:368: io.ReadAll(req.Body)`). A multi-gigabyte POST body would
  cause OOM.
- The GraphQL endpoint (`/graphql`) does not enforce query complexity limits.

**Risk rating:** MEDIUM.

**See:** `api-gateway-security.md`

### 6.4 Database Connection Pool Exhaustion

**Current GGID status:**
- pgx connection pools use default configuration. No `pool_max_conns` limit is
  explicitly enforced per service.
- Under load, each incoming request may hold a database connection. If the
  gateway forwards requests to all 5 backend services simultaneously, a single
  user request can consume 5+ DB connections.
- No circuit breaker exists between the gateway and backend services (though
  `circuit_breaker.go` middleware exists, it is not wired into the handler chain
  by default).

**Risk rating:** MEDIUM.

---

## 7. E — Elevation of Privilege

### 7.1 HasScope() Always Returns True for JWT-Authenticated Requests

**Attack scenario:** A regular user with a valid JWT accesses admin-only
endpoints because the gateway does not enforce per-route scope requirements.

**Current GGID status:**
- The `HasScope()` function (`apikey.go:60-71`) returns `true` when no scope
  restriction is in context:
  ```go
  func HasScope(ctx context.Context, scope string) bool {
      scopes, ok := ctx.Value(APIKeyScopesKey).([]string)
      if !ok {
          return true // No scope restriction if not using API key
      }
      ...
  }
  ```
- JWT-authenticated requests do NOT set `APIKeyScopesKey` in the context. The
  `JWTAuth` middleware (`middleware.go:567-574`) only sets `UserIDKey` and
  `TenantIDKey`. Therefore, `HasScope()` returns `true` for ALL JWT-authenticated
  requests.
- **No per-route scope check is enforced.** The gateway proxies all requests
  after JWT validation without checking if the user has the required scope for
  the target endpoint.
- JWT tokens issued by the Auth service do not include a `scope` or `roles`
  claim (`token_service.go:77-87` — claims are only `tenant_id`, `iss`, `sub`,
  `aud`, `iat`, `exp`, `jti`).

**Risk rating:** CRITICAL — any authenticated user can access any endpoint,
including user management, role assignment, and audit queries.

**See:** `oidc-scope-management.md`

### 7.2 Mass Assignment for Role Escalation

**Attack scenario:** An attacker adds extra fields (e.g., `"role": "admin"`,
`"is_admin": true`) to a legitimate PUT/PATCH request, and the backend deserializes
and persists them.

**Current GGID status:**
- The gateway's `injectTenantIntoBody()` function deserializes the JSON body
  into `map[string]any` and adds `tenant_id`. The backend services receive the
  full body.
- Identity service user update handlers accept JSON and deserialize into domain
  structs. If the struct includes fields like `IsAdmin` or `Role` with JSON tags,
  they could be set by the client.
- **Mitigation:** The domain models need verification — fields that should be
  server-controlled (roles, admin flags, status) must either not have JSON tags
  or must be explicitly overwritten server-side.

**Risk rating:** MEDIUM (needs per-struct verification).

**See:** `api-gateway-security.md`

### 7.3 JWT Claim Manipulation

**Attack scenario:** An attacker modifies JWT claims (e.g., changes `sub` to
another user's ID, or changes `tenant_id` to another tenant).

**Current GGID status:** **MITIGATED.**
- JWTs are RS256-signed. Any claim modification invalidates the signature.
- The gateway validates signature, issuer, and audience before extracting claims.
- The `tenant_id` claim is only extracted after successful JWT verification
  (though `TenantResolver` reads it pre-verification, it is overwritten by
  `JWTAuth` post-verification for the actual request context).

**Risk rating:** LOW.

### 7.4 Vertical Privilege Escalation (User to Admin)

**Attack scenario:** A regular user escalates to admin by exploiting missing
authorization checks in the Policy service or admin API.

**Current GGID status:**
- The Policy service has an RBAC + ABAC engine but it is not enforced at the
  gateway level. Any authenticated user can reach any backend endpoint.
- The admin API (`/api/v1/admin/*`, `/api/v1/gateway/*`) is JWT-protected but
  has no role check. Any authenticated user can toggle routes, reload config,
  and view backend statistics.
- User management endpoints (create, update, delete, lock/unlock) are not
  restricted to admin users.

**Risk rating:** CRITICAL — the authorization layer exists in Policy but is not
enforced on any request path.

---

## 8. Complete Attack Tree

**Top-level goal:** Compromise tenant data (exfiltrate, modify, or destroy)

```
[GOAL] Compromise Tenant Data
 |
 +-- [S1] Steal JWT (network/MITM/proxy)
 |    +-- No jti denylist → replay until expiry (15 min)
 |    +-- No token binding → use from any client
 |    +-- Access all endpoints (no scope enforcement)
 |    +-- Access cross-tenant data (tenant_id in JWT)
 |
 +-- [S2] Brute-force credentials
 |    +-- Rate limit: 5/min per IP (bypassable via IP rotation)
 |    +-- No per-account lockout after N failures
 |    +-- Argon2id hashing prevents offline cracking
 |
 +-- [S3] Exploit OAuth flow
 |    +-- State not validated at token exchange
 |    +-- In-memory state store (lost on restart, fails multi-instance)
 |    +-- Redirect URI strictly validated (mitigated)
 |    +-- PKCE enforced for public clients (mitigated)
 |
 +-- [E1] Access admin endpoints
 |    +-- HasScope() returns true for all JWT auth
 |    +-- No role check on /api/v1/admin/*
 |    +-- View all users, roles, audit events
 |    +-- Toggle routes, reload config
 |
 +-- [E2] Escalate via mass assignment
 |    +-- Inject admin/role fields in PUT/PATCH
 |    +-- Depends on struct serialization
 |
 +-- [I1] Read cross-tenant data
 |    +-- Forge X-Tenant-ID on public endpoints
 |    +-- List users from other tenants (if RLS bypassed)
 |    +-- RLS enforced in production (superuser in Docker)
 |
 +-- [I2] Extract data from verbose errors
 |    +-- err.Error() leaked in 500 responses
 |    +-- Reconnaissance: DB types, internal paths, connection errors
 |
 +-- [T1] Tamper audit trail
 |    +-- Delete events (no append-only enforcement)
 |    +-- HMAC chain detects but doesn't prevent
 |    +-- HMAC key co-located with DB
 |
 +-- [D1] Deny service to legitimate users
 |    +-- Exhaust rate limiter memory (no cleanup timer)
 |    +-- OOM via large POST body (no MaxBytesReader)
 |    +-- gRPC connection flood (no stream limits)
 |    +-- DB pool exhaustion (no per-service limits)
 |
 +-- [S4] Rogue internal service
      +-- No mTLS on gRPC
 |    +-- Connect to identity/policy/org gRPC
 |    +-- If network segmentation fails → full access
```

---

## 9. Risk Prioritization Matrix

| # | Threat | STRIDE | Likelihood | Impact | Risk Score | Current Mitigation | Recommended Fix | Priority |
|---|--------|--------|------------|--------|------------|-------------------|-----------------|----------|
| 1 | No per-route scope enforcement | E | HIGH | CRITICAL | **9.0** | None — HasScope() returns true for JWTs | Add scope/role claims to JWT; enforce per-route in gateway | **P0** |
| 2 | Admin API accessible to any authenticated user | E | HIGH | CRITICAL | **9.0** | JWT required only | Add role-based middleware for /admin/* and /gateway/* routes | **P0** |
| 3 | No JWT revocation (jti denylist) | S | MEDIUM | HIGH | **7.0** | Short TTL (15 min) | Redis-based jti denylist; DPoP token binding | **P0** |
| 4 | OAuth consent not audit-logged | R | HIGH | HIGH | **7.0** | None | Publish audit event on consent grant/deny | **P0** |
| 5 | No mTLS on gRPC servers | S | LOW-MED | HIGH | **6.0** | Network segmentation (K8s) | Add TLS credentials to all gRPC servers | **P1** |
| 6 | OAuth state CSRF incomplete | S | MEDIUM | HIGH | **6.5** | State required at authorize; not validated at token | Validate state at token exchange; move to Redis | **P1** |
| 7 | Error message information leakage | I | MEDIUM | MEDIUM | **5.0** | Org/Policy use writeServiceError | Replace err.Error() with generic messages in Auth service | **P1** |
| 8 | No request body size limit | D | MEDIUM | HIGH | **6.0** | None | Add http.MaxBytesReader in gateway | **P1** |
| 9 | Rate limiter memory unbounded | D | LOW-MED | MEDIUM | **4.5** | Cleanup() exists but not scheduled | Run Cleanup() on ticker; add max bucket count | **P1** |
| 10 | SAML signature wrapping | T | LOW-MED | HIGH | **5.5** | Signature verified on SignedInfo | Validate signed element is root assertion | **P1** |
| 11 | Admin actions not audit-logged | R | HIGH | MEDIUM | **5.5** | None | Add audit events for all admin operations | **P1** |
| 12 | No gRPC connection/stream limits | D | MEDIUM | MEDIUM | **5.0** | None | Add MaxConcurrentStreams, KeepaliveParams | **P1** |
| 13 | PII (email) in application logs | I | HIGH | MEDIUM | **5.5** | Structured logging (no bodies) | Hash/redact email in log messages | **P1** |
| 14 | No DB pool limits per service | D | LOW-MED | MEDIUM | **4.0** | pgx defaults | Set explicit pool_max_conns per service | **P2** |
| 15 | Rate limit bypass via IP rotation | D | MEDIUM | MEDIUM | **5.0** | Per-IP token bucket | Add per-account and per-username rate limiting | **P2** |
| 16 | HMAC key co-located with DB | T | LOW | MEDIUM | **3.5** | HMAC chain exists | Store HMAC key in separate secrets manager | **P2** |
| 17 | CORS allows all origins | I | LOW | LOW-MED | **3.0** | `Access-Control-Allow-Origin: *` | Restrict to known frontend domains in production | **P2** |
| 18 | Audit events async fire-and-forget | R | LOW | MEDIUM | **3.5** | NATS JetStream persistence | Add local fallback queue for delivery failures | **P2** |

**Risk score = (Likelihood × 0.5 + Impact × 0.5) on a 1-10 scale.**

---

## 10. GGID Security Posture Summary

### Strengths (What GGID Does Well)

1. **RS256 JWT with algorithm pinning** — `jwt.WithValidMethods(["RS256"])`
   prevents algorithm confusion attacks (see `jwt-algorithm-confusion.md`).

2. **Tenant resolver now secure** — JWT claims take priority over `X-Tenant-ID`
   header, preventing tenant spoofing on authenticated requests.

3. **Parameterized SQL queries** — No SQL injection vectors found. All user input
   flows through `$1, $2, ...` placeholders (see `sql-injection-iam-defense.md`).

4. **Argon2id password hashing** — Industry-standard memory-hard KDF for
   password storage and client secret hashing.

5. **OAuth flow fundamentals** — redirect_uri validation, PKCE enforcement,
   state parameter requirement, single-use authorization codes, atomic code
   consumption.

6. **SSRF protection on webhooks** — `NewSSRFSafeDeliverer` blocks private IPs,
   loopback, and cloud metadata endpoints with custom dialer.

7. **CSRF protection** — Double-submit cookie with crypto/rand-generated tokens,
   constant-time comparison.

8. **Refresh token rotation** — Replay detection with automatic session revocation
   on reuse of a rotated token.

9. **Audit HMAC chain** — Tamper-evident hash chain with verification endpoint
   for detecting post-hoc modification.

10. **Security headers middleware** — HSTS, X-Content-Type-Options, X-Frame-Options,
    Referrer-Policy applied to all responses.

### Critical Gaps (P0 — Fix Immediately)

1. **No authorization enforcement** — The RBAC/ABAC engine exists in the Policy
   service but is never consulted on any request path. Any authenticated user can
   access any endpoint including admin APIs. This is the single most critical
   security gap.

2. **No JWT revocation mechanism** — Stolen tokens are valid until expiry (15 min).
   No `jti` denylist, no session-based token invalidation, no DPoP binding.

3. **No OAuth consent audit trail** — Consent grants are not recorded, creating
   GDPR compliance exposure.

### Important Gaps (P1 — Fix Before Production)

4. **No mTLS on gRPC** — Internal service-to-service communication is unencrypted
   and unauthenticated.

5. **OAuth state CSRF incomplete** — State is not validated at token exchange and
   uses in-memory store that breaks multi-instance deployments.

6. **Verbose error messages** — Auth service returns raw `err.Error()` in 500
   responses, leaking internal system details.

7. **No request body size limits** — OOM risk from large POST bodies.

8. **SAML signature wrapping** — Not explicitly tested; the module may accept
   attacker-injected assertions.

9. **PII in logs** — Email addresses written to application logs without redaction.

10. **Admin actions not auditable** — No attribution for route changes, config
    reloads, or user management operations.

### Nice-to-Have (P2 — Hardening)

11. **Rate limiter memory management** — Schedule periodic cleanup, add max bucket
    count.

12. **DB pool limits** — Explicit per-service connection pool configuration.

13. **HMAC key separation** — Move audit HMAC key to dedicated secrets manager.

14. **CORS origin restriction** — Configure production-specific allowed origins
    instead of wildcard.

15. **Circuit breaker wiring** — The middleware exists but is not enabled in the
    default handler chain.

### Security Posture Scorecard

| Category | Score (1-10) | Notes |
|----------|-------------|-------|
| Authentication | **7.5** | Strong JWT, Argon2id, MFA. Lacks revocation and token binding. |
| Authorization | **2.0** | Engine exists but NOT enforced. Critical gap. |
| Data Protection | **7.0** | RLS, parameterized queries, encryption at rest. |
| Audit/Logging | **5.0** | HMAC chain is good; gaps in consent and admin logging. |
| Network Security | **4.0** | No mTLS, no gRPC limits, no body size limits. |
| Input Validation | **7.5** | SQL injection clean, redirect URI validated. |
| Error Handling | **5.0** | Mixed — some services good, Auth leaks errors. |
| DoS Resistance | **5.0** | Basic rate limiting; bypassable and incomplete. |
| **Overall** | **5.3** | Strong foundations, critical authorization gap. |

### Conclusion

GGID demonstrates solid cryptographic foundations and good defense-in-depth in
several areas (JWT algorithm pinning, SSRF protection, CSRF defense, refresh
token rotation). However, the system has one catastrophic gap: **the authorization
engine is not wired into any request path.** This means every authenticated user
effectively has admin-level access to every endpoint. Until per-route scope
enforcement and role-based access control are implemented in the gateway, the
system should not be deployed in production.

The second most urgent issue is the lack of JWT revocation — combined with no
scope enforcement, a single stolen token grants full admin access for its entire
TTL window.

Fixing the P0 items (authorization enforcement, JWT revocation, consent audit)
would raise the overall score from 5.3 to approximately 7.5, making GGID
suitable for production deployment with the P1 items addressed in a follow-up
hardening sprint.

---

*This threat model should be revisited after each major release or when new
services, endpoints, or authentication methods are added. The attack tree should
be expanded with mitigation branches as fixes are implemented.*
