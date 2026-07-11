# Role-Based Access Control (RBAC) Guide

> Complete guide to creating roles, assigning permissions, binding users, and enforcing access checks with the GGID Policy Engine.

---

## Overview

GGID implements RBAC through the **Policy Service**, which exposes REST endpoints for managing roles, permissions, and access checks. Roles group permissions; users are assigned roles; the policy engine evaluates requests at runtime.

```
User → assigned → Role → contains → Permissions
                                    ↓
                         Policy Engine evaluates at request time
```

**Key concepts:**

| Concept | Description |
|---------|-------------|
| **Permission** | A `<action>:<resource>` pair (e.g. `read:users`, `write:roles`) |
| **Role** | A named collection of permissions, scoped to a tenant |
| **Role Hierarchy** | Roles can have parent roles, inheriting their permissions |
| **Policy Check** | Runtime evaluation: does user X have permission for action Y on resource Z? |

---

## Prerequisites

- GGID Gateway running at `http://localhost:8080`
- A JWT with admin scope (see [5-Minute JWT Quickstart](../quickstart/5-minute-jwt.md))
- Default tenant ID: `00000000-0000-0000-0000-000000000001`

---

## 1. Create a Role

Roles group permissions under a named label (e.g. "Editor", "Viewer", "Admin").

```bash
JWT="your-admin-jwt"
TENANT="00000000-0000-0000-0000-000000000001"

curl -s -X POST http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Editor",
    "key": "editor",
    "description": "Can read and write users, but cannot delete"
  }' | jq .
```

**Response (201 Created):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Editor",
  "key": "editor",
  "description": "Can read and write users, but cannot delete",
  "tenant_id": "00000000-0000-0000-0000-000000000001"
}
```

> **Note:** The `key` field must be unique within the tenant. It serves as a human-readable identifier for the role.

### List Roles

```bash
curl -s http://localhost:8080/api/v1/roles?tenant_id=$TENANT \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

---

## 2. Assign Permissions to a Role

Permissions use the format `<action>:<resource>`.

### Available Actions

| Action | Description |
|--------|-------------|
| `read` | View / list resources |
| `write` | Create / update resources |
| `delete` | Remove resources |
| `publish` | Publish events / webhooks |
| `*` | Wildcard — matches all actions |

### Available Resources

| Resource | Description |
|----------|-------------|
| `users` | User accounts |
| `roles` | Role definitions |
| `orgs` | Organizations |
| `audit` | Audit logs |
| `security` | Security settings (MFA, certificates) |
| `*` | Wildcard — matches all resources |

### Assign Permissions

```bash
ROLE_ID="550e8400-e29b-41d4-a716-446655440000"

curl -s -X POST "http://localhost:8080/api/v1/roles/$ROLE_ID/permissions" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "permissions": ["read:users", "write:users", "read:roles"]
  }' | jq .
```

**Response (200 OK):**

```json
{
  "status": "ok",
  "role_id": "550e8400-e29b-41d4-a716-446655440000",
  "permissions": ["read:users", "write:users", "read:roles"]
}
```

### List All Permissions

```bash
curl -s http://localhost:8080/api/v1/permissions?tenant_id=$TENANT \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

---

## 3. Bind a Role to a User

Assign a role to a user through the Identity Service:

```bash
USER_ID="usr_abc123def456"

curl -s -X POST "http://localhost:8080/api/v1/users/$USER_ID/roles" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d "{\"role_id\":\"$ROLE_ID\"}" | jq .
```

### View User's Effective Permissions

```bash
curl -s "http://localhost:8080/api/v1/users/$USER_ID/permissions?tenant_id=$TENANT" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

This returns all permissions the user has, including those inherited through role hierarchy.

---

## 4. Role Hierarchy (Inheritance)

Roles can inherit permissions from parent roles. This allows building permission hierarchies:

```
Admin (all:* permissions)
  └── Manager (read:*, write:users)
       └── Editor (read:users, write:users)
            └── Viewer (read:users)
```

### Set Parent Role

