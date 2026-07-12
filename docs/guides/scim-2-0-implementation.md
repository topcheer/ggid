# SCIM 2.0 Implementation Guide

User provisioning, group sync, PATCH operations, filtering, pagination, error handling, bulk operations, and custom schemas.

## Overview

SCIM (System for Cross-domain Identity Management) 2.0 automates user and group lifecycle across applications. GGID implements both SCIM Provider (server) and SCIM Consumer (client) roles.

## Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/scim/v2/Users` | GET, POST | List, search, create users |
| `/scim/v2/Users/{id}` | GET, PUT, PATCH, DELETE | CRUD single user |
| `/scim/v2/Groups` | GET, POST | List, search, create groups |
| `/scim/v2/Groups/{id}` | GET, PUT, PATCH, DELETE | CRUD single group |
| `/scim/v2/Bulk` | POST | Bulk operations |
| `/scim/v2/ServiceProviderConfig` | GET | Capabilities discovery |

## User Provisioning

### Create User

```bash
POST /scim/v2/Users
Authorization: Bearer <scim-token>
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "userName": "jane@corp.com",
  "active": true,
  "emails": [{"value": "jane@corp.com", "primary": true, "type": "work"}],
  "displayName": "Jane Doe",
  "name": {"givenName": "Jane", "familyName": "Doe"},
  "phoneNumbers": [{"value": "+1-555-0100", "type": "mobile"}],
  "groups": [{"value": "grp-engineering", "display": "Engineering"}]
}
# → 201 Created with full user object including id
```

### Get User

```bash
GET /scim/v2/Users/uuid-here
# → 200 with user representation
```

### Update User (PUT — full replace)

```bash
PUT /scim/v2/Users/uuid-here
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "userName": "jane@corp.com",
  "active": true,
  "emails": [{"value": "jane.new@corp.com", "primary": true}],
  "displayName": "Jane Smith"
}
# → 200 with updated user
```

## PATCH Operations

PATCH enables partial updates with operation-based semantics:

```bash
PATCH /scim/v2/Users/uuid-here
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
  "Operations": [
    {
      "op": "replace",
      "path": "emails[type eq \"work\"].value",
      "value": "jane.work@corp.com"
    },
    {
      "op": "add",
      "path": "phoneNumbers",
      "value": [{"value": "+1-555-0200", "type": "home"}]
    },
    {
      "op": "remove",
      "path": "phoneNumbers[type eq \"mobile\"]"
    }
  ]
}
```

### Operation Types

| Op | Syntax | Example |
|----|--------|---------|
| `add` | Add value(s) at path | `{"op":"add","path":"groups","value":[{"value":"grp-sales"}]}` |
| `replace` | Replace value at path | `{"op":"replace","path":"active","value":false}` |
| `remove` | Remove value(s) at path | `{"op":"remove","path":"emails[type eq \"old\"]"}` |

### Path Filter Expressions

```
emails[type eq "work"].value      → Specific email
phoneNumbers[primary eq true]     → Primary phone
groups[display eq "Admins"]       → Group by name
addresses[type eq "home"]         → Home address
name.familyName                   → Simple attribute
```

## Group Sync

### Create Group

```bash
POST /scim/v2/Groups
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"],
  "displayName": "Engineering",
  "members": [
    {"value": "user-uuid-1", "display": "Jane Doe"},
    {"value": "user-uuid-2", "display": "John Smith"}
  ]
}
```

### Add/Remove Members (PATCH)

```bash
PATCH /scim/v2/Groups/grp-engineering
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
  "Operations": [
    {"op": "add", "path": "members", "value": [{"value": "user-uuid-3"}]},
    {"op": "remove", "path": "members[value eq \"user-uuid-2\"]"}
  ]
}
```

## Filtering (GET with query)

```bash
# Exact match
GET /scim/v2/Users?filter=userName eq "jane@corp.com"

# Contains
GET /scim/v2/Users?filter=displayName co "Jane"

# Starts with
GET /scim/v2/Users?filter=emails.value sw "jane"

# Multiple conditions
GET /scim/v2/Users?filter=active eq true and emails.value co "@corp.com"

# Complex
GET /scim/v2/Users?filter=(name.familyName eq "Doe") or (displayName co "Doe")
```

### Supported Operators

| Operator | Meaning |
|----------|---------|
| `eq` | Equals |
| `ne` | Not equals |
| `co` | Contains |
| `sw` | Starts with |
| `ew` | Ends with |
| `pr` | Present (has value) |
| `gt` / `ge` | Greater than / or equal |
| `lt` / `le` | Less than / or equal |
| `and` / `or` / `not` | Logical |

## Pagination

```bash
# Page 1, 100 results per page
GET /scim/v2/Users?startIndex=1&count=100
# → {
#   "schemas": ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
#   "totalResults": 1547,
#   "startIndex": 1,
#   "itemsPerPage": 100,
#   "Resources": [ ... 100 users ... ]
# }

# Next page
GET /scim/v2/Users?startIndex=101&count=100
```

Max `count` is 1000. Default is 100.

## Bulk Operations

```bash
POST /scim/v2/Bulk
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
  "failOnErrors": 5,
  "Operations": [
    {"method": "POST", "path": "/Users", "data": {...}},
    {"method": "PUT", "path": "/Users/uuid", "data": {...}},
    {"method": "PATCH", "path": "/Groups/grp-1", "data": {...}},
    {"method": "DELETE", "path": "/Users/old-uuid"}
  ]
}
# → 200 with per-operation results
```

| Setting | Default | Max |
|---------|---------|-----|
| `failOnErrors` | unlimited | — |
| Max operations per request | — | 1000 |

## Error Handling

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:Error"],
  "status": "409",
  "scimType": "uniqueness",
  "detail": "User with userName 'jane@corp.com' already exists"
}
```

| HTTP Status | scimType | When |
|-------------|----------|------|
| 400 | `invalidSyntax` | Malformed JSON |
| 400 | `invalidFilter` | Bad filter expression |
| 400 | `tooMany` | Count > 1000 |
| 401 | — | Missing/invalid auth |
| 403 | — | Insufficient permissions |
| 404 | — | Resource not found |
| 409 | `uniqueness` | Duplicate attribute |
| 500 | — | Internal server error |

## Custom Schema Extension

```json
{
  "schemas": [
    "urn:ietf:params:scim:schemas:core:2.0:User",
    "urn:ggid:schemas:extension:enterprise:1.0"
  ],
  "userName": "jane@corp.com",
  "urn:ggid:schemas:extension:enterprise:1.0": {
    "department": "Engineering",
    "costCenter": "CC-1001",
    "manager": {"value": "uuid-of-manager"},
    "clearanceLevel": "secret"
  }
}
```

## Authentication

```bash
# Bearer token (recommended)
Authorization: Bearer <scim-bearer-token>

# Basic auth (legacy)
Authorization: Basic base64(apiKey:secret)
```

SCIM tokens are separate from OAuth tokens — they have only SCIM permissions.

## See Also

- [Identity Lifecycle Automation](identity-lifecycle-automation.md)
- [Identity Federation Architecture](identity-federation-architecture.md)
- [Identity Provider Configuration](identity-provider-configuration.md)
- [OAuth Scope Design](oauth-scope-design.md)
