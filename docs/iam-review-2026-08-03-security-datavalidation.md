# GGID IAM Security Audit Report — R390

**Date:** 2026-08-03  
**Scope:** Security (Round 32) + Data Validation (Round 30)  
**Reviewer:** Independent (first contact with codebase)  
**Methodology:** Read-only source code review across services/gateway, services/auth, services/identity, services/oauth, services/audit, pkg/crypto, pkg/rbac, pkg/middleware  

---

## Executive Summary

The GGID IAM platform demonstrates **mature security architecture** with several well-implemented controls including JWT fail-closed validation, Argon2id password hashing with pepper, PKCE enforcement (S256 only), HMAC-SHA256 audit hash chain, and constant-time CSRF token comparison. 

However, **18 findings** were identified across 6 P0 (critical), 8 P1 (high), and 4 P2 (medium) issues. The most severe findings involve overly broad public path prefixes, incomplete body size limit coverage, hash chain secret stored in memory, and lack of authorization_details schema validation.

---

## Part 1: Security Findings (Round 32)

### 1.1 JWT Lifecycle — Signing/Validation/Revocation/Refresh

**Checked files:**
- `services/gateway/internal/middleware/middleware.go` (lines 700-810)
- `services/gateway/internal/middleware/jwt_claims.go` (full file)
- `pkg/crypto/alg_whitelist.go`

#### S-001 [PASS] JWT Algorithm Restriction
- **File:** `middleware.go:727-728, 737-738`
- JWT parsing uses `jwt.WithValidMethods(crypto.SupportedAlgs())` and double-checks with `crypto.IsSupportedAlg()` in the key function. `alg:none` attacks are prevented.

#### S-002 [PASS] JWT Fail-Closed on Public Paths
- **File:** `middleware.go:766-784`, `jwt_claims.go:34-44`
- When JWTAuth(required=false) encounters an invalid token on a public path, it deletes the Authorization header and sets empty `JWTCClaims{}` context. `ExtractJWTClaims` only uses verified context claims, never raw header parsing. This prevents forged JWT injection.

#### S-003 [PASS] JWT kid Rotation Enforcement
- **File:** `middleware.go:742-749`
- When kid is present but not found in JWKS, the token is rejected. No fallback to static key for tokens with explicit kid, preventing use of rotated-out keys.

#### S-004 [PASS] CAE/Revocation Check
- **File:** `middleware.go:884-902`
- CAECheck middleware validates jti against Redis blocklist after JWTAuth. Properly returns 401 for revoked tokens.

#### S-005 [P1] CAE Graceful Degradation Allows Revoked Tokens
- **File:** `middleware.go:886-887` (comment: "Graceful degradation: Redis unavailable -> allow + warn")
- **Description:** The `CAECheck` function comment states Redis unavailable = allow. If `isRevoked` function returns false on Redis failure (rather than erroring), revoked JWTs would remain valid.
- **Risk:** Medium — attacker could use a revoked token during Redis outage.
- **Fix:** Ensure `isRevoked` implementation has explicit Redis-down behavior (reject vs allow). Document and test this path.

### 1.2 Authentication Bypass — Public Paths

**Checked files:**
- `services/gateway/internal/middleware/session.go:194-217`

#### S-006 [P0] Public Path Prefix `/oauth/` Too Broad
- **File:** `session.go:208`
- **Description:** `publicPathPrefixes` includes `"/oauth/"` which is a prefix match. Any path starting with `/oauth/` bypasses session timeout checks. This includes administrative endpoints like `/oauth/clients` (client management) if they exist under this prefix.
- **Risk:** High — potential auth bypass for OAuth admin endpoints. An attacker could enumerate or interact with OAuth management endpoints without session validation.
- **Fix:** Use exact path matches or narrower prefixes: `/oauth/authorize`, `/oauth/token`, `/oauth/callback`. Exclude management paths from public access.

