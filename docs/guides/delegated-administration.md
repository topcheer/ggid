# Delegated Administration

Guide for hierarchical admin delegation, JIT elevation, and break-glass access in GGID.

## Overview

Delegated administration allows privileged users to grant scoped administrative permissions to others — with guardrails, full audit, and time-bounded constraints.

## Admin Scope Hierarchy

```
Super Admin (org-wide, all tenants)
  └─ Tenant Admin (single tenant)
       └─ Department Admin (subset of users)
            └─ Helpdesk (read-only + password reset)
```

| Scope | Capabilities | Delegable |
|-------|-------------|-----------|
| `admin:super` | Everything, including tenant mgmt | No |
| `admin:tenant` | Users, roles, orgs in one tenant | Yes → dept admin |
| `admin:dept` | Users in assigned departments | Yes → helpdesk |
| `admin:helpdesk` | Read users, reset passwords, unlock | No |

## Delegated Permissions

### Role Delegation Chains

An admin can delegate a subset of their own permissions:

```bash
# Tenant admin delegates helpdesk role to user
POST /api/v1/policy/delegations
{
  "granter": "admin@corp.com",
  "grantee": "helpdesk1@corp.com",
  "role": "helpdesk",
  "constraints": {
    "departments": ["engineering", "design"],
    "max_duration": "8h",
    "require_approval": false
  }
}
```

Chain limit: max 3 levels deep (A → B → C → D, not further).

```go
const MaxDelegationDepth = 3

func validateDelegationDepth(granterClaims jwt.MapClaims) error {
    depth := getDelegationDepth(granterClaims)
    if depth >= MaxDelegationDepth {
        return ErrMaxDelegationDepth
    }
    return nil
}
```

### Constraints

| Constraint | Description |
|-----------|-------------|
| Department scope | Can only manage users in listed departments |
| Time window | Permission valid only 09:00-17:00 |
| Max duration | Auto-expire after N hours |
| Approval required | Requires another admin to approve |
| IP restriction | Only from corporate VPN |

## JIT (Just-In-Time) Elevation

Temporary privilege elevation with auto-expiry:

```bash
# Request JIT elevation
POST /api/v1/auth/elevate
{
  "requested_scope": "admin:dept",
  "reason": "On-call incident response",
  "duration_minutes": 60
}
# → Requires MFA step-up
# → Returns elevated JWT with 60-min TTL

# Elevated token includes:
# {scope: "admin:dept", elevation_expires: 1700000000}
```

After TTL expires, token automatically loses elevated scope. No manual revocation needed.

### Approval Workflow

For high-privilege JIT requests:

```
Requester → Manager approval → Security team approval → Elevated access granted
                  (5 min SLA)         (15 min SLA)
```

## Guardrails

### Hard Limits

- Cannot delegate `admin:super`
- Cannot grant broader scope than you hold
- Cannot self-delegate
- Cannot bypass MFA requirement
- Delegation chain max depth: 3

### Automated Controls

```sql
-- Nightly check: active delegations exceeding constraints
SELECT * FROM delegations
WHERE status = 'active'
  AND (expires_at < NOW() OR dept_scope_violated = true);
```

Violations trigger automatic revocation + alert.

## Break-Glass Access

Emergency access when normal delegation fails:

```bash
# Break-glass requires dual authorization
POST /api/v1/auth/break-glass
{
  "requester": "admin@corp.com",
  "target_user": "compromised@corp.com",
  "action": "force_password_reset",
  "incident_id": "INC-2025-0142"
}
# → Requires: 2 admin approvals + SIEM alert + auto-audit
```

| Rule | Enforcement |
|------|------------|
| Dual control | Two separate admins must approve |
| Time-boxed | Access expires in 30 minutes |
| Full audit | Every action logged with SIEM forward |
| Rate limited | Max 3 break-glass events per day |
| Notification | Security team paged immediately |

## Audit Trail

Every delegation and elevation is logged:

```json
{
  "event": "delegation.granted",
  "granter": "admin@corp.com",
  "grantee": "helpdesk1@corp.com",
  "role": "helpdesk",
  "constraints": {"departments": ["engineering"], "max_duration": "8h"},
  "timestamp": "2025-01-15T10:30:00Z",
  "ip": "10.0.1.5",
  "session_id": "sess-abc"
}
```

Quarterly access review reports:
- Who delegated what to whom
- Whether delegated permissions were used
- Dormant delegations for cleanup

## Monitoring

| Alert | Threshold |
|-------|-----------|
| Delegation chain depth | >3 → block |
| JIT elevation frequency | >5/day/user → review |
| Break-glass usage | Any → page security |
| Delegation outside business hours | Review |
| Scope creep | Granted > held scope → block + alert |

## See Also

- [Access Reviews](access-reviews.md)
- [Conditional Access](conditional-access.md)
- [MFA Architecture](mfa-architecture.md)
- [Audit API](../api/audit-api.md)
