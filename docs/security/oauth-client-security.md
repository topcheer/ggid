# OAuth Client Security Audit

**Date**: 2025-01-XX  
**Scope**: OAuth2/OIDC client credential handling, token endpoint protection, information leakage  
**Auditor**: Backend Agent  
**Status**: Completed  

---

## Executive Summary

| Area | Finding | Severity | Status |
|------|---------|----------|--------|
| Client secret storage | Argon2id hash — no plaintext stored | PASS | Production-ready |
| Secret returned on creation | Plaintext shown once, never again | PASS | Best practice (RFC 7591) |
| Token endpoint rate limiting | Now enforced at gateway (10 req/min) | FIXED | Was missing, now patched |
| Error message info leakage | Generic message replaces internal detail | FIXED | Was leaking header name |
| Client onboarding handler | Plaintext secret not persisted/hashed | LOW | Demo endpoint — see note |

---

## 1. Client Secret Storage

### Finding: PASS

Client secrets are stored exclusively as **Argon2id hashes**. The implementation follows OWASP recommendations.

**Evidence:**

- `domain/models.go:31`: `ClientSecretHash string // Argon2id hash; empty for public clients`
- `oauth_service.go:128`: `hash, err := pkgcrypto.HashPassword(plaintextSecret)` — Argon2id hashing on creation
- `oauth_service.go:389`: `ok, _ := pkgcrypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)` — constant-time verification
- `par.go:161`: `verifyClientSecret()` wrapper delegates to `crypto.VerifyPassword`
- Database column: `client_secret_hash` in `oauth_clients` table (never stores plaintext)

**Secret generation:**

```go
func generateClientSecret() string {
    secret, _ := pkgcrypto.GenerateRandomToken(32)
    return "gcs_" + secret  // 32 bytes of crypto/rand, base64-encoded
}
```

- 256 bits of entropy from `crypto/rand`
- Prefixed with `gcs_` for identification
- No known weakness in generation

**Secret rotation** (`RotateClientSecret`):
- Requires old secret verification before rotation
- Old hash immediately replaced
- New plaintext returned once, never retrievable again

### Recommendation

No action needed. The Argon2id implementation is industry-standard.

---

## 2. Token Endpoint Rate Limiting

### Finding: FIXED (was missing)

**Before**: The `/oauth/token` endpoint had no specific rate limiting. The gateway's `RateLimiter.getLimit()` only handled `/api/v1/auth/login`, `/api/v1/auth/register`, and generic `/api/v1/` paths. The `/oauth/token` path fell through to `return 0` (no limit), making it vulnerable to:
- Client secret brute-force attacks
- Token enumeration
- DoS via expensive grant_type processing

**After**: Added `TokenLimit` field to `RateLimitConfig` with a default of **10 requests per minute** per IP, matching the gateway-level rate limiter pattern.

**Changes:**

```go
// ratelimit.go
type RateLimitConfig struct {
    LoginLimit    int  // 5/min
    RegisterLimit int  // 3/min
    TokenLimit    int  // 10/min — NEW
    APILimit      int  // 100/min
    Window        time.Duration
}

func (rl *RateLimiter) getLimit(path string) int {
    switch {
    case path == "/oauth/token":
        return rl.cfg.TokenLimit  // NEW
    // ... existing cases
    }
}
```

**Defense in depth**: The gateway's `TenantBucketLimiter` (token bucket per tenant+IP) and `MultiDimRateLimiter` (5-dimensional: tenant, user, API key, IP, endpoint) provide additional layers.

### Recommendation

For production with multiple gateway replicas, replace the in-memory rate limiter with a Redis-backed implementation for distributed enforcement. The existing `MultiDimRateLimiter` architecture supports this swap.

---

## 3. Error Message Information Leakage

### Finding: FIXED

**Before**: The `/oauth/token` endpoint returned:
```json
{
  "error": "invalid_request",
  "error_description": "valid X-Tenant-ID header required"
}
```

