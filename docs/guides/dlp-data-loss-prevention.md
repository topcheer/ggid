# DLP Data Loss Prevention

Data classification, DLP policies (in-transit/at-rest/in-use), pattern matching, endpoint/cloud DLP, data discovery, incident response for exfiltration, and GGID implementation.

## Data Classification

| Level | Label | Examples | Access |
|-------|-------|---------|--------|
| Public | L1 | Press releases, docs | Anyone |
| Internal | L2 | Department, org chart | Authenticated |
| Confidential | L3 | Email, phone, salary | Scoped access |
| Restricted | L4 | Passwords, keys, MFA secrets | Internal only (never exposed) |

## DLP Policies

### In-Transit (Network)

```yaml
dlp_in_transit:
  - rule: block_pii_to_external
    condition: |
      request.destination NOT IN internal_domains AND
      response.contains(email_pattern OR ssn_pattern OR card_pattern)
    action: block + log + alert

  - rule: mask_pii_in_api_response
    condition: response.contains(email_pattern)
    action: mask  # j***@corp.com
```

### At-Rest (Storage)

```yaml
dlp_at_rest:
  - rule: encrypt_confidential
    condition: data.classification >= L3
    action: aes256_gcm_encrypt

  - rule: no_plaintext_pii
    condition: column IN (email, phone, ssn, card_number)
    action: encrypt_or_hash

  - rule: audit_log_masking
    condition: field IN (email, phone, ip)
    action: mask_in_logs
```

### In-Use (Processing)

```yaml
dlp_in_use:
  - rule: no_bulk_export_without_approval
    condition: request.select_count > 100 AND data.classification >= L3
    action: require_approval + limit_to_100

  - rule: screen_share_warning
    condition: data.classification == L4
    action: warn + watermark
```

## Pattern Matching

| Pattern | Regex | Confidence |
|---------|-------|-----------|
| Email | `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}` | 99% |
| SSN (US) | `\d{3}-\d{2}-\d{4}` | 90% |
| Credit card | Luhn-validated 13-19 digits | 98% |
| API key | `(ak|sk|pk)-[a-zA-Z0-9]{32,}` | 99% |
| JWT | `eyJ[A-Za-z0-9_-]+\.eyJ` | 99% |
| Phone (E.164) | `\+\d{1,3}\d{4,14}` | 85% |
| IBAN | `[A-Z]{2}\d{2}[A-Z0-9]{11,30}` | 95% |

## Data Discovery & Inventory

```bash
# Scan all tables for PII columns
POST /api/v1/admin/dlp/discover
# → {
#   "tables": {
#     "users": {"pii_columns": ["email", "phone", "display_name"], "classification": "L3"},
#     "audit_events": {"pii_columns": ["actor_ip"], "classification": "L2"},
#     "oauth_clients": {"pii_columns": [], "classification": "L2"}
#   },
#   "unencrypted_pii": 0,
#   "total_pii_columns": 12
# }
```

## Incident Response for Data Exfiltration

```bash
# Detect potential exfiltration
GET /api/v1/audit/events?q=action=="data.export" && result=="success" && count>1000
# → Flag for review

# Containment
POST /api/v1/admin/security/block-ip {"ip": "suspect-ip"}
DELETE /api/v1/admin/users/{user_id}/sessions
PATCH /api/v1/admin/users/{user_id} {"status": "locked"}

# Evidence
POST /api/v1/audit/export {"query": "ip == suspect-ip", "include_chain": true}
```

## GGID DLP Implementation

| Layer | Implementation |
|-------|---------------|
| API gateway | PII pattern scan in responses, mask before egress |
| Audit logs | Auto-mask email/phone/IP in all log entries |
| Database | pgcrypto encryption for L3/L4 columns |
| Redis | No PII stored (session IDs only) |
| SIEM forward | Masked before sending to Splunk/ELK |
| Export | Bulk export requires admin approval + rate limited |

## Monitoring

| Metric | Alert |
|--------|-------|
| PII in logs | Any → masking failure |
| Blocked egress | Spike → misconfigured client or exfiltration |
| Unencrypted PII columns | Any → encrypt immediately |
| Bulk export without approval | Any → block + alert |

## See Also

- [Data Loss Prevention](data-loss-prevention.md)
- [Data Classification Implementation](data-classification-implementation.md)
- [Privacy by Design](privacy-by-design.md)
- [Audit Log Architecture](audit-log-architecture.md)
