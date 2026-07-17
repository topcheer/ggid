# SOAR Playbooks — Technical Guide

> Feature: Security Orchestration, Automation & Response (SOAR)
> Location: `services/audit/internal/soar/engine.go`
> Console: `/security/itdr-rules` (Playbooks tab)

## What It Does

The SOAR Playbook engine automatically executes predefined response actions when ITDR detections trigger. Playbooks enable security teams to respond to identity threats in milliseconds rather than hours, with consistent, auditable actions.

## Architecture

```
ITDR Detection (e.g., mfa_fatigue, token_theft)
         ↓
   Playbook Engine
   ┌────────────────────┐
   │ Match trigger type │
   │ ↓                  │
   │ Execute actions    │
   │ (sequential)       │
   │ ↓                  │
   │ Log results        │
   └────────────────────┘
         ↓
   Audit Trail + Notification
```

## Core Types

### PlaybookTrigger

Defines what events activate the playbook:

| Trigger Type | Description |
|-------------|-------------|
| `detection` | Fires when an ITDR detection of a specific rule is created |
| `severity` | Fires when detection severity reaches threshold (high/critical) |
| `rule_id` | Fires for a specific detection rule (e.g., `mfa_fatigue`) |
| `manual` | Manually triggered by an administrator |

### PlaybookAction

Each playbook contains a sequence of actions executed in order:

| Action Type | Description |
|-------------|-------------|
| `revoke_session` | Revoke the user's active sessions |
| `disable_account` | Temporarily disable the user account |
| `force_mfa` | Enforce MFA enrollment on next login |
| `notify_user` | Send email/notification to the user |
| `notify_admin` | Alert security team via email/webhook |
| `block_ip` | Add source IP to blocklist |

### Playbook

```go
type Playbook struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Enabled     bool              `json:"enabled"`
    Trigger     PlaybookTrigger   `json:"trigger"`
    Actions     []PlaybookAction  `json:"actions"`
    CreatedAt   string            `json:"created_at"`
    UpdatedAt   string            `json:"updated_at"`
}
```

## Workflows

### Create a Playbook

1. Navigate to `/security/itdr-rules` > Playbooks tab.
2. Click **Create Playbook**.
3. Configure the trigger (e.g., rule_id: `mfa_fatigue`).
4. Add actions in execution order (e.g., `revoke_session` → `notify_user` → `notify_admin`).
5. Enable the playbook.

### Example: MFA Fatigue Auto-Response

Trigger: `mfa_fatigue` detection created
Actions:
1. `revoke_session` — immediately invalidates active sessions
2. `force_mfa` — requires MFA re-enrollment
3. `notify_user` — emails user about suspicious activity
4. `notify_admin` — alerts security team

### Example: Token Theft Response

Trigger: `token_theft` detection (critical severity)
Actions:
1. `revoke_session` — kill all sessions
2. `disable_account` — lock the account
3. `block_ip` — block the attacker's IP
4. `notify_admin` — page on-call security

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/audit/itdr/playbooks` | GET | List all playbooks |
| `/api/v1/audit/itdr/playbooks` | POST | Create a playbook |
| `/api/v1/audit/itdr/playbooks/:id` | PUT | Update a playbook |
| `/api/v1/audit/itdr/playbooks/:id` | DELETE | Delete a playbook |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# List playbooks
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/itdr/playbooks" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Create MFA Fatigue auto-response playbook
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/audit/itdr/playbooks" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"name":"MFA Fatigue Response","trigger":{"type":"rule_id","value":"mfa_fatigue"},"actions":[{"type":"revoke_session"},{"type":"force_mfa"},{"type":"notify_admin"}],"enabled":true}'
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Playbook not firing | Disabled or trigger mismatch | Check enabled=true; verify trigger rule_id matches detection |
| Actions not executing | Service connection failure | Check inter-service gRPC connectivity |
| Action fails silently | Missing target (user deleted, IP already blocked) | Check audit logs for action execution errors |

## Best Practices

- **Order matters**: Put critical actions (revoke, disable) before notifications.
- **Test first**: Use manual trigger to test before enabling automatic execution.
- **Escalate severity**: Create separate playbooks for High vs Critical detections.
- **Audit execution**: Review playbook execution logs weekly for failures or tuning.
- **Avoid loops**: Don't create playbooks that trigger on their own notification actions.
