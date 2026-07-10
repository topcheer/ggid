# RBAC and ABAC Guide

Role-Based Access Control (RBAC) and Attribute-Based Access Control (ABAC) in
GGID: role hierarchy, permission inheritance, ABAC policy syntax, condition
operators, dynamic policy evaluation, and enterprise scenarios.

---

## Table of Contents

- [RBAC Overview](#rbac-overview)
- [Role Hierarchy](#role-hierarchy)
- [Permission Inheritance](#permission-inheritance)
- [ABAC Overview](#abac-overview)
- [ABAC Policy Syntax](#abac-policy-syntax)
- [Condition Operators](#condition-operators)
- [Dynamic Policy Evaluation](#dynamic-policy-evaluation)
- [Enterprise Scenarios](#enterprise-scenarios)
- [API Reference](#api-reference)

---

## RBAC Overview

RBAC assigns permissions to roles, and users to roles. GGID implements a
hierarchical RBAC model with tenant-scoped roles.

### Core Concepts

| Concept | Description | Example |
|---------|-------------|---------|
| Role | Named collection of permissions | `admin`, `viewer`, `editor` |
| Permission | Action on a resource | `users:read`, `users:write` |
| Assignment | User-role binding (with scope + expiry) | `user@tenant → admin` |
| Scope | Where the role applies | `tenant`, `org`, `resource` |

### Default Roles

| Role | Key | Permissions |
|------|-----|-------------|
| Super Admin | `super_admin` | All resources, all actions |
| Admin | `admin` | User/org/role management, audit review |
| Security Admin | `security_admin` | MFA enforcement, impersonation, security monitoring |
| Editor | `editor` | Create/update resources, no delete |
| Viewer | `viewer` | Read-only access |

---

## Role Hierarchy

Roles form a tree. Permissions cascade downward:

```
super_admin (all permissions)
├── admin (all tenant permissions)
│   ├── security_admin (security + MFA + impersonation)
│   └── editor (create + update)
│       └── viewer (read-only)
└── service_account (M2M API access)
```

### Create Custom Role

```bash
curl -X POST https://iam.example.com/api/v1/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "Content Manager",
    "key": "content_manager",
    "description": "Manage content but not users",
    "permissions": ["content:read", "content:write", "content:publish"],
    "parent_role": "editor"
  }'
```

### List Roles

```bash
curl https://iam.example.com/api/v1/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

---

## Permission Inheritance

A child role inherits all permissions from its parent. The evaluation is
recursive up the hierarchy.

### Evaluation Flow

```
1. Gather user's roles (direct + group-derived)
2. For each role, gather permissions (direct + inherited from ancestors)
3. Check if requested permission is in the set
4. If RBAC allows → allow
5. If ABAC policy exists → evaluate conditions
6. Both must pass → allow
```

### Example

```
Roles:
  viewer → { users:read, roles:read }
  editor → parent: viewer, adds { users:write, roles:write }
  admin  → parent: editor, adds { users:delete, roles:delete }

User Alice has role: editor
→ Alice has: { users:read, roles:read, users:write, roles:write }
→ Alice does NOT have: { users:delete, roles:delete }
```

---

## ABAC Overview

ABAC extends RBAC with attribute-based conditions. While RBAC answers "is
this user allowed?", ABAC answers "is this user allowed **under these
circumstances?**".

### When to Use ABAC

| Scenario | RBAC? | ABAC? |
|----------|:-----:|:-----:|
| "Admins can manage users" | Yes | No |
| "Users can edit their own profile" | No | Yes |
| "Only during business hours" | No | Yes |
| "Only from corporate IP range" | No | Yes |
| "Managers can approve reports in their department" | No | Yes |
| "Read-only on weekends" | No | Yes |

---

## ABAC Policy Syntax

GGID uses a JSON-based policy language:

```json
{
  "id": "policy-uuid",
  "name": "Edit Own Profile",
  "description": "Users can edit their own profile only",
  "effect": "allow",
  "actions": ["users:update"],
  "resources": ["users/{user_id}"],
  "conditions": {
    "subject.user_id": { "equals": "resource.owner_id" }
  }
}
```

### Policy Structure

| Field | Type | Description |
|-------|------|-------------|
| `effect` | `allow` / `deny` | Decision if conditions match |
| `actions` | `[]string` | Resource actions (`users:read`, `*`) |
| `resources` | `[]string` | Resource patterns (`users/*`, `*`) |
| `conditions` | `map[string]Condition` | Attribute comparisons |
| `priority` | `int` | Higher priority evaluated first (deny wins ties) |

### Template Variables

| Variable | Source | Example |
|----------|--------|---------|
| `subject.user_id` | JWT claim | `"550e8400-..."` |
| `subject.role` | JWT claim | `"admin"` |
| `subject.department` | User attribute | `"Engineering"` |
| `subject.ip` | Request IP | `"192.168.1.50"` |
| `subject.time` | Current time | `"2024-01-15T14:30:00Z"` |
| `resource.owner_id` | Resource attribute | `"550e8400-..."` |
| `resource.tenant_id` | Resource attribute | `"00000000-..."` |
| `resource.department` | Resource attribute | `"Engineering"` |

---

## Condition Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `equals` | Strict equality | `subject.role equals "admin"` |
| `not_equals` | Inequality | `subject.role not_equals "viewer"` |
| `in` | Membership in list | `subject.department in ["Eng", "QA"]` |
| `not_in` | Not in list | `subject.role not_in ["viewer"]` |
| `contains` | String/list contains | `subject.groups contains "admins"` |
| `starts_with` | String prefix | `subject.ip starts_with "10.0."` |
| `ends_with` | String suffix | `subject.email ends_with "@corp.com"` |
| `less_than` | Numeric/date comparison | `subject.age less_than 65` |
| `greater_than` | Numeric/date | `resource.risk_score greater_than 7` |
| `between` | Range check | `subject.time between "09:00" and "17:00"` |
| `regex` | Regex match | `subject.email regex ".*@corp\\.com$"` |
| `cidr` | IP in CIDR range | `subject.ip cidr "10.0.0.0/8"` |

### Boolean Logic

```json
{
  "conditions": {
    "any_of": [
      { "subject.role": { "equals": "admin" } },
      { "all_of": [
        { "subject.role": { "equals": "manager" } },
        { "subject.department": { "equals": "resource.department" } }
      ]}
    ]
  }
}
```

---

## Dynamic Policy Evaluation

### Evaluation Pipeline

```
Request arrives →
  1. RBAC check: user has required role permission?
     └── No → DENY
     └── Yes → continue
  2. ABAC check: applicable policies for action+resource?
     └── No policies → ALLOW (RBAC sufficient)
     └── Policies found → evaluate each:
         ├── Deny policy matches → DENY (deny wins)
         ├── Allow policy matches → ALLOW
         └── No match → DENY (default deny)
  3. Return decision
```

### Policy Caching

Policies are cached in-memory per tenant with a 5-minute TTL:

```yaml
policy:
  cache:
    enabled: true
    ttl: 300s
    max_size: 10000
  evaluation_timeout: 100ms
```

### Performance

| Metric | Target | P99 |
|--------|--------|-----|
| Single policy eval | < 1ms | 5ms |
| 10-policy eval | < 5ms | 15ms |
| Cache hit rate | > 95% | - |

---

## Enterprise Scenarios

### Scenario 1: Department-Scoped Access

"Managers can only see employees in their own department."

```json
{
  "name": "Department Scope",
  "effect": "allow",
  "actions": ["users:read"],
  "resources": ["users/*"],
  "conditions": {
    "all_of": [
      { "subject.role": { "in": ["manager", "director"] } },
      { "subject.department": { "equals": "resource.department" } }
    ]
  }
}
```

### Scenario 2: Time-Based Access

"Service desk access only during business hours."

```json
{
  "name": "Business Hours Only",
  "effect": "allow",
  "actions": ["*"],
  "resources": ["*"],
  "conditions": {
    "subject.time": { "between": "09:00,17:00" },
    "subject.day_of_week": { "not_in": ["Saturday", "Sunday"] }
  }
}
```

### Scenario 3: IP-Based Step-Up

"Sensitive operations require corporate IP or MFA."

```json
{
  "name": "Corporate Network or MFA",
  "effect": "allow",
  "actions": ["users:delete", "roles:delete"],
  "resources": ["*"],
  "conditions": {
    "any_of": [
      { "subject.ip": { "cidr": "10.0.0.0/8" } },
      { "subject.mfa_verified": { "equals": true } }
    ]
  }
}
```

### Scenario 4: Resource Owner

"Users can edit resources they own."

```json
{
  "name": "Resource Owner Edit",
  "effect": "allow",
  "actions": ["documents:update", "documents:delete"],
  "resources": ["documents/*"],
  "conditions": {
    "subject.user_id": { "equals": "resource.owner_id" }
  }
}
```

### Scenario 5: Temporary Elevated Access

"Grant admin role for 2 hours (break-glass)."

```bash
# Assign time-limited role
curl -X POST .../users/{id}/roles \
  -d '{
    "role_id": "admin-role-id",
    "scope": "tenant",
    "expires_at": "2024-01-15T14:00:00Z"
  }'
```

After `expires_at`, the role is automatically revoked.

---

## API Reference

### Create Policy

```bash
curl -X POST https://iam.example.com/api/v1/policies \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "Department Scope",
    "effect": "allow",
    "actions": ["users:read"],
    "resources": ["users/*"],
    "conditions": { ... },
    "priority": 100
  }'
```

### List Policies

```bash
curl https://iam.example.com/api/v1/policies \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

### Evaluate Policy (Debug)

```bash
curl -X POST https://iam.example.com/api/v1/policies/evaluate \
  -d '{
    "subject": { "role": "manager", "department": "Eng", "user_id": "u1" },
    "action": "users:read",
    "resource": { "type": "users", "owner_id": "u1", "department": "Eng" }
  }'
```

```json
{
  "decision": "allow",
  "matched_policies": ["policy-uuid-1"],
  "evaluation_time_ms": 2
}
```
