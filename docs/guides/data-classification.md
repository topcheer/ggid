# Data Classification Policy Guide

This guide covers implementing data classification in GGID — levels, PII inventory, handling rules, masking, retention, and cross-border restrictions.

## Classification Levels

| Level | Examples | Encryption | Access | Retention |
|-------|---------|-----------|--------|----------|
| **Public** | Org name, public keys, docs | TLS in transit | Anyone | Indefinite |
| **Internal** | Role names, policy rules, configs | TLS + at-rest | Authenticated users | 2 years |
| **Confidential** | User email, audit logs, phone | TLS + at-rest + RLS | Authorized (RBAC) | 1-7 years |
| **Restricted** | Passwords, MFA secrets, keys | TLS + at-rest + encryption | Service only (no human access) | Until account deletion |

## PII Inventory

### PII Fields in GGID

| Field | Classification | Storage | Masked in Audit |
|-------|---------------|---------|----------------|
| `email` | Confidential | PostgreSQL (RLS) | Yes (`a***@e***.com`) |
| `phone` | Confidential | PostgreSQL (RLS) | Yes (`+1234567***`) |
| `username` | Internal | PostgreSQL (RLS) | No |
| `password_hash` | Restricted | PostgreSQL (Argon2id) | Never stored in audit |
| `mfa_secret` | Restricted | PostgreSQL (encrypted) | Never |
| `ip_address` | Confidential | Audit log | Partial (last octet) |
| `user_agent` | Internal | Audit log | No |

## Handling Rules

### Public
- No restrictions on distribution
- No encryption required beyond TLS
- Can be cached by CDN

### Internal
- Available to all authenticated users in the tenant
- Encrypted at rest (database-level)
- Not shared across tenants (RLS enforced)

### Confidential
- Requires RBAC permission (`resource:read`)
- PII obfuscation in audit logs
- Cannot be exported without admin approval
- Encrypted at rest + RLS

### Restricted
- No human access (service-to-service only)
- Encrypted with AES-256-GCM
- Never appears in logs, audit, or API responses
- Rotated on schedule

## Data Masking

### PII Obfuscation in Audit

```go
// pkg/pii/obfuscate.go
func Obfuscate(data map[string]interface{}) map[string]interface{} {
    result := make(map[string]interface{})
    for k, v := range data {
        switch k {
        case "email":
            result[k] = maskEmail(v.(string))  // alice@example.com → a***@e***.com
        case "phone":
            result[k] = maskPhone(v.(string))  // +1234567890 → +1234567***
        case "ssn", "national_id":
            result[k] = "***REDACTED***"
        default:
            result[k] = v
        }
    }
    return result
}
```
### API Response Masking

For users without `pii:read` scope, PII fields are masked:

```json
{
  "email": "a***@e***.com",
  "phone": "+1234567***"
}
```

For users with `pii:read` scope:

```json
{
  "email": "alice@example.com",
  "phone": "+1234567890"
}
```

## Retention by Classification

| Level | Default | Configurable | Legal Hold |
|-------|---------|-------------|------------|
| Public | Indefinite | No | N/A |
| Internal | 2 years | Yes (1-5y) | Freezes deletion |
| Confidential (audit) | 7 years | Yes (1-10y) | Freezes deletion |
| Restricted | Account deletion | No | N/A |

## Cross-Border Restrictions

See [Data Sovereignty](../research/data-sovereignty.md) for full details.

| Classification | EU→US | CN→any | Within EU |
|---------------|-------|-------|----------|
| Public | Allowed | Allowed | Allowed |
| Internal | SCC required | Blocked | Allowed |
| Confidential | SCC + TIA | Blocked | Allowed |
| Restricted | Blocked | Blocked | Same-country only |

## Compliance Mapping

| Standard | Classification Requirement |
|----------|--------------------------|
| GDPR Art. 5(1)(f) | Integrity & confidentiality |
| HIPAA 164.312(a)(2)(iv) | Encryption & decryption |
| SOC 2 CC6.1 | Data classification & handling |
| ISO 27001 A.8.2 | Information classification |
| PIPL Art. 28-32 | Sensitive PI handling |

## Implementation Checklist

- [ ] Data classification policy documented
- [ ] PII inventory maintained
- [ ] Masking applied in audit logs
- [ ] RBAC enforces classification levels
- [ ] Retention policies configured
- [ ] Cross-border restrictions enforced
- [ ] Legal hold mechanism documented
- [ ] Annual classification review

## See Also

- [Data Retention Policy](data-retention-policy.md)
- [GDPR Compliance](../research/gdpr-compliance.md)
- [Data Sovereignty](../research/data-sovereignty.md)
- [Compliance Guide](compliance-guide.md)
