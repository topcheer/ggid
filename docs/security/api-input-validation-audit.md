# KB-362: API Input Validation Audit

## Summary

Audit of all POST/PUT endpoints across auth, identity, policy, and audit services
for input validation, SQL injection protection, length limits, and error leakage.

## SQL Injection Protection: PASS

All database queries use parameterized queries (`$1`, `$2` via pgx).
The `fmt.Sprintf` calls found in the codebase only interpolate internal column
name constants (e.g., `userColumns`, `mfaColumns`), never user input.

Examples of safe patterns:
- `pool.QueryRow(ctx, "SELECT ... WHERE email = $1", email)` — parameterized
- `fmt.Sprintf("SELECT %s FROM users WHERE ...", userColumns)` — constant columns
- `fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", cols, table, col)` — config-driven, not user input

**No SQL injection vulnerabilities found.**

## Required Field Validation: MOSTLY PASS

### Auth Service
- **login**: Validates `username` and `password` are non-empty (implicit via auth provider)
- **register**: Checks `username`, `email`, `password` presence; email format validated via `validateEmail()`
- **password/reset**: Validates `token` is required
- **password/forgot**: Validates `email` is required
- **MFA verify**: Validates `code` is required

### Identity Service
- **create user**: Validates `username`, `email`, `password` via domain layer
- **update user**: Validates UUID format for path param

### Policy Service
- **roles/assign**: Validates `role_id` is required; admin check via `isAdminRequest()`
- **policy conflicts**: Validates JSON body structure

### Audit Service
- **webhook create**: Validates `url` field present
- **CCM run**: Body optional (runs all controls)

**Gap**: Some POST endpoints accept empty body silently (e.g., session revoke).
Non-critical — these use query params or path params instead.

## Length Limits: PARTIAL

- **Username**: No explicit max length (DB column is `varchar(200)`)
- **Password**: No max length enforced at API layer (bcrypt truncates at 72 bytes)
- **Email**: DB column is `varchar(255)`, API validates format but not length
- **JWT token**: Parsed by library which has built-in limits
- **Request body**: No explicit body size limit on most endpoints

**Recommendation**: Add middleware for max body size (1MB) in v1.1.

## Error Response Leakage: FIXED

### Before
- `multihash_handler.go:78` — returned `err.Error()` on 500 (could leak internal crypto details)

### After
- Changed to generic "internal rehash error" message
- All other 500 errors already use generic messages

### Remaining `err.Error()` on non-500 responses (safe):
- 400 Bad Request: shows validation error to client (intentional)
- 401 Unauthorized: shows token parse error (limited info)

## Tenant Isolation: PASS

- All queries go through `setTenantRLS()` which sets PostgreSQL row-level security
- Gateway forwards `X-Tenant-ID` header from JWT claims
- No cross-tenant data access possible without JWT manipulation

## Authorization: PASS

- Admin-only endpoints check `isAdminRequest()` which validates:
  - `X-User-Role` header (set by gateway from JWT scopes)
  - `X-Is-Admin` header (set by gateway for admin scope)
  - `X-Scopes` header (contains all JWT scopes)
- Regular user tokens without `admin` scope get 403 on admin endpoints
