# GGID API Versioning Strategy

How GGID manages API versions, deprecations, and breaking changes.

---

## Versioning Scheme

### URI-Based (Implemented)

GGID uses URI path versioning:

```
/api/v1/users
/api/v1/auth/login
/api/v1/policies/check
```

The version number (`v1`, `v2`) is embedded in the URL path.

### Why URI Versioning?

| Approach | Pros | Cons | Used? |
|----------|------|------|:-----:|
| **URI path** (`/v1/`) | Explicit, cacheable, simple | URL changes on major version | Yes |
| Header (`Api-Version: 2`) | Clean URLs, no URL change | Hard to test in browser, poor caching | No |
| Query param (`?v=2`) | Simple | Breaks caching, easy to forget | No |
| Content negotiation | RESTful, flexible | Complex, poor tooling support | No |

URI versioning is the most practical for a developer-facing API with SDK support.

---

## What Constitutes a Breaking Change?

### Breaking (Requires New Version)

| Change | Example |
|--------|---------|
| Remove field from response | Remove `username` from user object |
| Change field type | `age: int` → `age: string` |
| Change URL structure | `/users/{id}` → `/user/{id}` |
| Change HTTP method | `POST /login` → `PUT /login` |
| Change error format | Different error response structure |
| Remove endpoint | Delete `/api/v1/legacy-endpoint` |
| Change default behavior | Default pagination from 50 to 100 |
| Require new mandatory field | Add required `tenant_id` to request body |

### Non-Breaking (Allowed in Same Version)

| Change | Example |
|--------|---------|
| Add new endpoint | `GET /api/v1/users/{id}/sessions` |
| Add optional request field | Optional `include_deleted` query param |
| Add new response field | Add `last_login` to user object |
| Add new error code | New 429 for rate limiting |
| Change documentation | Clarify field descriptions |
| Improve performance | Faster response times |
| Change internal implementation | Switch database, refactor code |

---

## Current Version: v1

All GGID endpoints are currently under `/api/v1/`.

### v1 Contract

- JSON request/response bodies
- Bearer token authentication (JWT RS256)
- `X-Tenant-ID` header required
- Offset-based pagination (`page`, `page_size`)
- Consistent error format (see [API Conventions](./api-conventions.md))

---

## Deprecation Policy

### Deprecation Timeline

```
Day 0:     v2 released, v1 endpoint marked deprecated
Day 0:     Deprecation + Sunset headers added to v1 responses
Day 180:   v1 endpoint still functional (6-month overlap)
Day 365:   v1 endpoint removed (1 year after deprecation)
```

### Deprecation Headers

Deprecated endpoints include these headers in every response:

```
Deprecation: true
Sunset: Sat, 10 Jul 2025 00:00:00 GMT
Link: <https://iam.example.com/api/v2/users>; rel="successor-version"
```

### Deprecation Announcement

1. **Release notes** — Documented in [CHANGELOG](./CHANGELOG.md)
2. **API response headers** — `Deprecation` and `Sunset` headers
3. **SDK warnings** — SDK logs deprecation warning to stderr
4. **Console banner** — Admin Console shows deprecation notice

---

## Version Migration

### When v2 is Released

```
/api/v1/users    → still works (deprecated, 6-month overlap)
/api/v2/users    → new version with changes
```

### Migration Steps

1. **Review changelog** — understand what changed
2. **Update SDK** — upgrade to latest SDK version (supports v2)
3. **Test against v2** — use the `/api/v2/` prefix
4. **Switch base URL** — change from `/api/v1/` to `/api/v2/`
5. **Monitor for deprecation warnings** — fix any remaining v1 calls

### SDK Version Support

| SDK | v1 Support | v2 Support | Auto-Detection |
|-----|:----------:|:----------:|:--------------:|
| Go | Forever (cached binary) | New version | Configurable base path |
| Node.js | Forever | New version | Configurable base path |
| Java | Forever | New version | Configurable base path |
| Python | Forever | New version | Configurable base path |

---

## Backward Compatibility Guarantees

### v1 Lifetime Commitments

1. **Existing fields will not be removed** — fields present in v1.0 remain
2. **Existing endpoints will not change URLs** — paths are frozen
3. **Error format will not change** — same JSON structure
4. **Authentication will not change** — JWT RS256 + JWKS

### New Fields

GGID may add new response fields in v1 without incrementing the version.

**Clients must ignore unknown fields** (forward compatibility):

```python
# Correct: ignore unknown fields
user = response.json()
name = user.get("username")  # don't access user["username"] directly

# Wrong: breaks if field is renamed in future
name = user["username"]  # KeyError risk
```

---

## OpenAPI Spec Versioning

The OpenAPI spec (`docs/openapi.yaml`) includes `info.version`:

```yaml
openapi: 3.1.0
info:
  title: GGID API
  version: 1.0.0
```

Each API version has its own spec:
- `docs/openapi.yaml` — v1 spec
- `docs/openapi-v2.yaml` — v2 spec (when released)

---

## FAQ

### Q: How often will GGID release new API versions?

Major versions (v2, v3) are released only when breaking changes are necessary.
Target: every 2-3 years. Minor changes (new endpoints, new fields) happen
continuously within the current version.

### Q: Will I be forced to upgrade?

No. v1 will remain functional for at least 1 year after v2 release. After
that, v1 may be removed, but you control your deployment timeline
(self-hosted).

### Q: How do I know if an endpoint is deprecated?

1. Check response headers for `Deprecation: true`
2. Check the [CHANGELOG](./CHANGELOG.md)
3. SDK logs warnings to stderr for deprecated endpoints

### Q: Can I pin to a specific version forever?

Yes. Since GGID is self-hosted, you can remain on a specific release version
indefinitely. However, you won't receive security patches or new features.
