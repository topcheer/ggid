# Identity Lifecycle Automation Guide

## Lifecycle Stages

```
Pre-hire → Provisioning → Active → Dormant → Deprovisioning → Archived
```

## Pre-Hire (Day -7)

- Identity record created in `pending` status
- Email alias reserved
- Default group membership assigned
- MFA enrollment email sent

## Provisioning (Day 0)

```bash
# SCIM 2.0 auto-provisioning
POST /api/v1/identity/scim/v2/Users
{
  "userName": "jane@corp.com",
  "active": true,
  "emails": [{"value":"jane@corp.com","primary":true}],
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"]
}
```

| Source | Mechanism |
|--------|-----------|
| HR system | Scheduled sync → SCIM API |
| SSO JIT | On first login, auto-provision |
| Admin manual | Console create user |
| API programmatic | POST /api/v1/identity/users |

## Active Phase

- Periodic access reviews (quarterly)
- Automated role assignment based on attributes (ABAC)
- Session monitoring for anomalies

## Dormant Detection

```sql
-- Users with no login in 90 days
SELECT id, email, last_login_at
FROM users
WHERE status = 'active'
  AND last_login_at < NOW() - INTERVAL '90 days';
```

| Trigger | Action |
|---------|--------|
| 30 days inactive | Email reminder |
| 60 days inactive | Disable MFA bypass |
| 90 days inactive | Set `dormant`, revoke tokens |
| 180 days inactive | Auto-deprovision |

## Deprovisioning

```bash
# Webhook triggered on deprovision
POST https://app.example.com/hooks/user-deprovisioned
{
  "user_id": "uuid",
  "action": "deprovision",
  "effective_at": "2025-01-15T00:00:00Z",
  "reason": "departure"
}
```

Deprovisioning checklist:
1. Revoke all sessions and tokens
2. Disable MFA factors
3. Remove from all groups
4. Set status to `suspended`
5. Retain audit log (7 years)
6. Notify connected apps via webhook

## Archival

After 365 days suspended:
- Anonymize PII (GDPR right to erasure)
- Keep audit records with hashed identifier
- Archive to cold storage

## Automation Rules

```yaml
rules:
  - name: "Auto-deprovision contractors"
    condition: "employment_type = 'contractor' AND end_date < today()"
    action: deprovision
    notify: [manager, it@corp.com]

  - name: "Dormant detection"
    schedule: "0 2 * * *"
    condition: "last_login_at < NOW() - 90 days"
    action: set_dormant
```

## See Also

- [SCIM 2.0 Integration](scim-integration.md)
- [GDPR Compliance](gdpr-compliance.md)
- [Access Reviews](access-reviews.md)
