# HR JML Automation — Technical Guide

> Feature: Joiner/Mover/Leaver Lifecycle Automation
> Location: `services/identity/internal/server/lifecycle_handler.go`, `dormant_handler.go`
> Endpoints: `/api/v1/users/lifecycle/*`, `/api/v1/hr/*`

## What It Does

GGID automates the full identity lifecycle — from new hire (Joiner) through role changes (Mover) to departure (Leaver) — by integrating with HR systems. This ensures access is granted, modified, and revoked in sync with employment status changes, eliminating manual provisioning delays and orphaned accounts.

## HR Sync Workflow

```
HR System (Workday/BambooHR/SAP)
         ↓
    HR Connector (poll/push)
         ↓
    GGID JML Engine
    ┌──────────────────────────┐
    │ Match HR events to users │
    │ ↓                        │
    │ Evaluate lifecycle rules  │
    │ ↓                        │
    │ Execute provisioning     │
    │ actions                  │
    └──────────────────────────┘
         ↓
    Audit Trail + Notifications
```

### Connector Types

| Type | Mechanism | Latency |
|------|-----------|--------|
| **REST Poll** | Periodic GET to HR API | 5-60 min |
| **SCIM Push** | HR system pushes SCIM events | Real-time |
| **Webhook** | HR system sends HTTP webhook on change | Seconds |

## JML Event Mapping

### Joiner (New Employee)

**Trigger**: New employee record appears in HR system.

| Step | Action |
|------|--------|
| 1 | Create GGID user account |
| 2 | Assign default group based on department |
| 3 | Assign baseline role based on job title |
| 4 | Send welcome email with setup instructions |
| 5 | Schedule onboarding workflow (MFA enrollment, device enrollment) |
| 6 | Notify manager |

### Mover (Role/Department Change)

**Trigger**: Employee's department, title, or manager changes in HR.

| Step | Action |
|------|--------|
| 1 | Update user profile (department, title) |
| 2 | Revoke access from old groups/roles |
| 3 | Grant access to new groups/roles |
| 4 | Trigger access review for sensitive changes |
| 5 | Audit log: "Mover event: old_dept → new_dept" |
| 6 | Notify user of access changes |

### Leaver (Termination/Resignation)

**Trigger**: Employee status changes to terminated/resigned in HR.

| Step | Action |
|------|--------|
| 1 | **Immediately** revoke all active sessions |
| 2 | **Immediately** disable account login |
| 3 | Revoke OAuth tokens and refresh tokens |
| 4 | Revoke device certificates (SCEP) |
| 5 | Remove from groups and distribution lists |
| 6 | Archive user data per retention policy |
| 7 | Notify manager and IT team |
| 8 | Schedule data deletion after grace period |

> **Critical**: Steps 1-4 must execute within seconds of termination to prevent data exfiltration.

## Dormant Detection Timeline

Accounts with no activity become dormant through staged progression:

```
Day 0:    Last activity (login/API call)
Day 30:   Warning notification sent to user
Day 60:   Admin notification
Day 90:   Account marked DORMANT
Day 120:  Account SUSPENDED (login blocked)
Day 180:  Account ARCHIVED (hidden from directory)
Day 365:  Account DELETED (data purged)
```

All thresholds are configurable per-tenant.

## Ghost Account Reconciliation

Ghost accounts exist in GGID but have no corresponding HR record:

1. **Detection**: Cross-reference GGID user list against HR employee data.
2. **Classification**:
   - **Orphan**: Created manually, never in HR (e.g., service accounts).
   - **Stale**: HR record deleted but GGID account remains.
   - **Pre-hire**: HR record not yet synced (timing gap).
3. **Action policy**: Flag, auto-disable, or auto-delete based on classification.
4. **Allowlist**: Service accounts can be exempted from ghost detection.

## Lifecycle Rules Configuration

```json
{
  "name": "Auto-suspend dormant 90d",
  "trigger": "dormant_threshold_exceeded",
  "threshold_days": 90,
  "action": "suspend",
  "notify_admin": true,
  "notify_user": false
}
```

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/hr/connectors` | GET/POST | Manage HR connectors |
| `/api/v1/hr/dormant` | GET | List dormant accounts |
| `/api/v1/users/lifecycle/rules` | GET/POST | Lifecycle automation rules |
| `/api/v1/identity/user-lifecycle/stages` | GET | User lifecycle stages |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# List dormant users
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/hr/dormant" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Create lifecycle rule
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/users/lifecycle/rules" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"name":"Leaver auto-disable","trigger":"hr_termination","action":"disable","notify_admin":true}'

# Get user lifecycle stage
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/identity/user-lifecycle/stages?user_id=admin" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Joiner not created | Connector sync delay or HR record missing | Check connector status; trigger manual sync |
| Leaver still has access | JML rule not configured or webhook failed | Manually disable account; verify HR connector |
| Ghost accounts flagged incorrectly | Service accounts not allowlisted | Add service accounts to ghost allowlist |
| Dormant user re-enabled | User logged in during dormant period | System correctly detected activity — this is expected |

## Best Practices

- **Webhook over poll**: Use webhook connectors for near-real-time leaver response.
| **Test leaver flow**: Regularly test termination to ensure <60 second disable.
- **Quarterly ghost audit**: Run reconciliation quarterly even with automated rules.
- **Service account allowlist**: Exempt legitimate non-HR accounts from ghost detection.
- **Document exceptions**: Any manual account creation should be documented for audit.
- **Monitor dormant progression**: Track accounts approaching suspension thresholds.
