# Data Classification Implementation

4-level taxonomy, PII scanner configuration, masking rules, DLP policy authoring, cross-border rules, and audit evidence.

## Classification Taxonomy

| Level | Name | Description | Examples |
|-------|------|-------------|---------|
| L1 | Public | Intentionally public, no harm if disclosed | Display name, org name, public docs |
| L2 | Internal | Internal business data, limited disclosure | Department, role, group membership |
| L3 | Confidential | PII, cause harm if disclosed | Email, phone, address, audit logs |
| L4 | Restricted | Highest sensitivity, severe harm if disclosed | Password hash, MFA secret, recovery codes, SSN, encryption keys |

## Field-Level Classification

```yaml
classification:
  users:
    id: L1          # UUID — not guessable
    email: L3       # PII
    display_name: L1
    phone: L3       # PII
    department: L2
    password_hash: L4
    mfa_secret: L4
    status: L2
    created_at: L2
    last_login_at: L3
    
  audit_events:
    event_id: L1
    actor_user_id: L3
    actor_ip: L3
    action: L2
    details: L3     # May contain PII
    
  oauth_clients:
    client_id: L1
    client_secret: L4
    redirect_uris: L1
    scopes: L2
```

## PII Scanner

### Automated Discovery

```bash
# Scan database columns for PII patterns
ggid scan pii --database ggid --format json
# Output:
# {
#   "users.email": {"type": "email", "confidence": 99, "level": "L3"},
#   "users.phone": {"type": "phone", "confidence": 95, "level": "L3"},
#   "users.ssn": {"type": "ssn", "confidence": 100, "level": "L4"},
#   "audit.details": {"type": "mixed", "confidence": 70, "level": "L3"}
# }
```

### Scan Patterns

| PII Type | Detection Pattern |
|----------|------------------|
| Email | RFC 5322 regex |
| Phone | E.164 / NANP patterns |
| SSN | `\d{3}-\d{2}-\d{4}` |
| Credit card | Luhn algorithm |
| IP address | IPv4/IPv6 regex |
| JWT | `eyJ...` base64 prefix |
| API key | `ak-*`, `sk-*` prefixed tokens |

### Column Encryption Scanner

```sql
-- Find L3/L4 columns not encrypted
SELECT table_name, column_name 
FROM information_schema.columns 
WHERE table_schema = 'public'
  AND column_name IN ('email', 'phone', 'ssn', 'password_hash', 'mfa_secret')
  AND table_name NOT IN (
    SELECT table_name FROM pg_encrypted_columns  -- hypothetical view
  );
```

## Masking Rules

### Per-Type Masking

| Data Type | Mask Format | Example |
|-----------|------------|---------|
| Email | First char + domain | `j***@corp.com` |
| Phone | First 6 + mask | `+1-555-01**` |
| IP address | First 3 octets | `10.0.1.*` |
| SSN | Last 4 only | `***-**-1234` |
| Credit card | Last 4 only | `**** **** **** 1234` |
| Full name | First initial | `J. ***` |
| JWT/token | First 8 chars | `eyJhbGci*` |

### Implementation

```go
func MaskField(field string, piiType string) string {
    switch piiType {
    case "email":
        parts := strings.SplitN(field, "@", 2)
        return string(parts[0][0]) + "***@" + parts[1]
    case "phone":
        if len(field) > 6 {
            return field[:6] + strings.Repeat("*", len(field)-6)
        }
        return "****"
    case "ip":
        parts := strings.Split(field, ".")
        if len(parts) == 4 {
            return parts[0] + "." + parts[1] + "." + parts[2] + ".*"
        }
        return "masked"
    }
}
```

### Where Masking Applies

| Context | Masking | Rationale |
|---------|---------|-----------|
| Audit logs | Yes | Prevent PII in log aggregation |
| SIEM events | Yes | Prevent PII in external systems |
| Error messages | Yes | No PII in stack traces |
| Admin UI | Partial (depends on admin scope) | Need-to-know basis |
| API responses | No (full data, encrypted in transit) | Client needs full data |
| Debug dumps | Yes | Prevent accidental PII leak |

## DLP Policy Authoring

### Policy Structure

```yaml
dlp_policies:
  - name: "prevent-ssn-in-logs"
    description: "Block SSN from appearing in any log output"
    match:
      classification: L4
      fields: [ssn]
    action: mask
    destinations: [logs, siem, error_messages]
    severity: critical
    alert: security-team

  - name: "restrict-pii-export"
    description: "Require approval for bulk PII export"
    match:
      classification: [L3, L4]
      row_count: ">100"
    action: require_approval
    approver: "data-protection-officer"
    ttl: 3600

  - name: "block-token-in-url"
    description: "Prevent tokens from appearing in query parameters"
    match:
      pattern: "(access_token|api_key|secret)"
      location: query_params
    action: block
    severity: critical
```

### Enforcement Points

```
Request → Gateway
  ├── DLP: Check URL params for tokens/PII → Block if matched
  ├── DLP: Check request body for PII in wrong context
  ├── Process request
  └── DLP: Check response for PII before logging → Mask if needed
```

## Cross-Border Rules

```yaml
cross_border:
  # EU data stays in EU
  eu_origin:
    storage_region: "eu-west-1"
    replication: "eu-only"
    classification_allowed: [L1, L2, L3, L4]
    cross_border_transfer: "denied"
    
  # US data
  us_origin:
    storage_region: "us-east-1"
    replication: "us-only"
    cross_border_transfer: "approved_if_dpa_exists"
    
  # China data (PIPL)
  cn_origin:
    storage_region: "cn-north-1"
    replication: "cn-only"
    cross_border_transfer: "requires_government_approval"
```

### Transfer Approval

```bash
# Request cross-border data transfer
POST /api/v1/admin/data-transfer/request
{
  "classification": "L3",
  "source_region": "eu-west-1",
  "destination": "us-analytics.corp.com",
  "purpose": "Aggregated analytics (anonymized)",
  "legal_basis": "GDPR Art. 49 - explicit consent"
}
# → Requires DPO approval
```

## Audit Evidence

### Classification Register

```bash
# Export current data classification inventory
GET /api/v1/admin/classification/export
# → CSV with: table, column, classification, encryption_status, masking_rule, last_scanned
```

### DLP Compliance Report

```bash
# Monthly DLP compliance report
GET /api/v1/admin/dlp/report?period=2025-01
# → {
#   "policies_enforced": 12,
#   "violations_blocked": 47,
#   "masking_applied": 125000,
#   "cross_border_denied": 3,
#   "approval_pending": 1
# }
```

### Evidence Retention

| Evidence Type | Retention | Storage |
|--------------|-----------|---------|
| Classification register | 7 years | Audit archive |
| DLP violation logs | 7 years | Audit archive |
| Cross-border approval records | 7 years | Audit archive |
| PII scan results | 2 years | Operational storage |

## See Also

- [Privacy by Design](privacy-by-design.md)
- [Database Security](database-security.md)
- [Compliance Framework Mapping](compliance-framework-mapping.md)
- [Audit Log Architecture](audit-log-architecture.md)
