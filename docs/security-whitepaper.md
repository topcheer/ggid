# GGID Security Whitepaper

> Threat model and security architecture for the GGID IAM Platform.

---

## 1. Threat Model (STRIDE)

| Threat | Category | Mitigation |
|--------|----------|------------|
| Forged JWT tokens | Spoofing | RS256 signatures + JWKS rotation, key ID in header |
| Stolen credentials | Spoofing | MFA (TOTP/WebAuthn), password history, lockout policy |
| Tenant data leakage | Information Disclosure | PostgreSQL Row-Level Security (RLS) per tenant_id |
| Privilege escalation | Elevation of Privilege | RBAC + ABAC policy engine, least-privilege defaults |
| Replay attacks | Repudiation | JWT `iat`/`exp` + nonce + refresh token rotation |
| Audit log tampering | Repudiation | Append-only NATS JetStream + immutable storage |
| DoS / brute force | Denial of Service | Per-tenant rate limiting + account lockout + IP allowlist |
| Supply chain attack | Tampering | Pinned dependencies, Docker image scanning, signed builds |

---

## 2. Authentication Security

### 2.1 Password Storage

- **Algorithm**: bcrypt (cost 12)
- **Never stored in plaintext** — only the hash is persisted
- **Password history**: configurable (default 5) — rejects reused passwords
- **Password policy**: min 8 chars, upper+lower+digit+special

### 2.2 JWT Security

```
Header: { "alg": "RS256", "kid": "key-2024-v1", "typ": "JWT" }
Payload: {
  "iss": "ggid",
  "sub": "user-uuid",
  "aud": "tenant-uuid",
  "exp": 1735689600,
  "iat": 1735603200,
  "jti": "unique-token-id",
  "scope": "users:read users:write",
  "tenant_id": "tenant-uuid",
  "roles": ["admin", "user_manager"]
}
```

- **Signing**: RS256 (asymmetric) — private key never leaves Auth Service
- **Key rotation**: JWKS endpoint supports multiple keys, `kid` header selects correct key
- **Lifetime**: Access token 15min, Refresh token 7d (configurable per tenant)
- **Revocation**: Token blacklist in Redis + JWKS key revocation

### 2.3 MFA / Passwordless

| Method | Standard | Use Case |
|--------|----------|----------|
| TOTP | RFC 6238 (Google Authenticator, Authy) | Standard MFA |
| WebAuthn / Passkey | W3C WebAuthn, FIDO2 | Phishing-resistant MFA |
| Email OTP | Custom | Passwordless login |
| SMS OTP | Custom | Phone-based verification (less secure) |

### 2.4 Session Management

- Sessions stored in Redis with TTL
- Concurrent session limit per user (configurable, default 5)
- Session revocation on password change / MFA enrollment
- Refresh token rotation: each refresh issues a new refresh token, old one invalidated

---

## 3. Authorization Security

### 3.1 RBAC + ABAC Policy Engine

```
Decision = RBAC(user_roles, required_permissions)
         AND ABAC(user_attrs, resource_attrs, environment, action)
```

- **RBAC**: Role → Permission mapping, evaluated first (fast path)
- **ABAC**: Attribute-based conditions (e.g., "user.department == resource.department")
- **Default deny**: If no policy explicitly allows, access is denied
- **Policy caching**: Evaluated policies cached in Redis (5min TTL)

### 3.2 Role Hierarchy

- Parent roles inherit child role permissions
- Prevents circular inheritance (validated at assignment time)
- Temporary role assignment with TTL (auto-expire)

---

## 4. Multi-Tenant Isolation

### 4.1 Data Isolation

```sql
-- Every table has tenant_id column
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Policy: users can only see their tenant's data
CREATE POLICY tenant_isolation ON users
  FOR ALL
  USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

- **Defense in depth**: Isolation at both application (WHERE tenant_id = ?) and database (RLS) levels
- **Connection-level isolation**: `SET LOCAL app.tenant_id` per transaction
- **Cross-tenant queries**: Impossible even with SQL injection (PostgreSQL enforces RLS)

### 4.2 Per-Tenant Configuration

| Setting | Scope | Default |
|---------|-------|---------|
| Rate limits | Per tenant per endpoint | 100 req/min |
| Password policy | Per tenant | min 8, complexity on |
| MFA requirement | Per tenant | Optional |
| Session lifetime | Per tenant | Access 15m, Refresh 7d |
| IP allowlist | Per tenant | Disabled |
| JWT claims | Per tenant | Standard claims |

---

## 5. Network Security

### 5.1 TLS / mTLS

- All external traffic TLS 1.2+ (TLS 1.3 preferred)
- Internal service-to-service: mTLS via service mesh (optional, Istio/Linkerd)
- JWT signing keys: RS256, 2048-bit RSA minimum

### 5.2 API Security

- **CORS**: Per-origin configurable, no wildcard in production
- **CSRF**: SameSite=Strict cookies for session-based auth
- **Headers**: X-Content-Type-Options, X-Frame-Options, Strict-Transport-Security
- **Body size**: Per-route limits (login: 4KB, file upload: configurable)
- **Bot detection**: User-Agent analysis + behavioral fingerprinting

---

## 6. Audit & Compliance

### 6.1 Audit Trail

Every security-relevant event is logged:

| Event Type | Trigger | Fields |
|------------|---------|--------|
| `user.login.success` | Successful login | user_id, ip, user_agent, tenant_id |
| `user.login.failed` | Failed login | username, ip, reason |
| `user.register` | New registration | user_id, email, tenant_id |
| `user.password.change` | Password changed | user_id |
| `user.mfa.enable` | MFA enrolled | user_id, method |
| `role.assign` | Role granted | user_id, role_id, assigned_by |
| `policy.evaluate` | Policy decision | user_id, resource, decision |
| `token.revoke` | Token revoked | user_id, token_jti |

### 6.2 Log Integrity

- Events published to NATS JetStream (durable, ordered)
- Consumer writes to PostgreSQL (queryable) + S3 (archival)
- Hash chaining: each event includes hash of previous event (tamper detection)
- Retention: configurable per tenant (default 90 days hot, 7 years cold)

### 6.3 Compliance Frameworks

| Framework | Status |
|-----------|--------|
| SOC 2 Type II | Architecture ready, audit pending |
| GDPR | Data export/deletion endpoints, right-to-be-forgotten |
| HIPAA | BAA-ready (PHI fields configurable, encryption at rest) |
| ISO 27001 | Control mapping documented |

---

## 7. Incident Response

### 7.1 Anomaly Detection

- Failed login spike detection (> 10/min per IP → alert)
- Geographic anomaly (login from new country → step-up auth)
- Impossible travel detection (login from 2 distant IPs within 1h)
- Token reuse detection (same refresh token from 2 IPs → revoke)

### 7.2 Breach Response

1. **Revoke all tokens**: `POST /api/v1/admin/revoke-all` (admin only)
2. **Force password reset**: `POST /api/v1/admin/force-reset` (marks all users requiring reset)
3. **Lock affected tenant**: Disable tenant, redirect to maintenance page
4. **Export audit trail**: Preserve evidence for forensic analysis

---

## 8. Dependency Security

- **Go modules**: `go mod tidy` + `govulncheck` in CI
- **Docker images**: Distroleless base, Trivy scan in CI
- **npm packages**: `npm audit` + Dependabot
- **Secrets**: Never in git, stored in Vault / cloud KMS, injected as env vars
