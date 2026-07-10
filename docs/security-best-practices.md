# Security Best Practices

Production security hardening checklist for GGID deployments. Covers TLS,
cipher suites, HSTS, secret management, JWT key rotation, password policy,
rate limiting, CORS, CSP headers, and audit logging.

> **See also**: [Security Checklist](security-checklist.md),
> [Security Hardening](security-hardening.md).

---

## TLS Configuration

### Minimum TLS Version

```nginx
# nginx.conf
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
ssl_prefer_server_ciphers off;
```
### TLS 1.3 Only (Recommended)

```nginx
ssl_protocols TLSv1.3;
```
### Certificate Management

| Item | Requirement |
|------|-------------|
| CA | Let's Encrypt, DigiCert, or internal PKI |
| Key size | RSA 2048+ or ECDSA P-256+ |
| Validity | 90 days (Let's Encrypt auto-renew) |
| OCSP stapling | Enabled |
| HSTS | max-age=63072000; includeSubDomains; preload |

### Security Headers

```
Strict-Transport-Security: max-age=63072000; includeSubDomains; preload
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: geolocation=(), microphone=(), camera=()
```

---

## Secret Management

### Secret Rotation Schedule

| Secret | Rotation | Method |
|--------|----------|--------|
| JWT signing key | 90 days | Dual-key overlap (see below) |
| DB password | 90 days | Vault dynamic secrets |
| Redis password | 90 days | Vault dynamic secrets |
| SMTP password | 180 days | Manual update |
| LDAP bind password | 180 days | Manual update |
| OAuth client secrets | 180 days | Per-client rotation |
| API keys | 365 days | Admin-triggered |
| TLS certificates | 90 days | ACME auto-renew |

### Vault Integration

```yaml
secrets:
  provider: vault
  vault:
    address: https://vault.internal:8200
    auth_method: kubernetes
    role: ggid-auth
    paths:
      database: secret/ggid/db
      jwt_key: secret/ggid/jwt-signing
      redis: secret/ggid/redis
```

---

## JWT Key Rotation

### Dual-Key Overlap

```
Phase 1 (Day 1-7): Key-A active (signs tokens), Key-B accepted (validates)
Phase 2 (Day 8): Switch — Key-B signs, Key-A accepted
Phase 3 (Day 9-15): Key-B active, Key-A accepted for lingering tokens
Phase 4 (Day 16): Remove Key-A
```

```bash
# Generate new key
openssl genpkey -algorithm RSA -out jwt-new.key -pkeyopt rsa_keygen_bits:2048
openssl req -new -x509 -key jwt-new.key -out jwt-new.crt -days 365

# Register as accepted key
curl -X POST .../admin/oauth/signing-keys \
  -F 'key=@jwt-new.crt' -F 'status=accept_only'

# After 7 days, promote to primary
curl -X PATCH .../admin/oauth/signing-keys/primary \
  -d '{ "key_id": "new-key-id" }'
```

---

## Password Policy

```yaml
security:
  password:
    min_length: 12
    require_uppercase: true
    require_lowercase: true
    require_digit: true
    require_special: true
    max_age_days: 90
    history_count: 12            # Can't reuse last 12 passwords
    lockout_threshold: 5
    lockout_duration_minutes: 30
    breach_check: true           # Check against HaveIBeenPwned API
```

---

## Rate Limiting

```yaml
rate_limit:
  auth:
    login: 10/min/IP
    register: 5/min/IP
    password_reset: 3/min/IP
  api:
    authenticated: 1000/min/user
    unauthenticated: 100/min/IP
  admin:
    api: 100/min/user
```

---

## CORS Configuration

```yaml
cors:
  allowed_origins:
    - https://console.example.com
    - https://app.example.com
  allowed_methods: [GET, POST, PUT, PATCH, DELETE, OPTIONS]
  allowed_headers: [Authorization, Content-Type, X-Tenant-ID, X-Request-ID]
  expose_headers: [X-Request-ID, X-RateLimit-Remaining]
  allow_credentials: true
  max_age: 3600
```

> Never use `allowed_origins: ["*"]` with `allow_credentials: true`.

---

## CSP Headers

```
Content-Security-Policy:
  default-src 'self';
  script-src 'self';
  style-src 'self' 'unsafe-inline';
  img-src 'self' data: https:;
  font-src 'self';
  connect-src 'self' https://api.example.com;
  frame-ancestors 'none';
  form-action 'self';
  base-uri 'self';
  object-src 'none';
```

---

## Audit Logging

### Required Audit Events

| Category | Events |
|----------|--------|
| Authentication | All logins (success + failure), logouts, token refresh |
| Authorization | Role assignment/revocation, policy changes |
| Data access | User profile views, audit log queries |
| Admin actions | Config changes, impersonation, bulk operations |
| Security | Account lock/unlock, MFA changes, password resets |

### Log Integrity

- Hash chain verification (SHA-256 linked list)
- Daily automated chain verification
- Alert on chain break detection

### Retention

| Storage | Duration | Purpose |
|---------|----------|---------|
| Hot (DB) | 90 days | Fast query |
| Warm (archive) | 1 year | Compliance |
| Cold (S3) | 7 years | Legal hold |

---

## Production Checklist

### Pre-Launch

- [ ] TLS 1.2+ enforced (TLS 1.3 preferred)
- [ ] HSTS enabled (max-age >= 63072000)
- [ ] All security headers present (CSP, X-Frame-Options, etc.)
- [ ] CORS configured with explicit origins (no wildcard)
- [ ] JWT signed with RS256/ES256 (not HS256)
- [ ] JWT signing key rotation procedure documented
- [ ] Password policy enforced (min 12 chars, complexity)
- [ ] Rate limiting enabled on all endpoints
- [ ] All secrets in Vault (not in env files or code)
- [ ] Database uses RLS (Row-Level Security)
- [ ] Audit logging enabled for all categories
- [ ] Audit hash chain verified daily

### Ongoing

- [ ] JWT keys rotated every 90 days
- [ ] DB/Redis passwords rotated every 90 days
- [ ] TLS certificates auto-renewing
- [ ] Dependency vulnerability scan weekly (`govulncheck`)
- [ ] Penetration test annually
- [ ] Access review quarterly (remove stale roles)
- [ ] Audit log retention verified
- [ ] Backup restore tested monthly