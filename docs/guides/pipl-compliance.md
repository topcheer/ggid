# PIPL Compliance Guide

中国个人信息保护法（PIPL）合规指南 — 数据处理、跨境传输、同意管理、DPO工作流。

## Overview

The Personal Information Protection Law (PIPL) is China's comprehensive data protection law, effective November 1, 2021. This guide maps PIPL requirements to GGID features.

## Key Requirements

| PIPL Article | Requirement | GGID Support |
|-------------|-------------|-------------|
| Art. 13-16 | Lawful basis for processing | Tenant config |
| Art. 17-19 | Informed consent | OAuth consent store |
| Art. 24-27 | Data minimization | Per-field user data API |
| Art. 38-43 | Cross-border transfer restrictions | Data residency (per-region deploy) |
| Art. 44-50 | Data subject rights | Profile self-service + export |
| Art. 51-59 | Security obligations | TLS + RLS + audit hash chain |
| Art. 52-53 | DPO appointment | Audit trail for DPO review |
| Art. 55-56 | PIA (Privacy Impact Assessment) | Compliance reports |

## Data Handling

### Data Localization (Art. 38)

Personal information of Chinese nationals must be stored on servers in mainland China:

```yaml
# Deploy separate GGID instance in China region
tenants:
  - id: "cn-tenant-uuid"
    region: "cn-beijing"
    database: "postgres-cn.internal.cn"
    redis: "redis-cn.internal.cn"
    nats: "nats-cn.internal.cn"
```

**No cross-border replication** of personal data. Only anonymized analytics may be exported.

### Sensitive Personal Information (Art. 28-32)

PIPL defines "sensitive personal information" as data that could harm dignity or safety if leaked:

| Category | Examples | GGID Handling |
|----------|----------|--------------|
| Identity | ID card number, passport | PII obfuscation in audit |
| Biometric | Face, fingerprint | WebAuthn (biometrics stay on device) |
| Financial | Bank account | Not collected by GGID |
| Health | Medical records | Not collected by GGID |
| Minor | Under 14 data | Special consent required |

For sensitive data, GGID:
- Stores hashes only (Argon2id), never plaintext
- Obfuscates PII in audit logs
- Encrypts at rest (AES-256-GCM)

## Consent Management (Art. 17-19)

### Separate Consent

PIPL requires **separate consent** for sensitive information (unlike GDPR's bundled consent):

```json
{
  "consent_records": [
    {
      "purpose": "user_authentication",
      "data_categories": ["username", "password_hash"],
      "lawful_basis": "contract",
      "consent_required": false
    },
    {
      "purpose": "security_monitoring",
      "data_categories": ["ip_address", "user_agent", "login_time"],
      "lawful_basis": "legitimate_interest",
      "consent_required": false
    },
    {
      "purpose": "behavioral_analytics",
      "data_categories": ["page_views", "clicks"],
      "lawful_basis": "consent",
      "consent_required": true,
      "withdrawal_url": "/privacy/consent"
    }
  ]
}
```

### Consent Withdrawal (Art. 16)

Users can withdraw consent at any time:

```bash
curl -X DELETE https://api.ggid.example.com/api/v1/consent/behavioral_analytics \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

## Cross-Border Transfer (Art. 38-43)

### Requirements

Before transferring personal data outside China, one of:
1. **CAC security assessment** (for critical infrastructure)
2. **PI Protection Certification** (for certified organizations)
3. **Standard Contract** (CAC standard contract, registered)

### GGID Configuration

```yaml
# Block all cross-border data transfers
data_residency:
  region: "cn"
  cross_border_transfer: false
  anonymize_for_analytics: true
```

## Data Subject Rights

| Right (Article) | Implementation | API |
|-----------------|---------------|-----|
| Access (Art. 45) | Export user data | `GET /users/me/export` |
| Copy (Art. 45) | Machine-readable export | `GET /users/me/export?format=json` |
| Correction (Art. 46) | Update profile | `PUT /users/me` |
| Deletion (Art. 47) | GDPR erase endpoint | `DELETE /users/me` |
| Withdraw consent (Art. 16) | Revoke consent | `DELETE /consent/{purpose}` |
| Explanation (Art. 48) | Processing disclosure | `/privacy/disclosure` |

## DPO Workflow (Art. 52-53)

### DPO Responsibilities

Organizations processing large-scale personal information must appoint a DPO:

```bash
# DPO can query all processing records
curl https://api.ggid.example.com/api/v1/audit/compliance/report?type=gdpr \
  -G --data-urlencode "start_date=2025-01-01" \
  -H "Authorization: Bearer $DPO_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Privacy Impact Assessment (PIA) — Art. 55-56

Required before:
- Processing sensitive personal information
- Automated decision-making
- Cross-border transfer
- Entrusting processing to third parties

GGID supports PIA by providing:
- Data flow documentation (audit trail)
- Security measure inventory
- Compliance report generation

## PIPL vs GDPR Comparison

| Aspect | PIPL | GDPR |
|--------|------|------|
| Consent model | Separate consent for sensitive | Explicit consent |
| Cross-border | CAC approval required | Adequacy decision |
| Data localization | Required for CN nationals | Recommended but not mandated |
| DPO | Required for large-scale | Required for high-risk |
| Penalties | Up to 5% annual revenue or ¥50M | Up to 4% annual revenue or €20M |
| Children | Under 14 special rules | Under 16 special rules |

## Compliance Checklist

- [ ] Data stored on China-based servers
- [ ] No cross-border transfer of personal data
- [ ] Separate consent for sensitive information
- [ ] Consent withdrawal mechanism tested
- [ ] Data subject rights endpoints functional
- [ ] DPO appointed and has API access
- [ ] PIA conducted for high-risk processing
- [ ] Security measures documented
- [ ] Breach notification process (60 min to authorities)
- [ ] CAC standard contract signed (if transferring)

## See Also

- [GDPR Compliance](../research/gdpr-compliance.md)
- [Data Sovereignty](../research/data-sovereignty.md)
- [Compliance Guide](compliance-guide.md)
- [Data Retention Policy](data-retention-policy.md)
