# Data Subject Rights Automation

GDPR DSAR workflow, automated data discovery, export generation, deletion cascade, consent withdrawal, 30-day SLA, and compliance audit trail.

## DSAR Types (GDPR Articles)

| Right | Article | API Endpoint | Action |
|-------|---------|-------------|--------|
| Access | Art. 15 | `GET /users/{id}?expand=all` | Export all user data |
| Rectification | Art. 16 | `PATCH /users/{id}` | Correct inaccurate data |
| Erasure | Art. 17 | `DELETE /users/{id}` | Anonymize + purge |
| Restriction | Art. 18 | `PATCH /users/{id} {"status":"restricted"}` | Freeze processing |
| Portability | Art. 20 | `GET /users/{id}/export` | Machine-readable export |
| Object | Art. 21 | `POST /consent/object` | Stop specific processing |

## Automated Data Discovery

```bash
# Find all data for a user across all tables
POST /api/v1/admin/dsar/discover
{"user_id": "uuid"}
# → {
#   "tables": {
#     "users": {"rows": 1, "fields": ["email","display_name","phone"]},
#     "audit_events": {"rows": 342, "fields": ["actor_user_id","ip"]},
#     "sessions": {"rows": 0, "fields": []},
#     "oauth_grants": {"rows": 3, "fields": ["client_id","scope"]},
#     "webauthn_credentials": {"rows": 2, "fields": ["device_name"]}
#   },
#   "redis_keys": 5,
#   "total_records": 348
# }
```

## Export Generation

```bash
POST /api/v1/admin/dsar/export
{
  "user_id": "uuid",
  "format": "json",
  "include": ["users", "audit_events", "oauth_grants", "sessions"]
}
# → 202 {"export_id": "exp-uuid", "status": "processing"}
# → Async job collects data, packages, signs
```

### Export Format

```json
{
  "user": {"id":"uuid","email":"jane@corp.com","display_name":"Jane Doe"},
  "audit_events": [...],
  "oauth_grants": [...],
  "sessions": [...],
  "consents": [...],
  "mfa_factors": [{"type":"totp","enrolled_at":"..."}],
  "export_metadata": {
    "generated_at": "2025-01-15T10:00:00Z",
    "request_id": "dsar-uuid",
    "signed_by": "GGID"
  }
}
```

## Deletion Cascade

```go
func executeErasure(userID string) error {
    // 1. Revoke sessions + tokens
    revokeAllSessions(userID)
    revokeAllTokens(userID)

    // 2. Remove OAuth grants
    oauthStore.RemoveAllForUser(userID)

    // 3. Remove MFA factors
    mfaStore.RemoveAllForUser(userID)

    // 4. Remove group memberships
    groupStore.RemoveAllForUser(userID)

    // 5. Anonymize user record
    userStore.Update(userID, map[string]interface{}{
        "email":       "anonymized_" + hash(userID),
        "display_name": "Deleted User",
        "phone":        nil,
        "status":       "archived",
    })

    // 6. Retain audit logs (anonymized) for 7 years
    // audit_events: actor_user_id → hashed

    // 7. Delete Redis keys
    redis.Del("user:" + userID + ":*")

    audit.Log("dsar.erasure_completed", map[string]interface{}{
        "user_id": userID,
        "tables_anonymized": 6,
    })

    return nil
}
```

## Consent Withdrawal Handling

```bash
# Withdraw all consent → triggers data purge workflow
DELETE /api/v1/consent?user_id=uuid&all=true
# → Revokes OAuth tokens, stops data sharing, queues erasure
```

## 30-Day SLA Enforcement

```go
func checkSLACompliance() {
    openRequests := dsarStore.GetOpenRequests()
    for _, req := range openRequests {
        daysRemaining := 30 - daysSince(req.CreatedAt)
        if daysRemaining <= 7 {
            alert.Send("dsar_sla_warning", req, daysRemaining)
        }
        if daysRemaining <= 0 {
            alert.Send("dsar_sla_breach", req) // Escalate to DPO
        }
    }
}
```

## Audit Trail

Every DSAR action is logged for compliance evidence:

```json
{
  "event": "dsar.request_received",
  "user_id": "uuid",
  "request_type": "erasure",
  "deadline": "2025-02-14",
  "status": "completed",
  "completed_at": "2025-01-20T10:00:00Z"
}
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| DSAR completion within SLA | 100% | Any breach → DPO escalation |
| Export generation time | <5 min | >30 min → optimize |
| Deletion cascade completeness | 100% | Any orphaned data → fix |

## See Also

- [Privacy by Design](privacy-by-design.md)
- [Compliance Framework Mapping](compliance-framework-mapping.md)
- [Consent Management Design](consent-management-design.md)
- [Data Classification Implementation](data-classification-implementation.md)
