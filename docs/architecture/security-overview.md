# Security Architecture Overview

> How GGID secures authentication, authorization, multi-tenancy, and audit data.

---

## Authentication Flow

```
┌──────────┐    1. Login (username + password)    ┌──────────────┐
│  Client   │ ──────────────────────────────────▶ │  API Gateway  │
│           │                                     │  (:8080)      │
└──────────┘                                     └──────┬───────┘
      ▲                                                 │
      │  5. JWT + Refresh Token                         │ 2. Forward to Auth Service
      │                                                 ▼
┌──────────┐                                     ┌──────────────┐
│  Gateway  │ ◀──── 4. Verified JWT ─────────── │ Auth Service  │
│  (JWT v)  │                                     │  (Argon2id)   │
└──────────┘                                     └──────┬───────┘
      │                                                 │
      │  3. Verify password hash                        │ 6. Check LDAP/OAuth/SAML
      │                                                 ▼
      │                                           ┌──────────────┐
      │                                           │ AuthProvider  │
      │                                           │ Chain         │
      │                                           └──────────────┘
```

### Token Lifecycle

| Token | Lifetime | Storage | Purpose |
|-------|----------|---------|---------|
| Access Token (JWT) | 15 min | Client memory / cookie | API authentication |
| Refresh Token | 7 days | HttpOnly cookie / Redis | Obtain new access tokens |
| ID Token (OIDC) | 15 min | Client | User identity claims |
| API Key | No expiry | Server-side config | Service-to-service auth |

### JWT Verification Chain

1. Extract Bearer token from `Authorization` header
2. Verify RS256 signature against JWKS (cached 15 min)
3. Verify `iss`, `aud`, `exp`, `nbf` claims
4. Verify `jti` not replayed (Redis SETNX)
5. Extract `sub`, `tenant_id`, `roles`, `scope` claims
6. Forward to backend with `X-Tenant-ID` header

---

## P0 Security Measures (All Implemented)

### CSRF & State Validation
- OAuth state parameter uses **crypto/rand** (not math/rand)
- State is **one-time use** (Redis SETNX + TTL)
- State is **cross-client isolated** (state bound to client_id)

### Token Security
- JWT signed with **RS256** (asymmetric, not HMAC)
- **JWKS endpoint** for key rotation
- `jti` claim tracked with Redis SETNX (anti-replay)
- `JWT_SECRET` empty → `log.Fatal` (no silent bypass)
- `iss` parameter enforced in auth redirect

### Authorization Enforcement
- **`hasAdminScope()`** guards all `/api/v1/admin/*` routes
- **`HasScope()`** does actual scope matching (not always-true)
- **Tenant claim priority**: JWT `tenant_id` overrides `X-Tenant-ID` header
- Rate limiter wired into production `Handler()` chain
- Security headers (HSTS, X-Content-Type-Options, X-Frame-Options, CSP)

### Secrets Management
- Password hashing: **Argon2id** (memory-hard)
- Password **pepper** support (HMAC before hash)
- API keys stored as SHA-256 hash (never plaintext)
- Database passwords in secrets manager (not env files)

---

## Multi-Tenant Isolation (Defense in Depth)

### Row-Level Security (PostgreSQL)

Every tenant-scoped table has RLS policies:

```sql
-- Enable RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

-- Policy: users can only see their tenant's data
CREATE POLICY tenant_isolation ON users
  USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

### Tenant Resolution (Priority Order)

1. **JWT claim** `tenant_id` (highest priority — prevents spoofing)
2. **API key** tenant mapping
3. **`X-Tenant-ID` header** (lowest — can be spoofed)

```go
// Gateway middleware resolves tenant
func resolveTenant(r *http.Request) string {
    // 1. JWT claim (verified by signature)
    if claims := getJWTClaims(r); claims != nil {
        return claims.TenantID
    }
    // 2. API key
    if apiKey := getAPIKey(r); apiKey != "" {
        return resolveTenantFromAPIKey(apiKey)
    }
    // 3. Header (untrusted)
    return r.Header.Get("X-Tenant-ID")
}
```

---

## Audit Hash Chain (Tamper Detection)

Every audit event is linked to the previous one via a cryptographic hash chain:

```
Event₁ → hash₁ = SHA256(Event₁.data)
Event₂ → hash₂ = SHA256(hash₁ ‖ Event₂.data)
Event₃ → hash₃ = SHA256(hash₂ ‖ Event₃.data)
```

### Verification

```sql
SELECT verify_hash_chain() FROM audit_events WHERE tenant_id = $1;
-- Returns: { verified: true, tampered_events: [], total_events: 15423 }
```

### Tamper Detection

- **Field modification**: Hash mismatch on any event
- **Deletion**: Chain breaks at gap point
- **Insertion**: Hash mismatch on event after insertion point
- **Replay**: Duplicate `event_id` detected

---

## Security Headers

| Header | Value | Purpose |
|--------|-------|---------|
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` | Force HTTPS |
| `X-Content-Type-Options` | `nosniff` | Prevent MIME sniffing |
| `X-Frame-Options` | `DENY` | Prevent clickjacking |
| `Content-Security-Policy` | `default-src 'self'` | Prevent XSS |
| `X-XSS-Protection` | `1; mode=block` | Legacy XSS protection |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Limit referrer leakage |

---

## Rate Limiting

| Scope | Limit | Algorithm |
|-------|-------|-----------|
| Per-IP (global) | 100 req/s | Token bucket |
| Login attempts | 5 per 60s | Sliding window |
| API key | 1000 req/min | Token bucket |
| Password reset | 3 per hour | Sliding window |

Rate limiter runs **before** auth middleware in the handler chain.

---

## STRIDE Threat Model Summary

| Threat | Mitigation | Status |
|--------|-----------|--------|
| **Spoofing** | JWT RS256 + JWKS + anti-replay | Implemented |
| **Tampering** | RLS + audit hash chain | Implemented |
| **Repudiation** | Audit log (NATS JetStream + hash chain) | Implemented |
| **Information Disclosure** | PII obfuscation + tenant isolation | Implemented |
| **Denial of Service** | Rate limiting + circuit breaker | Implemented |
| **Elevation of Privilege** | RBAC + ABAC + scope enforcement | Implemented |

---

*See also: [Architecture Overview](overview.md) | [Security Hardening Guide](../guides/security-hardening.md) | [Production Checklist](../deploy/production-checklist.md) | [Gap Closure Report](../research/gap-closure-report.md)*

*Last updated: 2025-07-11*