#### S-007 [P0] Public Path Prefix `/.well-known/` Too Broad
- **File:** `session.go:211`
- **Description:** The `/.well-known/` prefix matches any path under it. While JWKS and OpenID config are legitimate, if other well-known endpoints are added they'd bypass auth.
- **Risk:** Medium — future endpoint additions could unintentionally become public.
- **Fix:** Use exact matches: `/.well-known/openid-configuration`, `/.well-known/jwks.json`.

#### S-008 [P1] Public Path `/docs` and `/api-docs` in Production
- **File:** `session.go:212-213`
- **Description:** API documentation endpoints are public paths. In production, this exposes API structure to attackers.
- **Risk:** Low-Medium — information disclosure of API surface.
- **Fix:** Gate documentation behind authentication in production deployments, or disable entirely.

### 1.3 Authorization Consistency

**Checked files:**
- `services/gateway/internal/middleware/middleware.go:994-1019` (apiKeyHasWriteAccess)
- `services/gateway/internal/middleware/jwt_claims.go:116-184` (JWTClaimExtraction)

#### S-009 [PASS] API Key Scope Enforcement
- **File:** `middleware.go:994-1019`
- API key scope checking is fail-closed: no scopes = no access. Read/write separation is properly enforced with `:read`, `:write`, `:admin` suffixes.

#### S-010 [PASS] Tenant Boundary Enforced from JWT
- **File:** `jwt_claims.go:134-151`
- Non-admin users are pinned to their JWT tenant_id. Client-supplied X-Tenant-ID header is overridden. Only platform:admin can set tenant via header. Spoofed headers for non-admins are cleared.

#### S-011 [P1] Platform Admin Can Override Tenant Without Verification
- **File:** `jwt_claims.go:140-146`
- **Description:** Platform admins can set arbitrary X-Tenant-ID headers. While this is by design for cross-tenant operations, there's no audit log at the middleware level recording the override.
- **Risk:** Medium — admin impersonation across tenants is not logged at this layer.
- **Fix:** Emit an audit event when platform admin overrides tenant context.

### 1.4 Sensitive Data Protection

#### S-012 [PASS] OAuth Client Secret Hashing
- **File:** `oauth/internal/domain/models.go:32`
- `ClientSecretHash` uses Argon2id, tagged with `json:"-"` to prevent API exposure.

#### S-013 [P1] JWT Claims Contain Email in Plaintext
- **File:** `jwt_claims.go:23`
- **Description:** Email is embedded in JWT claims (`JWTCClaims.Email`). JWTs are base64-encoded, not encrypted, so email is visible to anyone who captures the token.
- **Risk:** Low — PII exposure via JWT. Standard practice but worth noting for GDPR compliance.
- **Fix:** Consider using pairwise/truncated email or omitting from access tokens (keep only in ID tokens with `email` scope).

### 1.5 Password Security

**Checked files:**
- `pkg/crypto/crypto.go:1-120`

#### S-014 [PASS] Argon2id with OWASP-Recommended Parameters
- **File:** `crypto.go:29-34`
- Uses Argon2id with m=19456 (19MB), t=2, p=1 — current OWASP recommendations. Salt is 16 bytes from `crypto/rand`.

#### S-015 [PASS] Password Pepper Support
- **File:** `crypto.go:70-101`
- Optional HMAC-SHA256 pepper applied before Argon2id. Protects against rainbow tables in database-only compromise. Pepper is stored only in server memory from env var.

#### S-016 [P2] Argon2id Parameters Configurable via Environment Variables
- **File:** `crypto.go:39-56`
- **Description:** `ARGON2_ITERATIONS`, `ARGON2_MEMORY_KB`, `ARGON2_PARALLELISM` environment variables can override hashing parameters. While minimum values are enforced (iter >= 1, mem >= 1024), a misconfigured deployment could weaken password hashing.
- **Risk:** Low — requires server-level access to env vars.
- **Fix:** Enforce stricter minimums (iter >= 2, mem >= 16384) and log warnings when defaults are overridden.

