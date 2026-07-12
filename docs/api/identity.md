# Identity Service API Reference

Complete REST API reference for GGID's Identity service — users, groups, roles, lock/unlock, and SCIM 2.0.

**Base URL**: `https://api.ggid.example.com`

## Users

### List Users

```
GET /api/v1/users?page=1&page_size=20&search=alice&status=active
```

**Response** (200):
```json
{
  "items": [{
    "id": "uuid", "username": "alice", "email": "alice@example.com",
    "name": "Alice Chen", "phone": "+1234567890", "status": "active",
    "created_at": "2025-01-01T00:00:00Z", "last_login": "2025-01-24T14:30:00Z"
  }],
  "page": 1, "page_size": 20, "total": 142, "total_pages": 8
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
```json
{"username":"bob","email":"bob@example.com","password":"SecurePass1!","name":"Bob Smith"}
```
**Response** (201): `{"id":"uuid","username":"bob","status":"active"}`

### Update User
```
PUT /api/v1/users/{id}
```
```json
{"email":"new@example.com","phone":"+1234567890","name":"Bob Smith Jr"}
```

### Delete User
```
DELETE /api/v1/users/{id}
```
Soft-delete (marks inactive).

### Lock / Unlock
```
POST /api/v1/users/{id}/lock   {"reason":"Security investigation"}
POST /api/v1/users/{id}/unlock
```

### Import / Export
```
POST /api/v1/users/import
GET /api/v1/users/export?format=csv
```

## Groups

### List Groups
```
GET /api/v1/groups
```

### Create Group
```
POST /api/v1/groups
{"name":"Engineering","description":"Engineering team"}
```

### Add/Remove Members
```
POST /api/v1/groups/{id}/members   {"user_id":"uuid"}
DELETE /api/v1/groups/{id}/members/{user_id}
```

## Roles

### List/Create/Delete
```
GET /api/v1/roles
POST /api/v1/roles   {"key":"developer","name":"Developer"}
DELETE /api/v1/roles/{id}
```

### Assign/Revoke
```
POST /api/v1/users/{user_id}/roles   {"role_id":"uuid"}
DELETE /api/v1/users/{user_id}/roles/{role_id}
GET /api/v1/users/{user_id}/roles
```

## SCIM 2.0

### User CRUD (SCIM)
```
GET /scim/v2/Users?filter=userName eq "alice"
POST /scim/v2/Users
PATCH /scim/v2/Users/{id}
DELETE /scim/v2/Users/{id}
```

### SCIM Bulk
```
POST /scim/v2/Bulk
{"schemas":["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],"Operations":[...]}
```

## Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | `invalid_argument` | Malformed request |
| 409 | `already_exists` | Duplicate username/email |
| 404 | `not_found` | User not found |

## See Also
- [Identity API (detailed)](identity-api.md)
- [REST API Reference](rest-api.md)
- [Policy API](policy.md)
