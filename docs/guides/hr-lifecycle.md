# HR Lifecycle Automation — Technical Guide

> Feature: HR-Driven Identity Lifecycle (Joiner/Mover/Leaver)
> Location: `services/identity/internal/server/dormant_handler.go`, `lifecycle_handler.go`
> Endpoints: `/api/v1/hr/*`, `/api/v1/users/lifecycle/*`

## What It Does

GGID automates the identity lifecycle by integrating with HR systems to detect dormant accounts, reconcile ghost accounts, and manage Joiner/Mover/Leaver (JML) workflows — ensuring no orphaned or stale identities remain active.

## Components

### 1. HR Connectors

Connect to external HR systems (Workday, BambooHR, SAP SuccessFactors) to sync employee status:

- **Connector types**: REST API polling, SCIM push, webhook event-driven.
- **Sync frequency**: Configurable (hourly/daily/real-time).
- **Status mapping**: HR status → GGID lifecycle state (active, suspended, terminated).

**API:** `/api/v1/hr/connectors` (GET/POST/PUT/DELETE)

### 2. Dormant Detection

Identifies inactive user accounts that should be suspended or archived:

- **Threshold**: Configurable inactivity period (default: 90 days).
- **Lifecycle states**: Active → Dormant → Suspended → Archived.
- **Triggers**: No login, no API calls, no session activity.
- **Auto-action**: Optional automatic suspension after threshold.

**API:** `/api/v1/hr/dormant` (GET — list dormant users)

### 3. Ghost Account Reconciliation

Detects "ghost" accounts — identities that exist in GGID but have no corresponding HR record:

- **Reconciliation**: Cross-references GGID users against HR system employee records.
- **Actions**: Flag for review, auto-disable, or auto-delete.
- **Audit trail**: Every ghost detection and action is logged.

### 4. JML (Joiner/Mover/Leaver) Integration

Automated provisioning based on HR events:

| Event | Trigger | GGID Action |
|-------|---------|-------------|
| **Joiner** | New employee in HR | Create user, assign default role + group |
| **Mover** | Department/role change | Update groups, revoke old access, grant new |
| **Leaver** | Termination in HR | Revoke sessions, disable account, archive |

**API:** `/api/v1/identity/user-lifecycle/stages`, `/api/v1/users/lifecycle/rules`

## Lifecycle Rules

Rules define automated actions for lifecycle transitions:

```json
{
  "name": "Auto-suspend after 90 days inactive",
  "trigger": "dormant_threshold_exceeded",
  "threshold_days": 90,
  "action": "suspend",
  "notify_admin": true
}
```

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/hr/connectors` | GET/POST | List or create HR connectors |
| `/api/v1/hr/dormant` | GET | List dormant users |
| `/api/v1/users/lifecycle/rules` | GET/POST | Manage lifecycle rules |
| `/api/v1/identity/user-lifecycle/stages` | GET | Get lifecycle stage for users |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# List dormant accounts
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/hr/dormant" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# List HR connectors
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/hr/connectors" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Create lifecycle rule
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/users/lifecycle/rules" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"name":"Auto-suspend 90d","trigger":"dormant_threshold_exceeded","threshold_days":90,"action":"suspend"}'
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Dormant list empty | No users inactive past threshold | Lower threshold or wait longer |
| Connector sync fails | Invalid credentials or HR API down | Verify connector config; check HR system status |
| Ghost accounts not detected | Reconciliation not run | Trigger manual reconciliation via API |
| Leaver not disabled | JML rule not configured | Create lifecycle rule for termination events |

## Best Practices

- **Set conservative thresholds**: 90 days dormant, then suspend — never auto-delete without review.
- **Monthly ghost reconciliation**: Run monthly to catch accounts not in HR.
- **Automate leaver actions**: Critical for security — terminated employees must lose access immediately.
- **Test JML rules**: Use a test HR feed before production deployment.
