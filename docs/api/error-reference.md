# Error Reference

> Complete error code reference. Every HTTP status + error code + meaning + fix.

---

## HTTP Status Codes

### 400 Bad Request

| Error Code | Meaning | Fix |
|------------|---------|-----|
| `validation_error` | Missing or invalid field | Check required fields in request body |
| `invalid_json` | Malformed JSON body | Validate JSON syntax before sending |
| `invalid_filter` | Bad SCIM filter syntax | Review filter operators (eq, ne, co, sw, pr) |
| `invalid_password` | Password doesn't meet policy | Min 8 chars, uppercase, lowercase, number |

### 401 Unauthorized

| Error Code | Meaning | Fix |
|------------|---------|-----|
| `missing_token` | No Authorization header | Add `Authorization: Bearer <jwt>` |
| `invalid_token` | JWT signature/expiry invalid | Refresh token or re-login |
| `token_expired` | JWT past `exp` claim | Use refresh token to get new JWT |
| `invalid_credentials` | Wrong username or password | Verify credentials |

### 403 Forbidden

| Error Code | Meaning | Fix |
|------------|---------|-----|
| `insufficient_scope` | JWT missing required scope | Assign role with needed permission |
| `insufficient_role` | User lacks required role | Assign role via `/api/v1/users/{id}/roles` |
| `forbidden` | Policy check denied | Review RBAC/ABAC policies |
| `tenant_mismatch` | JWT tenant != requested tenant | Use correct `X-Tenant-ID` header |

### 404 Not Found

| Error Code | Meaning | Fix |
|------------|---------|-----|
| `not_found` | Resource doesn't exist | Verify resource ID |
| `route_not_found` | Unknown API path | Check API docs for correct endpoint |

### 409 Conflict

| Error Code | Meaning | Fix |
|------------|---------|-----|
| `user_exists` | Username already registered | Use different username |
| `role_key_exists` | Role key not unique in tenant | Use different `key` field |
| `conflict` | Duplicate resource | Check existing resources first |

### 429 Too Many Requests

| Error Code | Meaning | Fix |
|------------|---------|-----|
| `rate_limited` | Request rate exceeded | Wait for `Retry-After` seconds |
| `login_rate_limited` | Too many login attempts | Wait 60s or restart auth container |

### 500 Internal Server Error

| Error Code | Meaning | Fix |
|------------|---------|-----|
| `internal_error` | Unhandled server error | Check server logs |
| `database_error` | DB connection/query failed | Check `DB_HOST`, `DB_PASSWORD` |
| `nats_error` | NATS publish failed | Check `NATS_URL` connectivity |

### 502 Bad Gateway

| Error Code | Meaning | Fix |
|------------|---------|-----|
| `upstream_unavailable` | Backend service down | Check service health, restart container |

---

## SCIM Error Types

| scimType | HTTP | Meaning |
|----------|------|---------|
| `invalidSyntax` | 400 | Malformed JSON |
| `invalidVers` | 400 | Unsupported schema version |
| `tooMany` | 400 | Too many results for filter |
| `uniqueness` | 409 | Attribute must be unique |
| `mutability` | 400 | Attempted to modify read-only attribute |
| `sensitive` | 400 | Sensitive attribute access denied |

---

## OAuth Error Codes

| Error Code | Meaning |
|------------|---------|
| `invalid_request` | Missing required parameter |
| `invalid_client` | Client authentication failed |
| `invalid_grant` | Authorization code or refresh token invalid |
| `unauthorized_client` | Client not authorized for this grant type |
| `unsupported_grant_type` | Grant type not supported |
| `invalid_scope` | Requested scope exceeds allowed |
| `access_denied` | Resource owner denied consent |
| `server_error` | Authorization server error |

---

## Error Response Format

All errors use a consistent format:

```json
{
  "error": "error_code",
  "message": "Human-readable description",
  "request_id": "req_abc123"
}
```

Rate limit responses include `Retry-After` header:

```
HTTP/1.1 429 Too Many Requests
Retry-After: 30
```

---

*See: [REST API Reference](rest-api.md) | [Admin API](admin-api.md) | [Troubleshooting](../guides/troubleshooting.md)*

*Last updated: 2025-07-11*
