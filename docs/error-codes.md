# GGID Error Codes Reference

Complete reference for all HTTP status codes, business error codes, and error response formats.

---

## Error Response Format

All GGID API errors return a consistent JSON structure:

```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message": "Email address is not valid",
    "details": {
      "field": "email",
      "value": "not-an-email"
    },
    "request_id": "req-abc123"
  }
}
```

| Field | Type | Always Present | Description |
|-------|------|:--------------:|-------------|
| `code` | string | Yes | Machine-readable error code |
| `message` | string | Yes | Human-readable description (English) |
| `details` | object | No | Additional context (field name, validation info) |
| `request_id` | string | Yes | Correlation ID for support/debugging |

---

## HTTP Status Codes

### 2xx Success

| Code | When | Example Endpoint |
|------|------|-----------------|
| 200 OK | Successful GET, PATCH, PUT | `GET /api/v1/users/{id}` |
| 201 Created | Resource created | `POST /api/v1/users` |
| 204 No Content | Deleted successfully | `DELETE /api/v1/users/{id}` |

### 4xx Client Errors

| Code | Error Code | Description |
|------|-----------|-------------|
| 400 | `INVALID_ARGUMENT` | Malformed request, missing required field, validation error |
| 401 | `UNAUTHENTICATED` | Missing/expired JWT, invalid credentials |
| 403 | `PERMISSION_DENIED` | RBAC/ABAC denied, insufficient role |
| 404 | `NOT_FOUND` | Resource doesn't exist |
| 405 | `METHOD_NOT_ALLOWED` | Wrong HTTP method for endpoint |
| 409 | `ALREADY_EXISTS` | Duplicate username, email, role key |
| 422 | `UNPROCESSABLE_ENTITY` | Valid JSON but business rule violation |
| 429 | `RATE_LIMITED` | Rate limit exceeded |

### 5xx Server Errors

| Code | Error Code | Description |
|------|-----------|-------------|
| 500 | `INTERNAL` | Unexpected server error (logged + request_id) |
| 502 | `BAD_GATEWAY` | Backend service returned invalid response |
| 503 | `UNAVAILABLE` | Backend service down, circuit breaker open |

---

## Business Error Codes

### Authentication

| Code | HTTP | Description |
|------|------|-------------|
| `INVALID_CREDENTIALS` | 401 | Wrong username or password |
| `ACCOUNT_LOCKED` | 403 | Account locked due to failed attempts |
| `ACCOUNT_INACTIVE` | 403 | Account is deactivated |
| `MFA_REQUIRED` | 200 | Login successful but MFA challenge needed |
| `MFA_INVALID_CODE` | 401 | Wrong MFA code |
| `MFA_SETUP_REQUIRED` | 403 | MFA not set up but policy requires it |
| `TOKEN_EXPIRED` | 401 | JWT has expired |
| `TOKEN_INVALID` | 401 | JWT signature invalid or malformed |
| `REFRESH_TOKEN_REVOKED` | 401 | Refresh token was already used (rotation) |

### User Management

| Code | HTTP | Description |
|------|------|-------------|
| `USER_NOT_FOUND` | 404 | User ID doesn't exist |
| `USERNAME_EXISTS` | 409 | Username already taken in tenant |
| `EMAIL_EXISTS` | 409 | Email already registered in tenant |
| `EMAIL_NOT_VERIFIED` | 403 | Email verification required by policy |
| `PASSWORD_TOO_WEAK` | 400 | Password doesn't meet complexity rules |
| `PASSWORD_IN_HISTORY` | 400 | Password was used recently |
| `PASSWORD_EXPIRED` | 403 | Password past expiration date |

### Roles & Permissions

| Code | HTTP | Description |
|------|------|-------------|
| `ROLE_NOT_FOUND` | 404 | Role ID doesn't exist |
| `ROLE_KEY_EXISTS` | 409 | Role key already taken in tenant |
| `PERMISSION_DENIED` | 403 | RBAC/ABAC policy denied access |
| `CIRCULAR_ROLE_REFERENCE` | 400 | Role hierarchy creates a cycle |

### Organizations

| Code | HTTP | Description |
|------|------|-------------|
| `ORG_NOT_FOUND` | 404 | Organization ID doesn't exist |
| `MEMBER_EXISTS` | 409 | User already member of this org |
| `MEMBER_NOT_FOUND` | 404 | User is not a member |

### OAuth

| Code | HTTP | Description |
|------|------|-------------|
| `INVALID_CLIENT` | 401 | Unknown client_id or invalid secret |
| `INVALID_GRANT` | 400 | Authorization code expired or already used |
| `INVALID_REDIRECT_URI` | 400 | Redirect URI doesn't match registered |
| `INVALID_SCOPE` | 400 | Requested scope not allowed for client |
| `UNSUPPORTED_GRANT_TYPE` | 400 | Grant type not configured for client |

### SCIM

| Code | HTTP | Description |
|------|------|-------------|
| `INVALID_SYNTAX` | 400 | Malformed SCIM request |
| `INVALID_FILTER` | 400 | Bad SCIM filter expression |
| `UNIQUENESS` | 409 | Duplicate SCIM resource |

---

## Error Handling by SDK

### Go

```go
var apiErr *ggid.APIError
if errors.As(err, &apiErr) {
    fmt.Println(apiErr.Code)       // "USERNAME_EXISTS"
    fmt.Println(apiErr.StatusCode) // 409
}
```

### Node.js

```typescript
catch (err: any) {
    const code = err.response?.data?.error?.code;
    // "USERNAME_EXISTS"
}
```

### Java

```java
catch (GGIDException e) {
    e.getStatusCode();  // 409
    e.getMessage();     // "Username already exists"
}
```

---

## Multi-Language Error Messages

GGID supports localized error messages via `Accept-Language` header:

```bash
GET /api/v1/users/999
Accept-Language: zh-CN
```

Response:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "用户不存在",
    "request_id": "req-abc123"
  }
}
```

### Supported Locales

| Locale | Language |
|--------|----------|
| `en-US` | English (default) |
| `zh-CN` | Chinese (Simplified) |
| `ja-JP` | Japanese |
| `ko-KR` | Korean |
| `es-ES` | Spanish |
| `de-DE` | German |
| `fr-FR` | French |

### Fallback

If the requested locale is not available, GGID falls back to English (`en-US`).
