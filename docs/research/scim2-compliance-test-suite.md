# SCIM 2.0 Compliance and Test Suite Design

> Comprehensive reference for SCIM 2.0 (RFC 7643/7644) protocol compliance, GGID implementation assessment, and test suite design.

---

## Table of Contents

1. [SCIM 2.0 Overview](#1-scim-20-overview)
2. [Filter Syntax (RFC 7644 Section 3.4.2)](#2-filter-syntax-rfc-7644--342)
3. [Bulk Operations (RFC 7644 Section 3.7)](#3-bulk-operations-rfc-7644--37)
4. [ETag / If-Match / If-None-Match (RFC 7644 Section 3.14)](#4-etag--if-match--if-none-match-rfc-7644--314)
5. [Pagination (RFC 7644 Section 3.4.2)](#5-pagination-rfc-7644--342)
6. [PATCH Operations (RFC 7644 Section 3.5.2)](#6-patch-operations-rfc-7644--352)
7. [Sort (RFC 7644 Section 3.4.2)](#7-sort-rfc-7644--342)
8. [GGID SCIM Compliance Assessment](#8-ggid-scim-compliance-assessment)
9. [Test Suite Design](#9-test-suite-design)

---

## 1. SCIM 2.0 Overview

SCIM (System for Cross-domain Identity Management) is an HTTP-based protocol designed to automate user provisioning, updating, and de-provisioning across applications and identity providers. It is defined by two IETF standards-track RFCs:

| RFC | Title | Scope |
|-----|-------|-------|
| **RFC 7643** | SCIM Core Schema | Resource models (User, Group), attribute definitions, schema extension mechanism |
| **RFC 7644** | SCIM Protocol | REST endpoints, CRUD operations, filtering, pagination, sorting, bulk, PATCH, ETag |

A third RFC, **RFC 7642**, defines the use cases, requirements, and terminology.

### 1.1 Core Resource Types (RFC 7643)

#### User Resource

```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "id": "2819c223-7f76-453a-919d-413861904646",
  "externalId": "701984",
  "userName": "bjensen@example.com",
  "name": {
    "formatted": "Ms. Barbara J Jensen III",
    "familyName": "Jensen",
    "givenName": "Barbara",
    "middleName": "Jane",
    "honorificPrefix": "Ms.",
    "honorificSuffix": "III"
  },
  "displayName": "Barbara Jensen",
  "nickName": "Barb",
  "profileUrl": "https://login.example.com/bjensen",
  "emails": [
    {
      "value": "bjensen@example.com",
      "type": "work",
      "primary": true
    },
    {
      "value": "babs@jensen.org",
      "type": "home"
    }
  ],
  "phoneNumbers": [
    { "value": "555-555-8377", "type": "work" },
    { "value": "555-555-8223", "type": "mobile" }
  ],
  "addresses": [
    {
      "type": "work",
      "streetAddress": "100 Universal City Plaza",
      "locality": "Hollywood",
      "region": "CA",
      "postalCode": "91608",
      "country": "USA",
      "formatted": "100 Universal City Plaza\nHollywood, CA 91608 USA",
      "primary": true
    }
  ],
  "active": true,
  "title": "VP Engineering",
  "userType": "Employee",
  "preferredLanguage": "en",
  "locale": "en-US",
  "timezone": "America/Los_Angeles",
  "groups": [
    {
      "value": "e9e30dba-f08f-4109-8486-d5c6a331660a",
      "$ref": "https://example.com/v2/Groups/e9e30dba-f08f-4109-8486-d5c6a331660a",
      "display": "Tour Guides",
      "type": "direct"
    }
  ],
  "meta": {
    "resourceType": "User",
    "created": "2010-01-23T04:56:22Z",
    "lastModified": "2011-05-13T04:42:34Z",
    "location": "https://example.com/v2/Users/2819c223-7f76-453a-919d-413861904646",
    "version": "W\/\"a330bc54f0671c9\""
  }
}
```

#### Group Resource

```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"],
  "id": "e9e30dba-f08f-4109-8486-d5c6a331660a",
  "displayName": "Tour Guides",
  "members": [
    {
      "value": "2819c223-7f76-453a-919d-413861904646",
      "$ref": "https://example.com/v2/Users/2819c223-7f76-453a-919d-413861904646",
      "display": "Barbara Jensen",
      "type": "User"
    },
    {
      "value": "902c246b-6245-4190-8e05-00816be7344a",
      "$ref": "https://example.com/v2/Users/902c246b-6245-4190-8e05-00816be7344a",
      "display": "Babs Jensen",
      "type": "User"
    }
  ],
  "meta": {
    "resourceType": "Group",
    "created": "2010-01-23T04:56:22Z",
    "lastModified": "2011-05-13T04:42:34Z",
    "location": "https://example.com/v2/Groups/e9e30dba-f08f-4109-8486-d5c6a331660a"
  }
}
```

### 1.2 Enterprise User Extension (RFC 7643 Section 4.3)

The Enterprise User extension adds organizational attributes commonly needed by HR systems:

```json
{
  "schemas": [
    "urn:ietf:params:scim:schemas:core:2.0:User",
    "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
  ],
  "id": "4c98a632-afa2-45ac-afc6-6c4f61d22e5d",
  "userName": "kcheng@example.com",
  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User": {
    "employeeNumber": "701984",
    "costCenter": "4130",
    "organization": "Universal Studios",
    "division": "Theme Park",
    "department": "Tour Operations",
    "manager": {
      "value": "26118915-6010-4e2f-8f7b-8e7b8b8f8e7b",
      "$ref": "https://example.com/v2/Users/26118915-6010-4e2f-8f7b-8e7b8b8f8e7b",
      "displayName": "Barbara Jensen"
    }
  }
}
```

### 1.3 Standard Schema URNs

| URN | Purpose |
|-----|---------|
| `urn:ietf:params:scim:schemas:core:2.0:User` | Core User schema |
| `urn:ietf:params:scim:schemas:core:2.0:Group` | Core Group schema |
| `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User` | Enterprise User extension |
| `urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig` | Service provider capabilities |
| `urn:ietf:params:scim:schemas:core:2.0:ResourceType` | Resource type definition |
| `urn:ietf:params:scim:api:messages:2.0:ListResponse` | Paginated list response |
| `urn:ietf:params:scim:api:messages:2.0:Error` | Error response |
| `urn:ietf:params:scim:api:messages:2.0:PatchOp` | PATCH operation request |

### 1.4 Content-Type and Media Types

All SCIM requests and responses MUST use one of:

- `application/scim+json` — standard SCIM media type (RFC 7644 Section 2.1)
- `application/json` — acceptable per RFC, but `scim+json` is preferred

### 1.5 Standard Endpoints

| Endpoint | Methods | Description |
|----------|---------|-------------|
| `/Users` | GET, POST | List/create users |
| `/Users/{id}` | GET, PUT, PATCH, DELETE | Retrieve/replace/modify/delete user |
| `/Groups` | GET, POST | List/create groups |
| `/Groups/{id}` | GET, PUT, PATCH, DELETE | Retrieve/replace/modify/delete group |
| `/Bulk` | POST | Bulk operations |
| `/ServiceProviderConfig` | GET | Server capability advertisement |
| `/ResourceTypes` | GET | Available resource types |
| `/Schemas` | GET | Schema introspection |
| `/.search` | POST | POST-based search (SCIM POST binding) |

---

## 2. Filter Syntax (RFC 7644 Section 3.4.2)

SCIM defines a powerful filter language for querying resources via the `filter` query parameter.

### 2.1 Comparison Operators

| Operator | Meaning | Example |
|----------|---------|---------|
| `eq` | Equals | `userName eq "bjensen"` |
| `ne` | Not equal | `emails.type ne "work"` |
| `co` | Contains (substring) | `displayName co "Jensen"` |
| `sw` | Starts with | `userName sw "bjen"` |
| `ew` | Ends with | `userName ew "@example.com"` |
| `pr` | Present (has value) | `emails pr` or `title pr` |
| `gt` | Greater than | `meta.lastModified gt "2023-01-01T00:00:00Z"` |
| `ge` | Greater than or equal | `age ge 21` |
| `lt` | Less than | `age lt 65` |
| `le` | Less than or equal | `age le 64` |

### 2.2 Logical Operators

| Operator | Meaning | Example |
|----------|---------|---------|
| `and` | Logical AND | `active eq true and userName sw "a"` |
| `or` | Logical OR | `title eq "VP" or title eq "Director"` |
| `not` | Logical NOT | `not (emails co "example.com")` |

### 2.3 Grouping with Parentheses

Parentheses control evaluation order:

```
(userName eq "bjensen" or userName eq "jsmith") and active eq true
```

Without parentheses, `and` binds tighter than `or`:

```
userName eq "bjensen" or userName eq "jsmith" and active eq true
-- equivalent to: userName eq "bjensen" or (userName eq "jsmith" and active eq true)
```

### 2.4 Complex Attribute Paths

SCIM supports filtering on nested attributes of complex multi-valued attributes:

```
emails[type eq "work" and value co "@example.com"].value
```

This selects the `value` sub-attribute from the email entry where `type` is "work" and `value` contains "@example.com".

**Examples:**

```
# Filter users with a specific work email
emails[type eq "work"].value eq "bjensen@example.com"

# Filter users whose work address is in a specific city
addresses[type eq "work"].locality eq "Hollywood"

# Filter by manager in enterprise extension
urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department eq "Tour Operations"
```

### 2.5 Complete Filter Examples

```
# Simple equality
filter=userName eq "bjensen"

# Multiple conditions
filter=active eq true and (emails co "example.com" or emails co "example.org")

# Existence check
filter=title pr and displayName pr

# Numeric comparison
filter=age ge 30 and age le 50

# Date comparison
filter=meta.lastModified gt "2023-06-01T00:00:00Z"

# Negation
filter=not (active eq false)

# Complex multi-valued attribute
filter=emails[type eq "work"].value sw "admin@"

# Combined with grouping
filter=(name.familyName co "Jen" or name.familyName co "Smi") and active eq true
```

### 2.6 Common Filter Implementation Pitfalls

| Pitfall | Description | Impact |
|---------|-------------|--------|
| **Case sensitivity** | RFC says `eq` is case-insensitive for string attributes by default. Many implementations treat it as case-sensitive. | Filters may miss results |
| **Quoting** | Filter values MUST be double-quoted. Single quotes or unquoted values cause parse failures. | 400 Bad Request |
| **URL encoding** | The `filter` parameter is in a URL query string, so special characters (`"`, `(`, `)`, spaces) must be percent-encoded. | Broken filters |
| **`pr` with value** | `pr` (present) should NOT have a comparison value: `title pr` (correct), not `title pr "VP"` (incorrect). | Parse error |
| **OR precedence** | `and` has higher precedence than `or` without parentheses. Misunderstanding causes incorrect result sets. | Wrong results |
| **Complex path indexing** | `emails[type eq "work"].value` is a filter on the emails array, not array indexing. Returns ALL matching values. | Unexpected results |
| **Schema URN in path** | Extension attributes require the full URN prefix: `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department`. Missing it causes "attribute not found". | 400 Bad Request |
| **SQL injection** | SCIM filters must be parsed into a safe AST and translated to parameterized SQL. String concatenation is a critical security vulnerability. | Data breach |
| **Pagination + filter** | `totalResults` must reflect the filtered count, not the total unfiltered count. | Incorrect pagination |
| **Empty filter** | An empty `filter=` parameter may be treated as "no filter" or as a parse error. RFC is ambiguous. Inconsistent behavior across providers. | Unexpected results |

---

## 3. Bulk Operations (RFC 7644 Section 3.7)

Bulk operations allow a client to submit multiple SCIM operations in a single HTTP POST request, reducing network round-trips.

### 3.1 Endpoint

```
POST /scim/v2/Bulk
Content-Type: application/scim+json
Authorization: Bearer <token>
```

### 3.2 BulkRequest Structure

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
  "failOnErrors": 5,
  "Operations": [
    {
      "method": "POST",
      "path": "/Users",
      "bulkId": "id-001",
      "data": {
        "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
        "userName": "alice@example.com",
        "emails": [{"value": "alice@example.com", "type": "work"}]
      }
    },
    {
      "method": "POST",
      "path": "/Users",
      "bulkId": "id-002",
      "data": {
        "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
        "userName": "bob@example.com"
      }
    },
    {
      "method": "PATCH",
      "path": "/Users/bulkId:id-001",
      "data": {
        "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
        "Operations": [
          {
            "op": "replace",
            "path": "displayName",
            "value": "Alice Smith"
          }
        ]
      }
    },
    {
      "method": "DELETE",
      "path": "/Users/2819c223-7f76-453a-919d-413861904646"
    }
  ]
}
```

### 3.3 Key Fields

| Field | Type | Description |
|-------|------|-------------|
| `failOnErrors` | integer | Number of errors after which the server SHOULD stop processing. `0` = process all. If omitted, server SHOULD process all. |
| `Operations[].method` | string | HTTP method: POST, PUT, PATCH, or DELETE |
| `Operations[].path` | string | Resource path (e.g., `/Users`, `/Users/{id}`) |
| `Operations[].bulkId` | string | Client-generated identifier for cross-referencing within the bulk request |
| `Operations[].data` | object | Request body for POST/PUT/PATCH operations (omitted for DELETE) |

### 3.4 BulkId References

The `bulkId` enables forward references within a single bulk request. In the example above, operation 3 patches `/Users/bulkId:id-001` — the server resolves this to the actual ID returned by operation 1's POST.

This enables dependent operations (create a user, then add them to a group) in a single request.

### 3.5 BulkResponse Format

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkResponse"],
  "Operations": [
    {
      "location": "https://example.com/scim/v2/Users/902c246b-6245-4190-8e05-00816be7344a",
      "method": "POST",
      "bulkId": "id-001",
      "status": "201"
    },
    {
      "location": "https://example.com/scim/v2/Users/a987cb32-4e22-4aef-8b21-ba44c3f12e7d",
      "method": "POST",
      "bulkId": "id-002",
      "status": "201"
    },
    {
      "location": "https://example.com/scim/v2/Users/902c246b-6245-4190-8e05-00816be7344a",
      "method": "PATCH",
      "status": "200"
    },
    {
      "method": "DELETE",
      "status": "204"
    }
  ]
}
```

### 3.6 Error Handling and Partial Success

When an operation within a bulk request fails, the response contains an error response for that specific operation:

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkResponse"],
  "Operations": [
    {
      "method": "POST",
      "bulkId": "id-001",
      "status": "201",
      "location": "https://example.com/scim/v2/Users/902c246b-6245-4190-8e05-00816be7344a"
    },
    {
      "method": "POST",
      "bulkId": "id-002",
      "status": "409",
      "response": {
        "schemas": ["urn:ietf:params:scim:api:messages:2.0:Error"],
        "detail": "userName 'bob@example.com' already exists",
        "status": "409"
      }
    }
  ]
}
```

**`failOnErrors` semantics:**

| Value | Behavior |
|-------|----------|
| `0` or omitted | Server processes ALL operations regardless of errors |
| `n` (positive) | Server SHOULD stop processing after `n` errors. Operations already submitted are still included in the response. |
| `1` | All-or-nothing: server stops at the first error. Earlier operations MAY be rolled back. |

### 3.7 Max Operations Limits

The server advertises maximum bulk limits via `ServiceProviderConfig`:

```json
{
  "bulk": {
    "supported": true,
    "maxOperations": 1000,
    "maxPayloadSize": 1048576
  }
}
```

| Field | Description |
|-------|-------------|
| `maxOperations` | Maximum number of operations per bulk request |
| `maxPayloadSize` | Maximum payload size in bytes |

If a request exceeds these limits, the server returns `413 Request Entity Too Large` or `400 Bad Request`.

### 3.8 Processing Rules

- Operations are processed sequentially in the order they appear in the array.
- `bulkId` references MUST only reference operations earlier or within the same bulk request.
- A `bulkId` reference that cannot be resolved results in `400 Bad Request` for that operation.
- DELETE operations do not have a `data` field.
- The server MAY impose per-operation rate limits.

---

## 4. ETag / If-Match / If-None-Match (RFC 7644 Section 3.14)

SCIM supports optimistic concurrency control through HTTP ETag headers, allowing clients to detect concurrent modifications and avoid lost updates.

### 4.1 ETag on GET Responses

When a server supports ETags (advertised via `ServiceProviderConfig.etag.supported`), GET responses include an `ETag` header:

```
GET /scim/v2/Users/2819c223-7f76-453a-919d-413861904646
```

Response:

```
HTTP/1.1 200 OK
Content-Type: application/scim+json
ETag: W/"e952813f1f0a4f52"

{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "id": "2819c223-7f76-453a-919d-413861904646",
  "userName": "bjensen@example.com",
  "meta": {
    "resourceType": "User",
    "version": "W/\"e952813f1f0a4f52\""
  }
}
```

The ETag value is also mirrored in `meta.version`.

### 4.2 If-Match on PUT/PATCH/DELETE

The `If-Match` header enables optimistic locking. The server proceeds only if the current resource version matches:

```
PATCH /scim/v2/Users/2819c223-7f76-453a-919d-413861904646
If-Match: W/"e952813f1f0a4f52"
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
  "Operations": [
    {"op": "replace", "path": "displayName", "value": "Babs Jensen"}
  ]
}
```

| `If-Match` Value | Behavior |
|-------------------|----------|
| `W/"e952813f1f0a4f52"` | Proceed only if ETag matches. Otherwise 412. |
| `*` | Proceed only if the resource exists. Otherwise 404. |
| (omitted) | Proceed unconditionally (no version check). |

**412 Precondition Failed response:**

```
HTTP/1.1 412 Precondition Failed
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:Error"],
  "detail": "Resource version does not match If-Match precondition",
  "status": "412"
}
```

### 4.3 If-None-Match for Conditional GET

The `If-None-Match` header enables efficient caching. If the resource has not changed, the server returns `304 Not Modified`:

```
GET /scim/v2/Users/2819c223-7f76-453a-919d-413861904646
If-None-Match: W/"e952813f1f0a4f52"
```

Response if unchanged:

```
HTTP/1.1 304 Not Modified
ETag: W/"e952813f1f0a4f52"
```

| `If-None-Match` Value | Behavior |
|------------------------|----------|
| `W/"e952813f1f0a4f52"` | Return 304 if ETag matches; otherwise return full resource. |
| `*` | Return 304 only if the resource exists. Used for existence checks. |
| (omitted) | No conditional processing. |

### 4.4 ETag Generation Strategies

| Strategy | Example | Trade-offs |
|----------|---------|------------|
| Content hash | `W/"sha256(abridged)"` | Accurate; recomputes on every read |
| Updated timestamp | `W/"2023-06-01T12:00:00Z"` | Simple; may collide if multiple updates in same timestamp resolution |
| Database version column | `W/"v42"` | Reliable if DB supports it; simple to implement |
| Row hash | `W/"md5(row_data)"` | Tamper-resistant; higher CPU cost |

Weak ETags (`W/` prefix) are preferred for SCIM since byte-level equality is not required.

---

## 5. Pagination (RFC 7644 Section 3.4.2)

### 5.1 Query Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `startIndex` | 1 | 1-based index of the first result to return |
| `count` | Implementation-defined | Maximum number of results per page (non-negative integer) |

```
GET /scim/v2/Users?startIndex=1&count=10
GET /scim/v2/Users?startIndex=11&count=10
```

### 5.2 ListResponse Format

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
  "totalResults": 1234,
  "startIndex": 1,
  "itemsPerPage": 10,
  "Resources": [
    { /* User resource 1 */ },
    { /* User resource 2 */ },
    ...
    { /* User resource 10 */ }
  ]
}
```

| Field | Description |
|-------|-------------|
| `totalResults` | Total number of results matching the query (before pagination). The server MAY return an estimate. |
| `startIndex` | The 1-based index of the first result in the current page |
| `itemsPerPage` | The number of resources returned in this page |
| `Resources` | Array of resource objects (note: capital "R") |

### 5.3 Key Rules

1. **`startIndex` is 1-based**, not 0-based. `startIndex=1` returns results starting from the first match.
2. **`itemsPerPage` reflects actual returned count**, not the requested `count`. The last page may have fewer items.
3. **`totalResults` must be consistent** with the filter. If a filter narrows results, `totalResults` reflects the filtered count.
4. **Server MAY cap `count`** at a maximum (advertised in `ServiceProviderConfig.filter.maxResults`).
5. **Empty results** return `totalResults: 0` and an empty `Resources: []` array (not `null`).

### 5.4 Cursor-Based Alternatives

SCIM 2.0 uses offset-based pagination (`startIndex` + `count`). For large datasets, cursor-based pagination is more efficient but requires extension:

**Proposed extension (non-standard):**

```
GET /scim/v2/Users?cursor=eyJpZCI6IjEyMyJ9&count=10
```

```json
{
  "schemas": [
    "urn:ietf:params:scim:api:messages:2.0:ListResponse",
    "urn:com:example:scim:extension:cursor:2.0:ListResponse"
  ],
  "totalResults": 1234567,
  "itemsPerPage": 10,
  "nextCursor": "eyJpZCI6IjEzMyJ9",
  "Resources": [ ... ]
}
```

**Trade-offs:**

| Aspect | Offset-Based (Standard) | Cursor-Based (Extension) |
|--------|------------------------|--------------------------|
| Random access | Yes (can jump to page 5) | No (must traverse sequentially) |
| Performance at scale | O(n) — OFFSET gets expensive | O(1) — uses index seek |
| Stability under concurrent writes | May skip/duplicate | Stable (cursor encodes position) |
| SCIM compliance | Standard | Non-standard extension |

---

## 6. PATCH Operations (RFC 7644 Section 3.5.2)

SCIM PATCH allows partial resource updates. It uses a specialized request format with an array of operations.

### 6.1 Request Format

```
PATCH /scim/v2/Users/2819c223-7f76-453a-919d-413861904646
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
  "Operations": [
    {
      "op": "replace",
      "path": "displayName",
      "value": "Barbara Jensen III"
    }
  ]
}
```

### 6.2 Operation Types

#### ADD (`add`)

Adds or appends a value. For singular attributes, `add` behaves like `replace`. For multi-valued attributes, `add` appends to the collection.

```json
{
  "op": "add",
  "path": "emails",
  "value": [
    {
      "value": "babs@jensen.org",
      "type": "home"
    }
  ]
}
```

Add with path targeting a sub-attribute:

```json
{
  "op": "add",
  "path": "addresses[type eq \"work\"].postalCode",
  "value": "90210"
}
```

Add without a path (updates the whole resource):

```json
{
  "op": "add",
  "value": {
    "nickName": "Babs",
    "title": "VP Engineering"
  }
}
```

#### REPLACE (`replace`)

Replaces all matching values. For multi-valued attributes, replaces the entire array unless a path filter narrows the target.

```json
{
  "op": "replace",
  "path": "displayName",
  "value": "Barbara Jensen III"
}
```

Replace a specific email's value:

```json
{
  "op": "replace",
  "path": "emails[type eq \"work\"].value",
  "value": "bjensen@newcompany.com"
}
```

Replace the entire emails array:

```json
{
  "op": "replace",
  "path": "emails",
  "value": [
    {"value": "new@example.com", "type": "work", "primary": true}
  ]
}
```

Replace active status:

```json
{
  "op": "replace",
  "path": "active",
  "value": false
}
```

#### REMOVE (`remove`)

Removes matching attributes or array elements. The `value` field is NOT used for `remove`.

Remove a singular attribute:

```json
{
  "op": "remove",
  "path": "nickName"
}
```

Remove a specific email:

```json
{
  "op": "remove",
  "path": "emails[type eq \"home\" and value co \"jensen.org\"]"
}
```

Remove all emails:

```json
{
  "op": "remove",
  "path": "emails"
}
```

Remove a specific group membership:

```json
{
  "op": "remove",
  "path": "groups[value eq \"e9e30dba-f08f-4109-8486-d5c6a331660a\"]"
}
```

### 6.3 Path Expression Grammar

```
PATH = attrPath
     | attrPath "[" valFilter "]"
     | attrPath "[" valFilter "]" "." subAttr

attrPath  = [schemaUrn ":"] attributeName ["." subAttr]
valFilter = attrExp | filterExp
```

**Path resolution examples:**

| Path | Resolves to |
|------|-------------|
| `displayName` | The `displayName` singular attribute |
| `name.familyName` | The `familyName` sub-attribute of the `name` complex attribute |
| `emails` | The entire `emails` multi-valued array |
| `emails[type eq "work"]` | All email entries where `type` equals "work" |
| `emails[type eq "work"].value` | The `value` of all work emails |
| `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department` | The `department` field in the enterprise extension |

### 6.4 Complex Attribute Manipulation

Adding a member to a group:

```json
{
  "op": "add",
  "path": "members",
  "value": [
    {
      "value": "2819c223-7f76-453a-919d-413861904646",
      "$ref": "https://example.com/v2/Users/2819c223-7f76-453a-919d-413861904646",
      "display": "Barbara Jensen",
      "type": "User"
    }
  ]
}
```

Removing a specific member:

```json
{
  "op": "remove",
  "path": "members[value eq \"2819c223-7f76-453a-919d-413861904646\"]"
}
```

### 6.5 PATCH Error Handling

| Scenario | HTTP Status | Error Detail |
|----------|-------------|--------------|
| Invalid JSON | 400 | `invalidRequest` / "malformed request body" |
| Unknown operation type | 400 | `invalidSyntax` |
| Path targets non-existent attribute | 400 | `invalidPath` |
| Path filter matches nothing (for `replace`/`remove`) | 200 (no-op) or 400 | RFC allows either; SHOULD be 204 or 200 with unchanged resource |
| `remove` on a required attribute (e.g., `id`, `userName`) | 400 | `mutability` is `readOnly` or `immutable` |
| Resource not found | 404 | `ResourceNotFound` |
| If-Match precondition fails | 412 | `Precondition Failed` |

### 6.6 Response

A successful PATCH returns the updated resource with `200 OK`:

```
HTTP/1.1 200 OK
Content-Type: application/scim+json
ETag: W/"new-version-hash"

{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  ...
}
```

Alternatively, the server MAY return `204 No Content` with no body (and an updated ETag header).

---

## 7. Sort (RFC 7644 Section 3.4.2)

### 7.1 Sort Parameters

| Parameter | Values | Description |
|-----------|--------|-------------|
| `sortBy` | Attribute name | The attribute to sort by |
| `sortOrder` | `ascending` or `descending` | Sort direction (default: `ascending`) |

```
GET /scim/v2/Users?sortBy=userName&sortOrder=ascending
GET /scim/v2/Users?sortBy=meta.lastModified&sortOrder=descending
```

### 7.2 Sorting on Sub-Attributes

```
GET /scim/v2/Users?sortBy=name.familyName&sortOrder=ascending
```

### 7.3 Sorting on Extension Attributes

```
GET /scim/v2/Users?sortBy=urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department
```

### 7.4 Sort Capability Advertisement

The server advertises sort support in `ServiceProviderConfig`:

```json
{
  "sort": {
    "supported": true
  }
}
```

If sort is not supported, `sortBy` and `sortOrder` parameters are ignored (results returned in arbitrary order).

### 7.5 Combined with Filter and Pagination

```
GET /scim/v2/Users?filter=active%20eq%20true&sortBy=userName&sortOrder=ascending&startIndex=1&count=20
```

Processing order: filter first, then sort, then paginate.

---

## 8. GGID SCIM Compliance Assessment

### 8.1 Implementation Location

GGID's SCIM 2.0 implementation is in the Identity service:

- `services/identity/internal/scim/handler.go` — User endpoints, ServiceProviderConfig, ResourceTypes
- `services/identity/internal/scim/groups.go` — Group endpoints (currently mock-backed)
- `services/identity/internal/server/http.go` — Route registration

### 8.2 Implemented Features

| Feature | Status | Details |
|---------|--------|---------|
| **GET /Users** (list) | Partial | Pagination via `startIndex` + `count`; no filter, no sort |
| **POST /Users** (create) | Partial | Creates user; does not persist `externalId`, `name.familyName`, `phoneNumbers`, `addresses` |
| **GET /Users/{id}** | Working | Returns user by UUID |
| **PUT /Users/{id}** (replace) | Partial | Only updates `displayName` and `active`; ignores most attributes |
| **PATCH /Users/{id}** | Minimal | Only handles `displayName` and `active` paths; no complex attribute path support |
| **DELETE /Users/{id}** | Working | Returns `204 No Content` |
| **GET /Groups** (list) | Partial | Uses mock data; supports basic `displayName eq` filter only |
| **POST /Groups** (create) | Partial | Returns created group but does not persist (UUID generated but not stored) |
| **GET /Groups/{id}** | Stub | Searches mock data |
| **PATCH /Groups/{id}** | Stub | Returns hardcoded 200, does not apply operations |
| **DELETE /Groups/{id}** | Stub | Returns `204 No Content` without checking existence |
| **GET /ServiceProviderConfig** | Working | Reports `patch: true`, `filter: true`, `sort: true`, `etag: false`, `bulk: false` |
| **GET /ResourceTypes** | Working | Returns User + Group resource type definitions |
| **Content-Type** | Working | Uses `application/scim+json` |
| **Error format** | Working | Uses standard `ErrorResponse` with schemas/detail/status |

### 8.3 Missing Features (Gaps)

| Gap | RFC Reference | Priority | Description |
|-----|---------------|----------|-------------|
| **SCIM Filter engine** | RFC 7644 Section 3.4.2 | Critical | No filter parser. Only `displayName eq` for Groups (string split). Users have zero filter support. |
| **Sort support** | RFC 7644 Section 3.4.2 | High | `sortBy` and `sortOrder` not parsed in SCIM endpoints (the REST API has it, but SCIM layer does not) |
| **ETag / If-Match** | RFC 7644 Section 3.14 | Medium | Not implemented. `ServiceProviderConfig` reports `etag: false`. |
| **Bulk endpoint** | RFC 7644 Section 3.7 | Medium | `/scim/v2/Bulk` not registered. `ServiceProviderConfig` reports `bulk: false`. |
| **Schemas endpoint** | RFC 7643 Section 7 | Medium | `/scim/v2/Schemas` not implemented. Required for schema introspection. |
| **POST .search** | RFC 7644 Section 3.4.3 | Low | POST-based search not supported |
| **EnterpriseUser extension** | RFC 7643 Section 4.3 | High | No `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User` support |
| **`attributes` / `excludedAttributes`** | RFC 7644 Section 3.4.2.5 / 3.9 | Medium | No attribute projection support |
| **`meta.created` / `meta.lastModified` / `meta.version`** | RFC 7643 Section 5 | High | `meta` only has `resourceType`; missing timestamps and version |
| **`meta.location` URL** | RFC 7643 Section 5 | High | `Location` not populated for Users (only Groups). Should include full URL. |
| **Full PATCH path engine** | RFC 7644 Section 3.5.2 | Critical | PATCH only handles `displayName` and `active`. No support for `emails`, `name`, `phoneNumbers`, complex paths, or multi-valued arrays. |
| **Groups persistence** | RFC 7644 Section 3 | Critical | Groups use `getMockGroups()` — hardcoded mock data. No database-backed storage. |
| **Group PATCH application** | RFC 7644 Section 3.5.2 | Critical | `patchGroup()` returns hardcoded 200 without applying operations. |
| **`userName` uniqueness** | RFC 7643 Section 4.1.1 | Medium | Create always returns `409 Conflict` on any error, not `400` for validation errors. |
| **`password` attribute** | RFC 7643 Section 4.1.1 / RFC 7644 Section 3.5.1 | Low | `changePassword` advertised as `true` but no SCIM-level password change. |
| **`externalId` persistence** | RFC 7643 Section 5 | Medium | Accepted in request but not persisted or returned. |
| **`ListResponse.Resources` typing** | RFC 7644 Section 3.4.2 | Low | `ListResponse.Resources` is `[]SCIMUser` — cannot hold `SCIMGroup` for Group list responses (Groups uses `map[string]any` workaround). |
| **POST Location header** | RFC 7644 Section 3.3 | Low | `POST /Users` response lacks `Location` header. |
| **`PUT` attribute immutability** | RFC 7643 Section 7 | Low | `PUT` allows partial attribute updates but SCIM `PUT` is full replacement (should set unspecified attributes to defaults). |
| **Error response scimType** | RFC 7644 Section 3.12 | Medium | `ErrorResponse` lacks `scimType` field (e.g., `invalidFilter`, `invalidSyntax`, `invalidPath`, `tooMany`, `uniqueness`). |

### 8.4 Recommended Test Suite Approach

**Primary: Run the official [scim2/test-suite](https://github.com/scim2/test-suite)**

The scim2/test-suite is an Apache 2.0 licensed Go test suite that:

- Parses RFC 7642/7643/7644 requirements with their RFC 2119 keywords (MUST, SHOULD, MAY)
- Maps each requirement to Go test functions
- Runs black-box compliance tests against a SCIM server
- Generates a pass/fail/warn compliance report
- Supports feature discovery via `/ServiceProviderConfig` and `/ResourceTypes`
- Tests for optional features (filter, patch, bulk, sort, etag) in soft mode when not advertised

**How to run it against GGID:**

```bash
# Clone the test suite
git clone https://github.com/scim2/test-suite.git
cd test-suite

# Run against GGID's SCIM endpoint
go test ./compliance/ -v -count=1 \
  -scim.url=http://localhost:8081/scim/v2 \
  -scim.token=<bearer-token>

# Force-test features GGID doesn't advertise yet
go test ./compliance/ -v -count=1 \
  -scim.url=http://localhost:8081/scim/v2 \
  -scim.token=<bearer-token> \
  -scim.force=filter,patch,sort,etag,bulk
```

**Secondary: Custom integration tests**

The scim2/test-suite is in "Initial Draft" status with limited coverage. Supplement with custom integration tests in `test/integration/scim/` that cover GGID-specific flows (tenant isolation, role-to-group mapping, etc.).

### 8.5 Gap Closure Priority

| Phase | Gaps to Close | Estimated Effort |
|-------|---------------|-----------------|
| **Phase 1: Core Compliance** | SCIM filter engine, full PATCH path engine, `meta` timestamps/location, `scimType` in errors, `externalId` persistence | 3-5 days |
| **Phase 2: Enterprise Features** | EnterpriseUser extension, Sort in SCIM endpoints, `attributes`/`excludedAttributes` projection, Groups database persistence | 3-5 days |
| **Phase 3: Advanced** | Bulk endpoint, ETag/If-Match/If-None-Match, Schemas endpoint, POST .search | 3-5 days |
| **Phase 4: Validation** | Run scim2/test-suite, fix failures, achieve 90%+ compliance | 2-3 days |

---

## 9. Test Suite Design

### 9.1 Test Categories

```
SCIM Test Suite
├── 1. CRUD Operations
│   ├── Users (GET/POST/PUT/PATCH/DELETE)
│   └── Groups (GET/POST/PUT/PATCH/DELETE)
├── 2. Query Operations
│   ├── Filter (all operators)
│   ├── Sort (sortBy, sortOrder)
│   ├── Pagination (startIndex, count)
│   └── Attribute Projection (attributes, excludedAttributes)
├── 3. Advanced Operations
│   ├── Bulk
│   ├── ETag / If-Match / If-None-Match
│   └── POST .search
├── 4. Schema Discovery
│   ├── ServiceProviderConfig
│   ├── ResourceTypes
│   └── Schemas
├── 5. Error Handling
│   ├── 400 Invalid Request
│   ├── 401 Unauthorized
│   ├── 403 Forbidden
│   ├── 404 Not Found
│   ├── 409 Conflict
│   ├── 412 Precondition Failed
│   ├── 413 Payload Too Large
│   ├── 500 Internal Server Error
│   └── 501 Not Implemented
└── 6. Conformance
    ├── Content-Type validation
    ├── Response schema validation
    └── HTTP method handling
```

### 9.2 Test Cases: CRUD Operations

#### 9.2.1 Users CRUD

| ID | Test | Request | Expected |
|----|------|---------|----------|
| U-001 | Create user | `POST /Users` with valid `userName` | `201 Created`, `Location` header, valid `id` |
| U-002 | Create user — duplicate userName | `POST /Users` with existing `userName` | `409 Conflict`, `scimType: uniqueness` |
| U-003 | Create user — missing userName | `POST /Users` without `userName` | `400 Bad Request`, `scimType: invalidSyntax` |
| U-004 | Get user by ID | `GET /Users/{id}` | `200 OK`, full user resource |
| U-005 | Get user — non-existent ID | `GET /Users/nonexistent-uuid` | `404 Not Found` |
| U-006 | List users | `GET /Users` | `200 OK`, `ListResponse` with `schemas`, `totalResults`, `Resources` |
| U-007 | Replace user (PUT) | `PUT /Users/{id}` with full resource | `200 OK`, all attributes updated |
| U-008 | PATCH user — replace displayName | `PATCH /Users/{id}` with replace op | `200 OK`, updated `displayName` |
| U-009 | PATCH user — replace active | `PATCH /Users/{id}` with `active: false` | `200 OK`, `active: false` |
| U-010 | PATCH user — add email | `PATCH /Users/{id}` with add email op | `200 OK`, new email appended |
| U-011 | PATCH user — remove email | `PATCH /Users/{id}` with remove path filter | `200 OK`, email removed |
| U-012 | PATCH user — replace specific email | `PATCH /Users/{id}` with `emails[type eq "work"].value` | `200 OK`, only work email updated |
| U-013 | DELETE user | `DELETE /Users/{id}` | `204 No Content` |
| U-014 | DELETE user — non-existent | `DELETE /Users/nonexistent` | `404 Not Found` |
| U-015 | Create user with EnterpriseUser extension | `POST /Users` with enterprise schema | `201 Created`, extension attributes persisted |

#### 9.2.2 Groups CRUD

| ID | Test | Request | Expected |
|----|------|---------|----------|
| G-001 | Create group | `POST /Groups` with `displayName` | `201 Created`, valid `id`, `Location` header |
| G-002 | Create group — missing displayName | `POST /Groups` without `displayName` | `400 Bad Request` |
| G-003 | Get group by ID | `GET /Groups/{id}` | `200 OK`, group with members |
| G-004 | List groups | `GET /Groups` | `200 OK`, `ListResponse` |
| G-005 | PATCH group — add member | `PATCH /Groups/{id}` with add members op | `200 OK`, member in `members[]` |
| G-006 | PATCH group — remove member | `PATCH /Groups/{id}` with remove path | `200 OK`, member removed |
| G-007 | PATCH group — replace displayName | `PATCH /Groups/{id}` with replace displayName | `200 OK` |
| G-008 | PUT group (full replace) | `PUT /Groups/{id}` with full group resource | `200 OK` |
| G-009 | DELETE group | `DELETE /Groups/{id}` | `204 No Content` |
| G-010 | Get group — non-existent | `GET /Groups/nonexistent` | `404 Not Found` |

### 9.3 Test Cases: Filter

| ID | Filter Expression | Expected Result |
|----|-------------------|-----------------|
| F-001 | `userName eq "bjensen"` | Exact match |
| F-002 | `userName ne "bjensen"` | All except bjensen |
| F-003 | `displayName co "Jensen"` | Contains "Jensen" |
| F-004 | `userName sw "bjen"` | Starts with "bjen" |
| F-005 | `userName ew "@example.com"` | Ends with "@example.com" |
| F-006 | `title pr` | Has title set |
| F-007 | `active eq true` | All active users |
| F-008 | `meta.lastModified gt "2023-01-01T00:00:00Z"` | Modified after date |
| F-009 | `age ge 30 and age le 50` | Age range |
| F-010 | `title eq "VP" or title eq "Director"` | Multiple titles |
| F-011 | `not (active eq false)` | All active users |
| F-012 | `emails[type eq "work"].value eq "bjensen@example.com"` | Work email match |
| F-013 | `emails[type eq "work" and value co "@example.com"]` | Complex multi-valued |
| F-014 | `(name.familyName co "Jen") and active eq true` | Grouped + logical |
| F-015 | `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department eq "Eng"` | Extension attribute |
| F-016 | Empty filter | `400 Bad Request`, `scimType: invalidFilter` |
| F-017 | Invalid operator `userName xx "bjensen"` | `400 Bad Request`, `scimType: invalidFilter` |
| F-018 | Unterminated string `userName eq "bjensen` | `400 Bad Request`, `scimType: invalidFilter` |

### 9.4 Test Cases: Sort

| ID | Request | Expected |
|----|---------|----------|
| S-001 | `GET /Users?sortBy=userName&sortOrder=ascending` | Sorted ascending by userName |
| S-002 | `GET /Users?sortBy=userName&sortOrder=descending` | Sorted descending |
| S-003 | `GET /Users?sortBy=name.familyName` | Sorted by sub-attribute |
| S-004 | `GET /Users?sortBy=meta.created&sortOrder=ascending` | Sorted by creation date |
| S-005 | `sortBy` without `sortOrder` | Default ascending |
| S-006 | No sort params | Unspecified order (valid) |
| S-007 | Sort + filter + pagination combined | Filter applied first, then sorted, then paginated |

### 9.5 Test Cases: Pagination

| ID | Request | Expected |
|----|---------|----------|
| P-001 | `GET /Users?startIndex=1&count=10` | First 10 results, `totalResults` >= 10 |
| P-002 | `GET /Users?startIndex=11&count=10` | Next 10 results |
| P-003 | `GET /Users?startIndex=1&count=0` | `Resources: []`, `totalResults` = full count |
| P-004 | `GET /Users?count=1` | 1 result, `itemsPerPage: 1` |
| P-005 | `GET /Users?startIndex=999999` | `Resources: []`, `totalResults` unchanged |
| P-006 | `startIndex` defaults to 1 when omitted | First page |
| P-007 | `count` defaults to server max when omitted | Valid page |
| P-008 | `totalResults` matches filtered count | `totalResults` = count after filter |
| P-009 | Empty result set | `totalResults: 0`, `Resources: []` (not null) |
| P-010 | Combined with filter | `totalResults` reflects filtered total |

### 9.6 Test Cases: Bulk

| ID | Request | Expected |
|----|---------|----------|
| B-001 | POST with 2 create operations | `200 OK`, 2 operations in response, both `status: 201` |
| B-002 | POST with `bulkId` cross-reference | Operation 2 references `bulkId` from operation 1 |
| B-003 | POST with `failOnErrors: 1` and first op fails | Only first operation processed, second not attempted |
| B-004 | POST with `failOnErrors: 0` and one op fails | All operations processed, failed one has error status |
| B-005 | POST with DELETE + POST mixed | Both operations succeed |
| B-006 | POST exceeding `maxOperations` | `413 Payload Too Large` or `400 Bad Request` |
| B-007 | POST with unresolvable `bulkId` reference | `400 Bad Request` for that operation |
| B-008 | POST with PATCH referencing bulkId | `bulkId` resolved to created resource's ID |

### 9.7 Test Cases: ETag

| ID | Request | Expected |
|----|---------|----------|
| E-001 | `GET /Users/{id}` | Response has `ETag` header |
| E-002 | `PUT /Users/{id}` with correct `If-Match` | `200 OK`, resource updated |
| E-003 | `PUT /Users/{id}` with stale `If-Match` | `412 Precondition Failed` |
| E-004 | `PUT /Users/{id}` with `If-Match: *` and resource exists | `200 OK` |
| E-005 | `PUT /Users/{id}` with `If-Match: *` and resource does not exist | `404 Not Found` |
| E-006 | `GET /Users/{id}` with `If-None-Match` matching ETag | `304 Not Modified` |
| E-007 | `GET /Users/{id}` with `If-None-Match` not matching | `200 OK` with full resource |
| E-008 | `PATCH` with stale ETag | `412 Precondition Failed` |
| E-009 | `DELETE` with correct `If-Match` | `204 No Content` |
| E-010 | `DELETE` with stale `If-Match` | `412 Precondition Failed` |

### 9.8 Test Cases: Error Handling

| ID | Scenario | Expected Status | Expected `scimType` |
|----|----------|-----------------|---------------------|
| ER-001 | Invalid JSON body | 400 | `invalidSyntax` |
| ER-002 | Unsupported HTTP method | 405 | — |
| ER-003 | Missing required attribute | 400 | `invalidSyntax` |
| ER-004 | Invalid filter expression | 400 | `invalidFilter` |
| ER-005 | Unknown attribute in path | 400 | `invalidPath` |
| ER-006 | Duplicate `userName` | 409 | `uniqueness` |
| ER-007 | Resource not found | 404 | — |
| ER-008 | Unauthorized (missing token) | 401 | — |
| ER-009 | Forbidden (insufficient scope) | 403 | — |
| ER-010 | `POST /Bulk` exceeds max payload | 413 | — |
| ER-011 | Unsupported feature (e.g., bulk when not advertised) | 501 | — |
| ER-012 | Internal server error | 500 | — |
| ER-013 | ETag precondition failed | 412 | — |

### 9.9 Test Cases: Schema Discovery

| ID | Request | Expected |
|----|---------|----------|
| SD-001 | `GET /ServiceProviderConfig` | `200 OK`, includes `patch`, `bulk`, `filter`, `sort`, `etag`, `changePassword`, `authenticationSchemes` |
| SD-002 | `GET /ResourceTypes` | `200 OK`, includes User and Group types with `schema`, `endpoint`, `schemaExtensions` |
| SD-003 | `GET /ResourceTypes/User` | `200 OK`, User resource type definition |
| SD-004 | `GET /ResourceTypes/Group` | `200 OK`, Group resource type definition |
| SD-005 | `GET /Schemas` | `200 OK`, ListResponse with schema definitions |
| SD-006 | `GET /Schemas/urn:ietf:params:scim:schemas:core:2.0:User` | `200 OK`, User schema with all attributes |
| SD-007 | `GET /Schemas/urn:ietf:params:scim:schemas:core:2.0:Group` | `200 OK`, Group schema with all attributes |

### 9.10 Test Cases: Conformance

| ID | Test | Expected |
|----|------|----------|
| C-001 | All responses use `application/scim+json` | `Content-Type: application/scim+json` header |
| C-002 | All resources include `schemas` array | `schemas` present and non-empty |
| C-003 | `ListResponse` has correct `schemas` URN | `urn:ietf:params:scim:api:messages:2.0:ListResponse` |
| C-004 | Error responses use correct schema | `urn:ietf:params:scim:api:messages:2.0:Error` |
| C-005 | Resource `id` is immutable | `PUT`/`PATCH` cannot change `id` |
| C-006 | `meta.resourceType` matches endpoint | User resources have `"resourceType": "User"` |
| C-007 | `Resources` key in ListResponse is capitalized | `"Resources"` not `"resources"` |
| C-008 | `Location` header on POST | `POST /Users` returns `Location` header with resource URL |
| C-009 | Unsupported method returns 405 | `POST /Users/{id}` returns `405 Method Not Allowed` |
| C-010 | `PATCH` returns updated resource or 204 | `PATCH` returns `200 OK` with body, or `204 No Content` |

### 9.11 Recommended Test Implementation

```go
// test/integration/scim/scim_test.go
//go:build integration

package scim_test

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "testing"
)

const scimBaseURL = "http://localhost:8081/scim/v2"

func scimRequest(t *testing.T, method, path string, body any) (*http.Response, map[string]any) {
    t.Helper()
    var reqBody io.Reader
    if body != nil {
        b, _ := json.Marshal(body)
        reqBody = bytes.NewReader(b)
    }
    req, err := http.NewRequest(method, scimBaseURL+path, reqBody)
    if err != nil {
        t.Fatal(err)
    }
    req.Header.Set("Content-Type", "application/scim+json")
    req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        t.Fatal(err)
    }

    var result map[string]any
    if resp.Body != nil {
        defer resp.Body.Close()
        json.NewDecoder(resp.Body).Decode(&result)
    }
    return resp, result
}

func assertSCIMError(t *testing.T, resp *http.Response, body map[string]any, expectedStatus int) {
    t.Helper()
    if resp.StatusCode != expectedStatus {
        t.Errorf("expected status %d, got %d", expectedStatus, resp.StatusCode)
    }
    schemas, _ := body["schemas"].([]any)
    found := false
    for _, s := range schemas {
        if s == "urn:ietf:params:scim:api:messages:2.0:Error" {
            found = true
            break
        }
    }
    if !found {
        t.Errorf("error response missing Error schema URN: %v", body)
    }
    if body["status"] != fmt.Sprintf("%d", expectedStatus) {
        t.Errorf("error status field mismatch: got %v, expected %d", body["status"], expectedStatus)
    }
}
```

**Example test case implementation:**

```go
func TestCreateUser_Success(t *testing.T) {
    createReq := map[string]any{
        "schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
        "userName":    "test-scim-user@example.com",
        "displayName": "Test SCIM User",
        "emails": []map[string]string{
            {"value": "test-scim-user@example.com", "type": "work"},
        },
        "active": true,
    }

    resp, body := scimRequest(t, "POST", "/Users", createReq)

    if resp.StatusCode != http.StatusCreated {
        t.Fatalf("expected 201, got %d: %v", resp.StatusCode, body)
    }
    if resp.Header.Get("Location") == "" {
        t.Error("missing Location header on POST /Users response")
    }
    if body["id"] == nil || body["id"] == "" {
        t.Error("response missing 'id' field")
    }
    if body["userName"] != "test-scim-user@example.com" {
        t.Errorf("userName mismatch: got %v", body["userName"])
    }
}

func TestCreateUser_DuplicateConflict(t *testing.T) {
    createReq := map[string]any{
        "schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
        "userName": "duplicate@example.com",
    }

    // First create should succeed
    resp1, _ := scimRequest(t, "POST", "/Users", createReq)
    if resp1.StatusCode != http.StatusCreated {
        t.Fatalf("first create should succeed, got %d", resp1.StatusCode)
    }

    // Second create with same userName should conflict
    resp2, body := scimRequest(t, "POST", "/Users", createReq)
    assertSCIMError(t, resp2, body, http.StatusConflict)
}

func TestFilterUser_Equals(t *testing.T) {
    // Create a user first
    createReq := map[string]any{
        "schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
        "userName": "filter-test@example.com",
    }
    scimRequest(t, "POST", "/Users", createReq)

    // Filter by exact userName
    resp, body := scimRequest(t, "GET", "/Users?filter=userName+eq+%22filter-test%40example.com%22", nil)

    if resp.StatusCode != http.StatusOK {
        t.Fatalf("expected 200, got %d", resp.StatusCode)
    }
    total := int(body["totalResults"].(float64))
    if total < 1 {
        t.Errorf("expected totalResults >= 1, got %d", total)
    }
    resources := body["Resources"].([]any)
    if len(resources) == 0 {
        t.Error("expected at least 1 result")
    }
}
```

### 9.12 Test Execution Matrix

| Category | Test Count | GGID Status |
|----------|-----------|-------------|
| Users CRUD | 15 | 8 passing, 7 failing (PATCH, PUT gaps) |
| Groups CRUD | 10 | 0 passing (all mock-backed) |
| Filter | 18 | 0 passing (no filter engine) |
| Sort | 7 | 0 passing (not implemented in SCIM layer) |
| Pagination | 10 | 6 passing (basic pagination works) |
| Bulk | 8 | 0 passing (not implemented) |
| ETag | 10 | 0 passing (not implemented) |
| Error Handling | 13 | 4 passing (basic errors work) |
| Schema Discovery | 7 | 4 passing (ServiceProviderConfig + ResourceTypes work) |
| Conformance | 10 | 5 passing |
| **Total** | **108** | **~27 passing (25%)** |

---

## Appendix A: SCIM Error Response Reference

| HTTP Status | scimType | Description |
|-------------|----------|-------------|
| 400 | `invalidFilter` | Filter expression is invalid |
| 400 | `invalidSyntax` | Request body is malformed |
| 400 | `invalidPath` | PATCH path expression is invalid |
| 400 | `tooMany` | Too many results or operations |
| 400 | `invalidVers` | Unsupported SCIM protocol version |
| 400 | `invalidValue` | Attribute value is invalid |
| 401 | — | Authentication required or failed |
| 403 | — | Insufficient permissions |
| 404 | — | Resource not found |
| 405 | — | HTTP method not supported |
| 409 | `uniqueness` | Attribute uniqueness constraint violated |
| 412 | — | ETag precondition failed (If-Match) |
| 413 | — | Request payload too large |
| 500 | — | Internal server error |
| 501 | — | Feature not implemented |

## Appendix B: Useful References

- [RFC 7643 — SCIM Core Schema](https://www.rfc-editor.org/rfc/rfc7643)
- [RFC 7644 — SCIM Protocol](https://www.rfc-editor.org/rfc/rfc7644)
- [RFC 7642 — SCIM Definitions and Requirements](https://www.rfc-editor.org/rfc/rfc7642)
- [scim2/test-suite — Official Compliance Test Suite](https://github.com/scim2/test-suite)
- [WSO2 SCIM 2.0 Patch Operations](https://is.docs.wso2.com/en/6.0.0/apis/scim2-patch-operations/)
- [SimpleCloud.info — SCIM Community](https://simplecloud.info)