```bash
EDITOR_ROLE_ID="550e8400-e29b-41d4-a716-446655440000"
VIEWER_ROLE_ID="660e8400-e29b-41d4-a716-446655440001"

curl -s -X POST "http://localhost:8080/api/v1/roles/$VIEWER_ROLE_ID/parent" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d "{\"parent_id\":\"$EDITOR_ROLE_ID\"}" | jq .
```

After this, users with the Viewer role automatically inherit all Editor permissions.

---

## 5. Policy Check (Runtime Enforcement)

The policy check endpoint evaluates whether a user has permission for a specific action:

```bash
curl -s -X POST http://localhost:8080/api/v1/policies/check \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "'"$USER_ID"'",
    "resource_type": "users",
    "action": "write",
    "resource": "users"
  }' | jq .
```

**Response (Allowed):**

```json
{
  "allowed": true,
  "reason": "role_permission_match",
  "matched_by": "role:editor"
}
```

**Response (Denied):**

```json
{
  "allowed": false,
  "reason": "no_matching_permission",
  "matched_by": ""
}
```

---

## 6. Pre-Built Role Templates

GGID includes compliance templates with pre-configured roles and policies. See the [ABAC Policy Guide](./abac-policy.md) for template usage.

---

## 7. Enforce RBAC in Your Application

### Go (with SDK)

```go
package main

import (
    "net/http"

    ggid "github.com/ggid/ggid/sdk/go"
)

func main() {
    client := ggid.New("http://localhost:8080", ggid.WithAPIKey(os.Getenv("GGID_API_KEY")))

    mux := http.NewServeMux()

    // Auth middleware verifies JWT
    mux.Handle("/", client.Middleware(http.HandlerFunc(apiHandler)))

    // RequirePermission checks policy engine
    mux.Handle("/api/users", client.RequirePermission("users", "read")(listUsersHandler))
    mux.Handle("/api/users/", client.RequirePermission("users", "write")(editUsersHandler))

    http.ListenAndServe(":8081", mux)
}
```

### Express.js (with Node SDK)

```javascript
const { GGIDMiddleware } = require('@ggid/sdk-node');
const app = require('express')();

app.use('/api', GGIDMiddleware({
  gatewayURL: 'http://localhost:8080',
  secret: process.env.JWT_SECRET,
}));

// Scope-based check
function requireScope(scope) {
  return (req, res, next) => {
    if (!req.ggid?.scopes?.includes(scope)) {
      return res.status(403).json({ error: 'insufficient_scope', required: scope });
    }
    next();
  };
}

app.get('/api/users', requireScope('read:users'), listUsers);
app.delete('/api/users/:id', requireScope('delete:users'), deleteUser);
```

---

## Best Practices

1. **Start with least privilege**: Assign minimal permissions, add more as needed.
2. **Use role hierarchy**: Create base roles (Viewer) and extend upward (Editor, Admin).
3. **Avoid wildcard in production**: `*:*` grants full access — reserve for system-level roles only.
4. **Audit role assignments**: Regularly review who has which roles via the Audit Service.
5. **Use descriptive role names**: `editor`, `billing-admin`, `support-agent` — not `role1`, `role2`.

---

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/roles` | Create a role |
| `GET` | `/api/v1/roles?tenant_id=X` | List roles |
| `GET` | `/api/v1/roles/{id}` | Get role details |
| `DELETE` | `/api/v1/roles/{id}` | Delete a role |
| `POST` | `/api/v1/roles/{id}/permissions` | Assign permissions |
| `POST` | `/api/v1/roles/{id}/parent` | Set parent (inheritance) |
| `GET` | `/api/v1/permissions?tenant_id=X` | List all permissions |
| `POST` | `/api/v1/users/{id}/roles` | Bind role to user |
| `GET` | `/api/v1/users/{id}/permissions` | User's effective permissions |
| `POST` | `/api/v1/policies/check` | Runtime access check |

---

*See also: [ABAC Policy Guide](./abac-policy.md) | [RBAC Quickstart](../quickstart/rbac-permissions.md) | [API Reference](../api-reference.md)*

*Last updated: 2025-07-11*
