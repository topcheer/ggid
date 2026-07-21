# ADR: Dynamic RBAC Replacing Hardcoded Admin Prefixes

> **Status**: Approved
> **Date**: 2026-07-22
> **Author**: arch_pm
> **Implementor**: ggcxf_backend

## Context

The gateway's RBAC middleware (`services/gateway/internal/middleware/rbac.go`) uses a hardcoded `adminPrefixes` array to determine which API paths require admin-level access. Every time a new admin endpoint or role is added, the code must be modified, rebuilt, and redeployed.

**Current hardcoded paths:**
```
/api/v1/users, /api/v1/audit/, /api/v1/policies, /api/v1/webhooks,
/api/v1/oauth/clients, /api/v1/roles, /api/v1/admin/, /api/v1/settings/,
/api/v1/system/, /api/v1/tenants, /api/v1/impersonate
```

## Existing Infrastructure

The DB already has a `role_route_permissions` table:
```sql
role_id          UUID  -- FK to roles.id
route_prefix     TEXT  -- e.g. "/api/v1/users"
permission_level TEXT  -- "read" | "write" | "admin" (default: "read")
```

Additionally:
- `roles` table: role definitions with `key` (e.g. "platform:admin") and `name` (display name)
- `user_roles` table: maps users to roles
- `role_permissions` table: maps roles to fine-grained permissions (e.g. "users:read")

Redis is available for caching (already used by auth/identity services).

## Design

### Principle: DB-driven, Redis-cached, Code-stable

The middleware reads route→role mappings from DB (via Redis cache), eliminating code changes for new roles or endpoints.

### Architecture

```
Request → JWT extract roles → Check Redis cache for route permissions
                                   ↓ (cache miss)
                              Query PostgreSQL
                                   ↓
                         Cache result (TTL: 60s)
                                   ↓
                         Allow/Deny based on permission_level
```

### Data Flow

1. **JWT contains `roles` claim** (array of role display names, e.g. `["Administrator", "Tenant Administrator"]`)
2. **Gateway resolves role names to role IDs** via Redis-cached `roles` lookup table
3. **Gateway checks `role_route_permissions`** for the request path prefix
4. **Permission decision**: if any of the user's roles has `permission_level >= required_level` for the matching route prefix, access is granted

### Permission Levels

| Level | Meaning | Example |
|-------|---------|---------|
| `read` | GET requests only | Viewing user list |
| `write` | GET + POST/PATCH/DELETE | Creating users, editing roles |
| `admin` | All methods including dangerous ones | Tenant management, system config |

### Fallback Strategy

To avoid lockout if DB/Redis are unavailable:

1. **Cache warm-start**: On gateway startup, pre-load all `role_route_permissions` into memory
2. **Fallback to hardcoded**: If Redis AND DB are both unreachable, fall back to the current hardcoded `adminPrefixes` (kept as `defaultAdminPrefixes` for safety)
3. **Platform admin bypass**: Users with `platform:admin` or `Administrator` role always pass (superuser)

### Seed Data Migration

Insert default route permissions matching current hardcoded behavior:

```sql
-- Find the "Administrator" role ID
WITH admin_role AS (
  SELECT id FROM roles WHERE key = 'platform:admin' OR name = 'Administrator' LIMIT 1
)
INSERT INTO role_route_permissions (role_id, route_prefix, permission_level)
SELECT id, prefix, 'admin' FROM admin_role
CROSS JOIN (VALUES
  ('/api/v1/users'), ('/api/v1/audit/'), ('/api/v1/policies'),
  ('/api/v1/webhooks'), ('/api/v1/oauth/clients'), ('/api/v1/roles'),
  ('/api/v1/admin/'), ('/api/v1/settings/'), ('/api/v1/system/'),
  ('/api/v1/tenants'), ('/api/v1/impersonate')
) AS t(prefix)
ON CONFLICT (role_id, route_prefix) DO NOTHING;
```

Similar seeds for `tenant:admin` (write level) and `manager` (read level for most, write for own resources).

### Redis Cache Structure

```
Key: rbac:routes:{role_id}
Value: JSON array of {route_prefix, permission_level}
TTL: 60 seconds

Key: rbac:roles:name2id
Value: JSON map of {role_name → role_id}
TTL: 300 seconds (5 min, roles change rarely)
```

Cache invalidation: when role_route_permissions are modified via admin API, delete the corresponding Redis key.

### Interface Change

```go
// RequireAdminScope signature stays the same — no breaking change
func RequireAdminScope(next http.Handler) http.Handler { ... }

// Internal: isAdminEndpoint becomes DB-driven
func (gw *Gateway) isAdminEndpoint(path string, roles []string) bool {
    // 1. Check platform admin bypass
    // 2. Check Redis cache for route permissions
    // 3. Fallback to defaultAdminPrefixes if cache/DB unavailable
}
```

### Public Path Exemptions

Paths in `publicPaths` (login, register, healthz, etc.) are always accessible regardless of RBAC. The `isAdminEndpoint` function only applies to non-public paths.

## Implementation Plan (for ggcxf_backend)

### Step 1: Migration
- Create migration to seed `role_route_permissions` with current admin paths
- Ensure all existing roles (Administrator, Platform Administrator, Tenant Administrator) have appropriate entries

### Step 2: Redis Cache Layer
- Add `RBACCache` struct in `services/gateway/internal/middleware/`
- Methods: `GetRoutePermissions(roleID uuid.UUID) []RoutePermission`
- Cache miss → query DB → cache result

### Step 3: Middleware Update
- Replace `isAdminEndpoint(path string)` with `isAdminEndpoint(path string, userRoles []string) bool`
- Use cache layer for lookups
- Keep `defaultAdminPrefixes` as fallback

### Step 4: Admin API for Route Permissions
- `GET /api/v1/admin/rbac/routes` — list all route permissions
- `POST /api/v1/admin/rbac/routes` — add/update route permission for a role
- `DELETE /api/v1/admin/rbac/routes/{role_id}/{prefix}` — remove

## Tradeoffs

| Aspect | Current (hardcoded) | Proposed (DB-driven) |
|--------|--------------------|--------------------|
| Adding new role | Code change + rebuild + redeploy | DB insert |
| Performance | O(1) array scan | O(1) with Redis cache, O(n) on cache miss |
| Flexibility | Rigid | Dynamic |
| Lockout risk | None | Low (fallback to hardcoded) |
| Complexity | Minimal | Moderate (cache + DB) |

## Risks & Mitigations

1. **Cache stampede**: Mitigated by 60s TTL + warm-start
2. **Role name changes**: Role ID is stable FK; name resolution cached separately
3. **New route not in DB**: Falls through to public path check; if not public, requires admin (safe default)

## Acceptance Criteria

1. New role can access admin endpoints by inserting a DB row — no code change
2. Existing hardcoded behavior preserved as fallback
3. Gateway starts correctly with empty `role_route_permissions` table (uses defaults)
4. Redis cache hit ratio > 95% under normal operation
5. Zero-downtime deployment (no API contract change)
