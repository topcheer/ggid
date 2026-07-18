# GGID Security Hardening Guide

**Version**: v1.0-stable  
**Scope**: Production security hardening checklist and implementation guide  

---

## 1. Authentication & Authorization

### 1.1 Public Path Whitelist

All non-public paths require a valid JWT. The public path whitelist is defined in `services/gateway/internal/middleware/session.go`:

```go
var publicPathPrefixes = []string{
    "/api/v1/auth/login",
    "/api/v1/auth/register",
    "/api/v1/auth/refresh",
    "/api/v1/auth/password/forgot",
    "/api/v1/auth/password/reset",
    "/api/v1/auth/social/",
    "/oauth/",              // OAuth2 flows (token, authorize, etc.)
    "/api/v1/oauth/register",
    "/saml/",               // SAML federation
    "/.well-known/",        // OIDC discovery, JWKS
    "/docs",                // API documentation
    "/api-docs",
    "/login",
    "/register",
    "/forgot-password",
}
```

**Audit Result**: PASS. No POST/PUT/DELETE endpoint outside the whitelist is accessible without JWT authentication.

### 1.2 Admin Authorization

The `roles/assign` endpoint has an explicit admin guard (commit `d9b75ba9`). All other admin endpoints go through the gateway's JWT validation + policy service RBAC check.

**For production**: Verify that the Policy service is connected and enforcing role-based access control. The gateway proxies to Policy for authorization decisions.

### 1.3 Recommendations

- Add a periodic automated test that attempts POST/PUT/DELETE without a token on all non-public paths
- Consider adding a "deny by default" middleware for new routes (any new route must explicitly declare its auth requirement)

---

## 2. CORS Configuration

### 2.1 Current Implementation

GGID implements multi-layer CORS:

1. **Global CORS** (`middleware/cors.go`): Configurable via `CORS_ALLOWED_ORIGINS` environment variable
2. **Per-tenant CORS** (`middleware/per_tenant_cors.go`): Each tenant can customize allowed origins
3. **Strict default** (`middleware/security_headers.go`): Tenants without explicit CORS config have NO allowed origins

### 2.2 Security Features

- Origin validation: Exact match required (no wildcard with credentials)
- Preflight caching: `Access-Control-Max-Age: 3600`
- Credential support: `Access-Control-Allow-Credentials: true` (only with explicit origins)
- Method restriction: `GET, POST, PUT, PATCH, DELETE, OPTIONS`
- Header restriction: `Authorization, Content-Type, X-Tenant-ID, X-Request-ID, X-API-Key`

### 2.3 Production Configuration

```bash
# Set explicit origins in production
CORS_ALLOWED_ORIGINS=https://ggid-console.iot2.win,https://admin.ggid.dev

# NEVER use * in production with credentials
```

**Audit Result**: PASS. CORS implementation follows OWASP best practices.

### 2.4 Known Issue

The default CORS config allows `*` in some test paths. This is overridden in production by the env var. The `TenantCORSMiddleware` in `security_headers.go` enforces strict defaults (no origins) when tenant has no explicit config.

---

## 3. JWT Key Rotation

### 3.1 Implementation

JWT signing keys support graceful rotation via `RotatingKeyProvider` (`services/oauth/internal/service/key_rotation.go`):

```go
type RotatingKeyProvider struct {
    current     *rsa.PrivateKey   // active signing key
    currentID   string            // key identifier (kid)
    previous    *rsa.PrivateKey   // old key (valid during grace period)
    previousID  string
    rotatedAt   time.Time
    gracePeriod time.Duration     // default: 24h
}
```

### 3.2 Rotation Flow

1. `RotateKey()` generates a new RSA-2048 key
2. Current key is demoted to "previous"
3. Previous key remains valid for verification during grace period (24h default)
4. After grace period, `CleanupExpired()` removes the old key
5. JWKS endpoint (`/.well-known/jwks.json`) publishes both keys during transition

### 3.3 Key Rotation Log

Rotation events are persisted in PostgreSQL (`key_rotation_log` table):

```sql
CREATE TABLE key_rotation_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_type TEXT NOT NULL,      -- jwt_signing, scep_ca, webhook_hmac
    old_key_id TEXT,
    new_key_id TEXT NOT NULL,
    status TEXT DEFAULT 'active', -- active, grace, expired
    rotated_at TIMESTAMPTZ DEFAULT now(),
    grace_expires_at TIMESTAMPTZ
);
```

### 3.4 Automation Status

**Manual rotation**: Available via `POST /api/v1/auth/key-rotation/rotate` API endpoint.

**Automated rotation**: NOT YET IMPLEMENTED. For production hardening, add a scheduled job:

```go
// Recommended: rotate every 90 days via cron or k8s CronJob
ticker := time.NewTicker(24 * time.Hour)
for range ticker.C {
    if time.Since(provider.RotatedAt()) > 90*24*time.Hour {
        provider.RotateKey()
    }
    provider.CleanupExpired()
}
```

### 3.5 Recommendations

1. Implement a periodic key rotation goroutine (every 90 days)
2. Add Prometheus metrics for key age and rotation events
3. Alert on rotation failures
4. Document the rotation procedure in the runbook

---

## 4. Password Security

### 4.1 Argon2id Parameters (KB-325)

