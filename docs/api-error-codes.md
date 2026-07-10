# GGID API Error Codes Reference

Complete reference for all GGID error codes with HTTP status, description, and
retry guidance.

---

## Error Response Format

All errors follow a standard JSON structure:

```json
{
  "error": "invalid_request",
  "code": "AUTH_INVALID_CREDENTIALS",
  "description": "The username or password is incorrect.",
  "request_id": "req-abc-123",
  "retry_after": null
}
```

---

## AUTH_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `AUTH_INVALID_CREDENTIALS` | 401 | Wrong username or password | Yes (after delay) |
| `AUTH_ACCOUNT_LOCKED` | 423 | Account locked after too many failures | After lockout expires |
| `AUTH_ACCOUNT_SUSPENDED` | 403 | Account is suspended by admin | No |
| `AUTH_ACCOUNT_NOT_FOUND` | 404 | User does not exist | No |
| `AUTH_EMAIL_NOT_VERIFIED` | 403 | Email verification required | No |
| `AUTH_TOKEN_EXPIRED` | 401 | Access token has expired | Use refresh token |
| `AUTH_TOKEN_INVALID` | 401 | Token signature invalid or malformed | No |
| `AUTH_TOKEN_REVOKED` | 401 | Token has been revoked | Re-authenticate |
| `AUTH_REFRESH_TOKEN_INVALID` | 401 | Refresh token is invalid or reused | Re-authenticate |
| `AUTH_MFA_REQUIRED` | 403 | MFA verification needed | With MFA code |
| `AUTH_MFA_INVALID_CODE` | 401 | Wrong MFA code | Yes (limited) |
| `AUTH_PASSWORD_TOO_WEAK` | 400 | Password doesn't meet policy | No |
| `AUTH_PASSWORD_REUSE` | 400 | Password matches recent password | No |
| `AUTH_SESSION_EXPIRED` | 401 | Session has expired | Re-authenticate |

---

## OAUTH_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `OAUTH_INVALID_CLIENT` | 401 | Client ID or secret invalid | No |
| `OAUTH_INVALID_GRANT` | 400 | Authorization code is expired or invalid | No |
| `OAUTH_INVALID_REQUEST` | 400 | Missing required parameter | Fix and retry |
| `OAUTH_INVALID_SCOPE` | 400 | Requested scope not allowed | No |
| `OAUTH_UNAUTHORIZED_CLIENT` | 403 | Client not authorized for grant type | No |
| `OAUTH_UNSUPPORTED_GRANT_TYPE` | 400 | Grant type not supported | No |
| `OAUTH_REDIRECT_URI_MISMATCH` | 400 | Redirect URI doesn't match registration | No |
| `OAUTH_ACCESS_DENIED` | 403 | User denied consent | No |
| `OAUTH_SERVER_ERROR` | 500 | Internal OAuth server error | Yes (backoff) |

---

## POLICY_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `POLICY_ACCESS_DENIED` | 403 | RBAC/ABAC check failed | No |
| `POLICY_ROLE_EXISTS` | 409 | Role with same key already exists | No |
| `POLICY_ROLE_NOT_FOUND` | 404 | Role does not exist | No |
| `POLICY_INVALID_PERMISSION` | 400 | Permission string malformed | No |
| `POLICY_QUOTA_EXCEEDED` | 429 | Maximum roles for tier reached | No |

---

## IDENTITY_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `IDENTITY_USER_EXISTS` | 409 | Username or email already registered | No |
| `IDENTITY_USER_NOT_FOUND` | 404 | User does not exist | No |
| `IDENTITY_INVALID_EMAIL` | 400 | Email format invalid | No |
| `IDENTITY_INVALID_USERNAME` | 400 | Username format invalid | No |
| `IDENTITY_DUPLICATE_EMAIL` | 409 | Email already in use | No |
| `IDENTITY_ORG_NOT_FOUND` | 404 | Organization does not exist | No |

---

## SCIM_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `SCIM_INVALID_FILTER` | 400 | SCIM filter syntax error | Fix and retry |
| `SCIM_RESOURCE_NOT_FOUND` | 404 | SCIM resource (User/Group) not found | No |
| `SCIM_TOO_MANY` | 400 | Too many resources returned | Add pagination |
| `SCIM_INVALID_SCHEMA` | 400 | Request schema doesn't match SCIM spec | No |
| `SCIM_MUTABILITY` | 409 | Attempted to modify read-only attribute | No |

---

## TENANT_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `TENANT_NOT_FOUND` | 404 | Tenant ID does not exist | No |
| `TENANT_SUSPENDED` | 403 | Tenant is suspended | No |
| `TENANT_QUOTA_EXCEEDED` | 429 | Tenant has exceeded tier quota | Upgrade tier |
| `TENANT_INVALID_HEADER` | 400 | Missing or invalid X-Tenant-ID header | No |

---

## RATE_LIMIT_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `RATE_LIMIT_EXCEEDED` | 429 | Rate limit hit | After Retry-After |
| `RATE_LIMIT_QUOTA_EXCEEDED` | 429 | Monthly/daily quota exceeded | Next period |

### 429 Response Headers

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1721034600
Retry-After: 42
```

---

## WEBHOOK_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `WEBHOOK_DELIVERY_FAILED` | N/A | Webhook delivery to subscriber failed | Auto-retry (3x) |
| `WEBHOOK_SIGNATURE_INVALID` | 401 | Webhook signature verification failed | No |
| `WEBHOOK_EVENT_UNKNOWN` | 400 | Event type not recognized | No |

---

## Validation Errors (400)

```json
{
  "error": "validation_failed",
  "code": "VALIDATION_ERROR",
  "details": [
    { "field": "email", "message": "must be a valid email address" },
    { "field": "password", "message": "must be at least 12 characters" }
  ]
}
```

---

## Retry Guidance

| HTTP Status | Retry? | Backoff Strategy |
|-------------|--------|-----------------|
| 400 | No | Fix request |
| 401 | No | Re-authenticate |
| 403 | No | Fix permissions |
| 404 | No | Check resource ID |
| 409 | No | Handle conflict |
| 429 | Yes | Exponential: 1s, 2s, 4s, 8s (max 60s) |
| 500 | Yes | Exponential: 1s, 2s, 4s (max 3 retries) |
| 502, 503 | Yes | Fixed 5s delay (up to 10 retries) |

---

## References

- [API Reference](./api-reference.md) — Endpoint documentation
- [Rate Limiting](./api-rate-limiting.md) — Rate limit details
- [SDK Guide](./sdk-guide.md) — SDK error handling
