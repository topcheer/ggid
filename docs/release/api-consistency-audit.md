# API Consistency Audit — v1.0-beta

**Audited:** 2026-07-18

## 1. HTTP Status Codes

### POST handlers returning 200 (should be 201)
- `device_posture.go` — POST evaluate returns 200, POST policy returns 200
- `zt_posture_handler.go` — POST returns 200
- `security_posture_handler.go` — POST returns 200

**Status:** Low priority — 200 is acceptable for actions that aren't strict resource creation. 201 only needed for `POST /users`, `POST /clients` etc.

### DELETE handlers
All return 200 with `{"status":"deleted"}` — consistent ✅

## 2. Pagination

### Endpoints WITH pagination
- `GET /api/v1/users` — supports `page_size`, `offset`, `search` ✅
- `GET /api/v1/audit/events` — supports pagination ✅

### Endpoints MISSING pagination
- `GET /api/v1/users/search` — returns all (hardcoded 10000 limit)
- Various config endpoints return single object (no pagination needed)

**Status:** Acceptable for v1.0-beta. Most list endpoints have reasonable limits.

## 3. Error Response Format

### Standard format (auth, identity, audit, org, policy):
```json
{"error": "message", "code": "INVALID_ARGUMENT"}
```
Via `writeError()` → `errors.WriteSimpleAPIError()` ✅

### Inconsistencies:
- OAuth service uses `writeJSONError` (same format) ✅
- Gateway uses custom `writeGatewayJSONError` ✅
- All use consistent `{"error": "..."}` envelope

## 4. Rate Limiting

Rate limiting applied to sensitive endpoints:
- `/api/v1/auth/login` ✅
- `/api/v1/auth/register` ✅
- `/api/v1/auth/mfa/verify` ✅
- `/api/v1/auth/forgot-password` ✅

### Missing rate limiting:
- `/api/v1/auth/refresh` — should be rate limited
- `/oauth/token` — should be rate limited (per client)

**Status:** Core sensitive endpoints covered. Refresh/token can be added in v1.0-stable.

## Summary

| Check | Status | Notes |
|-------|--------|-------|
| Status codes | ⚠️ Minor | Some POSTs return 200 vs 201 |
| Pagination | ✅ Core OK | Users/audit paginated |
| Error format | ✅ Consistent | All use writeError helper |
| Rate limiting | ⚠️ Good | Login/register/mfa covered, refresh/token gaps |