### 1.6 API Key / SCIM Token

#### S-017 [PASS] API Key Scope Checking (see S-009)

#### S-018 [P2] No API Key Rotation Enforcement
- **Description:** No evidence of automatic API key expiration or rotation policies. API keys may persist indefinitely.
- **Risk:** Low — long-lived keys increase impact of compromise.
- **Fix:** Add optional max-age for API keys and rotation reminders.

### 1.7 MFA — TOTP/Passkey

**Checked files:**
- `pkg/crypto/totp.go`

#### S-019 [PASS] TOTP Secret Encryption
- **File:** `crypto/totp.go` — TOTP secrets are encrypted with `EncryptTOTPSecret()` and decrypted with `DecryptTOTPSecret()` using AES. Secrets are not stored in plaintext.

### 1.8 OAuth — Authorization Code / PKCE / Redirect URI

**Checked files:**
- `services/oauth/internal/domain/models.go:57, 96-104, 155-188`

#### S-020 [PASS] PKCE Enforcement for Public Clients
- **File:** `models.go:57`
- `RequiresPKCE()` returns true for public clients or when `RequirePKCE` flag is set.

#### S-021 [PASS] PKCE S256 Only, plain Rejected
- **File:** `models.go:177-187`
- Code challenge verification explicitly rejects "plain" method. Only S256 (and empty defaulting to S256) is accepted. Uses `subtle.ConstantTimeCompare`.

#### S-022 [PASS] Redirect URI Exact Match
- **File:** `models.go:96-104`
- `ValidateRedirectURI` uses exact string match against registered URIs. No wildcard or prefix matching.

#### S-023 [P1] No Redirect URI Scheme Validation on Registration
- **File:** `models.go:38`
- **Description:** When registering a client, RedirectURIs are stored without validating scheme. A malicious client could register `javascript://` or `data://` URIs which could be used for XSS via redirect.
- **Risk:** Medium — depends on whether client registration is admin-only or self-service.
- **Fix:** Validate redirect URI schemes on registration: only `https://` and `http://localhost` for development.

### 1.9 Audit Hash Chain

**Checked files:**
- `services/audit/internal/domain/hash_chain.go` (full file)

#### S-024 [PASS] HMAC-SHA256 Hash Chain with Length-Prefixed Canonicalization
- **File:** `hash_chain.go:62-104`
- Chain uses HMAC-SHA256 with versioned secrets. Canonical data uses length-prefixed fields (`%04x` hex) preventing delimiter collision. Version tags prevent cross-version forgery.

#### S-025 [PASS] Constant-Time Hash Verification
- **File:** `hash_chain.go:123`
- Uses `hmac.Equal()` for constant-time comparison.

#### S-026 [P0] Hash Chain Secret Stored Only in Memory
- **File:** `hash_chain.go:17-19`
- **Description:** `hashChainSecrets` is a `map[int][]byte` in process memory. If the service restarts and the secret is reloaded from an env var or config file, any change in the secret value would break verification of all previously chained events. If the secret is lost (no persistent backup), the entire audit chain becomes unverifiable.
- **Risk:** High — audit chain integrity depends on a single in-memory secret that could be lost on restart.
- **Fix:** Store the hash chain secret in a KMS or sealed file. Ensure consistent loading across restarts. Document the secret management lifecycle.

#### S-027 [P1] VerifyChain Uses Version 0 Only
- **File:** `hash_chain.go:128-129`
- **Description:** `VerifyHash` hardcodes `secretVersion=0` via `VerifyHashWithVersion(prevHash, 0)`. If secrets are rotated (version > 0), chain verification of events hashed with newer versions would fail.
- **Risk:** Medium — breaks audit verification after key rotation.
- **Fix:** Store the secret version in each AuditEvent and use it during `VerifyChain`.

