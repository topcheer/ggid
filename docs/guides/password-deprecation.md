# Password Deprecation Policy (KB-074)

## Overview

GGID's password deprecation policy provides a controlled pathway to eliminate passwords entirely — transitioning users to passkeys, biometrics, and SSO while maintaining backward compatibility.

## Deprecation Stages

| Stage | Name | User Impact | Timeline |
|-------|------|-------------|----------|
| 0 | **Enabled** | Password + passkey both work | Baseline |
| 1 | **Discouraged** | UI promotes passkey; password still works | Week 1-2 |
| 2 | **Conditional** | Password requires additional MFA | Week 3-4 |
| 3 | **Deprecated** | Password blocked for enrolled users | Week 5+ |
| 4 | **Removed** | Password hash deleted from DB | Week 8+ |

## Configuration

```yaml
password_policy:
  deprecation_stage: 2  # conditional
  require_alternative_before_disable: true
  grace_period_days: 14
  auto_enroll_passkey: true
  notification_template: "password-deprecation-notice"
```

## Per-Group Stages

Different groups can be at different stages simultaneously:

```http
PUT /api/v1/auth/password-deprecation
Content-Type: application/json

{
  "group_id": "engineering",
  "stage": 3,
  "effective_date": "2025-02-01"
}
```

## Enforcement Logic

```
Login attempt with password:
  1. Check user's group deprecation stage
  2. Stage 0-1: Allow (stage 1 shows banner)
  3. Stage 2: Allow only if additional MFA provided
  4. Stage 3: Deny if user has enrolled passkey/SSO
  5. Stage 4: Deny always (password hash deleted)
```

## Monitoring

```http
GET /api/v1/admin/password-deprecation/status
```

```json
{
  "total_users": 1247,
  "by_stage": {
    "stage_0": 892,
    "stage_1": 241,
    "stage_2": 89,
    "stage_3": 25
  },
  "passkey_enrolled": 356,
  "password_removed": 0,
  "avg_days_to_removal": 12
}
```

## Best Practices

- **Never skip stages** — users need time to enroll alternatives
- **Require enrollment** — block stage 3 unless user has ≥1 non-password method
- **Communicate early** — send notifications at stage 1 and 2
- **Admin fallback** — always keep emergency password access for break-glass accounts
