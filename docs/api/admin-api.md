# Admin API Reference

> Complete reference for GGID administration endpoints. All require admin JWT with `X-Tenant-ID` header.

---

## Authentication

All admin endpoints require:

```
Authorization: Bearer <admin-jwt>
X-Tenant-ID: <tenant-uuid>
```

Admin JWT must have `admin` role or `*:*` scope.

---

## User Management

### Create User

```
POST /api/v1/users
```

**Request:**
```json
{
  "username": "newuser",
  "email": "user@example.com",
  "password": "SecurePass123!",
  "display_name": "New User",
  "status": "active"
}
```

**Response (201):**
```json
{
  "id": "usr_abc123",
  "username": "newuser",
  "email": "user@example.com",
  "status": "active",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "created_at": "2025-07-11T12:00:00Z"
}
```

**Errors:** 400 (invalid input), 409 (username exists)

### List Users

```
GET /api/v1/users?page=1&page_size=20&search=alice
```

**Response (200):**
```json
{
  "users": [...],
  "total": 42,
  "page": 1,
  "page_size": 20
}
```

### Get User

```
GET /api/v1/users/{user_id}
```

### Update User

```
PUT /api/v1/users/{user_id}
```

**Request:**
```json
{
  "email": "newemail@example.com",
  "display_name": "Updated Name",
  "status": "active"
}
```

### Delete User

```
DELETE /api/v1/users/{user_id}
```

### Assign Role to User

```
POST /api/v1/users/{user_id}/roles
```

**Request:**
```json
{
  "role_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Get User Permissions

```
GET /api/v1/users/{user_id}/permissions
```

**Response (200):**
```json
{
  "permissions": ["read:users", "write:users"],
  "roles": ["editor"]
}
```

---

## Role Management

### Create Role

```
POST /api/v1/roles
```

**Request:**
```json
{
  "name": "Editor",
  "key": "editor",
  "description": "Can read and write users"
}
```

**Response (201):**
```json
{
  "id": "550e8400-...",
  "name": "Editor",
  "key": "editor",
  "tenant_id": "..."
}
```

**Errors:** 400 (empty key), 409 (key already exists)

### List Roles

```
GET /api/v1/roles
```

### Assign Permissions to Role

```
POST /api/v1/roles/{role_id}/permissions
```

**Request:**
```json
{
  "permissions": ["read:users", "write:users"]
}
```

### Set Parent Role (Hierarchy)

```
POST /api/v1/roles/{role_id}/parent
```

**Request:**
```json
{
  "parent_id": "parent-role-uuid"
}
```

---

## Organization Management

### Create Organization

```
POST /api/v1/orgs
```

**Request:**
```json
{
  "name": "Engineering",
  "description": "Engineering Department",
  "parent_id": ""
}
```

**Response (201):**
```json
{
  "id": "org_abc123",
  "name": "Engineering",
  "tenant_id": "..."
}
```

### List Organizations

```
GET /api/v1/orgs
```

### Get Organization

```
GET /api/v1/orgs/{org_id}
```

### Update Organization

```
PUT /api/v1/orgs/{org_id}
```

### Delete Organization

```
DELETE /api/v1/orgs/{org_id}
```

---

## Policy Management

### Create Policy (ABAC)

```
POST /api/v1/policies
```

**Request:**
```json
{
  "name": "Deny outside business hours",
  "effect": "deny",
  "actions": ["read", "write"],
  "resources": ["financial_data"],
  "priority": 300
}
```

### Check Permission

```
POST /api/v1/policies/check
```

**Request:**
```json
{
  "user_id": "usr_abc123",
  "action": "write",
  "resource": "users",
  "resource_type": "users"
}
```

**Response (200):**
```json
{
  "allowed": true,
  "reason": "role_permission_match"
}
```

### Evaluate with Attributes

```
POST /api/v1/policies/evaluate
```

### Dry-Run Test

```
POST /api/v1/policies/dry-run
```

### Apply Compliance Template

```
POST /api/v1/policies/from-template/{template_id}
```

Templates: `pci-dss`, `hipaa`, `soc2`, `gdpr`

---

## Audit Query

### Query Audit Events

```
GET /api/v1/audit/events?limit=50&offset=0&actor_type=user&action=login
```

**Query Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `limit` | int | Results per page (default 50, max 200) |
| `offset` | int | Pagination offset |
| `actor_type` | string | Filter: `user`, `system`, `api_key` |
| `action` | string | Filter: `login`, `register`, `delete`, etc. |
| `resource_type` | string | Filter: `users`, `roles`, `orgs` |
| `start_time` | ISO8601 | Events after this time |
| `end_time` | ISO8601 | Events before this time |

**Response (200):**
```json
{
  "events": [
    {
      "id": "evt_001",
      "tenant_id": "...",
      "actor_type": "user",
      "actor_id": "usr_abc123",
      "action": "login",
      "resource_type": "auth",
      "resource_id": "",
      "metadata": {"ip": "192.168.1.1"},
      "timestamp": "2025-07-11T12:00:00Z",
      "hash": "abc123..."
    }
  ],
  "total": 15423
}
```

### Verify Hash Chain Integrity

```
GET /api/v1/audit/verify
```

**Response (200):**
```json
{
  "verified": true,
  "total_events": 15423,
  "tampered_events": []
}
```

---

## Admin System

### Gateway Statistics

```
GET /api/v1/admin/stats
```

**Response (200):**
```json
{
  "uptime": "72h15m",
  "requests_total": 154223,
  "routes": 84,
  "services": 7
}
```

### List Route Configuration

```
GET /api/v1/admin/routes
```

### Toggle Route (Enable/Disable)

```
POST /api/v1/admin/routes/{prefix}/toggle
```

---

## Common Error Codes

| Code | HTTP Status | Meaning |
|------|-------------|---------|
| `unauthorized` | 401 | Missing or invalid JWT |
| `forbidden` | 403 | Insufficient role/scope |
| `not_found` | 404 | Resource doesn't exist |
| `conflict` | 409 | Duplicate resource |
| `validation_error` | 400 | Invalid input |
| `rate_limited` | 429 | Too many requests |
| `internal_error` | 500 | Server error |

---

*See: [API Reference](../api-reference.md) | [Error Codes](error-codes.md) | [RBAC Guide](../guides/role-based-access.md) | [ABAC Guide](../guides/abac-policy.md)*

*Last updated: 2025-07-11*