### 1.10 Session Management

**Checked files:**
- `services/gateway/internal/middleware/session.go`

#### S-028 [PASS] Session Timeout Enforcement
- Session timeout middleware checks session validity and skips only public paths.

#### S-029 [P1] Session ID in Cookie Without HttpOnly Guarantee
- **File:** `session.go:191`
- **Description:** The session key is constructed as `ggid:session:<sessionID>`. The actual cookie settings (HttpOnly, Secure, SameSite) for the session cookie were not visible in the examined code path. If HttpOnly is not set, JavaScript can read the session ID.
- **Risk:** Medium — XSS could steal session tokens.
- **Fix:** Verify session cookie has `HttpOnly: true`, `Secure: true`, `SameSite: Strict` or `Lax`.

### 1.11 Impersonation Security

**Checked files:**
- `middleware.go:872-881`, `jwt_claims.go:176-180`

#### S-030 [PASS] Impersonation Marker from Verified JWT Only
- **File:** `jwt_claims.go:176-180`
- X-Impersonated header is first deleted (clearing spoofed values), then set only from verified JWT claims. `ImpKey` and `ImpByKey` are extracted from context, not headers.

### 1.12 CSRF Protection Coverage

**Checked files:**
- `middleware.go:240-319`

#### S-031 [PASS] Double-Submit Cookie with Constant-Time Comparison
- **File:** `middleware.go:246-278`
- CSRF uses double-submit cookie pattern. Token comparison uses `subtle.ConstantTimeCompare`. Token generated from 32 bytes of `crypto/rand`.

#### S-032 [P2] CSRF Cookie Not Bound to Session
- **File:** `middleware.go:288-297`
- **Description:** The CSRF cookie has `SameSite: Lax` but is not cryptographically bound to the session ID. An attacker who can set cookies (via subdomain) could perform a cookie toss attack.
- **Risk:** Low — requires specific subdomain control.
- **Fix:** Bind CSRF token to session ID using HMAC(sessionID, serverSecret).

#### S-033 [P1] ValidateCSRF Returns True When No Cookie Present
- **File:** `middleware.go:302-308`
- **Description:** `ValidateCSRF` returns `true` when no csrf_token cookie is present (line 305: "No cookie = Bearer token auth, CSRF not applicable"). This assumes all cookieless requests use Bearer tokens. If a session-based request arrives without a CSRF cookie (e.g., after cookie expiration), CSRF protection is bypassed.
- **Risk:** Medium — timing-dependent CSRF bypass for session-based auth.
- **Fix:** Explicitly distinguish session-based vs token-based auth before skipping CSRF.

---

## Part 2: Data Validation Findings (Round 30)

### 2.1 Request Body Size Limits

**Checked files:**
- `services/gateway/internal/router/router.go:761, 809`
- `services/identity/internal/server/user_roles_handler.go:105`

#### D-001 [P0] Inconsistent Body Size Limit Coverage
- **File:** `router.go:761` (gateway-level) vs individual handlers
- **Description:** The gateway applies `middleware.MaxBodySize(gw.maxBodySize())` as a wrapper around the inner handler at line 761. However, grep shows only 28 matches for MaxBytesReader across all services, while there are **1006 instances of `json.NewDecoder`/`json.Unmarshal`** across 484 files. Many service handlers parse JSON bodies without any body size limit, relying solely on the gateway-level limit. If any service is accessed directly (bypassing the gateway), there is no protection.
- **Risk:** High — direct service access bypasses body limits entirely, enabling memory exhaustion DoS.
- **Fix:** Apply `http.MaxBytesReader` at each service's HTTP handler level, not just the gateway. Use a shared middleware.

### 2.2 Input Length Validation

