# Consent Management Architecture

Consent lifecycle, record schema, consent receipts, purpose binding, dashboard, API, regulatory requirements, and versioning.

## Consent Lifecycle

```
Collect → Store → Manage → Withdraw → Audit
   │        │        │          │        │
   ▼        ▼        ▼          ▼        ▼
 Screen   Record   Dashboard  Revoke   Evidence
```

## Consent Record Schema

```json
{
  "consent_id": "uuid",
  "user_id": "uuid",
  "client_id": "oauth-client-123",
  "scopes": ["openid", "profile", "email"],
  "purpose": "Profile access for dashboard",
  "granted_at": "2025-01-15T10:00:00Z",
  "expires_at": "2025-07-15T10:00:00Z",
  "version": 2,
  "withdrawn_at": null,
  "receipt_uri": "https://auth.ggid.dev/consent/uuid/receipt"
}
```

## Consent Receipt (JWT-based)

```json
{
  "iss": "https://auth.ggid.dev",
  "sub": "user-uuid",
  "iat": 1700000000,
  "consent_id": "uuid",
  "purposes": [{"purpose": "identity", "data": ["email"], "lawful_basis": "consent"}],
  "jurisdiction": "EU",
  "sig": "..."
}
```

Receipts are immutable, stored 7 years for compliance evidence.

## Consent API

```bash
# Grant consent
POST /api/v1/consent {user_id, client_id, scopes, purpose}

# List consents
GET /api/v1/consent?user_id=uuid

# Withdraw consent
DELETE /api/v1/consent/{id}
# → All tokens from this consent revoked

# Get receipt
GET /api/v1/consent/{id}/receipt
```

## Consent Versioning

When consent terms change, version increments:

| Version | Change | Action |
|---------|--------|--------|
| 1 | Initial consent | User grants |
| 2 | New scope added | Re-consent required |
| 3 | Purpose changed | Re-consent required |

Users must re-consent when version increases. Old version retained for audit.

## Regulatory Requirements

| Law | Consent Requirement |
|-----|-------------------|
| GDPR | Explicit, granular, withdrawable, unambiguous |
| ePrivacy | Consent for tracking (cookies, pixels) |
| CCPA | Opt-out (not consent-based, but similar mechanism) |
| LGPD | Explicit, free, informed, withdrawable |

## Monitoring

| Metric | Alert |
|--------|-------|
| Consent denial rate | >20% → client over-requesting |
| Withdrawal rate | >10% → trust issue |
| Expired consents | Batch → notify users |

## See Also

- [Consent Management Design](consent-management-design.md)
- [Privacy by Design](privacy-by-design.md)
- [DSAR Automation](dsar-automation.md)
- [OAuth Client Scoped Permissions](oauth-client-scoped-permissions.md)