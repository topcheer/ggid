# Security Audit Checklist

> STRIDE, OWASP Top 10, and GGID security verification checklist.

---

## STRIDE Threat Model

| Threat | GGID Control | Verified |
|--------|-------------|----------|
| **Spoofing** | JWT RS256 + JWKS + jti anti-replay | ✅ |
| **Tampering** | PostgreSQL RLS + audit hash chain | ✅ |
| **Repudiation** | NATS JetStream audit pipeline | ✅ |
| **Info Disclosure** | PII obfuscation + tenant isolation | ✅ |
| **DoS** | Rate limiting + circuit breaker | ✅ |
| **Elevation of Privilege** | RBAC + ABAC + scope enforcement | ✅ |

---

## OWASP Top 10 (2021)

| # | Category | GGID Control |
|---|----------|-------------|
| A01 | Broken Access Control | RBAC + ABAC + RLS tenant isolation |
| A02 | Cryptographic Failures | RS256 JWT, Argon2id, TLS 1.3, gRPC mTLS |
| A03 | Injection | Parameterized queries (pgx), input validation |
| A04 | Insecure Design | Threat-modeled (STRIDE), defense in depth |
| A05 | Security Misconfiguration | Security headers, HSTS, no default secrets |
| A06 | Vulnerable Components | `go mod tidy`, dependabot |
| A07 | Auth Failures | Rate limiting, MFA, account lockout |
| A08 | Data Integrity Failures | Audit hash chain, signed webhooks |
| A09 | Logging Failures | Structured logging + NATS audit pipeline |
| A10 | SSRF | Webhook URL validation, IP allowlist |

---

## JWT Security Checklist

- [ ] RS256 (not HS256)
- [ ] Short expiry (15 min access token)
- [ ] jti anti-replay (Redis SETNX)
- [ ] iss claim verified
- [ ] aud claim verified
- [ ] Refresh token rotation

## CSRF Protection

- [ ] OAuth state parameter (crypto/rand)
- [ ] SameSite cookies
- [ ] Origin header validation

## SQL Injection

- [ ] Parameterized queries only (pgx `$1`)
- [ ] No string concatenation in SQL
- [ ] RLS enforced on all tenant tables

---

## Verification Commands

```bash
# Check JWT algorithm
echo $JWT | cut -d. -f1 | base64 -d | jq .alg  # Must be RS256

# Verify hash chain
curl -s .../api/v1/audit/verify | jq .verified  # Must be true

# Check security headers
curl -sI .../ | grep -E 'Strict-Transport|X-Frame|Content-Security'

# Test rate limiting
for i in $(seq 1 200); do curl -s -o /dev/null -w '%{http_code}
' .../api/v1/users; done | sort | uniq -c
```

---

*See: [Security Hardening](security-hardening.md) | [Security Overview](../architecture/security-overview.md)*

*Last updated: 2025-07-11*