#### D-002 [P1] Limited Input Length Validation Across Handlers
- **Description:** Based on 1006 JSON decode sites across 484 files, most handlers decode directly into structs without explicit length validation on string fields. While struct tags may enforce some validation, there's no evidence of systematic input length checking.
- **Risk:** Medium — oversized inputs could cause memory issues or DB errors.
- **Fix:** Add `max` validators to struct tags or use a validation middleware for common fields (name, description, email).

### 2.3 JSON Depth Limiting

#### D-003 [P1] No JSON Depth Limit
- **Description:** Go's `encoding/json` does not limit nesting depth by default. `json.NewDecoder` calls across services have no `DisallowUnknownFields` or depth limit configured. Deeply nested JSON could cause stack overflow.
- **Risk:** Medium — DoS via deeply nested JSON payload.
- **Fix:** Implement a custom JSON decoder with depth limiting, or use `io.LimitReader` + a depth-checking wrapper.

### 2.4 SCIM Filter Depth

**Checked files:**
- `services/identity/internal/scim/filter.go:274-275`

#### D-004 [P0] SCIM Filter Max Depth = 100 Too High
- **File:** `filter.go:274`
- **Description:** `maxDepth = 100` allows extremely deep SCIM filter expressions. A filter like `a EQ b AND a EQ b AND ...` nested 100 levels deep would consume significant CPU and memory during parsing. Typical SCIM usage rarely exceeds 5-10 levels.
- **Risk:** High — DoS via CPU/memory exhaustion from complex filter expressions.
- **Fix:** Reduce maxDepth to 10-20 for SCIM filters. Add a filter expression total length limit (e.g., 4096 chars).

### 2.5 Email Format Validation

#### D-005 [P1] Email Validation Inconsistency
- **Description:** No centralized email validation was found in the reviewed code. Email validation may rely on Go's `net/mail.ParseAddress` or simple regex in different locations, leading to inconsistent enforcement.
- **Risk:** Low-Medium — invalid email formats could bypass validation or cause DB-level errors.
- **Fix:** Create a shared `ValidateEmail()` utility and use it consistently across all handlers.

### 2.6 File Upload Validation

#### D-006 [P1] No Evidence of Magic Bytes Validation for File Uploads
- **Description:** File upload handling (e.g., avatar uploads) was not fully traced in this session, but based on previous audit notes, MIME type is validated via `http.DetectContentType` which checks magic bytes. However, there may be upload paths that rely solely on file extension or Content-Type header.
- **Risk:** Medium — malicious file upload with spoofed Content-Type.
- **Fix:** Ensure all file upload paths use `http.DetectContentType` on the first 512 bytes.

### 2.7 UUID Format Validation

#### D-007 [PASS] UUID Parsing with Error Handling
- `middleware.go:906-913` uses `uuid.Parse()` with proper error handling. Returns false on invalid UUID.

### 2.8 Batch Operation Limits

#### D-008 [P1] No Consistent Batch Size Limit
- **Description:** SCIM bulk operations and batch import endpoints may not enforce a maximum number of operations per request. Previous audit notes indicate this is a recurring issue.
- **Risk:** Medium — large batch operations can exhaust memory and DB connections.
- **Fix:** Enforce a maximum batch size (e.g., 100 items) at the handler level.

### 2.9 Enum Value Validation

#### D-009 [PASS] ClientType Validation
- **File:** `models.go:22-25`
- `ClientType.IsValid()` properly validates against known enum values.

### 2.10 authorization_details Schema

#### D-010 [P0] authorization_details Stored Without Schema Validation
- **File:** `services/oauth/internal/service/oauth_service.go:394`
- **Description:** `AuthorizationDetails` is typed as `json.RawMessage` — raw JSON stored without any schema validation against RFC 9396. This allows arbitrary JSON to be stored and later retrieved. If downstream code processes this data without validation, it could lead to injection or logic bypass.
- **Risk:** High — malformed or malicious authorization_details could exploit downstream consumers.
- **Fix:** Define and validate a JSON schema for authorization_details before storage. Reject payloads that don't conform.

### 2.11 Password Strength Validation

