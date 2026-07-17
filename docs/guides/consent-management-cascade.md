# Consent Withdrawal Cascade — Technical Guide

> Feature: GDPR-Compliant Consent Cascade + Erasure
> Location: `services/identity/internal/server/consent_registry_handler.go`, `services/oauth/internal/server/consent_receipt.go`

## What It Does

When a user withdraws consent or requests data erasure (GDPR Article 17), GGID propagates the change across all dependent systems — revoking OAuth tokens, removing data from connected applications, and generating a complete audit trail.

## Consent Withdrawal Cascade

When consent is withdrawn via `/api/v1/identity/consent/registry`:

```
User withdraws consent for purpose X
         ↓
   ┌──────────────────────────────┐
   │ 1. Update consent_records     │
   │    status → 'withdrawn'       │
   │    withdrawn_at → now()       │
   │    withdrawn_reason → text    │
   │ ↓                             │
   │ 2. Revoke OAuth tokens        │
   │    linked to this consent     │
   │ ↓                             │
   │ 3. Revoke OAuth refresh       │
   │    tokens for affected scopes │
   │ ↓                             │
   │ 4. Notify dependent clients   │
   │    via backchannel-logout     │
   │ ↓                             │
   │ 5. Log cascade audit trail    │
   └──────────────────────────────┘
```

## GDPR Article 17 Erasure

Right to erasure ('right to be forgotten'):

1. **Identity service**: Anonymize user record (name, email, phone → hashes).
2. **Audit service**: Retain audit events but remove PII fields.
3. **OAuth service**: Revoke all tokens and client grants.
4. **Policy service**: Remove role assignments and policy bindings.
5. **SCIM outbound**: Send DELETE to all connected applications.
6. **Sessions**: Revoke all active sessions immediately.
7. **Audit trail**: Generate erasure certificate with timestamp.

> **Note**: Audit logs retain event records (minus PII) for compliance — financial regulations may require retention of transaction evidence.

## Consent Receipt (GDPR Article 7)

Every consent grant generates a receipt:

```json
{
  "receipt_id": "uuid",
  "tenant_id": "...",
  "user_id": "...",
  "purposes": ["marketing"],
  "scopes": ["read:profile"],
  "granted_at": "2026-07-18T10:00:00Z",
  "expires_at": "2027-07-18T10:00:00Z",
  "withdraw_url": "/api/v1/oauth/consent/withdraw",
  "policy_version": "1.2"
}
```

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/identity/consent/registry` | DELETE | Withdraw consent (triggers cascade) |
| `/api/v1/oauth/consent/:id/receipt` | GET | Get GDPR-compliant consent receipt |
| `/api/v1/oauth/consent/withdraw` | POST | Withdraw OAuth consent |
| `/api/v1/oauth/backchannel-logout` | POST | Notify clients of session invalidation |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# Withdraw consent (triggers cascade)
curl -k -H 'Accept-Encoding: identity' \
  -X DELETE "https://ggid.iot2.win/api/v1/identity/consent/registry?user_id=admin&purpose=marketing" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Get consent receipt
NEW_TOKEN="your-jwt-token"
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/oauth/consent/consent-123/receipt" \
  -H "Authorization: Bearer $NEW_TOKEN" -H "X-Tenant-ID: $TENANT"
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Tokens still valid after withdrawal | Cascade delayed or backchannel-logout failed | Manually revoke tokens via `/api/v1/oauth/revoke` |
| Erasure incomplete | Some services unreachable | Retry erasure; check each service pod |
| Audit events still show PII | Anonymization incomplete | Run manual anonymization job on audit tables |

## Best Practices

- **Test cascade regularly**: Verify token revocation propagates within seconds.
- **Document erasure**: Keep erasure certificates for regulatory audits.
- **Monitor backchannel-logout**: Ensure connected applications receive logout notifications.
- **Retain audit evidence**: Never delete audit events entirely — anonymize PII fields only.
