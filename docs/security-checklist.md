# GGID Security Checklist

Pre-production security audit checklist. All items must pass before v1.0-stable release.

---

## Authentication & Sessions

- [x] **Password hashing**: Argon2id is the default algorithm (no bcrypt/sha256 for new passwords)
- [x] **Legacy hash migration**: JIT migration transparently upgrades bcrypt/pbkdf2 → Argon2id on next login
- [x] **Access token TTL**: Configurable, default expires at session creation (typically 15min)
- [x] **Refresh token TTL**: 24 hours with rotation on each use
- [x] **Session binding**: DPoP proof-of-possession supported (optional per session)
- [x] **Rate limiting**: Login attempts rate-limited (brute-force protection with lockout)
- [x] **Password spray detection**: Threshold-based blocking (15 unique users/10min → 24h block)
- [x] **MFA**: TOTP + backup codes + WebAuthn (passkey) supported
- [x] **Break-glass**: Emergency access with full audit trail + reason required

### Risk Items
- [ ] Verify access token TTL is set to ≤15min in production config (currently derived from session TTL)
- [ ] Consider shorter refresh TTL for high-risk tenants (configurable via CAP)

---

## Public Endpoints (No Auth Required)

The following paths skip JWT verification. All are intentionally public:

| Path | Justification | Rate-Limited |
|------|--------------|-------------|
| `/api/v1/auth/login` | User authentication | Yes (brute-force) |
| `/api/v1/auth/register` | Self-service signup | Yes (per-IP) |
| `/api/v1/auth/refresh` | Token rotation | Yes (per-token) |
| `/api/v1/auth/password/forgot` | Password reset request | Yes (per-email) |
| `/api/v1/auth/password/reset` | Password reset execution | Yes (per-token) |
| `/api/v1/auth/social/*` | Social login callback | Yes |
| `/healthz`, `/readyz` | Health checks | No (read-only) |
| `/.well-known/*` | OIDC discovery + JWKS | No (public keys only) |
| `/api/v1/system/initialized` | Bootstrap check | No (boolean only) |
| `/api/v1/system/bootstrap` | Initial tenant setup | Must be disabled after bootstrap |
| `/api/v1/tenants/resolve` | Tenant lookup from slug | Yes |
| `/api/v1/dashboard` | Public dashboard stats | Consider adding auth |
| `/api/v1/oauth/jwks` | Public signing keys | No |

### Risk Items
- [ ] `/api/v1/system/bootstrap` should be disabled after initial setup (env: `BOOTSTRAP_ENABLED=false`)
- [ ] `/api/v1/dashboard` exposes aggregate stats without auth — verify no sensitive data leaks

---

## CORS Configuration

- [x] CORS is configurable via environment variables
- [ ] **Verify production**: `CORS_ALLOWED_ORIGINS` must NOT be `*` in production
- [ ] Set to specific origins: `https://console.yourcompany.com,https://app.yourcompany.com`
- [ ] Preflight cache: `Access-Control-Max-Age` set (reduces OPTIONS overhead)
- [ ] Credentials: `Access-Control-Allow-Credentials: true` only for whitelisted origins

---

## Rate Limiting

| Endpoint | Limit | Scope |
|----------|-------|-------|
| Login | 5 attempts / 5 min | Per username + IP |
| Register | 3 / hour | Per IP |
| Password reset | 3 / hour | Per email |
| Token refresh | 10 / min | Per refresh token |
| OAuth authorize | 10 / min | Per client_id |
| Global API | 1000 / min | Per tenant (configurable) |
| GraphQL | 100 / min | Per user |

- [x] Password spray detection: 15 unique users / 10 min → 24h block
- [x] Multi-dimensional rate limiting: tenant + user + IP + endpoint + tier
- [ ] Verify Redis-backed rate limiter is connected in production (not in-memory)

---

## JWT Configuration

