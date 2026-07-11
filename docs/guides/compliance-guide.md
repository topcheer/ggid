# Compliance Guide

This guide covers SOC 2, HIPAA, and GDPR compliance checklists, report generation, and data residency configuration in GGID.

## Overview

GGID provides the building blocks for regulatory compliance. This guide maps each framework's requirements to GGID features and provides actionable checklists.

## SOC 2 Type II

### Trust Service Criteria

#### CC6: Logical and Physical Access

| Control | GGID Feature | Status |
|---------|-------------|--------|
| CC6.1: User identification | JWT with `sub` claim | Done |
| CC6.2: User authentication | Password + MFA (TOTP, WebAuthn) | Done |
| CC6.3: Access authorization | RBAC + ABAC policy engine | Done |
| CC6.4: Access restriction | Per-request scope enforcement | Done |
| CC6.5: Access removal | User delete/lock + session revoke | Done |
| CC6.6: Physical access | Self-hosted (your responsibility) | N/A |
| CC6.7: System component inventory | Docker images + K8s manifests | Done |
| CC6.8: Access points | Gateway as single entry point | Done |

#### CC7: System Operations

| Control | GGID Feature | Status |
|---------|-------------|--------|
| CC7.1: Infrastructure management | Docker Compose / K8s | Done |
| CC7.2: Incident detection | Audit events + alert rules | Done |
| CC7.3: Infrastructure monitoring | Health endpoints + metrics | Done |
| CC7.4: Anomalies | Rate limiting + circuit breaker | Done |

#### CC8: Change Management

| Control | GGID Feature | Status |
|---------|-------------|--------|
| CC8.1: Authorize changes | Audit log for all config changes | Done |

### SOC 2 Report Generation

```bash
curl https://api.ggid.example.com/api/v1/audit/compliance/report?type=soc2 \
  -G --data-urlencode "start_date=2025-01-01" \
      --data-urlencode "end_date=2025-03-31" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

## HIPAA Security Rule

### Administrative Safeguards

| Standard | GGID Feature | Status |
|----------|-------------|--------|
| 164.308(a)(3): Workforce access | RBAC + access reviews | Done |
| 164.308(a)(1)(ii)(C): Sanction policy | Audit trail of all actions | Done |
| 164.308(a)(5): Security awareness | MFA enforcement | Done |

### Technical Safeguards

| Standard | GGID Feature | Status |
|----------|-------------|--------|
| 164.312(a)(1): Access control | RBAC + ABAC + RLS | Done |
| 164.312(b): Audit controls | Audit service + hash chain | Done |
| 164.312(c)(1): Integrity | Audit hash chain verification | Done |
| 164.312(d): Person/entity auth | JWT + MFA + WebAuthn | Done |
| 164.312(e)(1): Transmission security | TLS + gRPC TLS | Partial |
| 164.312(a)(2)(iv): Encryption/decryption | Argon2id + AES-256-GCM | Done |

### HIPAA Report

```bash
curl https://api.ggid.example.com/api/v1/audit/compliance/report?type=hipaa \
  -G --data-urlencode "start_date=2025-01-01" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

## GDPR

### Data Subject Rights

| Right (Article) | GGID Support | API Endpoint |
|-----------------|-------------|-------------|
| Access (Art 15) | User profile + audit history | `GET /users/{id}` |
| Rectification (Art 16) | Update user | `PUT /users/{id}` |
| Erasure (Art 17) | GDPR erase (anonymize PII) | `DELETE /users/{id}/gdpr-erase` |
| Restriction (Art 18) | Lock user | `POST /users/{id}/lock` |
| Portability (Art 20) | Export user data | `GET /users/{id}/export` |
| Objection (Art 21) | Revoke consents | `DELETE /oauth/consent` |

### Data Processing Records (Art 30)

GGID audit log serves as the record of processing activities:

```bash
curl https://api.ggid.example.com/api/v1/audit/compliance/report?type=gdpr \
  -G --data-urlencode "start_date=2025-01-01" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Breach Notification (Art 33)

GGID's alert system supports breach detection:

- Configure alert rules for suspicious access patterns
- SIEM forwarding to external incident response systems
- Audit hash chain for forensic evidence

## ISO 27001

| Annex A Control | GGID Feature |
|----------------|-------------|
| A.9: Access control | RBAC + ABAC + MFA |
| A.10: Cryptography | Argon2id + TLS + AES-256-GCM |
| A.12: Operations security | Audit + health monitoring |
| A.16: Incident management | Alert rules + SIEM forward |
| A.18: Compliance | Compliance reports |

## Compliance Checklist Template

### Pre-Audit

- [ ] Access review conducted in last 90 days
- [ ] All admin accounts have MFA enrolled
- [ ] Audit log retention >= 12 months
- [ ] Audit hash chain verified (no tampering)
- [ ] Password policy enforced
- [ ] Account lockout configured
- [ ] TLS 1.2+ enforced
- [ ] No plaintext PII in logs (PII obfuscation verified)
- [ ] Data residency requirements documented
- [ ] Incident response plan documented

### Evidence Collection

```bash
# Generate quarterly compliance report
curl https://api.ggid.example.com/api/v1/audit/compliance/report \
  -G --data-urlencode "type=soc2" \
      --data-urlencode "start_date=2025-01-01" \
      --data-urlencode "end_date=2025-03-31" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -o soc2_q1_2025.json

# Export audit log for auditor
https://api.ggid.example.com/api/v1/audit/export?format=csv&start_date=2025-01-01 \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Verify audit integrity
https://api.ggid.example.com/api/v1/audit/integrity/verify \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Data Residency

For multi-region compliance (EU, China, etc.), see [Data Sovereignty](../research/data-sovereignty.md).

## See Also

- [GDPR Compliance Research](../research/gdpr-compliance.md)
- [Data Sovereignty](../research/data-sovereignty.md)
- [Audit & SIEM Guide](audit-siem-guide.md)
- [Security Audit Checklist](security-audit-checklist.md)
- [Access Reviews](access-reviews.md)
