# Policy API Reference

Complete REST API reference for GGID's Policy service — roles, permissions, SoD, ABAC evaluation, dry-run, delegation, and templates.

**Base URL**: `https://api.ggid.example.com/api/v1`

## Roles

### List Roles

```
GET /api/v1/roles?page=1&page_size=20
```

```bash
curl https://api.ggid.example.com/api/v1/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Response** (200):
```json
{
  "items": [
    {
      "id": "role-uuid",
      "key": "admin",
      "name": "Administrator",
      "description": "Full system access",
      "permissions": ["users:write", "roles:write", "audit:read"],
      "created_at": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 5
}
```

### Create Role

```
POST /api/v1/roles
```

| Field | Required | Description |
|-------|----------|-------------|
| `key` | Yes | Unique role key (e.g., `developer`) |
| `name` | Yes | Display name |
| `description` | No | Role description |
| `permissions` | No | Array of `resource:action` strings |

```bash
curl -X POST https://api.ggid.example.com/api/v1/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "key": "developer",
    "name": "Developer",
    "permissions": ["users:read", "policy:check"]
  }'
```

**Errors**: 500 if `key` is empty (UNIQUE constraint on tenant_id+key)

### Get Role

```
GET /api/v1/roles/{id}
```

### Update Role

```
PUT /api/v1/roles/{id}
```

### Delete Role

```
DELETE /api/v1/roles/{id}
```

## Role Assignment

### Assign Role to User

```
POST /api/v1/users/{user_id}/roles
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/users/$USER_ID/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"role_id": "role-uuid"}'
```

### Revoke Role

```
DELETE /api/v1/users/{user_id}/roles/{role_id}
```

### Get User Roles

```
GET /api/v1/users/{user_id}/roles
```

**Response**:
```json
{
  "roles": [
    { "id": "uuid", "key": "admin", "name": "Administrator" }
  ]
}
```

## Permissions

### List Permissions

```
GET /api/v1/permissions
```

**Response**:
```json
{
  "permissions": [
    { "resource": "users", "action": "read", "description": "View users" },
    { "resource": "users", "action": "write", "description": "Create/update users" },
    { "resource": "roles", "action": "write", "description": "Manage roles" },
    { "resource": "audit", "action": "read", "description": "View audit log" },
    { "resource": "policy", "action": "check", "description": "Evaluate policies" }
  ]
}
```

## Policy Evaluation

### Check Permission

```
POST /api/v1/policies/check
```

| Field | Required | Description |
|-------|----------|-------------|
| `user_id` | Yes | User UUID |
| `resource` | Yes | Resource identifier (e.g., `document:report.pdf`) |
| `action` | Yes | Action (e.g., `read`, `write`, `delete`) |
| `context` | No | Additional ABAC attributes |

```bash
curl -X POST https://api.ggid.example.com/api/v1/policies/check \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "user_id": "user-uuid",
    "resource": "api:users",
    "action": "read",
    "context": { "ip": "192.168.1.50", "time": "business_hours" }
  }'
```

**Response**:
```json
{
  "allowed": true,
  "reason": "role:developer permits users:read",
  "policy_id": "policy-uuid",
  "matched_rules": ["rule-1"]
}
```

### Dry-Run Policy

```
POST /api/v1/policies/dry-run
```

Evaluates a policy definition without persisting it.

```bash
curl -X POST https://api.ggid.example.com/api/v1/policies/dry-run \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "rules": [
      {
        "effect": "allow",
        "resource": "document:*",
        "action": "read",
        "conditions": [
          { "attribute": "department", "operator": "eq", "value": "engineering" }
        ]
      }
    ],
    "test_cases": [
      { "user_id": "uuid", "resource": "document:spec", "action": "read" }
    ]
  }'
```

**Response**:
```json
{
  "results": [
    { "user_id": "uuid", "resource": "document:spec", "action": "read", "allowed": true }
  ]
}
```

## Policy Management

### List Policies

```
GET /api/v1/policies
```

### Create Policy

```
POST /api/v1/policies
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/policies \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "Engineering docs access",
    "effect": "allow",
    "resource": "document:*",
    "action": "read",
    "conditions": [
      { "group": "condition_groups", "operator": "and", "items": [
        { "attribute": "department", "operator": "eq", "value": "engineering" }
      ]}
    ]
  }'
```

### Delete Policy

```
DELETE /api/v1/policies/{id}
```

## Segregation of Duties (SoD)

### List SoD Rules

```
GET /api/v1/policies/sod
```

### Create SoD Rule

```
POST /api/v1/policies/sod
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/policies/sod \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "No self-grant",
    "conflicting_roles": ["admin", "auditor"],
    "action": "deny"
  }'
```

### Check SoD Violation

```
POST /api/v1/policies/sod/check
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/policies/sod/check \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"user_id": "uuid", "proposed_role_id": "role-uuid"}'
```

**Response**:
```json
{
  "violated": false,
  "conflicts": []
}
```

## Delegation

### Create Delegation

```
POST /api/v1/policies/delegation
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/policies/delegation \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "from_user_id": "manager-uuid",
    "to_user_id": "delegate-uuid",
    "role_id": "approver-uuid",
    "expires_at": "2025-06-30T23:59:59Z"
  }'
```

### List Delegations

```
GET /api/v1/policies/delegation?user_id={user_id}
```

### Revoke Delegation

```
DELETE /api/v1/policies/delegation/{id}
```

## Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | `invalid_argument` | Malformed request |
| 401 | `unauthorized` | Invalid token |
| 403 | `forbidden` | Insufficient scope |
| 404 | `not_found` | Role/policy not found |
| 409 | `already_exists` | Duplicate key |
| 422 | `sod_violation` | SoD conflict |

## See Also

- [REST API Reference](rest-api.md)
- [Audit API](audit-api.md)
- [Delegation Guide](../guides/delegation-guide.md)
- [Access Reviews](../guides/access-reviews.md)