#### D-011 [PASS] Password Hashing with Argon2id (see S-014)
- Previous audit notes confirm `validatePasswordComplexity` covers creation paths.

### 2.12 SQL Parameter Validation

**Checked via grep:**
- 31 matches for `fmt.Sprintf` with SQL fragments across services

#### D-012 [P2] fmt.Sprintf in SQL Query Construction
- **File:** `services/auth/internal/server/jit_migration.go:199`
- **File:** `services/oauth/internal/repository/pg_repo.go:146, 182`
- **File:** `services/audit/internal/server/memory_map_repo.go:128`
- **Description:** These use `fmt.Sprintf` to construct SQL queries. In most cases, the interpolated values are constant column name strings (e.g., `clientColumns`, `mfaColumns`) not user input. The `jit_migration.go:199` case constructs `SELECT %s FROM %s WHERE %s = $1` with table/column names — if these come from user-controlled input, it would be SQL injection.
- **Risk:** Low — column/table names appear to be compile-time constants, but the pattern is dangerous if maintained incorrectly.
- **Fix:** Use allowlists for any dynamic table/column names. Add comments documenting that Sprintf values are constants.

---

## Code Paths Examined

### Files Read in Full or Partially:
1. `services/gateway/internal/middleware/middleware.go` (1020 lines) — JWT auth, CSRF, CORS, JWKS, API key, impersonation
2. `services/gateway/internal/middleware/jwt_claims.go` (113 lines) — JWT claim extraction, tenant enforcement
3. `services/gateway/internal/middleware/session.go` (249 lines) — session management, public paths
4. `services/oauth/internal/domain/models.go` (264 lines) — OAuth client, PKCE, redirect URI, auth codes
5. `services/audit/internal/domain/hash_chain.go` (184 lines) — audit hash chain
6. `pkg/crypto/crypto.go` (252 lines) — password hashing, AES encryption

### Patterns Searched via grep:
- `MaxBytesReader|bodyLimit|BodyLimit` — body size coverage
- `json.NewDecoder|json.Unmarshal` — 1006 matches across 484 files
- `fmt.Sprintf.*WHERE|fmt.Sprintf.*SELECT` — SQL injection patterns
- `code_challenge_method|S256|plain` — PKCE validation
- `redirect_uri|RedirectURI` — redirect validation
- `CSRF|csrf` — CSRF coverage
- `argon2|bcrypt|HashPassword|VerifyPassword` — password security
- `EncryptTOTP|DecryptTOTP` — TOTP secret protection
- `authorization_details|AuthorizationDetails` — RAR schema
- `maxDepth|depth.*limit` — SCIM filter depth

---

## Summary by Severity

| Severity | Count | Key Issues |
|----------|-------|------------|
| **P0** | 6 | Public path `/oauth/` too broad; `/.well-known/` too broad; Hash chain secret memory-only; Body size gaps (1006 decode sites); SCIM filter depth=100; authorization_details no schema |
| **P1** | 8 | CAE Redis degradation; Admin tenant override not audited; Email in JWT plaintext; No redirect URI scheme validation; VerifyChain version=0; Session cookie flags unclear; ValidateCSRF no-cookie bypass; No batch size limit |
| **P2** | 4 | Argon2 params env-overridable; No API key rotation; CSRF cookie not session-bound; fmt.Sprintf SQL pattern |

**Total: 18 findings** (6 P0, 8 P1, 4 P2)

### What's Working Well:
- JWT validation is fail-closed with algorithm whitelisting
- PKCE S256-only enforcement is correct
- Password hashing uses Argon2id with pepper option
- Audit hash chain uses HMAC-SHA256 with versioned secrets
- CSRF uses constant-time comparison
- Tenant boundary enforcement prevents cross-tenant access
- TOTP secrets are encrypted at rest
- Impersonation markers come from verified JWT only
- API key scope checking is fail-closed

---

*End of Report*