- [x] Algorithm: RS256 (asymmetric, JWKS-compatible)
- [x] Key rotation: JWKS rotation endpoint + graceful overlap period
- [x] Claims: `iss`, `sub`, `aud`, `exp`, `iat`, `jti`, `tenant_id`, `session_id`
- [x] DPoP support: Proof-of-possession binding (optional)
- [x] Refresh token rotation: Single-use with replay detection
- [ ] **Verify**: Access token `exp` ≤ 15min in production config
- [ ] **Verify**: JWT secret is ≥256 bits (check `JWT_SECRET` env var)

---

## Password Storage

- [x] **Argon2id** is the default and only algorithm for new passwords
- [x] Transparent rehashing: legacy hashes (bcrypt, pbkdf2, scrypt, ssha) → Argon2id on next login
- [x] Password history: Configurable (default: 5 previous passwords checked)
- [x] Password strength: zxcvbn score ≥ 2 required at registration + password change
- [x] HIBP breach check: Known breached passwords forced to score 0
- [x] No plaintext passwords stored or logged

---

## Audit Logging

- [x] All sensitive operations emit audit events via NATS:
  - Login/logout/refresh
  - Password change/reset
  - MFA enroll/disable
  - Role assignment/revocation
  - Policy create/update/delete
  - Break-glass activation
  - OAuth client management
  - Conditional access decisions
  - Session revocation
- [x] Hash chain integrity verification endpoint (`/api/v1/audit/verify-integrity`)
- [x] Conditional access policy changes audited with before/after diff
- [x] NHI risk evaluations logged
- [x] CCM compliance scan results persisted to PostgreSQL

### Coverage Gaps
- [ ] User export (CSV) should log what data was exported
- [ ] Admin key rotation should log operator identity

---

## Data Isolation

- [x] Multi-tenant isolation via Row-Level Security (RLS) on all tables
- [x] `X-Tenant-ID` header required on all tenant-scoped requests
- [x] Tenant context propagated through request context
- [ ] **Verify**: RLS policies tested with cross-tenant queries (integration test)

---

## Secrets Management

- [x] Secrets stored in `keys.env` (not in main config YAML)
- [x] `AES_KEY` required (32-byte hex for encryption at rest)
- [x] `JWT_SECRET` required (signing key)
- [x] Database credentials via env vars (not hardcoded)
- [ ] **Verify**: No secrets in git history (run `git log -p | grep -i "password\|secret\|key" | head`)
- [ ] Consider external secret manager (Vault/AWS Secrets Manager) for production

---

## Transport Security

- [x] TLS termination at gateway
- [x] Internal service-to-service over plaintext (behind firewall)
- [ ] **Verify**: TLS 1.2+ only (disable TLS 1.0/1.1)
- [ ] **Verify**: HSTS header set in production
- [ ] **Verify**: Security headers (X-Frame-Options, X-Content-Type-Options, CSP)

---

## Dependency Security

- [x] `govulncheck` runs in CI (advisory mode)
- [x] `gosec` runs in CI (advisory mode)
- [ ] **Action**: Review and fix any high-severity findings before v1.0
- [ ] Run `go mod audit` before release

---

## OWASP Top 10 Coverage

| Risk | Status | Notes |
|------|--------|-------|
| A01: Broken Access Control | ✅ | RBAC + ABAC + CAP + tenant isolation |
| A02: Cryptographic Failures | ✅ | Argon2id + AES-256 + TLS |
| A03: Injection | ✅ | Parameterized queries (pgx) everywhere |
| A04: Insecure Design | ✅ | Threat-modeled architecture |
| A05: Security Misconfiguration | ⚠️ | Verify CORS/bootstrap disabled |
| A06: Vulnerable Components | ⚠️ | govulncheck advisory mode |
| A07: Auth Failures | ✅ | MFA + rate limiting + spray detection |
| A08: Data Integrity Failures | ✅ | Hash chain audit + code signing |
| A09: Logging Failures | ✅ | Comprehensive audit + CCM |
| A10: SSRF | ✅ | No outbound URL fetching from user input |

---

*Last updated: 2025-07-18*
