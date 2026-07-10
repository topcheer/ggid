# SCIM 2.0 Provisioning Protocol Reference

Complete protocol-level reference for SCIM 2.0 (RFC 7643/7644) provisioning
in GGID. Covers /Users and /Groups CRUD, bulk operations, PATCH semantics,
SCIM filter syntax, and pagination.

> For a quick-start tutorial, see [SCIM Provisioning Guide](scim-provisioning.md).

---

## Table of Contents

- [Protocol Overview](#protocol-overview)
- [Content Types](#content-types)
- [Users Resource](#users-resource)
- [Groups Resource](#groups-resource)
- [Bulk Operations](#bulk-operations)
- [PATCH Operations](#patch-operations)
- [SCIM Filter Syntax](#scim-filter-syntax)
- [Pagination](#pagination)
- [Sorting](#sorting)
- [ETag and Concurrency](#etag-and-concurrency)
- [Error Handling](#error-handling)

---

## Protocol Overview

SCIM (System for Cross-domain Identity Management) 2.0 is defined by:

| RFC | Title |
|-----|-------|
| RFC 7643 | SCIM Core Schema (Users, Groups, Enterprise User) |
| RFC 7644 | SCIM Protocol (CRUD, Bulk, PATCH, Search) |

### Base URL

```
https://iam.example.com/scim/v2
```

### Authentication

```
Authorization: Bearer <scim-api-key-or-jwt>
```

### Supported Operations

| Method | Path | Operation |
|--------|------|-----------|
| `POST` | `/Users` | Create user |
| `GET` | `/Users/{id}` | Retrieve user |
| `GET` | `/Users` | Search/list users |
| `PUT` | `/Users/{id}` | Replace user |
| `PATCH` | `/Users/{id}` | Modify user |
| `DELETE` | `/Users/{id}` | Delete user |
| `POST` | `/Groups` | Create group |
| `GET` | `/Groups/{id}` | Retrieve group |
| `GET` | `/Groups` | Search/list groups |
| `PUT` | `/Groups/{id}` | Replace group |
| `Patch` | `/Groups/{id}` | Modify group |
| `DELETE` | `/Groups/{id}` | Delete group |
| `POST` | `/Bulk` | Bulk operations |
| `GET` | `/ServiceProviderConfig` | SP config |
| `GET` | `/ResourceTypes` | Resource type definitions |
| `GET` | `/Schemas` | Schema definitions |

---

## Content Types

### Request

```
Content-Type: application/scim+json
```

> GGID also accepts `application/json` for compatibility.

### Response

```
Content-Type: application/scim+json; charset=utf-8
```

---

## Users Resource

### User Schema (RFC 7643 Core + Enterprise)

```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "externalId": "emp-12345",
  "userName": "jane.doe@example.com",
  "name": {
    "formatted": "Jane Doe",
    "familyName": "Doe",
    "givenName": "Jane",
    "middleName": "Marie",
    "honorificPrefix": "Dr.",
    "honorificSuffix": "PhD"
  },
  "displayName": "Jane Doe",
  "nickName": "JD",
  "profileUrl": "https://example.com/~jane",
  "title": "Senior Engineer",
  "userType": "Employee",
  "preferredLanguage": "en-US",
  "locale": "en_US",
  "timezone": "America/New_York",
  "active": true,
  "password": "TempPass123!",
  "emails": [
    {
      "value": "jane.doe@example.com",
      "type": "work",
      "primary": true
    },
    {
      "value": "jane.personal@gmail.com",
      "type": "home"
    }
  ],
  "phoneNumbers": [
    {
      "value": "+1-555-123-4567",
      "type": "work",
      "primary": true
    },
    {
      "value": "+1-555-987-6543",
      "type": "mobile"
    }
  ],
  "addresses": [
    {
      "type": "work",
      "streetAddress": "123 Main St",
      "locality": "San Francisco",
      "region": "CA",
      "postalCode": "94105",
      "country": "USA",
      "primary": true
    }
  ],
  "groups": [
    {
      "value": "group-id-1",
      "display": "Engineering",
      "type": "direct"
    }
  ],
  "x509Certificates": [
    {
      "value": "MIIDQjCCAiqgAwIBAg..."
    }
  ],
  "meta": {
    "resourceType": "User",
    "created": "2024-01-15T10:00:00Z",
    "lastModified": "2024-01-20T15:30:00Z",
    "location": "https://iam.example.com/scim/v2/Users/550e8400-...",
    "version": "W/\"a330bc54f0671c9\""
  }
}
```

### Enterprise User Extension

```json
{
  "schemas": [
    "urn:ietf:params:scim:schemas:core:2.0:User",
    "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
  ],
  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User": {
    "employeeNumber": "12345",
    "costCenter": "CC-1001",
    "organization": "Engineering",
    "division": "Platform",
    "department": "Identity",
    "manager": {
      "value": "manager-user-id",
      "displayName": "John Smith"
    }
  }
}
```

### Create User

```
POST /scim/v2/Users
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "userName": "jane.doe@example.com",
  "active": true,
  "name": {
    "givenName": "Jane",
    "familyName": "Doe"
  },
  "emails": [{
    "value": "jane.doe@example.com",
    "type": "work",
    "primary": true
  }]
}
```

**Response**: `201 Created` with full user representation.

### Retrieve User

```
GET /scim/v2/Users/550e8400-e29b-41d4-a716-446655440000
```

**Response**: `200 OK` with user representation.

### List Users

```
GET /scim/v2/Users
```

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
  "totalResults": 2,
  "startIndex": 1,
  "itemsPerPage": 2,
  "Resources": [
    { "id": "user-1", "userName": "alice@example.com", ... },
    { "id": "user-2", "userName": "bob@example.com", ... }
  ]
}
```

### Replace User (PUT)

`PUT` replaces the entire resource. Omitted attributes are removed (except
`id`, read-only attrs, and `password`).

```
PUT /scim/v2/Users/550e8400-e29b-41d4-a716-446655440000
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "userName": "jane.doe@example.com",
  "active": true,
  "name": {
    "givenName": "Jane",
    "familyName": "Smith"
  },
  "emails": [{
    "value": "jane.smith@example.com",
    "primary": true
  }]
}
```

### Delete User

```
DELETE /scim/v2/Users/550e8400-e29b-41d4-a716-446655440000
```

**Response**: `204 No Content`

---

## Groups Resource

### Group Schema

```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"],
  "id": "group-id-1",
  "displayName": "Engineering",
  "externalId": "ad-engineering",
  "members": [
    {
      "value": "550e8400-e29b-41d4-a716-446655440000",
      "display": "Jane Doe",
      "type": "User"
    },
    {
      "value": "group-id-2",
      "display": "Backend Team",
      "type": "Group"
    }
  ],
  "meta": {
    "resourceType": "Group",
    "created": "2024-01-15T10:00:00Z",
    "lastModified": "2024-01-20T15:30:00Z",
    "location": "https://iam.example.com/scim/v2/Groups/group-id-1"
  }
}
```

### Create Group

```
POST /scim/v2/Groups
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"],
  "displayName": "Engineering",
  "members": [
    {
      "value": "550e8400-e29b-41d4-a716-446655440000",
      "type": "User"
    }
  ]
}
```

### Add Member via PATCH

```
PATCH /scim/v2/Groups/group-id-1
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
  "Operations": [
    {
      "op": "add",
      "path": "members",
      "value": [
        {
          "value": "new-user-id",
          "type": "User"
        }
      ]
    }
  ]
}
```

### Remove Member via PATCH

```
PATCH /scim/v2/Groups/group-id-1
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
  "Operations": [
    {
      "op": "remove",
      "path": "members[value eq \"user-id-to-remove\"]"
    }
  ]
}
```

---

## Bulk Operations

The `/Bulk` endpoint processes multiple operations in a single request.

### Request

```
POST /scim/v2/Bulk
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
  "failOnErrors": 5,
  "Operations": [
    {
      "method": "POST",
      "path": "/Users",
      "bulkId": "user-1",
      "data": {
        "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
        "userName": "alice@example.com",
        "name": { "givenName": "Alice" }
      }
    },
    {
      "method": "POST",
      "path": "/Users",
      "bulkId": "user-2",
      "data": {
        "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
        "userName": "bob@example.com",
        "name": { "givenName": "Bob" }
      }
    },
    {
      "method": "POST",
      "path": "/Groups",
      "bulkId": "group-1",
      "data": {
        "schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"],
        "displayName": "New Team"
      }
    },
    {
      "method": "PATCH",
      "path": "/Groups/bulkId:group-1",
      "data": {
        "Operations": [{
          "op": "add",
          "path": "members",
          "value": [
            { "value": "bulkId:user-1", "type": "User" },
            { "value": "bulkId:user-2", "type": "User" }
          ]
        }]
      }
    }
  ]
}
```

### Response

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkResponse"],
  "Operations": [
    {
      "method": "POST",
      "bulkId": "user-1",
      "location": "https://iam.example.com/scim/v2/Users/alice-id",
      "status": { "code": "201", "description": "Created" }
    },
    {
      "method": "POST",
      "bulkId": "user-2",
      "location": "https://iam.example.com/scim/v2/Users/bob-id",
      "status": { "code": "201", "description": "Created" }
    },
    {
      "method": "POST",
      "bulkId": "group-1",
      "location": "https://iam.example.com/scim/v2/Groups/team-id",
      "status": { "code": "201", "description": "Created" }
    },
    {
      "method": "PATCH",
      "location": "https://iam.example.com/scim/v2/Groups/team-id",
      "status": { "code": "200", "description": "OK" }
    }
  ]
}
```

### Bulk ID References

The `bulkId` enables cross-operation references within the same bulk request.
GGID resolves `bulkId:` references before processing dependent operations.

### Error Handling in Bulk

| Setting | Behavior |
|---------|----------|
| `failOnErrors: null` | Process all operations; return errors in response |
| `failOnErrors: 1` | Stop on first error |
| `failOnErrors: N` | Stop after N errors |

### Limits

| Limit | Value |
|-------|-------|
| Max operations per request | 1000 |
| Max payload size | 1 MB |
| Max bulkId reference depth | 1 (no chained references) |

---

## PATCH Operations

PATCH provides partial modifications without requiring a full resource PUT.

### Operations

| Operation | Description |
|-----------|-------------|
| `add` | Add new value(s) to a multi-valued attribute |
| `replace` | Replace value(s) of an attribute |
| `remove` | Remove value(s) from an attribute |

### Add Example

```json
{
  "op": "add",
  "path": "emails",
  "value": [
    {
      "value": "jane.alt@example.com",
      "type": "work"
    }
  ]
}
```

### Replace Example

```json
{
  "op": "replace",
  "path": "name.familyName",
  "value": "Smith"
}
```

### Replace with Filter

```json
{
  "op": "replace",
  "path": "emails[type eq \"work\"].value",
  "value": "jane.new@example.com"
}
```

### Remove Example

```json
{
  "op": "remove",
  "path": "emails[type eq \"home\"]"
}
```

### Multiple Operations

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
  "Operations": [
    {
      "op": "replace",
      "path": "active",
      "value": false
    },
    {
      "op": "replace",
      "path": "displayName",
      "value": "Jane Smith (Inactive)"
    },
    {
      "op": "remove",
      "path": "password"
    },
    {
      "op": "add",
      "path": "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department",
      "value": "Former Engineering"
    }
  ]
}
```

### PATCH Response

On success: `200 OK` with the updated resource (if `Accept` includes the
resource type) or `204 No Content`.

---

## SCIM Filter Syntax

SCIM filters use a subset of filter expressions defined in RFC 7644 Section 3.4.2.2.

### Comparison Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equals | `userName eq "jane@example.com"` |
| `ne` | Not equals | `active ne true` |
| `co` | Contains | `displayName co "Jane"` |
| `sw` | Starts with | `userName sw "jane"` |
| `ew` | Ends with | `userName ew "@example.com"` |
| `pr` | Present (has value) | `emails pr` |
| `gt` | Greater than | `meta.lastModified gt "2024-01-01T00:00:00Z"` |
| `ge` | Greater or equal | `age ge 18` |
| `lt` | Less than | `age lt 65` |
| `le` | Less or equal | `age le 64` |

### Logical Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `and` | Logical AND | `active eq true and userType eq "Employee"` |
| `or` | Logical OR | `title eq "Manager" or title eq "Director"` |
| `not` | Logical NOT | `not (active eq false)` |

### Grouping

Parentheses control precedence:

```
(userType eq "Employee" and (title eq "Manager" or title eq "Director"))
```

### Complex Attribute Filters

Filter on sub-attributes of complex/multi-valued attributes:

```
emails[type eq "work" and value co "@example.com"].display

addresses[type eq "work"].locality eq "San Francisco"
```

### Common Filter Examples

```
# Find all active employees
GET /scim/v2/Users?filter=active eq true and userType eq "Employee"

# Find users in a department
GET /scim/v2/Users?filter=urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department eq "Engineering"

# Find by email
GET /scim/v2/Users?filter=emails[type eq "work" and value eq "jane@example.com"]

# Find users modified since date
GET /scim/v2/Users?filter=meta.lastModified gt "2024-01-01T00:00:00Z"

# Find users NOT in a group
GET /scim/v2/Users?filter=not (groups.value eq "group-id-1")
```

### URL Encoding

Filters must be URL-encoded in query strings:

```
# Raw filter:
filter=active eq true and userName sw "jane"

# URL-encoded:
filter=active%20eq%20true%20and%20userName%20sw%20%22jane%22
```

---

## Pagination

### Request Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `startIndex` | 1-based index of first result | 1 |
| `count` | Maximum results per page | 100 (max 1000) |

```
GET /scim/v2/Users?startIndex=1&count=50
```

### Response

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
  "totalResults": 1523,
  "startIndex": 1,
  "itemsPerPage": 50,
  "Resources": [ ... ]
}
```

### Pagination Logic

```
totalResults = 1523
startIndex  = 101
count       = 50

→ Returns items 101-150
→ itemsPerPage = 50
```

### Cursor-Based Pagination (Extension)

For very large datasets, GGID supports cursor-based pagination:

```
GET /scim/v2/Users?count=50&cursor=eyJpZCI6IjE1MSJ9
```

```json
{
  "totalResults": 1523,
  "itemsPerPage": 50,
  "cursor": "eyJpZCI6IjIwMSJ9",
  "Resources": [ ... ]
}
```

---

## Sorting

| Parameter | Description |
|-----------|-------------|
| `sortBy` | Attribute name to sort by |
| `sortOrder` | `ascending` (default) or `descending` |

```
GET /scim/v2/Users?sortBy=name.familyName&sortOrder=ascending
```

### Sortable Attributes

```
userName
displayName
name.givenName
name.familyName
meta.created
meta.lastModified
```

---

## ETag and Concurrency

GGID supports optimistic concurrency control via ETags.

### Response Headers

```
ETag: W/"a330bc54f0671c9"
```

### Conditional Requests

```
# Retrieve only if changed
GET /scim/v2/Users/user-id
If-None-Match: W/"a330bc54f0671c9"

# Update only if not changed (prevents lost updates)
PUT /scim/v2/Users/user-id
If-Match: W/"a330bc54f0671c9"
```

**Response if precondition fails**: `412 Precondition Failed`

---

## Error Handling

### Error Response Format

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:Error"],
  "detail": "Resource 550e8400-... not found",
  "status": "404",
  "scimType": null
}
```

### Status Codes

| Code | SCIM Meaning |
|------|-------------|
| `400` | Bad request (invalid syntax, filter error) |
| `401` | Unauthorized (missing/invalid auth) |
| `403` | Forbidden (insufficient permissions) |
| `404` | Resource not found |
| `409` | Conflict (e.g., duplicate userName) |
| `412` | Precondition failed (ETag mismatch) |
| `500` | Internal server error |
| `501` | Not implemented (e.g., unsupported filter operator) |

### scimType Values (for 400 errors)

| scimType | Description |
|----------|-------------|
| `invalidFilter` | Filter expression is invalid |
| `tooMany` | Too many results returned |
| `uniqueness` | Attribute uniqueness constraint violated |
| `invalidPath` | PATCH path is invalid |
| `noTarget` | PATCH target doesn't exist |
| `invalidValue` | Attribute value is invalid |
| `invalidVers` | Invalid feature in this version |
| `sensitive` | Sensitive attribute accessed without authorization |

### Conflict Example

```
POST /scim/v2/Users
{ "userName": "existing@example.com", ... }

→ 409 Conflict
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:Error"],
  "detail": "userName 'existing@example.com' already exists",
  "status": "409",
  "scimType": "uniqueness"
}
```

---

## ServiceProviderConfig

```
GET /scim/v2/ServiceProviderConfig
```

```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"],
  "documentationUri": "https://docs.example.com/scim",
  "patch": { "supported": true },
  "bulk": {
    "supported": true,
    "maxOperations": 1000,
    "maxPayloadSize": 1048576
  },
  "filter": {
    "supported": true,
    "maxResults": 1000
  },
  "changePassword": { "supported": true },
  "sort": { "supported": true },
  "etag": { "supported": true },
  "authenticationSchemes": [
    {
      "name": "OAuth Bearer Token",
      "description": "OAuth 2.0 Bearer Token authentication",
      "type": "oauthbearertoken",
      "specUri": "https://datatracker.ietf.org/doc/html/rfc6750"
    }
  ]
}
```

IdPs (Okta, Azure AD) query this endpoint during connector setup to discover
which SCIM features GGID supports.
