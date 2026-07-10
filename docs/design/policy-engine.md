# Design: RBAC + ABAC Hybrid Policy Engine

> **Status:** Implemented | **ADR:** [ADR-004](../adr/ADR-004-rls-for-multi-tenancy.md) (related)

## Problem Statement

GGID needs a policy engine that supports both:

1. **RBAC (Role-Based Access Control)** — Users have roles; roles have permissions.
   Simple, hierarchical, fast.

2. **ABAC (Attribute-Based Access Control)** — Access decisions based on
   attributes of the user, resource, action, and environment. Flexible,
   context-aware, dynamic.

Most IAM platforms support only RBAC. GGID combines both for maximum flexibility.

## Architecture

```
                    Permission Check Request
                    (user_id, resource, action, context)
                              │
                              ▼
                    ┌─────────────────┐
                    │  Policy Engine  │
                    │  (Policy Svc)   │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
     ┌────────────┐  ┌────────────┐  ┌────────────┐
     │  RBAC      │  │  ABAC      │  │  Deny      │
     │  Check     │  │  Rules     │  │  Override  │
     │            │  │            │  │            │
     │ Roles →    │  │ Attributes │  │ Explicit   │
     │ Permissions │  │ → Match    │  │ deny rules │
     └─────┬──────┘  └─────┬──────┘  └─────┬──────┘
           │               │               │
           └───────────────┼───────────────┘
                           │
                    ┌──────▼──────┐
                    │  Decision   │
                    │             │
                    │  allow/deny │
                    │  + reason   │
                    └─────────────┘
```

## Evaluation Algorithm

The policy engine evaluates requests in this order:

```
1. Check explicit DENY policies
   ├── If any deny policy matches → DENY (explicit override)

2. Check RBAC permissions
   ├── Get user's roles (including inherited from parent roles)
   ├── For each role, check if it grants (resource, action)
   └── If any role grants → ALLOW candidate

3. Check ABAC conditions
   ├── For each allow policy matching (resource, action)
   │   └── Evaluate conditions against request attributes
   ├── If conditions pass → ALLOW
   └── If conditions fail → continue

4. Default: DENY
```

**Deny always wins.** If a deny policy matches, access is denied regardless
of allow rules.

## RBAC Component

### Role Model

```sql
CREATE TABLE roles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    key             TEXT NOT NULL,           -- unique identifier (e.g. 'editor')
    name            TEXT NOT NULL,           -- display name
    description     TEXT,
    parent_role_id  UUID REFERENCES roles(id), -- inheritance
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, key)
);
```

### Permission Model

```sql
CREATE TABLE role_permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    role_id     UUID NOT NULL REFERENCES roles(id),
    resource    TEXT NOT NULL,   -- e.g. 'documents:sensitive'
    action      TEXT NOT NULL,   -- e.g. 'read', 'write', '*'
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

### Role Hierarchy (Inheritance)

Roles can inherit permissions from a parent role:

```
staff (base role)
  ├── permissions: documents:public → read
  │
  └── editor (inherits staff)
        ├── permissions: documents:drafts → write
        │
        └── admin (inherits editor)
              ├── permissions: documents:sensitive → *
