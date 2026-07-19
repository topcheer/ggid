# API Input Validation Audit

**Date**: 2026-07-19  
**Scope**: All POST/PUT endpoints across 7 services  
**KB**: KB-362  

---

## Summary

| Check | Status | Notes |
|-------|--------|-------|
| Required field validation | **GOOD** | Core endpoints (login, register, password) validate required fields |
| SQL injection protection | **PASS** | All queries use pgx parameterized statements ($1, $2) |
| JSON decode error handling | **GOOD** | 143 handlers decode JSON; ~90% check `err != nil` |
| Input length limits | **GAP** | No explicit max-length checks on most fields |
| Error information leakage | **PASS** | Fixed in KB-326 (no internal header names in errors) |

---

## 1. Required Field Validation

### Core Auth Endpoints — PASS

| Endpoint | Fields Validated | Missing Check |
|----------|-----------------|---------------|
| `POST /api/v1/auth/login` | username + password required | None |
| `POST /api/v1/auth/register` | username + email + password required | None |
| `POST /api/v1/auth/password/forgot` | email required | None |
| `POST /api/v1/auth/password/reset` | token + password required | None |
| `POST /api/v1/auth/password/change` | current + new password required | None |
| `POST /api/v1/auth/mfa/enroll` | method required | None |
| `POST /api/v1/auth/mfa/verify` | code required | None |

### Identity Endpoints — PASS

| Endpoint | Fields Validated |
|----------|-----------------|
| `POST /api/v1/users` | username + email + password required |
| `PUT /api/v1/users/:id` | Partial update — all fields optional |
| `POST /scim/v2/Users` | SCIM schema validation (userName required) |

### Policy Endpoints — PARTIAL

| Endpoint | Issue |
|----------|-------|
| `POST /api/v1/roles` | name + key required — validated |
| `POST /api/v1/roles/assign` | user_id + role_id required — validated |
| `POST /api/v1/policies` | name + effect required — validated |
| `POST /api/v1/policies/attribute-mapping` | attribute + value required |
| `POST /api/v1/policy/conflicts` | Optional body — OK |

### OAuth Endpoints — PASS

| Endpoint | Fields Validated |
|----------|-----------------|
| `POST /oauth/token` | grant_type required; client auth enforced |
| `POST /api/v1/oauth/clients` | client_name + redirect_uris required |
| `POST /api/v1/agents/register` | name + type + owner_user_id required |

---

## 2. SQL Injection Protection — PASS

All database access uses `pgx` parameterized queries:

```go
// CORRECT — parameterized
pool.QueryRow(ctx, `SELECT * FROM users WHERE id = $1 AND tenant_id = $2`, userID, tenantID)

// No string concatenation found in any production query
```

Verified across: `services/auth/internal/repository/`, `services/identity/internal/`, `services/policy/internal/`, `services/audit/internal/`, `services/oauth/internal/`.

---

## 3. JSON Decode Error Handling

143 `json.NewDecoder(r.Body).Decode()` calls found. Pattern analysis:

| Pattern | Count | Risk |
|---------|-------|------|
| `if err := Decode(); err != nil { writeError(400) }` | ~128 | GOOD — returns 400 |
| `json.NewDecoder(r.Body).Decode(&req)` (no error check) | ~15 | LOW — defaults to zero-value struct |

The unchecked decoders are mostly in secondary endpoints (config, DLP rules) where zero-value defaults are acceptable. Core endpoints all check.

---

## 4. Input Length Limits — GAP

**Current state**: No explicit max-length validation on:
- `username` (should be ≤128 chars)
- `email` (should be ≤256 chars)
- `password` (should be ≤128 chars to prevent DoS on Argon2id)
- `display_name` (should be ≤256 chars)

**Risk**: 
- Argon2id on a 1MB password could cause CPU exhaustion (mitigated by Go's default body limit in gateway middleware: 10MB)
- PostgreSQL TEXT columns accept unlimited length (waste of storage)

**Recommendation**: Add `validateInput()` helper for core auth endpoints in v1.1.

---

## 5. Error Information Leakage — PASS

Fixed in KB-326:
- `/oauth/token` errors no longer expose internal header names
- Generic error messages: "missing or invalid tenant context"
- Stack traces only in development mode (controlled by `LOG_LEVEL=debug`)
- No SQL error details returned to client (mapped to generic 500)

---

## 6. Duplicate Handling

| Endpoint | Behavior | Status |
|----------|----------|--------|
| `POST /api/v1/auth/register` | 409 Conflict on duplicate email/username | PASS |
| `POST /api/v1/users` | 409 Conflict | PASS |
| `POST /api/v1/roles` | UNIQUE(tenant_id, key) constraint → 409 | PASS |
| `POST /api/v1/oauth/clients` | UNIQUE(client_id) → 409 | PASS |

---

## Recommendations for v1.1

1. **Add `validateAuthInput()` helper**: Check length limits (username ≤128, password ≤128, email ≤256)
2. **Rate-limit decode failures**: Track repeated 400 responses per IP
3. **Add request body size middleware**: Enforce 1MB max for all POST/PUT
4. **Standardize validation error format**: `{"error": {"field": "username", "message": "required"}}`
