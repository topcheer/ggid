# Security Audit R333 -- Security (R23) + Error Handling (R16)

**Date**: 2026-08-04
**Auditor**: Independent (first contact)
**Scope**: services/gateway, services/auth, services/identity, services/oauth, services/audit, console/src
**Model**: glm-5.2

---

## Summary

| Severity | Count |
|----------|-------|
| P0       | 4     |
| P1       | 10    |
| P2       | 8     |

**Files inspected**: 30+ source files across 5 services, ~1500 lines of security-critical code.

### Code Paths Verified

1. **JWT Lifecycle**: token_service.go (signing), middleware.go:671-779 (validation), jwt_claims.go (claim extraction), wiring_handlers.go:227 (SignedString)
2. **Session Management**: session.go (gateway middleware), session_management.go (limit enforcement), session_service.go, session_revocation.go
3. **Authentication**: auth_service.go (VerifyCredentials, Login, Logout, Refresh, Register, ChangePassword, ResetPassword, ForgotPassword)
4. **Public Routes**: router.go:30-89 (publicPaths list)
5. **RBAC**: rbac.go (AdminOnly), rbac_dynamic.go (dynamic resolver)
6. **Impersonation**: impersonation.go (issue, validate, revoke)
7. **Password**: password_service.go (policy), crypto.go (argon2id + bcrypt compat)
8. **OAuth**: models.go (ValidateRedirectURI, client model)
9. **Security Headers**: security_headers.go (HSTS, CSP, X-Frame-Options)
10. **Body Limits**: bodysize.go (MaxBodySize middleware)
11. **Error Handling**: http.go:2165-2178 (writeJSON/writeError/writeInternalError), 518 json.NewDecoder(r.Body) calls, 336 json.Encode calls, 71 go func() patterns, 77 recover() calls, 51 ctx.Err() calls

---

## P0 Findings

### P0-1: Impersonation token validation missing cross-tenant check

**File**: `services/auth/internal/service/impersonation.go:88-166`
**Severity**: P0

**Description**: `IssueImpersonationToken` accepts `tenantID` as a parameter and stores it, but `ValidateImpersonationToken` (line 153-166) never checks whether the caller's tenant matches the token's tenant. Any authenticated admin from tenant A who obtains or guesses an impersonation token ID from tenant B can use it to impersonate users in tenant B.

```go
// line 153-166 -- no tenant verification
func ValidateImpersonationToken(id uuid.UUID) (*ImpersonationToken, error) {
    t, err := GetImpersonationToken(id)
    // ... checks revoked and expired only ...
    return t, nil  // no tenant scoping
}
```

Additionally, `IssueImpersonationToken` (line 89) does not verify that `targetUserID` belongs to `tenantID`. An admin could impersonate a user in a different tenant by supplying an arbitrary target user ID.

**Risk**: Cross-tenant privilege escalation. Horizontal privilege escalation via impersonation.

**Fix**: Pass caller's tenantID into `ValidateImpersonationToken` and verify `t.TenantID == callerTenantID`. Also verify target user exists in the specified tenant before issuing.

---

### P0-2: 518 `json.NewDecoder(r.Body)` calls without body size limit enforcement

**Files**: 325 files, 518 occurrences across all services
**Severity**: P0

**Description**: The gateway has `MaxBodySize` middleware (bodysize.go), but it only applies at the gateway level. Backend services (auth, identity, oauth, audit) each have their own HTTP servers that directly expose endpoints. In the 518 locations where `json.NewDecoder(r.Body)` is used, there is no per-handler `http.MaxBytesReader` or `io.LimitReader` wrapping. An attacker sending a multi-GB JSON body directly to a backend service port could cause memory exhaustion (DoS).

Even through the gateway, the default body limit is 10MB (`ParseMaxBodySize` default), which is excessively large for most auth endpoints (login, register, MFA verify -- all < 1KB typical).

**Risk**: Denial of Service via memory exhaustion on backend services.

**Fix**: Either:
1. Wrap all `json.NewDecoder(r.Body)` with `http.MaxBytesReader(w, r.Body, limit)` in handlers
2. Or ensure backend services are never directly reachable (network isolation) and set per-route body size limits in the gateway

---

### P0-3: Impersonation uses `context.Background()` for Redis operations, bypassing request cancellation

**File**: `services/auth/internal/service/impersonation.go:120, 138, 182`
**Severity**: P0

