# API Consistency Audit

**Date**: 2025-07-18  
**Version**: v1.0-beta  

---

## 1. Pagination Support

### Services with pagination ✅
- **Identity**: `listUsers` supports `page` + `page_size` query params
- **Audit**: `handleEvents` supports `page_size` with default 50
- **Policy**: `listRoles`, `listPolicies` support pagination

### Missing pagination ⚠️
Several identity endpoints return full lists without pagination:
- Group listing
- Org listing  
- Department/team listing
- NHI inventory
- Federation entities
- SCIM tokens

**Recommendation**: Add `page`/`page_size` support to these endpoints before v1.0-stable. For lists <1000 items, current behavior is acceptable.

### Audit score: 6/10

---

## 2. HTTP Status Codes

### Create endpoints return 201 ✅
All verified POST handlers return `http.StatusCreated`:
- `POST /users` → 201 ✅
- `POST /roles` → 201 ✅
- `POST /policies` → 201 ✅
- `POST /permissions` → 201 ✅
- `POST /oauth/clients` → 201 ✅
- `POST /webhooks` → 201 ✅
- `POST /register` → 201 ✅
- `POST /policies/sod/rules` → 201 ✅

### List endpoints return 200 ✅
All GET list handlers return `http.StatusOK`.

### Delete endpoints return 200 ✅
All DELETE handlers return 200 with `{status: "deleted"}`.

### Score: 9/10 ✅

---

## 3. Error Response Format

### Two formats in use ⚠️

**Format A** (auth service): `writeError(w, status, "message")`
```json
{"error": "message"}
```

**Format B** (pkg/errors): `WriteSimpleAPIError(w, status, code, message)`
```json
{"error": {"code": "ERROR_CODE", "message": "human-readable"}}
```

**Inconsistency**: Auth service uses Format A (simple string), while newer handlers (authz_check, TAP, conditional access) use Format B (structured).

**Recommendation**: Migrate all auth handlers to Format B for v1.0-stable. Format B is more useful for frontend i18n.

### Score: 5/10 ⚠️

---

## 4. Rate Limiting Coverage

| Endpoint | Rate Limited | Method |
|----------|-------------|--------|
| `/api/v1/auth/login` | ✅ | Per-user + per-IP (Redis) |
| `/api/v1/auth/register` | ✅ | Per-IP (middleware) |
| `/api/v1/auth/password/forgot` | ✅ | Per-email |
| `/api/v1/auth/password/reset` | ✅ | Per-token |
| `/api/v1/auth/refresh` | ✅ | Per-token |
| `/api/v1/oauth/authorize` | ✅ | Per-client |
| `/api/v1/oauth/token` | ⚠️ | Via gateway only |
| `/api/v1/auth/login-attempts/:username` (DELETE) | ⚠️ | Admin-only (no rate limit) |
| Global API | ✅ | Per-tenant bucket limiter |
| GraphQL | ✅ | Per-user (100/min) |

### Score: 8/10 ✅

---

## Summary

| Check | Score | Status |
|-------|-------|--------|
| Pagination | 6/10 | ⚠️ Some list endpoints missing |
| Status codes | 9/10 | ✅ Consistent 200/201 |
| Error format | 5/10 | ⚠️ Two formats in use |
| Rate limiting | 8/10 | ✅ All sensitive endpoints covered |
| **Overall** | **7/10** | **Acceptable for beta** |

### Priority fixes for v1.0-stable
1. Unify error response format to `{error: {code, message}}` 
2. Add pagination to identity list endpoints (groups, orgs, NHI)
3. Add rate limiting to DELETE login-attempts endpoint
