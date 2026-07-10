# Design: Multi-Tenant Row-Level Security (RLS)

> **Status:** Implemented | **ADR:** [ADR-004](../adr/ADR-004-rls-for-multi-tenancy.md)

## Problem Statement

GGID is a multi-tenant IAM platform. Multiple tenants share the same database,
but tenant data must be strictly isolated. A user in Tenant A must never see
Tenant B's users, roles, or audit events — even if application code has a bug.

## Design Goals

1. **Security-critical isolation** — cross-tenant data leakage must be impossible
2. **Defense in depth** — RLS is the last line of defense, not the only one
3. **Scalability** — support thousands of tenants without per-tenant schemas
4. **Operational simplicity** — one database, one migration path
5. **Performance** — minimal overhead from tenant filtering

## Solution

### Shared Tables with `tenant_id` Column

Every multi-tenant table includes a `tenant_id UUID NOT NULL` column:

```sql
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    username    TEXT NOT NULL,
    email       TEXT NOT NULL,
    -- ...
    UNIQUE(tenant_id, username),  -- uniqueness scoped per tenant
    UNIQUE(tenant_id, email)
);
```

### RLS Policy Definitions

```sql
-- Enable RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Force RLS even for table owners (defense in depth)
ALTER TABLE users FORCE ROW LEVEL SECURITY;

-- Policy: only rows where tenant_id matches the session variable
CREATE POLICY tenant_isolation ON users
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### Setting the Tenant Context

The application sets `app.tenant_id` at the start of each transaction:

```go
// In repository layer
func (r *Repository) WithTenant(ctx context.Context, tenantID uuid.UUID) (pgx.Tx, error) {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return nil, err
    }
    // SET LOCAL scopes to the transaction; reset on commit/rollback
    _, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID))
    if err != nil {
        tx.Rollback(ctx)
        return nil, err
    }
    return tx, nil
}
```

> **Important:** `SET LOCAL` does NOT support `$1` parameters in pgx v5.
Use `fmt.Sprintf` with a validated UUID string.

### Tables with RLS

| Table | RLS Column | Unique Constraint |
|-------|-----------|-------------------|
| `users` | `tenant_id` | `UNIQUE(tenant_id, username)` |
| `credentials` | `tenant_id` | `UNIQUE(tenant_id, user_id, type)` |
| `roles` | `tenant_id` | `UNIQUE(tenant_id, key)` |
| `role_permissions` | `tenant_id` | — |
| `user_roles` | `tenant_id` | `UNIQUE(tenant_id, user_id, role_id)` |
| `organizations` | `tenant_id` | — |
| `departments` | `tenant_id` | — |
| `teams` | `tenant_id` | — |
| `policies` | `tenant_id` | — |
| `audit_events` | `tenant_id` | — |

## Tenant Context Flow

```
HTTP Request (X-Tenant-ID header)
      │
      ▼
Gateway extracts X-Tenant-ID
      │
      ▼
Gateway injects into:
  - Query param (GET requests)
  - JSON body field (POST/PUT/PATCH)
      │
      ▼
Backend service handler
      │
      ▼
tenant.FromContext(ctx) extracts tenant_id
      │
      ▼
Repository sets SET LOCAL app.tenant_id = '<uuid>'
      │
      ▼
PostgreSQL RLS policy filters all rows
```

## Indexing Strategy

All indexes include `tenant_id` as the first column for efficient tenant-scoped lookups:

```sql
CREATE INDEX idx_users_tenant_username ON users (tenant_id, username);
CREATE INDEX idx_users_tenant_email    ON users (tenant_id, email);
CREATE INDEX idx_users_tenant_status   ON users (tenant_id, status);
CREATE INDEX idx_roles_tenant_key      ON roles (tenant_id, key);
CREATE INDEX idx_audit_tenant_time     ON audit_events (tenant_id, created_at DESC);
```

## Deployment Considerations

### Development (Docker Compose)

Docker Compose uses a superuser role (`ggid`) which **bypasses RLS**:
- `BYPASSRLS` is implicit for superusers
- This is acceptable for development but must be addressed for production

### Production

Create a non-superuser application role:

```sql
CREATE ROLE ggid_app WITH LOGIN PASSWORD '...' NOBYPASSRLS;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES TO ggid_app;
ALTER ROLE ggid_app NOBYPASSRLS;  -- explicit
```

### Verification

```sql
-- Check RLS is enabled and forced
SELECT relname, relrowsecurity, relforcerowsecurity
FROM pg_class WHERE relname IN ('users', 'roles', 'audit_events');
-- All should show: t, t

-- Test isolation (as ggid_app)
SET app.tenant_id = 'tenant-a-uuid';
SELECT count(*) FROM users;  -- returns only Tenant A users

SET app.tenant_id = 'tenant-b-uuid';
SELECT count(*) FROM users;  -- returns only Tenant B users
```

## Trade-offs

| Aspect | Benefit | Cost |
|--------|--------|------|
| Shared tables | Simple operations, one migration | No per-tenant backup |
| RLS enforcement | Defense in depth | Small per-query overhead |
| `tenant_id` in constraints | Correct uniqueness per tenant | Slightly larger indexes |
| `SET LOCAL` per transaction | Automatic cleanup on commit | Easy to forget in new code |

## Alternative Considered: Database-per-Tenant

Rejected because:
- Managing thousands of databases is operationally expensive
- Schema migrations become per-database
- Cross-tenant analytics is difficult
- Connection pool management is complex

However, the architecture supports database-per-tenant as a deployment
variant for tenants requiring physical isolation (regulated industries).
