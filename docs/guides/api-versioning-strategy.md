# API Versioning Strategy

Guide for API versioning, deprecation, and backward compatibility in GGID.

## Versioning Approach: URL Path

GGID uses URL path versioning — the most explicit and cache-friendly approach:

```
GET /api/v1/users
GET /api/v2/users
```

| Approach | Pros | Cons | GGID Decision |
|----------|------|------|---------------|
| URL path | Explicit, cacheable, easy routing | URL pollution | ✅ Chosen |
| Header | Clean URLs | Not cacheable, hard to debug | ❌ |
| Content negotiation | RESTful ideal | Complex, poor tooling support | ❌ |

## Version Lifecycle

```
v1 released → v2 released → v1 deprecated → v1 sunset → v1 removed
   T=0           T+12mo        T+18mo          T+24mo
```

| Phase | Duration | Behavior |
|-------|----------|----------|
| Active | Until next major | Full support, bug fixes |
| Deprecated | 6 months | `Deprecation` + `Sunset` headers, no new features |
| Sunset | 3 months | `429 Too Many Requests` on 10% of calls (circuit breaker) |
| Removed | — | `410 Gone` |

### Deprecation Headers

```http
HTTP/1.1 200 OK
Deprecation: Sun, 30 Jun 2025 00:00:00 GMT
Sunset: Sat, 31 Dec 2025 00:00:00 GMT
Link: </api/v2/users>; rel="successor-version"
```

## Breaking vs Non-Breaking Changes

### Non-Breaking (no version bump)

- Adding optional request field
- Adding response field
- Adding new endpoint
- Adding enum value (with default)
- Loosening validation

### Breaking (requires version bump)

- Removing/renaming field
- Changing field type
- Adding required field
- Removing endpoint
- Changing status code semantics
- Tightening validation
- Changing error format

## Backward Compatibility Rules

Within the same major version (e.g., all v1.x):

```go
// Adding a field — safe
type UserResponse struct {
    ID       string `json:"id"`
    Email    string `json:"email"`
    Phone    string `json:"phone,omitempty"` // NEW: optional, safe
    Metadata struct {
        Department string `json:"department,omitempty"` // NEW: nested, safe
    } `json:"metadata,omitempty"`
}
```

Old clients ignore unknown fields. New clients handle absent fields via `omitempty`.

## SDK Synchronization

| Event | SDK Action | Timeline |
|-------|-----------|----------|
| v2 released | Publish v2 SDK alongside v1 | T+0 |
| v1 deprecated | Mark v1 SDK deprecated, log warning | T+12mo |
| v1 sunset | v1 SDK throws error on import | T+18mo |
| v1 removed | Archive v1 SDK repository | T+24mo |

```bash
# SDKs support both versions during transition
ggid.users.list({ version: "v2" })  // Explicit
ggid.users.list()                    // Defaults to latest stable
```

## Migration Tools

```bash
# Automated migration analyzer
ggid migrate analyze --from v1 --to v2
# Output:
# Breaking changes: 3
#   - users.phone renamed to users.mobile
#   - roles.permissions is now array (was string)
#   - DELETE /users/{id} returns 204 (was 200)
# Auto-fixable: 2
# Manual review: 1
```

## Version Matrix

| API Version | Status | GGID Release |
|-------------|--------|-------------|
| v1 | Active | 1.0 |
| v2 | Active | 2.0 |

gRPC versions use package suffix: `ggid.auth.v1`, `ggid.auth.v2`.

## Changelog Format

```markdown
## v2.0.0 (2025-01-15)

### Breaking
- `users.phone` renamed to `users.mobile`
- `roles.permissions` changed from string to array
- Removed `GET /api/v1/users/search` (use `GET /api/v2/users?filter=`)

### Added
- `PATCH /api/v2/users/{id}/mfa` endpoint
- `users.metadata` field for custom attributes

### Fixed
- Rate limit headers now include `X-RateLimit-Remaining`
```

## See Also

- [gRPC vs REST](grpc-vs-rest.md)
- [REST API Reference](../api/rest-api.md)
- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
