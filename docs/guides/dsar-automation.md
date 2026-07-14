# DSAR Automation

GDPR/CCPA data subject request types, intake workflow, identity verification, data discovery, automated retrieval, review & redaction, delivery format, retention enforcement, and deadline tracking.

## Request Types

| Right | Law | API | Action |
|-------|-----|-----|--------|
| Access | GDPR Art. 15 | GET /users/{id}?expand=all | Export all data |
| Deletion | GDPR Art. 17 | DELETE /users/{id} | Anonymize + purge |
| Portability | GDPR Art. 20 | GET /users/{id}/export | Machine-readable export |
| Rectification | GDPR Art. 16 | PATCH /users/{id} | Correct data |
| Restriction | GDPR Art. 18 | PATCH status=restricted | Freeze processing |
| Object | GDPR Art. 21 | POST /consent/object | Stop processing |
| Know | CCPA | GET /users/{id}/categories | What data we hold |
| Delete | CCPA | DELETE /users/{id} | Delete personal data |

## Intake Workflow

```
1. User submits request via portal or email
2. GGID creates DSAR ticket with 30-day deadline
3. Identity verification (re-auth + MFA)
4. Automated data discovery across all services
5. Data retrieval + redaction
6. Review by DPO (for sensitive cases)
7. Delivery to user (secure download or email)
8. Audit trail + compliance evidence
```

## Identity Verification

```bash
POST /api/v1/dsar/submit
{
  "user_id": "uuid",
  "request_type": "access",
  "verification": {
    "password": "...",
    "mfa_code": "123456"
  }
}
# → Identity must be re-verified before processing DSAR
```

## Data Discovery

```bash
POST /api/v1/admin/dsar/discover
{"user_id": "uuid"}
# → Scans: users, audit_events, sessions, oauth_grants,
#          webauthn_credentials, groups, consents, api_keys
# → Returns table → row count → PII fields mapping
```

## Automated Retrieval

```go
func executeDSAR(userID string, requestType string) (*DSARResult, error) {
    data := map[string]interface{}{}

    // Collect from all services
    data["user"] = userService.Get(userID)
    data["audit_events"] = auditService.QueryByUser(userID)
    data["sessions"] = sessionService.ListForUser(userID)
    data["oauth_grants"] = oauthService.ListGrants(userID)
    data["mfa_factors"] = mfaService.ListFactors(userID)
    data["consents"] = consentService.ListForUser(userID)

    // Redact third-party PII
    redactThirdPartyPII(data)

    // Generate export
    export := packageExport(data, requestType)

    // Audit
    audit.Log("dsar.completed", map[string]interface{}{
        "user_id": userID,
        "type": requestType,
        "records_collected": countRecords(data),
    })

    return export, nil
}
```

## Delivery Format

| Format | Use Case |
|--------|---------|
| JSON | Machine-readable (portability) |
| CSV | Spreadsheet (access) |
| PDF | Human-readable (access) |
| ZIP | Multiple files (large exports) |

Delivery via signed download URL (TTL 24h) or encrypted email.

## Retention Enforcement

| Data Type | Retention | After DSAR Deletion |
|-----------|-----------|-------------------|
| User PII | Deleted immediately | Anonymized |
| Audit logs | 7 years | actor_user_id → hashed |
| Backups | 30 days | Overwritten naturally |
| Redis cache | Immediate | Deleted |
| SIEM forwarded | Per SIEM policy | Masked on next forward |

## Deadline Tracking

```go
func checkDSARDeadlines() {
    for _, req := range dsarStore.GetOpen() {
        daysLeft := 30 - daysSince(req.CreatedAt)

        switch {
        case daysLeft <= 0:
            alert.Send("dsar_breach", req) // Escalate to DPO
        case daysLeft <= 7:
            alert.Send("dsar_warning", req)
        case daysLeft <= 14:
            notify(req.Assignee, "DSAR due in 14 days")
        }
    }
}
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| DSAR completion within SLA | 100% | Any breach → DPO |
| Average completion time | <5 days | >20 days → bottleneck |
| Identity verification failures | <2% | High → user friction |

## See Also

- [Data Subject Rights Automation](data-subject-rights-automation.md)
- [Privacy by Design](privacy-by-design.md)
- [Consent Management Design](consent-management-design.md)
- [Compliance Framework Mapping](compliance-framework-mapping.md)
