# GGID API Conventions

Design conventions and standards for the GGID REST API.

---

## URL Structure

### Base URL

```
https://iam.example.com/api/v1/
```

| Part | Description |
|------|-------------|
| `/api` | API root namespace |
| `/v1` | API version (see [Versioning](#versioning)) |

### Resource Naming

| Rule | Example | Correct | Incorrect |
|------|---------|---------|-----------|
| Plural nouns for collections | `/users` | Yes | ~~`/user`~~ |
| Kebab-case for multi-word | `/password-reset` | Yes | ~~`/passwordReset`~~ |
| UUID for resource IDs | `/users/550e8400-...` | Yes | ~~`/users/123`~~ |
| Nested for relationships | `/users/{id}/roles` | Yes | ~~`/user-roles?user_id=`~~ |
| No trailing slash | `/users` | Yes | ~~`/users/`~~ |
| No file extensions | `/users` | Yes | ~~`/users.json`~~ |

### HTTP Methods

| Method | Purpose | Idempotent | Example |
|--------|---------|:----------:|---------|
| `GET` | Read resource(s) | Yes | `GET /users/{id}` |
| `POST` | Create resource | No | `POST /users` |
| `PUT` | Full update | Yes | `PUT /users/{id}` |
| `PATCH` | Partial update | No | `PATCH /users/{id}` |
| `DELETE` | Remove resource | Yes | `DELETE /users/{id}` |

---

## Request Format

### Content-Type

```
Content-Type: application/json
```

All request bodies are JSON. Form-encoded and multipart are not supported
(except for specific file upload endpoints).

### Required Headers

| Header | Required | Description |
|--------|:--------:|-------------|
| `Authorization: Bearer <jwt>` | Yes (except public) | JWT access token |
| `X-Tenant-ID: <uuid>` | Yes | Tenant identifier |
| `Content-Type: application/json` | For POST/PUT/PATCH | JSON body marker |
| `Idempotency-Key: <uuid>` | Optional for POST | Prevents duplicate processing |

---

## Response Format

### Success Response

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "john.doe",
  "email": "john@example.com",
  "created_at": "2024-07-10T12:00:00Z"
}
```

### Collection Response

```json
{
  "users": [
    {"id": "...", "username": "john.doe", ...},
    {"id": "...", "username": "jane.doe", ...}
  ],
  "total": 142,
  "page": 1,
  "page_size": 50
}
```

### Timestamps

All timestamps are ISO 8601 UTC:

```
"created_at": "2024-07-10T12:00:00.000Z"
```

---

## Pagination

### Offset-Based (Default)

```
GET /api/v1/users?page=1&page_size=50
```

| Parameter | Default | Max | Description |
|-----------|---------|-----|-------------|
| `page` | 1 | — | Page number (1-based) |
| `page_size` | 50 | 200 | Items per page |

Response includes pagination metadata:

```json
{
  "users": [...],
  "total": 142,
  "page": 1,
  "page_size": 50
}
```

### Cursor-Based (for large datasets)

```
GET /api/v1/audit/events?cursor=eyJpZCI6IjEyMyJ9&limit=100
```

| Parameter | Description |
|-----------|-------------|
| `cursor` | Opaque base64-encoded cursor from previous response |
| `limit` | Max items (default 100, max 500) |

```json
{
  "events": [...],
  "next_cursor": "eyJpZCI6IjQ1NiJ9",
  "has_more": true
}
```

---

## Filtering

### Query Parameter Filters

```
GET /api/v1/users?status=active&email_domain=example.com&created_after=2024-01-01
```

| Operator | Syntax | Example |
|----------|--------|---------|
| Equals | `field=value` | `status=active` |
| Not equals | `field__ne=value` | `status__ne=locked` |
| Greater than | `field__gt=value` | `created_at__gt=2024-01-01` |
| Less than | `field__lt=value` | `created_at__lt=2024-12-31` |
| In list | `field__in=a,b,c` | `status__in=active,pending` |
| Contains | `field__contains=sub` | `email__contains=example` |
| Prefix | `field__prefix=abc` | `username__prefix=jo` |

### Multiple Filters

Multiple filters are combined with AND:

```
GET /users?status=active&role=admin&created_after=2024-01-01
```

This returns users that are active AND have admin role AND were created after Jan 1.

---

## Sorting

```
GET /api/v1/users?sort=username
GET /api/v1/users?sort=-created_at
GET /api/v1/users?sort=last_name,first_name
```

| Syntax | Description |
|--------|-------------|
| `sort=name` | Ascending by name |
| `sort=-name` | Descending by name |
| `sort=a,b` | Multi-field: sort by a, then b |

Default sort: `created_at DESC` for most collections.

---

## Error Format

All errors follow a consistent structure:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "User not found",
    "details": {
      "user_id": "550e8400-..."
    },
    "request_id": "req-abc123"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `code` | string | Machine-readable error code |
| `message` | string | Human-readable description |
| `details` | object | Additional context (optional) |
| `request_id` | string | For support reference |

### Error Codes

| Code | HTTP Status | Description |
|------|:-----------:|-------------|
| `INVALID_ARGUMENT` | 400 | Missing or invalid field |
| `UNAUTHENTICATED` | 401 | Missing/expired JWT |
| `PERMISSION_DENIED` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource doesn't exist |
| `ALREADY_EXISTS` | 409 | Duplicate resource |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL` | 500 | Server error |
| `UNAVAILABLE` | 503 | Service temporarily unavailable |

### Field-Level Validation Errors

```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message": "Validation failed",
    "details": {
      "fields": [
        {"field": "email", "message": "Invalid email format"},
        {"field": "password", "message": "Must be at least 8 characters"}
      ]
    }
  }
}
```

---

## Versioning

### URI-Based

```
/api/v1/users    → current
/api/v2/users    → future (when breaking changes needed)
```

### Compatibility Rules

| Change | Allowed in v1? |
|--------|:-:|
| Add new endpoint | Yes |
| Add new response field | Yes |
| Add optional request field | Yes |
| Remove a field | No (breaking) |
| Change field type | No (breaking) |
| Change URL structure | No (breaking) |

### Deprecation

1. Mark endpoint as deprecated in OpenAPI spec
2. Add `Deprecation` and `Sunset` headers to responses
3. Support for at least 6 months before removal

```
Deprecation: Sun, 01 Jan 2025 00:00:00 GMT
Sunset: Sat, 01 Jul 2025 00:00:00 GMT
```

---

## Idempotency

### Idempotency-Key Header

For POST and PUT requests that might be retried:

```bash
POST /api/v1/users
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000
Content-Type: application/json

{"username": "john.doe", "email": "john@example.com"}
```

### Behavior

1. Server stores the response keyed by `Idempotency-Key` (24h TTL)
2. If same key is seen again, returns the stored response without re-executing
3. Prevents duplicate user creation, double charges, etc.
4. Key must be unique per tenant

### Safe Retries

```python
import uuid

key = str(uuid.uuid4())
try:
    response = create_user(data, headers={"Idempotency-Key": key})
except TimeoutError:
    # Safe to retry with same key
    response = create_user(data, headers={"Idempotency-Key": key})
```

---

## Rate Limiting

See [API Rate Limits](./api-rate-limits.md) for full details.

### Headers on Every Response

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1720612860
```

### When Rate Limited (429)

```
Retry-After: 60
```

---

## Common Patterns

### Bulk Operations

```bash
# Bulk create
POST /api/v1/users/bulk
[
  {"username": "user1", "email": "user1@example.com"},
  {"username": "user2", "email": "user2@example.com"}
]
```

Response includes per-item results:

```json
{
  "created": 2,
  "failed": 0,
  "results": [
    {"username": "user1", "id": "...", "status": "created"},
    {"username": "user2", "id": "...", "status": "created"}
  ]
}
```

### Soft Delete

DELETE marks resources as `status: deleted` rather than removing them. This
preserves referential integrity (e.g., audit logs can still reference the user).

### Envelope vs Bare

GGID returns bare objects for single resources and wrapped arrays for collections:

```json
// Single resource (bare)
{"id": "...", "username": "..."}

// Collection (wrapped)
{"users": [...], "total": 142}
```

### Localization

```
Accept-Language: zh-CN
```

Error messages are localized based on the `Accept-Language` header.
Field names in errors are always in English (machine-readable).

---

## HTTP Status Code Reference

| Code | Meaning | When |
|------|---------|------|
| 200 | OK | Successful GET, PUT, PATCH |
| 201 | Created | Successful POST |
| 204 | No Content | Successful DELETE |
| 400 | Bad Request | Invalid input |
| 401 | Unauthorized | Missing/expired token |
| 403 | Forbidden | Permission denied |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Duplicate resource |
| 422 | Unprocessable Entity | Validation error |
| 429 | Too Many Requests | Rate limited |
| 500 | Internal Server Error | Unhandled server error |
| 502 | Bad Gateway | Backend unreachable |
| 503 | Service Unavailable | Backend unhealthy |
| 504 | Gateway Timeout | Backend timeout |
