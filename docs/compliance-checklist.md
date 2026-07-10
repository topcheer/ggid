# GGID Compliance Checklist

How GGID meets GDPR, SOC 2, and HIPAA requirements.

---

## GDPR (General Data Protection Regulation)

### Right to Erasure (Article 17)

```bash
# Delete user and all associated data
DELETE /api/v1/users/{user_id}
```

Cascade deletes across all tables:
- `users` → `credentials`, `user_roles`, `org_members`
- Audit events retain `actor_id` as NULL (compliance requires audit trail, not full erasure)
- Redis session data expires via TTL

**Status:** Implemented. Verified by test.

### Right to Data Export (Article 20)

```bash
GET /api/v1/users/{user_id}?format=json
```

Exports user profile, roles, org memberships, and recent audit events.

**Status:** Implemented.

### Data Processing Records (Article 30)

All data processing is logged via audit events:
- Who accessed data (`actor_id`)
- What action was performed (`action`)
- When (`timestamp`)
- From where (`ip_address`)

**Status:** Implemented (NATS JetStream + PostgreSQL).

### Consent Management

No explicit consent tracking built-in. For GDPR consent:
- Use user metadata to store consent flags
- Use audit events to record when consent was given/withdrawn

```bash
# Record consent
PUT /api/v1/users/{id}
{"metadata": {"consent_marketing": "2024-07-10T12:00:00Z"}}
```

**Status:** Partially supported (via metadata + audit trail).

### Data Residency

GGID is self-hosted. Data stays in your infrastructure. For multi-region:
- Deploy GGID in the required region
- Use region-specific DATABASE_URL
- No cross-region data flow unless explicitly configured

**Status:** Full control (self-hosted).

---

## SOC 2 Type II

### CC1: Control Environment

| Requirement | GGID Implementation | Status |
|-------------|---------------------|:------:|
| Access controls | RBAC + ABAC policy engine | Implemented |
| Least privilege | Role hierarchy with scoped permissions | Implemented |
| Segregation of duties | Tenant isolation via RLS | Implemented |

### CC2: Communication and Information

| Requirement | GGID Implementation | Status |
|-------------|---------------------|:------:|
| System monitoring | Prometheus + Grafana | Implemented |
| Alerting | Alert rules for security events | Implemented |
| Audit logging | NATS JetStream → PostgreSQL | Implemented |
| Change management | Audit trail for all config changes | Implemented |

### CC3: Risk Assessment

| Requirement | GGID Implementation | Status |
|-------------|---------------------|:------:|
| Vulnerability scanning | govulncheck + Trivy | Implemented |
| Penetration testing | OWASP Top 10 checklist | Documented |

### CC4: Monitoring Activities

| Requirement | GGID Implementation | Status |
|-------------|---------------------|:------:|
| Continuous monitoring | Prometheus metrics + health checks | Implemented |
| Anomaly detection | Audit anomaly rules API | Implemented |
| Incident response | Documented runbook | Documented |

### CC5: Control Activities

| Requirement | GGID Implementation | Status |
|-------------|---------------------|:------:|
| Authentication | MFA (TOTP, WebAuthn, Email OTP) | Implemented |
| Session management | JWT with rotation + blocklist | Implemented |
| Rate limiting | Per-IP and per-tenant | Implemented |

### CC6: Logical and Physical Access

| Requirement | GGID Implementation | Status |
|-------------|---------------------|:------:|
| Unique user identification | UUID-based user IDs | Implemented |
| Authentication methods | Password + MFA + WebAuthn | Implemented |
| Access revocation | Token blocklist + user lock | Implemented |
| Password policy | Configurable complexity + history | Implemented |

### CC7: System Operations

| Requirement | GGID Implementation | Status |
|-------------|---------------------|:------:|
| Change management | Git-based deployment + audit | Implemented |
| Configuration management | Environment variables + Helm | Implemented |
| Software vulnerability management | govulncheck CI + image scanning | Implemented |

### CC8: Change Management

| Requirement | GGID Implementation | Status |
|-------------|---------------------|:------:|
| Authorized changes | Admin role required for config | Implemented |
| Change documentation | Audit events for all changes | Implemented |

---

## HIPAA

### Administrative Safeguards

| Requirement | GGID Implementation | Status |
|-------------|---------------------|:------:|
| Access management (164.308(a)(4)) | RBAC + ABAC | Implemented |
| Audit controls (164.308(a)(1)(ii)(D)) | NATS audit pipeline | Implemented |
| Automatic logoff | Session timeout configurable | Implemented |

### Physical Safeguards

| Requirement | GGID Implementation | Status |
|-------------|---------------------|:------:|
| Facility access | Self-hosted (your responsibility) | N/A |
| Workstation security | Self-hosted (your responsibility) | N/A |

### Technical Safeguards

| Requirement | GGID Implementation | Status |
|-------------|---------------------|:------:|
| Access control (164.312(a)(1)) | JWT + RBAC + ABAC | Implemented |
| Audit controls (164.312(b)) | Audit events for all PHI access | Implemented |
| Integrity (164.312(c)(1)) | RLS prevents unauthorized data modification | Implemented |
| Authentication (164.312(d)) | Password + MFA + WebAuthn | Implemented |
| Transmission security (164.312(e)(1)) | TLS 1.3 for all connections | Implemented |

### Encryption

| Requirement | GGID Implementation | Status |
|-------------|---------------------|:------:|
| At rest (164.312(a)(2)(iv)) | PostgreSQL TDE / disk encryption | Configurable |
| In transit (164.312(e)(2)(ii)) | TLS 1.3 required | Implemented |
| Password hashing | Argon2id (irreversible) | Implemented |

### Audit Log Retention

HIPAA requires 6-year retention for audit logs:

```bash
# Configure retention (default is configurable)
PUT /api/v1/audit/retention
{"retention_days": 2190}  # 6 years
```

**Note:** Long retention requires audit_events table partitioning + archival.

### BAA (Business Associate Agreement)

GGID is open-source (Apache 2.0). No BAA is needed since you self-host.
Your organization's infrastructure provider (AWS, GCP, Azure) requires a BAA.

---

## ISO 27001

| Control | GGID Implementation | Status |
|---------|---------------------|:------:|
| A.5: Information security policies | Documented in security guides | Documented |
| A.6: Organization of information security | Tenant isolation, role separation | Implemented |
| A.7: Human resource security | Self-hosted (your responsibility) | N/A |
| A.8: Asset management | Self-hosted (your responsibility) | N/A |
| A.9: Access control | RBAC + ABAC + MFA | Implemented |
| A.10: Cryptography | Argon2id, AES-256-GCM, RSA-2048, TLS 1.3 | Implemented |
| A.12: Operations security | Audit logging, vulnerability scanning | Implemented |
| A.13: Communications security | TLS, mTLS capable, network policies | Implemented |
| A.14: System acquisition/development | Secure coding standards, govulncheck | Implemented |
| A.16: Incident management | Audit anomaly detection, incident runbook | Documented |
| A.18: Compliance | GDPR/SOC2/HIPAA checklists | Documented |
