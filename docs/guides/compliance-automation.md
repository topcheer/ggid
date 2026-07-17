# Compliance Automation — Technical Guide

> Feature: Compliance automation across SOC 2, ISO 27001, NIST CSF, NIS2, CRA
> Console: `/settings/compliance-dashboard`

## What It Does

GGID automates compliance evidence collection and control monitoring across multiple regulatory frameworks. Instead of manual spreadsheets and point-in-time audits, GGID continuously evaluates your control posture and generates real-time compliance reports.

## Compliance Features

### 1. Immutable Audit Log

Every identity event (login, MFA, role change, API call) is recorded in a tamper-evident audit log with hash-chain integrity.

- **Hash chain**: Each event includes a SHA-256 hash of the previous event, creating a verifiable chain.
- **Verification**: `/api/v1/audit/verify-integrity` validates the entire chain.
- **Retention**: Configurable retention policies (7 years for financial compliance).

**Framework mappings:** SOC 2 CC7.2, ISO 27001 A.12.4, NIST AU-2, NIS2 Article 21(1)(a).

### 2. Access Reviews & Certifications

Periodic access reviews ensure users only retain necessary permissions.

- **Campaigns**: Create quarterly/annual access certification campaigns.
- **Reviewers**: Managers or role owners review team access.
- **Actions**: Certify, modify, or revoke access per user.
- **Evidence**: Each campaign produces auditable evidence records.

**Framework mappings:** SOC 2 CC6.3, ISO 27001 A.9.2.5, NIST AC-2(3).

### 3. Data Loss Prevention (DLP)

Real-time PII detection and redaction in API responses.

- **6 PII types**: SSN, credit card, email, phone, API keys, IBAN.
- **5 redaction strategies**: Full mask, partial mask, email mask, tokenize, remove.
- **Audit trail**: Every PII match logged with timestamp and field name.

**Framework mappings:** SOC 2 CC6.7, ISO 27001 A.13.2, NIST SC-8, NIS2 Article 21(2)(b).

### 4. Encryption

Envelope encryption with per-tenant data keys (AES-256-GCM / SM4-GCM).

- **Per-tenant isolation**: Each tenant has unique DEKs.
- **Key rotation**: DEKs rotate automatically; KEK rotation is manual.
- **At-rest encryption**: All sensitive fields encrypted in PostgreSQL.

**Framework mappings:** SOC 2 CC6.1, ISO 27001 A.10.1, NIST SC-28, CRA Article 13.

### 5. ITDR (Identity Threat Detection)

8 MITRE ATT&CK-mapped detection rules for identity-based threats.

- **Detection types**: MFA fatigue, token theft, session hijack, consent phishing, etc.
- **Response**: Automated SOAR playbook execution.
- **Correlation**: External threat intel + internal ITDR detections.

**Framework mappings:** SOC 2 CC7.1, ISO 27001 A.16.1, NIST IR-4, NIS2 Article 21(2)(g).

## Framework Mapping Guide

### SOC 2 (Trust Services Criteria)

| GGID Feature | SOC 2 Criteria | Evidence Source |
|--------------|---------------|-----------------|
| Audit log | CC7.2 | `/api/v1/audit/events` |
| Access reviews | CC6.3 | Access certification campaigns |
| DLP | CC6.7 | DLP redaction logs |
| Encryption | CC6.1 | Encryption config + key inventory |
| ITDR | CC7.1 | ITDR detection records |
| MFA enforcement | CC6.1 | MFA enrollment stats |
| Session management | CC6.1 | Session timeout policy |
| Change management | CC8.1 | Policy version history |

### ISO 27001:2022

| GGID Feature | Control | Evidence Source |
|--------------|---------|-----------------|
| Access control | A.5.15 | RBAC role assignments |
| Authentication | A.5.17 | MFA enrollment stats |
| Audit logging | A.8.15 | Audit event stream |
| Encryption | A.8.24 | Encryption configuration |
| Vulnerability mgmt | A.8.8 | Threat intel matches |
| Incident mgmt | A.5.24 | ITDR + SOAR execution logs |

### NIST CSF 2.0

| GGID Feature | Function/Category | Evidence Source |
|--------------|-------------------|-----------------|
| Audit log | DE.AE-3 | Event analysis |
| Access control | PR.AC-1 | Role/permission matrix |
| Encryption | PR.DS-1 | Encryption status |
| ITDR | DE.CM-1 | Threat detection |
| Incident response | RS.RP-1 | SOAR playbook logs |
| Recovery | RC.RP-1 | Session revocation records |

### NIS2 Directive

| Requirement | GGID Control | Article |
|-------------|-------------|---------|
| Risk management | URE + CAE | Art. 21(1) |
| Incident handling | ITDR + SOAR | Art. 21(2)(b) |
| Access control | RBAC + ABAC + ReBAC | Art. 21(2)(d) |
| Training | MFA enforcement logs | Art. 21(2)(g) |
| Cryptography | AES-256-GCM encryption | Art. 21(2)(h) |

## Evidence Collection Workflow

```
1. Define scope: Select framework (SOC 2, ISO 27001, etc.)
2. Map controls: Associate GGID features to control IDs
3. Continuous monitoring: GGID evaluates controls in real-time
4. Evidence generation: Export reports via compliance dashboard
5. Auditor review: Share reports + provide read-only API access
6. Remediation: Address gaps identified by the dashboard
```

### Using the Compliance Dashboard

1. Navigate to `/settings/compliance-dashboard`.
2. Review framework donut charts — coverage percentages per framework.
3. Click any framework to expand gap details.
4. Export compliance report via `/api/v1/audit/compliance-report`.

## API Endpoints

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# Get compliance dashboard data
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/compliance-dashboard" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Generate compliance report
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/compliance-report?framework=soc2" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Export audit events as evidence
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/export?format=json&from=2026-01-01&to=2026-07-18" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Verify audit log integrity (tamper-evidence proof)
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/verify-integrity" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Dashboard shows 0% coverage | No controls mapped | Run compliance assessment first |
| Integrity verification fails | Manual DB changes or data corruption | Restore from backup; never modify audit tables directly |
| Export missing events | Date range filter or pagination | Adjust from/to params; increase page_size |
| Framework not listed | Not yet configured | Add framework mapping in policy settings |

## Best Practices

- **Automate evidence collection**: Schedule weekly compliance report exports.
- **Address gaps promptly**: Monitor the dashboard for coverage drops.
- **Map custom controls**: Extend framework mappings for industry-specific requirements.
- **Maintain audit integrity**: Never modify audit tables directly — always use the API.
- **Periodic access reviews**: Run campaigns quarterly for SOC 2 evidence.
- **Document remediation**: Keep records of how each gap was addressed for auditors.
