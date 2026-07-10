# GGID Security Whitepaper

> Threat model, security architecture, and mitigations for the GGID IAM Platform.

---

## 1. Threat Model (STRIDE)

| Threat | Category | Risk | Mitigation |
|--------|----------|------|------------|
| Forged JWT tokens | Spoofing | Critical | RS256 signatures + JWKS rotation + key ID in header |
| Cross-tenant data access | Information Disclosure | Critical | PostgreSQL Row-Level Security (RLS) per tenant_id |
| Brute-force password | Spoofing | High | Account lockout (5 attempts / 15 min lock) + rate limiting |
| Token replay | Repudiation | High | JWT `exp` (15 min access, 7d refresh) + refresh rotation |
| Privilege escalation | Tampering | High | RBAC + ABAC policy engine enforces least privilege |
| Audit log tampering | Repudiation | High | NATS JetStream (durable, at-least-once delivery) + append-only DB |
| Session hijacking | Spoofing | Medium | HTTP-only + Secure cookies + SameSite=Strict |
| CSRF | Spoofing | Medium | Double-submit cookie + SameSite cookies |
| XSS in Console | Tampering | Medium | React auto-escaping + CSP headers + no inline scripts |
| Man-in-the-middle | Information Disclosure | Critical | TLS 1.3 required + HSTS header |
| SQL injection | Tampering | High | Parameterized queries only (pgx) + RLS as backstop |
| Supply chain attack | Tampering | Medium | Go modules from verified sources + Docker image scanning |

---

## 2. Authentication Security

### 2.1 Password Storage
- **Algorithm**: bcrypt (cost 12)
- **Minimum length**: 12 characters
- **Complexity**: requires uppercase, lowercase, digit, special
- **History**: last 10 passwords checked, reuse rejected
- **Expiration**: configurable per tenant (default: 90 days)

### 2.2 JWT Security
```
Header: { "alg": "RS256", "kid": "key-2024-01", "typ": "JWT" }
Payload: {
  "iss": "ggid",
  "sub": "user-uuid",
  "aud": "tenant-uuid",
  "exp": 1700000000,
  "iat": 1699999100,
  "scope": "openid profile email",
  "roles": ["admin", "user_manager"],
  "tenant_id": "uuid"
}
```

- **Signing**: RS256 (asymmetric, 2048-bit RSA)
- **Key rotation**: JWKS endpoint, 90-day rotation, overlap period
- **Access token TTL**: 15 minutes (configurable per tenant)
- **Refresh token TTL**: 7 days, rotated on each use
- **Revocation**: token blacklist in Redis + revocation endpoint (RFC 7009)

### 2.3 Multi-Factor Authentication
| Method | Implementation | Use Case |
|--------|---------------|----------|
| TOTP | RFC 6238, 30s window, Google Authenticator compatible | Default MFA |
| WebAuthn / Passkey | CTAP 2.0, platform authenticators | Passwordless |
| Email OTP | 6-digit code, 10 min expiry | Verification |
| SMS OTP | Via Twilio (optional) | High-risk operations |

### 2.4 Session Management
- Server-side sessions in Redis (not stateless JWT-only)
- Session ID: 256-bit CSPRNG, HTTP-only + Secure cookie
- Concurrent session limit: 5 per user (configurable)
- Idle timeout: 30 min (configurable)
- Step-up auth for sensitive operations (require recent MFA)

---

## 3. Authorization Security

### 3.1 RBAC + ABAC
- **RBAC**: Roles → Permissions mapping (many-to-many)
- **ABAC**: Attribute-based conditions (IP, time, device, risk score)
- **Policy evaluation**: deny-by-default, explicit allow required
- **Role hierarchy**: parent roles inherit child permissions (planned)

### 3.2 Policy Decision Point
```
Request → Policy Engine
  → Check RBAC: user has role with required permission?
  → Check ABAC: conditions satisfied (IP allowlist, time window)?
  → Check Deny rules: explicit deny overrides allow
  → Decision: ALLOW or DENY
```

---

## 4. Network Security

### 4.1 TLS Configuration
- TLS 1.3 required (1.2 with PFS as fallback)
- HSTS: `max-age=63072000; includeSubDomains; preload`
- Cipher suites: ChaCha20-Poly1305, AES-GCM only
- Certificate: Let's Encrypt or customer-provided

### 4.2 API Gateway Defense
| Layer | Control |
|-------|---------|
| DDoS | Rate limiting per tenant per endpoint |
| Bot | User-Agent analysis + behavioral fingerprinting |
| IP | Per-tenant IP allowlist |
| Size | Request body size limit (per route) |
| CORS | Strict origin allowlist per tenant |
| Headers | X-Content-Type-Options, X-Frame-Options, CSP |

---

## 5. Data Security

### 5.1 At Rest
- PostgreSQL: TDE (Transparent Data Encryption) or disk-level encryption
- Backups: AES-256 encrypted, stored in separate region
- Secrets: never in plaintext config, Docker secrets / K8s secrets / Vault

### 5.2 In Transit
- All inter-service communication: mTLS (K8s) or VPC-private network
- Database connections: TLS required
- Redis: TLS optional (recommended for production)
- NATS: TLS required for JetStream

### 5.3 PII Handling
- Email and phone: stored in `users` table, encrypted at column level via pgcrypto
- Password: bcrypt hash only (never plaintext, never reversible)
- Audit logs: user_id reference only (no PII in audit events)
- Right to erasure: cascading delete across all services

---

## 6. Compliance

| Framework | Status |
|-----------|--------|
| OWASP Top 10 2021 | All categories addressed |
| SOC 2 Type II | Audit trail + access controls implemented |
| GDPR | Data export, right to erasure, data residency (per-tenant region planned) |
| HIPAA | Audit logging + encryption at rest/transit (BAA required) |
| ISO 27001 | Information security management controls implemented |

---

## 7. Incident Response

### Detection
- Real-time anomaly detection in audit service (unusual login patterns)
- Alerting via webhook (Slack/PagerDuty integration)
- Prometheus alert rules for security events

### Response
1. Revoke compromised JWTs (blacklist in Redis)
2. Force password reset for affected users
3. Lock affected tenant if breach confirmed
4. Generate forensic audit report (export from audit service)

---

## 8. Security Checklist for Production Deployment

- [ ] TLS certificates configured (not self-signed)
- [ ] Strong JWT signing key (2048-bit RSA minimum)
- [ ] Unique DB password (not default "ggid")
- [ ] Unique Redis password (not default "ggid-redis")
- [ ] LDAP bind password stored in secret
- [ ] Rate limits configured per tenant
- [ ] IP allowlist configured for admin endpoints
- [ ] Audit log retention policy set (minimum 90 days)
- [ ] Backup encryption enabled
- [ ] HSTS header enforced
- [ ] CSP header configured for Console
- [ ] Container images scanned (Trivy/Grype)
- [ ] Secret management (Vault, not env vars in prod)
