# Data Loss Prevention (DLP)

PII detection in API payloads, masking rules, redaction in audit logs, egress filtering, tenant-level policies, and exception workflow.

## PII Detection

| PII Type | Pattern | Confidence |
|----------|---------|-----------|
| Email | RFC 5322 | 99% |
| Phone | E.164 / NANP | 95% |
| SSN | `\d{3}-\d{2}-\d{4}` | 90% |
| Credit card | Luhn validated | 98% |
| JWT | `eyJ...` prefix | 99% |
| API key | `ak-`/`sk-` prefix | 99% |

## Masking Rules

| Endpoint Context | Field | Masking |
|-----------------|-------|---------|
| Audit log | email | `j***@corp.com` |
| Audit log | phone | `+1-555-01**` |
| Audit log | ip | `10.0.1.*` |
| SIEM forward | All PII | Masked |
| Error message | Any PII | Redacted |
| API response (full scope) | None | Full data |

## Egress Filtering

```yaml
egress_rules:
  - client: "external-oauth-client"
    block_fields: [ssn, salary, password_hash, mfa_secret]
    allow_fields: [email, display_name]

  - client: "internal-service"
    block_fields: [password_hash, mfa_secret]
    allow_fields: ["*"]

  - tenant: "tenant-restricted"
    block_endpoints: ["/api/v1/admin/*"]
```

## Tenant-Level DLP

```bash
PUT /api/v1/admin/tenants/{tenant_id}/dlp
{
  "pii_fields": ["email", "phone", "address"],
  "mask_in_logs": true,
  "block_external_export": true,
  "require_approval_for_bulk": true
}
```

## Exception Workflow

```bash
# Request DLP exception (e.g., analytics needs unmasked email)
POST /api/v1/admin/dlp/exceptions
{
  "requester": "data-team@corp.com",
  "field": "email",
  "purpose": "deduplication analytics",
  "duration_hours": 24,
  "anonymization": "hash"  # Not raw email
}
# → Requires DPO approval
```

## Monitoring

| Metric | Alert |
|--------|-------|
| PII in logs detected | Any → masking failure |
| Blocked egress attempts | Spike → misconfigured client |
| Bulk export without approval | Any → block + alert |

## See Also

- [Data Classification Implementation](data-classification-implementation.md)
- [Privacy by Design](privacy-by-design.md)
- [Audit Log Architecture](audit-log-architecture.md)
