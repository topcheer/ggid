# GGID API Conventions

## HTTP Status Codes

| Code | Usage |
|------|-------|
| 200 | GET, PUT, DELETE success |
| 201 | POST resource creation (users, roles, policies, clients) |
| 204 | No content (e.g. OPTIONS preflight) |
| 400 | Invalid request body, missing required fields |
| 401 | Missing or invalid authentication |
| 403 | Insufficient permissions |
| 404 | Resource not found |
| 409 | Conflict (duplicate resource) |
| 429 | Rate limited |
| 500 | Internal server error (never expose details) |
| 503 | Service unavailable (infra-level only) |

## Error Response Format

All errors MUST use `writeError(w, status, message)`:

```json
{
  "error": "invalid request body",
  "code": "INVALID_ARGUMENT",
  "request_id": "req-..."
}
```

Error codes map from HTTP status via `httpStatusToCode()`.

## Pagination

List endpoints SHOULD support:

| Param | Default | Description |
|-------|---------|-------------|
| `page_size` | 50 | Items per page (max 200) |
| `offset` | 0 | Skip N items |
| `search` | "" | Full-text search |

Response:
```json
{
  "items": [...],
  "total": 150,
  "next_offset": 50
}
```

## Authentication

- All endpoints (except `/healthz`, `/login`, `/register`) require `Authorization: Bearer <JWT>`
- Tenant context via `X-Tenant-ID` header (UUID)
- Internal service calls use `X-Internal-Secret` HMAC

## Rate Limiting

Sensitive endpoints MUST be rate limited:
- `/api/v1/auth/login` — 5 attempts/5min per IP
- `/api/v1/auth/register` — 3/hour per IP
- `/api/v1/auth/mfa/verify` — 3/5min per user
- `/api/v1/auth/forgot-password` — 3/hour per email

## Graceful Degradation

GET endpoints with optional dependencies MUST return empty results (200) when the dependency is not configured:
```json
{"items": [], "count": 0}
```
Never return 503 for feature-level endpoints. 503 is reserved for infra-level failures only.