This leaked:
- The exact name of an internal HTTP header (`X-Tenant-ID`)
- The multi-tenant architecture of the system
- Useful reconnaissance information for attackers

**After**:
```json
{
  "error": "invalid_request",
  "error_description": "missing or invalid tenant context"
}
```

The new message follows RFC 6749 error response format without revealing implementation details. Legitimate integrators still understand the problem (missing tenant context) without learning internal header names.

### Recommendation

Audit all error responses across services for similar leakage. A grep pattern:
```bash
grep -rn "X-Tenant-ID\|header required\|internal" services/*/internal/server/ --include="*.go"
```

---

## 4. Client Secret Exposure in API Responses

### Finding: PASS (with minor note)

**Secret visibility by endpoint:**

| Endpoint | Secret in Response? | Correct? |
|----------|---------------------|----------|
| `POST /oauth/register` (RFC 7591 DCR) | Yes — plaintext, one-time | YES (RFC 7591 §3.2.1) |
| `POST /api/v1/oauth/clients` (Create) | Yes — plaintext, one-time | YES |
| `GET /api/v1/oauth/clients` (List) | No — only hash in DB | YES |
| `GET /api/v1/oauth/clients/{id}` | No | YES |
| `POST /clients/{id}/rotate-secret` | Yes — new plaintext, one-time | YES |

**DCR response struct** (`oauth_service.go:1410`):
```go
type DynamicRegistrationResponse struct {
    ClientSecret string `json:"client_secret,omitempty"` // omitempty = absent for public clients
    // ...
}
```

The `omitempty` tag correctly omits the secret for public clients and on subsequent reads.

**Client onboarding handler** (`client_onboarding_handler.go`):
- This handler generates a plaintext secret using `crypto/rand` (32 hex chars = 128 bits)
- It returns the plaintext in the response but does NOT persist the hash to the database
- **Classification**: Demo/mock endpoint for UI testing — not wired to the OAuth service layer
- **Risk**: LOW — no actual OAuth clients are created through this path

### Recommendation

1. The onboarding handler should be either:
   - Removed from production routes (it's a demo stub)
   - Or wired to `OAuthService.CreateClient()` for proper hash storage
2. Add a `json:"-"` tag to `ClientSecretHash` in any response-facing structs to ensure the hash is never accidentally serialized

---

## 5. Summary of Changes

### Files Modified

| File | Change |
|------|--------|
| `services/oauth/internal/server/server.go:571` | Error message: `"valid X-Tenant-ID header required"` → `"missing or invalid tenant context"` |
| `services/gateway/internal/middleware/ratelimit.go` | Added `TokenLimit` field (default: 10/min) + `/oauth/token` case in `getLimit()` |
| `services/gateway/internal/middleware/coverage_sprint14_test.go` | Added `TestRateLimiter_TokenEndpoint` and `TestDefaultRateLimitConfig_TokenLimit` |
| `docs/security/oauth-client-security.md` | This document |

### Verification

```bash
# Build
go build ./...

# Test
go test ./services/oauth/internal/server/... -run TestExchange
go test ./services/gateway/internal/middleware/... -run TestRateLimiter_Token

# Lint
golangci-lint run ./services/oauth/... ./services/gateway/...
```

---

## Appendix: OAuth2 Security Best Practices Compliance

| Practice | RFC/Standard | GGID Status |
|----------|-------------|-------------|
| Client secret hashing | OWASP ASVS 2.4.4 | Argon2id |
| PKCE enforcement for public clients | RFC 7636 | Enforced |
| DPoP token binding | RFC 9449 | Implemented |
| PAR (Pushed Authorization Requests) | RFC 9126 | Implemented |
| RFC 7523 JWT client auth | RFC 7523 | Implemented |
| Token revocation | RFC 7009 | Implemented |
| Introspection auth required | RFC 7662 §2.1 | Enforced |
| Client secret rotation | RFC 7592 | Implemented |
| FAPI 2.0 security profile | FAPI 2.0 | Enforced |
| Rate limiting on token endpoint | OAuth2 Security BCP §4.4 | Implemented (this audit) |