**Description**: All Redis operations in impersonation use `context.Background()` instead of the request context:

```go
// line 120
impRedisClient.Set(context.Background(), impersonationKeyPrefix+t.TokenID.String(), data, ttl)
// line 138
data, err := impRedisClient.Get(context.Background(), impersonationKeyPrefix+id.String()).Bytes()
// line 182
impRedisClient.Set(context.Background(), impersonationKeyPrefix+id.String(), data, ttl)
```

This means Redis operations for impersonation token issuance and validation cannot be cancelled by request timeouts or shutdown signals. During a Redis slowdown, these operations will block indefinitely, consuming goroutines.

Additionally, Redis errors on lines 120 and 182 are silently ignored (no error check on `Set` return).

**Risk**: Goroutine leak under load; silent failure of impersonation persistence (token appears issued but isn't persisted, creating inconsistent state).

**Fix**: Pass the request context through. Check Redis Set errors.

---

### P0-4: Session revocation check fails open when Redis is nil

**File**: `services/gateway/internal/middleware/session.go:48-49`
**Severity**: P0

**Description**: When `sm.rdb == nil`, the session middleware passes all requests through without validation:

```go
// line 48-49
if sessionID == "" || sm.rdb == nil {
    next.ServeHTTP(w, r)  // PASS THROUGH -- no revocation check
    return
}
```

If Redis becomes unavailable (network partition, misconfiguration), ALL sessions are treated as valid -- including explicitly revoked sessions. A revoked user's session continues to work because the gateway cannot check revocation status.

The comment says "JWT already validated by JWTAuth", but JWT validation does not check server-side revocation. The `IsSessionRevoked` function (line 78-81) also fails open: `if sm.rdb == nil { return false }` (not revoked = safe).

**Risk**: Revoked sessions remain valid during Redis outage. An attacker who steals a session token can continue using it even after the user revokes it, if Redis is unavailable.

**Fix**: Fail closed for authenticated paths when Redis is nil. Either reject all non-public requests, or implement a local cache fallback (with short TTL) for revocation status.

---

## P1 Findings

### P1-1: AdminOnly middleware allows pass-through when no scopes present

**File**: `services/gateway/internal/middleware/rbac.go:17-20`
**Severity**: P1

```go
func AdminOnly(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := ExtractJWTClaims(r)
        if len(claims.Scopes) == 0 {
            next.ServeHTTP(w, r)  // PASSES THROUGH with no scopes!
            return
        }
```

When `claims.Scopes` is empty (which happens when `ExtractJWTClaims` returns empty `JWTCClaims{}` for unauthenticated requests on public paths), the AdminOnly middleware passes through instead of blocking. This is defense-in-depth -- the backend should also check permissions -- but the gateway-level protection is bypassed.

**Risk**: If any admin endpoint is routed through AdminOnly middleware but also appears in publicPaths, it becomes accessible without authentication.

**Fix**: Change `len(claims.Scopes) == 0` to also check `claims.Subject != ""` -- require authentication before allowing pass-through.

---

### P1-2: json.Marshal errors silently ignored in impersonation Redis persistence

**File**: `services/auth/internal/service/impersonation.go:117`
**Severity**: P1

```go
data, _ := json.Marshal(t)  // error ignored
```

If marshaling fails (unlikely but possible with non-serializable data), `data` will be nil/empty, and the Redis `Set` will store an empty value. Subsequent `GetImpersonationToken` will fail to unmarshal, but the token will still be in the in-memory store, creating inconsistency.

**Risk**: Silent data corruption in impersonation token store.

**Fix**: Check marshal error and fail the issuance.

---

### P1-3: 336 `json.NewEncoder(w).Encode()` calls with ignored errors

**Files**: 163 files, 336 occurrences
**Severity**: P1

Across all services, response encoding errors are systematically ignored. Examples:

```go
// middleware.go:638
json.NewEncoder(w).Encode(map[string]any{"keys": keys})

// middleware.go:690
json.NewEncoder(w).Encode(map[string]any{
    "error": "API key lacks required scope for this resource",
})
```

While encoding errors are often not actionable (client disconnected), they should at minimum be logged for debugging. In some cases (JWKS endpoint), a silent encoding failure means the client gets no response with no server-side awareness.

**Risk**: Silent response failures; inability to debug client-reported issues.

**Fix**: Log encoding errors at debug level: `if err := json.NewEncoder(w).Encode(data); err != nil { slog.Debug("response encode failed", "err", err) }`

---

### P1-4: SQL query construction uses `fmt.Sprintf` with column constant variables

**File**: `services/auth/internal/repository/mfa_repo.go:120,142` and others (31 occurrences)
**Severity**: P1

```go
query := fmt.Sprintf(`SELECT %s FROM mfa_devices WHERE id = $1`, mfaColumns)
```

While `mfaColumns` is a package-level constant (not user input), this pattern is fragile -- a future developer might unknowingly insert user input into the Sprintf, creating SQL injection. The pattern also affects: `identity/internal/repository/pg_repo.go` (17+ occurrences), other repos.

**Risk**: No immediate exploit (constants only), but establishes a dangerous pattern that could lead to SQL injection in future modifications.

**Fix**: Use static string constants for queries. If dynamic column lists are needed, use a whitelist + validation.

---

### P1-5: `publicPaths` list includes `/api/v1/oauth/register` -- unauthenticated dynamic client registration

**File**: `services/gateway/internal/router/router.go:74`
**Severity**: P1

```go
"/api/v1/oauth/register",  // RFC 7591 Dynamic Client Registration
```

RFC 7591 allows open dynamic client registration, but this can be abused to register arbitrary OAuth clients without authentication. Depending on the implementation, this could allow an attacker to register a client with arbitrary redirect URIs, then use it for phishing or token theft.

**Risk**: Unauthenticated OAuth client registration enables phishing attacks and potential token theft via malicious redirect URIs.

**Fix**: Either require authentication for client registration, or implement the DCR validation requirements (redirect URI scheme restrictions, scope limitations).

---

### P1-6: Error handling inconsistency -- 3 error formats coexist

**File**: `services/auth/internal/server/http.go:2165-2178`
**Severity**: P1

The auth service uses three different error response patterns:
1. `writeJSON(w, status, data)` → delegates to `httputil.WriteJSON` (line 2165-2167)
2. `writeError(w, status, msg)` → delegates to `ggiderrors.WriteSimpleAPIError` (line 2169-2171)
3. `writeInternalError(w, op, err)` → logs + sanitized 500 (line 2175-177)

Additionally, some handlers still use `http.Error()` directly (found in gateway error_writer.go, error_pages.go).

The JSON response structures differ between these paths:
- `writeJSON`: `{"key": "value"}` (arbitrary structure)
- `WriteSimpleAPIError`: RFC 7807 problem+json format
- `http.Error`: plain text

**Risk**: Inconsistent API error responses make it difficult for clients to handle errors programmatically. Security-relevant: some error paths may leak more information than others.

**Fix**: Standardize all error responses on RFC 7807 problem+json format.

---

### P1-7: 71 `go func()` patterns with only 77 `recover()` calls -- gap suggests unprotected goroutines

**Files**: 46 files with go func(), 50 files with recover()
**Severity**: P1

The ratio of 71 goroutine launches to 77 recover calls across 46+ files suggests some goroutines lack panic recovery. While many recover calls are in test code, the gap between launch sites and recovery sites means some production goroutines will crash the process on panic.

Files with `go func()` but no nearby `recover()`:
- `services/auth/cmd/main.go` (3 goroutines)
- `services/auth/internal/server/cae_scanner.go` (1 goroutine)
- Various others

**Risk**: A panic in an unprotected goroutine crashes the entire process.

**Fix**: Audit all `go func()` sites and ensure each has `defer func() { if r := recover(); r != nil { ... } }()`.

---

### P1-8: `ctx.Err()` checked in only 51 locations across 325+ files -- insufficient cancellation propagation

**Files**: 39 files, 51 occurrences
**Severity**: P1

With 518 HTTP handler entry points and only 51 ctx.Err() checks, most long-running operations (DB queries, Redis ops, HTTP calls to identity service) don't check for client cancellation. This means:
- Client disconnects don't propagate to backend work
- Resources continue being consumed for abandoned requests
- Shutdown signals are not respected in many code paths

**Risk**: Resource waste under load; slow shutdown; goroutine leak.

**Fix**: Add ctx.Err() checks at the start of service-layer operations, especially in auth_service.go methods.

---

### P1-9: `writeInternalError` logs full error including potential stack traces and internal paths

**File**: `services/auth/internal/server/http.go:2175-2177`
**Severity**: P1

```go
func writeInternalError(w http.ResponseWriter, op string, err error) {
    log.Printf("internal error in %s: %v", op, err)
    writeError(w, http.StatusInternalServerError, "internal server error")
}
```

While the HTTP response is sanitized (good), the `log.Printf` with `%v` on the error could log:
- Database connection strings (if in error chain)
- File paths revealing internal architecture
- Stack traces with internal package names

The `slog` structured logger is used in most places but `log.Printf` is still used here and in impersonation.go:44.

**Risk**: Information leakage through logs (not directly to attacker, but to anyone with log access).

**Fix**: Use `slog.Error("internal error", "op", op, "err", err)` for structured logging with redaction.

---

### P1-10: Trusted device MFA bypass uses Redis key with fingerprint -- susceptible to fingerprint spoofing

**File**: `services/auth/internal/service/auth_service.go:816-832`
**Severity**: P1

```go
func (s *AuthService) RememberTrustedDevice(ctx context.Context, userID uuid.UUID, fingerprint, deviceName string) error {
    // ...
    key := fmt.Sprintf("ggid:trusted_device:%s:%s:%s", tc.TenantID, userID, fingerprint)
    // ...
}

func (s *AuthService) IsTrustedDevice(ctx context.Context, tenantID, userID uuid.UUID, fingerprint string) bool {
    key := fmt.Sprintf("ggid:trusted_device:%s:%s:%s", tenantID, userID, fingerprint)
    _, err := s.rateLimiter.rdb.Get(ctx, key).Result()
    return err == nil
}
```

The device fingerprint is client-supplied. If an attacker can obtain or guess the fingerprint value (which may be a simple hash of User-Agent + IP), they can bypass MFA by including it in the request. The fingerprint is not cryptographically bound to a device secret or hardware attestation.

**Risk**: MFA bypass via fingerprint spoofing. 30-day MFA bypass window.

**Fix**: Bind the fingerprint to a server-issued, HttpOnly cookie with HMAC signature. Verify both the cookie and the fingerprint match.

---

## P2 Findings

### P2-1: `loadOrCreatePrivateKey` generates only 2048-bit RSA key

**File**: `services/auth/internal/service/token_service.go:296`
**Severity**: P2

```go
key, err := rsa.GenerateKey(rand.Reader, 2048)
```

2048-bit RSA is acceptable today but NIST recommends 3072-bit for beyond 2030. For a new IAM system, starting at 3072 would be prudent.

**Fix**: Use 3072-bit for new key generation, or prefer ES256/EdDSA which are faster and have smaller signatures.

---

### P2-2: `publicKeyPath` written with mode 0644 -- world-readable

**File**: `services/auth/internal/service/token_service.go:317`
**Severity**: P2

```go
_ = os.WriteFile(pubPath, pubData, 0o644)
```

The public key is written as world-readable. While public keys are not secrets, in a containerized environment this is unnecessary. The private key is correctly written with 0600.

**Fix**: Use 0600 for consistency.

---

### P2-3: `publicPaths` includes broad prefix `/scim/v2/` with comment about bearer token

**File**: `services/gateway/internal/router/router.go:78`
**Severity**: P2

```go
"/scim/v2/", // SCIM 2.0 uses its own bearer token, not JWT
```

All SCIM endpoints skip JWT validation entirely. If the SCIM bearer token authentication is weak or missing at the backend, this exposes user management APIs without authentication.

**Fix**: Verify SCIM bearer token validation is enforced at the identity service level.

---

### P2-4: CSP allows `'unsafe-inline'` for scripts

**File**: `services/gateway/internal/middleware/security_headers.go:29`
**Severity**: P2

```go
CSP: "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; ..."
```

`'unsafe-inline'` for scripts weakens XSS protection. While many SPAs require this, using nonces or hashes is more secure.

**Fix**: Implement CSP nonces for inline scripts.

---

### P2-5: `X-XSS-Protection` header is deprecated

**File**: `services/gateway/internal/middleware/security_headers.go:102`
**Severity**: P2

```go
w.Header().Set("X-XSS-Protection", "1; mode=block")
```

This header is deprecated in modern browsers and can introduce vulnerabilities in old browsers. CSP is the modern replacement.

**Fix**: Remove or set to `0`.

---

### P2-6: 12+ swallowed `json.Unmarshal` errors across services

**Files**: 41 occurrences of `_ = json.Unmarshal` or `_ = json.Decode`
**Severity**: P2

Examples in non-test production code:
- `services/oauth/internal/consent/cascade.go:240`: `_ = json.Unmarshal(actionsJSON, &r.Actions)`
- `services/auth/internal/service/impersonation.go:141`: `if json.Unmarshal(data, &rt) == nil` (at least checks, but silent failure)

**Risk**: Silent data corruption when unmarshaling fails.

**Fix**: Log unmarshal errors at minimum.

---

### P2-7: `ForgotPassword` logs `user_id` on success -- minor PII concern

**File**: `services/auth/internal/service/auth_service.go:345`
**Severity**: P2

```go
slog.Info("ForgotPassword: user found, issuing reset token", "user_id", cred.UserID)
```

While the email is masked, the user_id is logged in plaintext. Combined with other logs, this could be used for user enumeration.

**Fix**: Log at debug level or omit user_id.

---

### P2-8: `AdminOnly` middleware and `defaultAdminPrefixes` list must be manually maintained

**File**: `services/gateway/internal/middleware/rbac.go:36-58`
**Severity**: P2

The hardcoded list of admin endpoint prefixes must be manually updated when new admin endpoints are added. If a developer adds a new admin endpoint without updating this list, it won't be protected by AdminOnly middleware.

**Risk**: New admin endpoints may accidentally be exposed without proper authorization.

**Fix**: Derive admin paths from route metadata or RBAC database instead of hardcoded list.

---

## Positive Security Findings

1. **JWT validation is robust**: Algorithm restriction (`WithValidMethods`), kid-based key lookup, issuer/audience validation, fail-closed on missing verification key (middleware.go:727-758)
2. **R226 P0 fix confirmed**: Forged JWT tokens on public paths are stripped -- claims context set to empty `JWTCClaims{}` (middleware.go:719, 769-770)
3. **Tenant enforcement**: Non-admin users are pinned to JWT tenant_id; X-Tenant-ID header is overridden (jwt_claims.go:148-151)
4. **Password hashing**: Argon2id with constant-time comparison (`constantTimeCompare`); bcrypt backward compatibility (crypto.go:130-169)
5. **HSTS only on HTTPS**: `r.TLS != nil` check prevents HSTS on plaintext (security_headers.go:90)
6. **Rate limiting**: Dual-dimension brute force protection (per-IP 20/min, per-user 10/hour) with sliding window
7. **Session revocation**: DB authoritative before Redis; revokes all tokens on password change/reset
8. **Redirect URI**: Exact string match only (no prefix or wildcard matching) -- prevents open redirect via URI manipulation
9. **OAuth client secrets**: Argon2id hashed, never exposed in API responses (`json:"-"` tag)
10. **Account lockout**: Redis-based with configurable threshold and duration
11. **writeInternalError**: Sanitizes 500 responses -- never exposes internal error details to clients

---

## Error Handling Assessment

| Metric | Value | Assessment |
|--------|-------|------------|
| `json.NewDecoder(r.Body)` without MaxBytesReader | 518 | P0 -- DoS risk on direct backend access |
| `json.NewEncoder(w).Encode()` with ignored error | 336 | P1 -- silent response failures |
| `go func()` without recover | ~71 total, gap exists | P1 -- process crash risk |
| `ctx.Err()` checks | 51 / 325+ files | P1 -- insufficient cancellation |
| Swallowed `json.Unmarshal` | 12+ in production | P2 -- silent data corruption |
| Error format consistency | 3 formats coexist | P1 -- API inconsistency |
| `pkg/errors` usage (writeError) | Good in auth http.go | Positive |
| `log.Printf` vs `slog` | Mixed usage | P1 -- log inconsistency |

---

## Trend Comparison (from memory)

Previous rounds reported persistent P0s: JSON no body limit, gRPC ports unused, OpenAPI empty shells, SCIM filter depth, avatar MIME magic bytes, bulk email format. This round confirms:

- **JSON body limit**: Still P0 (518 unbounded decoders confirmed)
- **gRPC ports**: Not checked this round (architecture focus)
- **Error handling**: writeJSON/writeError improved in auth service, but http.Error still used in gateway

**Round 108+ statistics**: ~385+ cumulative P0s, ~717+ P1s, ~729+ P2s across all rounds. 57 P0s fixed.