```

When checking a user with the `admin` role, the engine collects permissions
from `admin` + `editor` + `staff`.

### Wildcard Matching

- `resource: *` matches all resources
- `action: *` matches all actions
- `resource: documents:*` matches `documents:drafts`, `documents:sensitive`, etc.

```go
func matchPermission(grantedResource, requestedResource string) bool {
    if grantedResource == "*" {
        return true
    }
    // Wildcard prefix match: "documents:*" matches "documents:anything"
    if strings.HasSuffix(grantedResource, ":*") {
        prefix := strings.TrimSuffix(grantedResource, "*")
        return strings.HasPrefix(requestedResource, prefix)
    }
    return grantedResource == requestedResource
}
```

## ABAC Component

### Policy Model

```sql
CREATE TABLE policies (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    name        TEXT NOT NULL,
    description TEXT,
    effect      TEXT NOT NULL CHECK (effect IN ('allow', 'deny')),
    actions     JSONB NOT NULL,      -- ["read", "write"]
    resources   JSONB NOT NULL,      -- ["documents:*"]
    conditions  JSONB,               -- {"department": "engineering", "clearance": ">= 3"}
    priority    INT DEFAULT 0,       -- higher = evaluated first
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

### Condition Evaluation

Conditions are JSON key-value pairs with operators:

```json
{
  "user.department": "engineering",
  "user.clearance_level": ">= 3",
  "resource.classification": "internal",
  "time.hour": "9-17",
  "ip.cidr": "10.0.0.0/8"
}
```

| Operator | Syntax | Example |
|----------|--------|---------|
| Equals | `value` | `"engineering"` |
| Not equals | `!=value` | `"!=engineering"` |
| Greater than | `>=N` | `">=3"` |
| Less than | `<=N` | `"<=5"` |
| Range | `N-M` | `"9-17"` (time hours) |
| CIDR | `cidr` | `"10.0.0.0/8"` |
| In list | `[a,b,c]` | `["eng","sales"]` |

### Attribute Sources

| Prefix | Source |
|--------|--------|
| `user.*` | User profile attributes (department, title, location) |
| `resource.*` | Resource metadata (type, classification, owner) |
| `action` | Requested action (read, write, delete) |
| `time.*` | Current time (hour, day_of_week, timezone) |
| `ip.*` | Client IP address |
| `env.*` | Environment variables / config |

### Example ABAC Policy

```json
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

This policy denies engineering users from writing to sensitive documents
outside business hours.

## Policy Check API

### Request

```bash
POST /api/v1/policies/check
{
  "user_id": "uuid-here",
  "resource": "documents:sensitive",
  "action": "read",
  "context": {
    "user.department": "engineering",
    "user.clearance_level": 5,
    "resource.classification": "internal"
  }
}
```

### Response

```json
{
  "allowed": true,
  "reason": "Role 'editor' grants 'read' on 'documents:*' (inherited from parent role 'staff')"
}
```

## Performance Optimization

### Role Resolution Caching

User roles (including inherited) are cached in Redis:
- Key: `ggid:roles:{tenant_id}:{user_id}`
- TTL: 5 minutes
- Invalidated on role assignment change

### Policy Indexing

Policies are indexed by `(tenant_id, resource_pattern)` in PostgreSQL for
fast lookup. Only policies matching the requested resource are loaded.

### Evaluation Complexity

| Step | Complexity |
|------|-----------|
| Role lookup (cached) | O(1) — Redis |
| Role resolution (uncached) | O(d) where d = hierarchy depth |
| RBAC permission check | O(r × p) where r = roles, p = permissions |
| ABAC policy evaluation | O(n) where n = matching policies |

Typical: < 3ms per check (p95).

## Policy Import/Export

```bash
# Export all policies
GET /api/v1/policies/export?tenant_id=...

# Import policies
POST /api/v1/policies/import
[
  {
    "name": "Imported Policy",
    "effect": "allow",
    "actions": ["read"],
    "resources": ["reports:*"]
  }
]
```

## Comparison with Other Platforms

| Feature | Auth0 | Keycloak | GGID |
|---------|-------|----------|------|
| RBAC | ✅ (Roles) | ✅ (Realm Roles) | ✅ |
| Role hierarchy | ❌ | ✅ (Composite) | ✅ |
| ABAC | ❌ | ❌ | ✅ |
| Attribute conditions | ❌ | Script-based | ✅ (JSON conditions) |
| Policy import/export | ❌ | ✅ (JSON) | ✅ |
| Wildcard resources | ❌ | ❌ | ✅ |
| Multi-tenant scoping | ✅ (Organizations) | ✅ (Realms) | ✅ (RLS) |
