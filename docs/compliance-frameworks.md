# Compliance Frameworks

How GGID maps to common compliance frameworks: GDPR, SOC 2, HIPAA, and
ISO 27001. Each section cross-references GGID features to specific control
requirements.

---

## Table of Contents

- [GDPR (General Data Protection Regulation)](#gdpr-general-data-protection-regulation)
- [SOC 2 (Service Organization Control 2)](#soc-2-service-organization-control-2)
- [HIPAA (Health Insurance Portability and Accountability Act)](#hipaa-health-insurance-portability-and-accountability-act)
- [ISO 27001 (Information Security Management)](#iso-27001-information-security-management)
- [Cross-Framework Feature Matrix](#cross-framework-feature-matrix)

---

## GDPR (General Data Protection Regulation)

The EU GDPR (Regulation 2016/679) governs processing of personal data.
GGID provides features to help satisfy GDPR requirements.

### Article 7: Consent Management

| Requirement | GGID Feature |
|-------------|-------------|
| Documented consent | OAuth consent screen with per-scope grants |
| Withdraw consent at any time | `oauth.consent.revoked` event + API: `DELETE /api/v1/oauth/consent/{client_id}` |
| Audit trail of consent | Audit events: `oauth.consent.granted`, `oauth.consent.revoked` |

```bash
# View user consents
curl https://iam.example.com/api/v1/oauth/consent \
  -H "Authorization: Bearer $TOKEN"

# Revoke consent for a specific app
curl -X DELETE https://iam.example.com/api/v1/oauth/consent/{client_id} \
  -H "Authorization: Bearer $TOKEN"
```

### Article 15: Right of Access (Data Portability)

| Requirement | GGID Feature |
|-------------|-------------|
| Export user data in machine-readable format | `GET /api/v1/admin/users/{id}/export` (JSON) |
| Include all associated data | User profile, sessions, groups, roles, audit events, OAuth grants |
| Complete within 30 days | Immediate export via API |

```bash
# Full data export for a user (GDPR Article 15)
curl https://iam.example.com/api/v1/admin/users/{user_id}/export \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -o user-data-export.json
```

```json
{
  "user": { "id": "...", "username": "...", "email": "..." },
  "sessions": [...],
  "groups": [...],
  "roles": [...],
  "oauth_grants": [...],
  "audit_events": [...],
  "mfa_factors": [...],
  "webauthn_credentials": [...]
}
```

### Article 17: Right to Erasure (Right to Be Forgotten)

| Requirement | GGID Feature |
|-------------|-------------|
| Delete all personal data | `DELETE /api/v1/admin/users/{id}?hard=true` |
| Anonymize audit logs (retain for compliance) | Configurable: replace PII with hashed ID |
| Revoke all tokens and sessions | Automatic on deletion |
| Propagate deletion to downstream systems | `user.deleted` webhook event |

```bash
# Hard delete with PII anonymization in audit logs
curl -X DELETE "https://iam.example.com/api/v1/admin/users/{user_id}?hard=true&anonymize=true" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Article 20: Right to Data Portability

| Requirement | GGID Feature |
|-------------|-------------|
| Structured, machine-readable format | JSON export |
| Common format | SCIM 2.0 compatible output |
| Direct transmission to another controller | Bulk export via SCIM `/Bulk` endpoint |

### Article 25: Data Protection by Design

| Requirement | GGID Feature |
|-------------|-------------|
| Minimize data collection | Configurable required fields per tenant |
| Encrypt data at rest | PostgreSQL TDE / disk encryption |
| Encrypt data in transit | TLS 1.3 mandatory |
| PII fields encrypted at column level | `pkg/pii` — AES-256-GCM column encryption |
| Row-Level Security per tenant | PostgreSQL RLS policies |

### Article 28: Processor Agreements

GGID processes data on behalf of the controller (your organization). Key
considerations:

| Requirement | GGID Feature |
|-------------|-------------|
| Sub-processor disclosure | Open source — no hidden sub-processors |
| Data residency | Configurable database region |
| Breach notification | Security event webhooks + audit alerts |
| Deletion on termination | `DELETE /api/v1/admin/tenant` (full tenant wipe) |

### GDPR Data Processing Register

```sql
CREATE TABLE gdpr_consent_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    purpose         VARCHAR(128) NOT NULL,    -- "marketing", "analytics", etc.
    consent_given   BOOLEAN NOT NULL,
    consent_text    TEXT NOT NULL,            -- Exact text shown to user
    consent_version VARCHAR(32) NOT NULL,     -- Policy version
    given_at        TIMESTAMPTZ NOT NULL,
    withdrawn_at    TIMESTAMPTZ,
    ip_address      INET,
    user_agent      TEXT
);
```

---

## SOC 2 (Service Organization Control 2)

SOC 2 audits assess controls across five Trust Service Principles: Security,
Availability, Processing Integrity, Confidentiality, and Privacy.

### CC1: Control Environment

| Control | GGID Feature |
|---------|-------------|
| Role-based access control | Policy Service — RBAC + ABAC engine |
| Segregation of duties | Per-role permissions, admin vs security_admin split |
| Management review | Audit log: `admin.config.changed`, `admin.impersonation.*` |

### CC2: Communication and Information

| Control | GGID Feature |
|---------|-------------|
| System alerts | Webhook events for security incidents |
| External communication | `user.deleted`, `user.login.failed` webhooks |
| Incident logging | All admin actions audit-logged |

### CC3: Risk Assessment

| Control | GGID Feature |
|---------|-------------|
| Vulnerability scanning | Dependency scanning in CI (`go vet`, `golangci-lint`) |
| Penetration test support | Read-only audit access, SCIM export for testing |
| Change management | Git-based workflow, PR review required, CI gates |

### CC4: Monitoring Activities

| Control | GGID Feature |
|---------|-------------|
| Continuous monitoring | Prometheus metrics for all services |
| Anomaly detection | Failed login rate alerts, token reuse detection |
| Audit log integrity | Hash chain verification (tamper-proof) |

### CC5: Control Activities

| Control | GGID Feature |
|---------|-------------|
| Access control enforcement | JWT validation on every request |
| Authentication | Multi-factor: TOTP, WebAuthn, SMS |
| Authorization | Per-request RBAC/ABAC evaluation |
| Data encryption | TLS 1.3 in transit, AES-256 at rest |

### CC6: Logical and Physical Access

| Control | GGID Feature |
|---------|-------------|
| Unique user IDs | Enforced unique username + email per tenant |
| Authentication | Password + MFA enforcement at tenant level |
| Access reviews | `GET /api/v1/admin/users/{id}/roles` — exportable |
| Session management | Configurable timeout, concurrent session limits |
| Privileged access | `super_admin`, `admin`, `security_admin` roles |
| Access revocation | `DELETE /api/v1/admin/users/{id}/sessions` — instant |

### CC7: System Operations

| Control | GGID Feature |
|---------|-------------|
| Change management | PR review + CI pipeline + deployment records |
| Configuration management | YAML configs + env vars, version controlled |
| Software vulnerabilities | `go mod audit` + Dependabot |
| Incident response | Audit trail + webhook alerts + session revocation |

### CC8: Change Management

| Control | GGID Feature |
|---------|-------------|
| Authorize changes | Branch protection: 2 reviewers for `pkg/` and `proto/` |
| Test changes | CI pipeline: lint → test → build → integration |
| Approve changes | PR merge requires all CI checks passing |
| Deploy changes | Docker image tagging + rolling deployment |

### CC9: Risk Mitigation

| Control | GGID Feature |
|---------|-------------|
| Business continuity | HA deployment: multi-AZ, auto-failover |
| Backup and recovery | PostgreSQL streaming replication + WAL archiving |
| Incident detection | Real-time security event webhooks |

### Availability

| Control | GGID Feature |
|---------|-------------|
| Capacity monitoring | Prometheus: CPU, memory, connection pool metrics |
| Performance monitoring | P99 latency tracking per endpoint |
| Disaster recovery | RPO < 1 min (streaming replication), RTO < 5 min |
| Backup testing | WAL archive + point-in-time recovery |

---

## HIPAA (Health Insurance Portability and Accountability Act)

HIPAA applies to Protected Health Information (PHI). GGID can be configured
for HIPAA-compliant deployments.

### Administrative Safeguards (45 CFR §164.308)

| Requirement | GGID Feature |
|-------------|-------------|
| Access management (§164.308(a)(4)) | RBAC + ABAC + per-tenant isolation |
| Access review (§164.308(a)(3)(ii)(C)) | Role audit: `GET /api/v1/admin/users/{id}/roles` |
| Access establishment (§164.308(a)(4)(ii)(C)) | JIT provisioning via LDAP/SCIM |
| Access modification (§164.308(a)(4)(ii)(C)) | Role assignment/revocation API |
| Termination (§164.308(a)(3)(ii)(C)) | `DELETE /api/v1/admin/users/{id}/sessions` + deactivate |
| Audit controls (§164.308(a)(1)(ii)(D)) | Comprehensive audit logging + hash chain |
| Integrity controls (§164.308(a)(1)(ii)(C)) | PII encryption (AES-256-GCM) |

### Physical Safeguards (45 CFR §164.310)

| Requirement | GGID Mitigation |
|-------------|----------------|
| Facility access | Not applicable (software) — infrastructure provider responsibility |
| Workstation security | WebAuthn hardware key enforcement (FIPS 140-2 Level 2) |
| Device controls | `device_type` tracking + device-bound credentials |

### Technical Safeguards (45 CFR §164.312)

| Requirement | GGID Feature |
|-------------|-------------|
| Access control (§164.312(a)) | JWT + RBAC/ABAC + tenant isolation |
| Audit controls (§164.312(b)) | Immutable audit log with hash chain verification |
| Integrity (§164.312(c)) | PII field encryption + HMAC signatures on webhooks |
| Person authentication (§164.312(d)) | Password + MFA (TOTP/WebAuthn) + step-up auth |
| Transmission security (§164.312(e)) | TLS 1.3 mandatory, mTLS for service-to-service |

### PHI Access Logging

HIPAA requires logging all access to PHI. Configure GGID to audit every
read of user data:

```yaml
audit:
  phi_mode: true
  log_phi_access: true        # Log every user data read
  include_fields:
    - actor_id
    - action                  # "read", "write", "delete"
    - resource_type           # "user", "session", "role"
    - resource_id
    - source_ip
    - purpose                 # "treatment", "payment", "operations"
  retention:
    minimum_years: 6          # HIPAA requires 6-year retention
```

### BAA (Business Associate Agreement)

When deploying GGID in a HIPAA-regulated environment:

| Requirement | Status |
|-------------|--------|
| Sign BAA with hosting provider | AWS / GCP / Azure offer BAA |
| Sign BAA with GGID support provider | Required if using managed GGID |
| Sub-contractor disclosure | All infrastructure providers must be BAA-covered |
| Breach notification | 60 days (HHS Breach Notification Rule) |

---

## ISO 27001 (Information Security Management)

ISO 27001 defines an Information Security Management System (ISMS). The
following maps GGID features to Annex A controls.

### A.5: Information Security Policies

| Control | GGID Feature |
|---------|-------------|
| A.5.1 Policies for information security | Tenant-level security policy configuration |
| A.5.2 Information security roles | Admin roles: super_admin, admin, security_admin |

### A.6: Organization of Information Security

| Control | GGID Feature |
|---------|-------------|
| A.6.1 Internal organization | Multi-tenant isolation, per-tenant admin |
| A.6.2 Mobile devices and teleworking | WebAuthn device management, session policies |
| A.6.3 Information security incident management | Security event webhooks + audit alerts |

### A.7: Human Resource Security

| Control | GGID Feature |
|---------|-------------|
| A.7.1 Prior to employment | Background checks (organizational, not software) |
| A.7.2 During employment | Role-based access, annual access review |
| A.7.3 Termination | Instant session revocation + account deactivation |

### A.8: Asset Management

| Control | GGID Feature |
|---------|-------------|
| A.8.1 Responsibility for assets | Per-tenant resource ownership |
| A.8.2 Information classification | PII tagging via `pkg/pii` |
| A.8.3 Media handling | Disk encryption, secure deletion |

### A.9: Access Control

| Control | GGID Feature |
|---------|-------------|
| A.9.1 Access control policy | RBAC + ABAC policy engine |
| A.9.2 User access management | JIT provisioning, role assignment API |
| A.9.3 User responsibilities | MFA enforcement, password policy |
| A.9.4 System and application access | Unique user IDs, session management, password policy |

```yaml
# ISO 27001 access control configuration
auth:
  password_policy:
    min_length: 12
    require_uppercase: true
    require_lowercase: true
    require_digit: true
    require_special: true
    max_age_days: 90
    history_count: 12

  mfa:
    required: true
    allowed_methods: ["totp", "webauthn"]

  session:
    max_duration: "8h"
    idle_timeout: "30m"
    max_concurrent: 2
```

### A.10: Cryptography

| Control | GGID Feature |
|---------|-------------|
| A.10.1 Cryptographic controls | TLS 1.3, AES-256-GCM, RSA-2048+ for JWT |
| Key management | Key rotation procedure, JWKS endpoint |
| Key strength | RSA 2048+ minimum, ECDSA P-256+ |

### A.12: Operations Security

| Control | GGID Feature |
|---------|-------------|
| A.12.1 Operational procedures | Rolling deployments, health checks |
| A.12.2 Protection from malware | Input validation, SQL injection prevention |
| A.12.3 Information backup | PostgreSQL replication + WAL archiving |
| A.12.4 Logging and monitoring | Comprehensive audit logging + Prometheus metrics |
| A.12.5 Control of operational software | CI/CD pipeline, immutable images |
| A.12.6 Technical vulnerability management | `go mod audit`, Dependabot, golangci-lint |

### A.13: Communications Security

| Control | GGID Feature |
|---------|-------------|
| A.13.1 Network security management | TLS 1.3, mTLS for internal services |
| A.13.2 Information transfer | HMAC-signed webhooks, encrypted sessions |

### A.14: System Acquisition, Development and Maintenance

| Control | GGID Feature |
|---------|-------------|
| A.14.1 Security requirements analysis | Threat model documented |
| A.14.2 Security in development and support | CI pipeline, code review, dependency scanning |
| A.14.3 Test data | Anonymized test data, no PII in tests |

### A.16: Information Security Incident Management

| Control | GGID Feature |
|---------|-------------|
| A.16.1 Management of incidents | Security event webhooks + alerts |
| A.16.2 Reporting events | `user.login.failed` rate alerting |
| A.16.3 Reporting weaknesses | Vulnerability disclosure policy |
| A.16.4 Response | Session revocation, account lockout, admin notification |

---

## Cross-Framework Feature Matrix

| GGID Feature | GDPR | SOC 2 | HIPAA | ISO 27001 |
|-------------|:----:|:-----:|:-----:|:---------:|
| Audit logging | Art. 30 | CC4, CC7 | §164.312(b) | A.12.4 |
| Hash chain verification | Art. 5(2) | CC4 | §164.312(b) | A.12.4 |
| RBAC + ABAC | Art. 25 | CC5, CC6 | §164.312(a) | A.9.1 |
| MFA enforcement | Art. 25 | CC5 | §164.312(d) | A.9.3 |
| PII encryption | Art. 25, 32 | CC5 | §164.312(a) | A.10.1 |
| Row-Level Security | Art. 25 | CC5 | §164.312(a) | A.9.4 |
| Data export (portability) | Art. 15, 20 | — | — | — |
| Right to erasure | Art. 17 | — | — | — |
| Consent management | Art. 7 | — | — | — |
| Session management | Art. 32 | CC6 | §164.312(a) | A.9.4 |
| TLS 1.3 encryption | Art. 32 | CC5 | §164.312(e) | A.13.1 |
| Key rotation | Art. 32 | CC5 | §164.312(a) | A.10.1 |
| Webhook events | Art. 33 | CC2 | §164.314 | A.16.1 |
| Password policy | Art. 32 | CC5 | §164.312(d) | A.9.3 |
| Rate limiting | Art. 32 | CC5 | §164.312(a) | A.12.1 |
| Immutable audit | Art. 5(2) | CC4 | §164.312(b) | A.12.4 |