| Parameter | Previous (v1.0-beta) | Current (v1.0-stable) | OWASP Recommendation |
|-----------|----------------------|----------------------|---------------------|
| Memory | 64 MB | 19 MB | >= 19 MB |
| Iterations | 3 | 2 | >= 2 |
| Parallelism | 2 | 1 | 1 (single-lane) |
| Salt Length | 16 bytes | 16 bytes | >= 16 bytes |
| Key Length | 32 bytes | 32 bytes | >= 32 bytes |

The tuned parameters target <100ms login latency while maintaining OWASP compliance. Existing password hashes verify with their embedded parameters (stored in the hash string format `argon2id$iter$mem$par$salt.hash`).

### 4.2 Password Pepper

A server-side HMAC-SHA256 pepper is configurable via `AUTH_PASSWORD_PEPPER` environment variable. This adds defense-in-depth against database-only compromise.

```bash
# Set in production (32+ random bytes, base64-encoded)
AUTH_PASSWORD_PEPPER=$(openssl rand -base64 32)
```

### 4.3 Password Policies

- Minimum length: configurable per tenant (default: 8)
- zxcvbn strength evaluation (KB-082)
- Password history (prevents reuse)
- Account lockout after failed attempts (Redis-backed)

---

## 5. Rate Limiting

### 5.1 Multi-Layer Architecture

| Layer | Scope | Mechanism |
|-------|-------|-----------|
| Gateway RateLimiter | Per-IP, per-endpoint | Fixed-window (in-memory) |
| TenantBucketLimiter | Per-tenant + IP | Token bucket |
| MultiDimRateLimiter | 5-dimensional | Burst + sustained |

### 5.2 Endpoint-Specific Limits

| Endpoint | Limit | Rationale |
|----------|-------|-----------|
| `/api/v1/auth/login` | 5/min | Brute-force protection |
| `/api/v1/auth/register` | 3/min | Spam prevention |
| `/oauth/token` | 10/min | Client secret brute-force (KB-326) |
| `/api/v1/*` (general) | 100/min | General API throttling |

### 5.3 Recommendations

- Replace in-memory limiter with Redis-backed for multi-replica deployments
- Add `X-RateLimit-Reset` header on all 429 responses
- Monitor rate limit hit rates via `ggid_rate_limit_hits_total` metric

---

## 6. Token Security

### 6.1 Token Types

| Token | Storage | TTL | Revocation |
|-------|---------|-----|------------|
| Access token | Stateless JWT | 15min | Redis blocklist |
| Refresh token | Redis + PG | 7d | Immediate (Redis DEL) |
| Session token | Redis | Configurable | Immediate (Redis DEL) |

### 6.2 Token Binding

- **DPoP** (RFC 9449): Tokens bound to client's public key
- **mTLS**: Optional mutual TLS for high-security clients
- **Token family tracking**: Refresh token rotation detects reuse

### 6.3 Recommendations

- Shorten access token TTL to 10min for high-security deployments
- Enable DPoP for all machine-to-machine clients
- Monitor refresh token rotation failures (indicates token theft)

---

## 7. Network Security

### 7.1 Security Headers

The gateway sets the following headers on all responses:

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
```

### 7.2 TLS

- TLS 1.2+ required (enforced at load balancer/ingress)
- HSTS with 1-year max-age
- Certificate management via cert-manager (K8s)

### 7.3 Recommendations

- Enable TLS 1.3 only for internal service-to-service communication
- Add certificate expiry monitoring
- Implement Perfect Forward Secrecy

---

## 8. Audit & Monitoring

### 8.1 Audit Events

All security-relevant actions are logged to the audit service:
- Login/logout (success + failure)
- Password changes
- Role assignments
- Privileged operations (PAM)
- CCM control evaluations

### 8.2 Metrics

Prometheus metrics exposed at `/metrics`:
- `ggid_auth_attempts_total{result, tenant_id}`
- `ggid_token_issuance_total{grant_type}`
- `ggid_rate_limit_hits_total{endpoint, tier}`
- `ggid_risk_evaluations_total{result}`

### 8.3 Recommendations

- Set up alerting on authentication failure spikes (>10/min per tenant)
- Monitor for impossible travel patterns
- Log all key rotation events to external SIEM

---

## 9. Dependency Security

### 9.1 Vulnerability Scanning

- `govulncheck` runs in CI (added in KB-315)
- Dependabot enabled for Go modules
- Regular `go mod tidy` + upgrade cycle

### 9.2 Current Status

See `docs/security/dependency-audit.md` for the latest dependency scan results.

---

## 10. Deployment Hardening Checklist

- [ ] Set `AUTH_PASSWORD_PEPPER` environment variable
- [ ] Configure `CORS_ALLOWED_ORIGINS` with explicit origins (no wildcard)
- [ ] Enable Redis-backed rate limiting for multi-replica deployments
- [ ] Set up automated JWT key rotation (90-day cycle)
- [ ] Configure SIEM forwarding for audit events
- [ ] Enable HSTS at ingress/load balancer layer
- [ ] Set up Prometheus alerting for auth failures
- [ ] Run `govulncheck` weekly
- [ ] Verify database connection uses TLS
- [ ] Restrict Redis with AUTH password + network policy

---

## References

- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [RFC 6749 - OAuth 2.0 Framework](https://tools.ietf.org/html/rfc6749)
- [RFC 7591 - OAuth 2.0 Dynamic Client Registration](https://tools.ietf.org/html/rfc7591)
- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)
- [Argon2id RFC 9106](https://tools.ietf.org/html/rfc9106)
