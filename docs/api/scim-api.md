# SCIM 2.0 API Reference

> SCIM (System for Cross-domain Identity Management) 2.0 endpoints for enterprise HR system integration.

---

## Base URL

```
http://localhost:8080/scim/v2
```

## Authentication

```
Authorization: Bearer <admin-jwt>
```

Content-Type for all requests: `application/scim+json`

---

## Users

### List Users

```
GET /scim/v2/Users?startIndex=1&count=20&filter=userName eq "alice"
```

**Query Parameters:**

| Param | Description |
|-------|-------------|
| `startIndex` | 1-based pagination offset (default 1) |
| `count` | Max results (default 100) |
| `filter` | SCIM filter expression (e.g. `userName eq "alice"`) |

**Response (200):**
```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
  "totalResults": 42,
  "startIndex": 1,
  "itemsPerPage": 20,
  "Resources": [
    {
      "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
      "id": "usr_abc123",
      "userName": "alice",
      "emails": [{"value": "alice@example.com", "primary": true}],
      "active": true,
      "displayName": "Alice Smith"
    }
  ]
}
```

### Get User

```
GET /scim/v2/Users/{id}
```

### Create User

```
POST /scim/v2/Users
```

**Request:**
```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "userName": "newuser",
  "emails": [{"value": "new@test.com", "primary": true}],
  "displayName": "New User",
  "active": true,
  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User": {
    "department": "Engineering",
    "employeeNumber": "EMP-001",
    "manager": {"value": "mgr-001", "displayName": "Boss"}
  }
}
```

**Response (201):**
```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "id": "usr_new001",
  "userName": "newuser",
  "active": true
}
```

**Errors:**
| Status | scimType | Meaning |
|-------|----------|---------|
| 400 | `invalidSyntax` | Malformed JSON |
| 400 | `invalidVers` | Unsupported schema version |
| 409 | `uniqueness` | userName already exists |

### Update User (PUT)

```
PUT /scim/v2/Users/{id}
```

Replaces all attributes. Omitted attributes are removed.

### Patch User (PATCH)

```
PATCH /scim/v2/Users/{id}
```

**Request:**
```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
  "Operations": [
    {
      "op": "replace",
      "path": "displayName",
      "value": ""Updated Name""
    },
    {
      "op": "replace",
      "path": "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department",
      "value": ""Security""
    }
  ]
}
```

**Supported Operations:** `replace`, `add`, `remove`

**Path Syntax:**
| Format | Example |
|--------|---------|
| Simple attribute | `displayName` |
| Nested (dot) | `emails[type eq "work"].value` |
| URN + dot | `urn:...:User.department` |
| URN + colon | `urn:...:User:department` |

### Delete User

```
DELETE /scim/v2/Users/{id}
```

**Response:** 204 No Content

---

## Groups

### List Groups

```
GET /scim/v2/Groups?filter=displayName eq "Admins"
```

### Create Group

```
POST /scim/v2/Groups
```

**Request:**
```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"],
  "displayName": "Engineering",
  "members": [
    {"value": "usr_001", "display": "Alice"},
    {"value": "usr_002", "display": "Bob"}
  ]
}
```

### Get / Update / Patch / Delete Group

Same patterns as Users: `GET/PUT/PATCH/DELETE /scim/v2/Groups/{id}`

---

## Bulk Operations

```
POST /scim/v2/Bulk
```

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
  "failOnErrors": 1,
  "Operations": [
    {"method": "POST", "path": "/Users", "data": {...}},
    {"method": "PATCH", "path": "/Groups/abc", "data": {...}}
  ]
}
```

---

## Service Provider Config

```
GET /scim/v2/ServiceProviderConfig
```

Returns supported features: filtering, patch, sort, etag, bulk.

---

## Resource Types

```
GET /scim/v2/ResourceTypes
```

Returns schema definitions for User and Group resources.

---

## Supported Filter Operators

| Operator | Example |
|----------|---------|
| `eq` | `userName eq "alice"` |
| `ne` | `active ne false` |
| `co` | `emails co "@example.com"` |
| `sw` | `userName sw "a"` |
| `pr` | `displayName pr` (present) |
| `and` | `active eq true and userName sw "a"` |
| `or` | `department eq "Eng" or department eq "Sales"` |

---

*See: [Admin API](admin-api.md) | [API Reference](../api-reference.md) | [Identity Service](../architecture/microservices.md)*

*Last updated: 2025-07-11*
