# Policy Engine Guide (RBAC + ABAC)

How to use GGID's hybrid RBAC + ABAC policy engine for authorization.

---

## Overview

GGID combines two access control models:

- **RBAC** — Permissions assigned to roles; users inherit through role assignment
- **ABAC** — Attribute-based conditions for fine-grained, context-aware access

```
Permission Check Request
  (user_id, resource, action, context)
         │
         ▼
  ┌──────────────────┐
  │  Policy Engine   │
  │                  │
  │  1. Deny rules   │ ──► If deny matches → DENY
  │  2. RBAC check   │ ──► If role grants → ALLOW candidate
  │  3. ABAC check   │ ──► If conditions pass → ALLOW
  │  4. Default      │ ──► deny-all or allow-all
  └──────────────────┘
```

**Deny always wins.** If any deny policy matches, access is denied regardless of allow rules.

---

## RBAC: Roles and Permissions

### Create a Role

```bash
POST /api/v1/roles
{
  "key": "editor",
  "name": "Content Editor",
  "description": "Can edit and publish content"
}
# Response: {"id": "role-uuid", "key": "editor", ...}
```

### Add Permissions to a Role

```bash
POST /api/v1/roles/{role_id}/permissions
[
  {"resource": "documents:drafts", "action": "read"},
  {"resource": "documents:drafts", "action": "write"},
  {"resource": "documents:*", "action": "read"}
]
```

### Wildcard Matching

| Permission | Matches |
|-----------|---------|
| `documents:*` / `read` | `documents:drafts` / `read`, `documents:published` / `read`, etc. |
| `*` / `*` | Everything (superuser) |
| `documents:drafts` / `*` | All actions on documents:drafts |

### Role Hierarchy

Roles can inherit permissions from a parent:

```bash
POST /api/v1/roles
{
  "key": "staff",
  "name": "Staff",
  "description": "Base staff role"
}

POST /api/v1/roles
{
  "key": "admin",
  "name": "Administrator",
  "parent_role_key": "staff"  <!-- inherits all staff permissions
}
```

When checking a user with `admin` role, the engine collects permissions from `admin` + `staff`.

### Assign Role to User

```bash
POST /api/v1/users/{user_id}/roles
{"role_id": "role-uuid"}
```

### List User's Effective Permissions

```bash
GET /api/v1/users/{user_id}/permissions
# Returns all permissions from all assigned roles (including inherited)
```

---

## ABAC: Attribute-Based Policies

### Create a Policy

```bash
POST /api/v1/policies
{
  "name": "Engineering Off-Hours Block",
  "effect": "deny",
  "actions": ["write"],
  "resources": ["documents:sensitive"],
  "conditions": {
    "user.department": "engineering",
    "time.hour": "<9 OR >17"
  },
  "priority": 10
}
```

### Condition Operators

| Operator | Syntax | Example |
|----------|--------|---------|
| Equals | `"value"` | `{"user.department": "engineering"}` |
| Not equals | `"!=value"` | `{"user.department": "!=engineering"}` |
| Greater than | `">=N"` | `{"user.clearance_level": ">=3"}` |
| Less than | `"<=N"` | `{"user.attempts": "<=5"}` |
| Range | `"N-M"` | `{"time.hour": "9-17"}` |
| CIDR | `"10.0.0.0/8"` | `{"ip.cidr": "10.0.0.0/8"}` |
| In list | `["a","b"]` | `{"user.department": ["eng","sales"]}` |

### Attribute Sources

| Prefix | Source | Example Attributes |
|--------|--------|-------------------|
| `user.*` | User profile | `department`, `title`, `location`, `clearance_level` |
| `resource.*` | Resource metadata | `classification`, `owner`, `type` |
| `action` | Requested action | `read`, `write`, `delete` |
| `time.*` | Current time | `hour`, `day_of_week`, `date` |
| `ip.*` | Client IP | `address`, `cidr` |
| `env.*` | Environment | `region`, `deployment` |

### Example Policies

**Time-based access:**
```json
{
  "name": "Business Hours Only",
  "effect": "allow",
  "actions": ["write"],
  "resources": ["reports:*"],
  "conditions": {"time.hour": "9-17", "time.day_of_week": "1-5"}
}
```

**IP restriction:**
```json
{
  "name": "Corporate Network Only",
  "effect": "allow",
  "actions": ["*"],
  "resources": ["admin:*"],
  "conditions": {"ip.cidr": "10.0.0.0/8"}
}
```

**Department-scoped access:**
```json
{
  "name": "Department Document Access",
  "effect": "allow",
  "actions": ["read", "write"],
  "resources": ["documents:*"],
  "conditions": {"resource.department": "user.department"}
}
```

---

## Policy Check API

### Check Permission

```bash
POST /api/v1/policies/check
{
  "user_id": "a1b2c3d4-...",
  "resource": "documents:sensitive",
  "action": "read",
  "context": {
    "user.department": "engineering",
    "user.clearance_level": 5,
    "resource.classification": "internal",
    "ip.cidr": "10.0.1.100"
  }
}
```

### Response

```json
{
  "allowed": true,
  "reason": "Role 'editor' grants 'read' on 'documents:*' (inherited from 'staff')"
}
```

### Denied Response

```json
{
  "allowed": false,
  "reason": "Policy 'Engineering Off-Hours Block' (deny) matched: time.hour=22 does not satisfy range 9-17"
}
```

---

## Default Policy Action

Configure the fallback when no policy matches:

```bash
GET /api/v1/policies/default-action
# {"default_action": "deny"}  ← secure default

PUT /api/v1/policies/default-action
{"default_action": "deny"}
```

- `deny` (recommended) — Deny all unless explicit allow rule matches
- `allow` — Allow all unless explicit deny rule matches (less secure)

---

## Policy Templates

Apply pre-built compliance templates:

```bash
# List available templates
GET /api/v1/policies/templates

# Apply a template
POST /api/v1/policies/from-template/soc2
# Creates all policies needed for SOC 2 compliance
```

---

## Policy Import / Export

```bash
# Export all policies (backup)
GET /api/v1/policies/export

# Import policies
POST /api/v1/policies/import
[
  {
    "name": "Imported Policy",
    "effect": "allow",
    "actions": ["read"],
    "resources": ["reports:*"],
    "conditions": {}
  }
]
```

---

## Policy Versioning

```bash
# List versions
GET /api/v1/policies/versions?policy_id={id}

# Snapshot current state as new version
POST /api/v1/policies/versions?policy_id={id}

# Rollback to specific version
POST /api/v1/policies/versions/rollback?policy_id={id}&version=3
```

---

## Performance

| Operation | Typical Latency |
|-----------|----------------|
| RBAC check (cached) | < 3ms |
| RBAC check (cold) | 5-15ms |
| ABAC evaluation | 2-8ms per policy |
| Full check (RBAC + ABAC) | p95 < 50ms |

Role permissions are cached in Redis (5min TTL). Policy evaluation loads only matching policies.
