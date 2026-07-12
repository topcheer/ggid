# Penetration Testing Guide

This guide covers penetration testing GGID deployments — scope definition, test scenarios per service, OAuth/JWT attack vectors, rate limit bypass, reporting, and remediation.

## Scope Definition

### In-Scope Targets

| Target | URL | Auth |
|--------|-----|------|
| API Gateway | https://api.ggid.example.com | JWT |
| Admin Console | https://console.ggid.example.com | Session |
| OAuth/OIDC | https://api.ggid.example.com/oauth | Client credentials |
| SCIM | https://api.ggid.example.com/scim/v2 | SCIM token |
| JWKS | https://api.ggid.example.com/.well-known/jwks.json | None |

### Out-of-Scope

- Infrastructure DoS (use rate limiting analysis instead)
- Social engineering
- Physical access
- Third-party libraries (separate dependency audit)

## Test Scenarios by Service

### Gateway

| Test | Attack Vector | Expected |
|------|--------------|----------|
| JWT none algorithm | `alg: none` header | Rejected (only RS256/ES256) |
| JWT algorithm confusion | `alg: HS256` with RSA key | Rejected |
| JWT expired | `exp` in past | 401 |
| JWT forged signature | Random signature | 401 |
| Tenant spoofing | X-Tenant-ID mismatch | JWT claim wins |
| Rate limit bypass | Rapid requests from multiple IPs | Token bucket per IP |
| SSRF via webhook | localhost/metadata URLs | Blocked (RFC 1918) |
| SQL injection in query params | `' OR 1=1` | Parameterized queries (safe) |
| XSS in headers | `<script>` in User-Agent | Sanitized |

### Auth Service

| Test | Attack Vector | Expected |
|--------|--------------|----------|
| Brute force | 100 logins/min | Rate limited after 5 |
| Credential stuffing | Known password list | Lockout after 5 failures |
| Password pepper bypass | Offline hash cracking | Argon2id + pepper (infeasible) |
| MFA TOTP brute force | 1M codes | TOTP window = 1 (3 codes max) |
| Session fixation | Reuse session post-login | Session regenerated |
| JWT jti replay | Reuse revoked token | jti blacklist catches |
| Refresh token reuse | Use rotated token | Token family revoked |

### OAuth Service

| Test | Attack Vector | Expected |
|--------|--------------|----------|
| Auth code interception | Steal code from redirect | PKCE prevents |
| Redirect URI bypass | Path traversal in redirect | Exact match only |
| Open redirect | Manipulate redirect_uri | Rejected |
| State CSRF | Missing/invalid state | Rejected |
| Implicit grant | response_type=token | Not supported |
| Scope escalation | Request admin scope | Limited to registered |
| Client impersonation | Guess client_secret | 401 invalid_client |
| Token introspection without auth | POST /introspect | 401 (requires client auth) |

### Identity Service

| Test | Attack Vector | Expected |
|--------|--------------|----------|
| User enumeration | Different login errors | Generic errors |
| IDOR | /users/{other_user_id} | RLS blocks cross-tenant |
| Mass assignment | Extra fields in POST | Whitelisted fields only |
| SCIM injection | Malformed SCIM payload | Validated |
| Bulk export without auth | GET /users/export | Requires admin scope |

### Audit Service

| Test | Attack Vector | Expected |
|--------|--------------|----------|
| Audit tampering | Modify audit_events row | Hash chain detects |
| Audit deletion | DELETE on audit event | Soft-delete only |
| Hash chain forgery | Recompute all hashes | Requires breaking SHA-256 |

## OWASP Top 10 Mapping

| OWASP | Test |
|-------|------|
| A01 Broken Access Control | IDOR, tenant spoofing, scope escalation |
| A02 Cryptographic Failures | JWT alg confusion, weak TLS |
| A03 Injection | SQLi in params, SCIM injection |
| A04 Insecure Design | No rate limit, no MFA |
| A05 Security Misconfiguration | Default creds, debug endpoints |
| A06 Vulnerable Components | go.mod audit |
| A07 Auth Failures | Brute force, credential stuffing |
| A08 Software/Data Integrity | Hash chain bypass |
| A09 Logging/Monitoring Failures | Missing audit events |
| A10 SSRF | Webhook URL injection |

## Testing Tools

```bash
# JWT analysis
python3 jwt_tool.py <jwt_token>

# OAuth testing
burp suite (OAuth scan extension)

# Rate limit testing
for i in $(seq 1 100); do curl -s -o /dev/null -w "%{http_code}\n" https://api.ggid.example.com/api/v1/auth/login -d '{...}'; done

# SSRF testing
curl -X POST https://api.ggid.example.com/api/v1/webhooks -d '{"url":"http://169.254.169.254/latest/meta-data/"}'

# Dependency audit
go list -m all | trufflehog --no-update
```

## Reporting

### Finding Template

```markdown
## Finding: [Title]
**Severity**: Critical / High / Medium / Low
**OWASP**: A0X
**Endpoint**: POST /api/v1/auth/login
**Description**: ...
**Steps to reproduce**: ...
**Proof of concept**: ...
**Remediation**: ...
**Status**: Open / Fixed / Verified
```

### Severity Scale

| Severity | Criteria |
|----------|----------|
| Critical | Data breach, auth bypass, RCE |
| High | Privilege escalation, token theft |
| Medium | Info disclosure, rate limit bypass |
| Low | Minor config issues, info leak |

## Remediation Tracking

| Finding ID | Severity | Status | Owner | Due Date |
|-----------|----------|--------|-------|----------|
| PEN-001 | High | Fixed | backend | 2025-02-01 |
| PEN-002 | Medium | Open | arch | 2025-03-01 |

## Post-Test Checklist

- [ ] All findings documented
- [ ] Severity assigned per OWASP
- [ ] Remediation plan for each finding
- [ ] Critical findings fixed before report
- [ ] Retest after fixes
- [ ] Final report delivered

## See Also

- [Threat Modeling](threat-modeling.md)
- [Security Audit Checklist](security-audit-checklist.md)
- [ITDR Implementation](itdr-implementation.md)
- [STRIDE Analysis](../research/stride-analysis.md)
