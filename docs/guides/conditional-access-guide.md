# Conditional Access Guide (KB-080)

## Overview

GGID's conditional access engine evaluates signals (device, location, risk, time) at login time to dynamically allow, step-up, or block authentication — replacing static access rules with context-aware policies.

## Signal Sources

| Signal | Source | Example |
|--------|--------|---------|
| Device trust | Device posture score | `device_trust >= 80` |
| IP reputation | Gateway feed | `ip_reputation = "malicious"` |
| Geo location | IP geolocation | `country = "CN"` |
| Risk score | URE composite | `risk_score < 30` |
| Time of day | Server clock | `hour BETWEEN 6 AND 22` |
| Group membership | User directory | `group = "engineering"` |
| MFA enrollment | Auth service | `mfa_enrolled = true` |

## Policy Structure

```json
{
  "name": "Block High-Risk Non-Corporate",
  "priority": 100,
  "conditions": [
    { "field": "risk_score", "op": ">=", "value": 60 },
    { "field": "device_managed", "op": "==", "value": false }
  ],
  "action": "deny",
  "reason": "High risk score from unmanaged device"
}
```

## Actions

| Action | Behavior |
|--------|----------|
| `allow` | Grant access |
| `step_up` | Require additional MFA |
| `deny` | Block with message |
| `quarantine` | Allow to limited sandbox session |

## Login Flow Integration

```
1. User submits credentials
2. Auth service validates password/passkey
3. Conditional access engine loads tenant policies (sorted by priority)
4. Evaluate each policy against collected signals
5. First matching policy wins → execute action
6. No match → default allow
```

## Configuration

### Create Policy
```http
POST /api/v1/auth/conditional-access
Content-Type: application/json

{
  "name": "Require MFA for Admins",
  "priority": 50,
  "conditions": [
    { "field": "group", "op": "==", "value": "admin" },
    { "field": "mfa_enrolled", "op": "==", "value": false }
  ],
  "action": "step_up"
}
```

### List Policies
```http
GET /api/v1/auth/conditional-access
```

## Best Practices

- **Priority ordering**: Higher priority = evaluated first (100 > 50 > 10)
- **Fail-safe default**: Default action is `allow` — set explicit deny rules for sensitive resources
- **Test with dry-run**: Evaluate against sample sessions before enabling
- **Combine signals**: Use multiple conditions for precise targeting
- **Audit trail**: All policy evaluations are logged to the audit chain
