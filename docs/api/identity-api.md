# Identity API Reference

Complete REST API reference for GGID's Identity service — user CRUD, lock/unlock, search, SCIM provisioning, and account linking.

**Base URL**: `https://api.ggid.example.com/api/v1`

## User CRUD

### List Users

```
GET /api/v1/users
```

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | int | Page (default 1) |
| `page_size` | int | Per page (max 100, default 20) |
| `search` | string | Search username/email |
| `status` | string | `active` / `locked` / `suspended` |
| `role` | string | Filter by role key |
| `org_id` | string | Filter by org membership |

```bash
curl "https://api.ggid.example.com/api/v1/users?page=1&page_size=20&search=alice" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Response** (200):
```json
{
  "items": [
    {
      "id": "uuid",
      "username": "alice",
      "email": "alice@example.com",
      "name": "Alice Chen",
      "phone": "+1234567890",
      "status": "active",
      "created_at": "2025-01-01T00:00:00Z",
      "last_login": "2025-01-24T14:30:00Z"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 142,
  "total_pages": 8
}
```

### Get User

```
GET /api/v1/users/{id}
```

### Get Current User

```
GET /api/v1/users/me
```

### Create User

```
POST /api/v1/users
```

Requires `users:write` scope.

```bash
curl -X POST https://api.ggid.example.com/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "username": "bob",
    "email": "bob@example.com",
    "password": "SecurePass1!",
    "name": "Bob Smith"
  }'
```

**Response** (201):
```json
{
  "id": "new-user-uuid",
  "username": "bob",
  "email": "bob@example.com",
  "status": "active"
}
```

### Update User

```
PUT /api/v1/users/{id}
```

```bash
curl -X PUT https://api.ggid.example.com/api/v1/users/$USER_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"email": "newemail@example.com", "phone": "+1234567890"}'
```

### Delete User

```
DELETE /api/v1/users/{id}
```

Soft-deletes (marks inactive). Audit event logged.

## User Status Management

### Lock User

```
POST /api/v1/users/{id}/lock
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/users/$USER_ID/lock \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"reason": "Security investigation"}'
```

### Unlock User

```
POST /api/v1/users/{id}/unlock
```

### Suspend User

```
POST /api/v1/users/{id}/suspend
```

## Bulk Operations

### Import Users

```
POST /api/v1/users/import
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/users/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "users": [
      {"username": "user1", "email": "u1@example.com", "password": "Pass1!"},
      {"username": "user2", "email": "u2@example.com", "password": "Pass1!"}
    ]
  }'
```

**Response** (200):
```json
{
  "imported": 2,
  "failed": 0,
  "errors": []
}
```

### Export Users

```
GET /api/v1/users/export?format=csv
```

```bash
curl https://api.ggid.example.com/api/v1/users/export?format=csv \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -o users.csv
```

## User Sessions

### List Sessions

```
GET /api/v1/users/{id}/sessions
```

### Revoke All Sessions

```
DELETE /api/v1/users/{id}/sessions
```

## SCIM 2.0

### Get User (SCIM)

```
GET /scim/v2/Users/{id}
```

### Create User (SCIM)

```
POST /scim/v2/Users
```

```bash
curl -X POST https://api.ggid.example.com/scim/v2/Users \
  -H "Authorization: Bearer $SCIM_TOKEN" \
  -H "Content-Type: application/scim+json" \
  -d '{
    "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
    "userName": "alice@example.com",
    "name": { "givenName": "Alice", "familyName": "Chen" },
    "emails": [{ "value": "alice@example.com", "primary": true }],
    "active": true
  }'
```

### Patch User (SCIM)

```
PATCH /scim/v2/Users/{id}
```

```bash
curl -X PATCH https://api.ggid.example.com/scim/v2/Users/$USER_ID \
  -H "Authorization: Bearer $SCIM_TOKEN" \
  -H "Content-Type: application/scim+json" \
  -d '{
    "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
    "Operations": [
      { "op": "replace", "path": "emails[type eq \"work\"].value", "value": "new@example.com" }
    ]
  }'
```

### Search Users (SCIM)

```
GET /scim/v2/Users?filter=userName eq "alice"
```

**Response** (SCIM format):
```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
  "totalResults": 1,
  "Resources": [
    {
      "id": "uuid",
      "userName": "alice@example.com",
      "active": true
    }
  ]
}
```

### Delete User (SCIM)

```
DELETE /scim/v2/Users/{id}
```

### SCIM Bulk

```
POST /scim/v2/Bulk
```

```bash
curl -X POST https://api.ggid.example.com/scim/v2/Bulk \
  -H "Authorization: Bearer $SCIM_TOKEN" \
  -H "Content-Type: application/scim+json" \
  -d '{
    "schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
    "failOnErrors": 1,
    "Operations": [
      { "method": "POST", "path": "/Users", "data": {...} },
      { "method": "PATCH", "path": "/Groups/xxx", "data": {...} }
    ]
  }'
```

## Account Linking

### Link External Identity

```
POST /api/v1/users/{id}/identities
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/users/$USER_ID/identities \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "provider": "google",
    "external_id": "google-user-id-123",
    "email": "alice@gmail.com"
  }'
```

### List Linked Identities

```
GET /api/v1/users/{id}/identities
```

### Unlink Identity

```
DELETE /api/v1/users/{id}/identities/{provider}
```

## Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | `invalid_argument` | Malformed request |
| 401 | `unauthorized` | Invalid token |
| 403 | `forbidden` | Insufficient scope |
| 404 | `not_found` | User not found |
| 409 | `already_exists` | Duplicate username/email |

## See Also

- [REST API Reference](rest-api.md)
- [Policy API](policy-api.md)
- [OAuth API](oauth-api.md)
- [SCIM Provisioning Guide](../guides/scim-provisioning.md)
