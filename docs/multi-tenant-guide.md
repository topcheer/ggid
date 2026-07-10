# GGID Multi-Tenant Architecture Guide

How GGID isolates tenant data and manages multi-tenancy.

---

## Overview

GGID is a multi-tenant IAM platform. Multiple tenants share the same
infrastructure while maintaining strict data isolation.

### Isolation Model: Shared Database with RLS

```
┌─────────────────────────────────────┐
│         PostgreSQL Database          │
│                                     │
│  ┌──────────┐  ┌──────────┐        │
│  │ Tenant A │  │ Tenant B │  ...   │
│  │  users   │  │  users   │        │
│  │  roles   │  │  roles   │        │
│  │  orgs    │  │  orgs    │        │
│  └──────────┘  └──────────┘        │
│                                     │
│  RLS Policy: WHERE tenant_id =     │
│    current_setting('app.tenant_id') │
└─────────────────────────────────────┘
```

All tenants share the same tables, separated by `tenant_id` column and
PostgreSQL Row-Level Security (RLS).

---

## Tenant Lifecycle

### Create a Tenant

```sql
INSERT INTO tenants (id, name)
VALUES ('aabbcc00-0000-0000-0000-000000000001', 'Acme Corp');
```

### Default Tenant

GGID ships with a default tenant:

```
ID: 00000000-0000-0000-0000-000000000001
Name: Default
```

### Tenant ID Format

Tenant IDs are UUIDs (UUIDv4). They are required on every API request via
the `X-Tenant-ID` header.

---

## How RLS Works

### Table Structure

Every multi-tenant table has a `tenant_id UUID NOT NULL` column:

```sql
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    username    TEXT NOT NULL,
    email       TEXT NOT NULL,
    UNIQUE(tenant_id, username),   -- uniqueness scoped per tenant
    UNIQUE(tenant_id, email)
);
```

### RLS Policy

```sql
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON users
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### Tenant Context

The application sets `app.tenant_id` at the start of each transaction:

```go
tx, _ := pool.Begin(ctx)
// SET LOCAL scopes to this transaction; auto-resets on commit
tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID))
// All queries in this transaction are now tenant-scoped
rows, _ := tx.Query(ctx, "SELECT * FROM users WHERE email = $1", email)
tx.Commit(ctx)  // tenant context auto-cleared
```

> **Note:** `SET LOCAL` does NOT support `$1` parameters in pgx v5.
> Use `fmt.Sprintf` with a validated UUID string.

---

## Tenant Context Flow

```
HTTP Request
  │
  ├─ Header: X-Tenant-ID: 00000000-0000-0000-0000-000000000001
  │
  ▼
Gateway
  │ Extracts X-Tenant-ID
  ├─ Injects as query param (GET): ?tenant_id=00000000-...
  ├─ Injects in JSON body (POST/PUT): {"tenant_id": "00000000-..."}
  │
  ▼
Backend Service
  │ tenant.FromContext(ctx) extracts tenant_id
  │
  ▼
Repository Layer
  │ tx.Exec("SET LOCAL app.tenant_id = '00000000-...'")
  │
  ▼
PostgreSQL RLS
  │ Automatically filters: WHERE tenant_id = '00000000-...'
  │ Even if application code has a bug, DB enforces isolation
```

---

## Data Isolation Guarantees

| Scenario | Isolation Method | Result |
|----------|-----------------|--------|
| User query without tenant | RLS filters by session var | Only sees own tenant's users |
| SQL injection in WHERE clause | RLS adds `AND tenant_id = ...` | Still isolated |
| Forgotten WHERE clause | RLS enforces on all queries | Still isolated |
| Direct DB access (superuser) | Superuser bypasses RLS | NOT isolated (use non-superuser app role) |
| Cross-tenant join | Not possible (different tenant_ids) | Empty result |

### Verification

```sql
-- Check RLS is enabled and forced
SELECT relname, relrowsecurity, relforcerowsecurity
FROM pg_class
WHERE relname IN ('users', 'roles', 'audit_events');
-- Should show: t, t for all multi-tenant tables

-- Test isolation (as app user, NOT superuser)
SET app.tenant_id = 'tenant-a-uuid';
SELECT count(*) FROM users;  -- returns only Tenant A's users

SET app.tenant_id = 'tenant-b-uuid';
SELECT count(*) FROM users;  -- returns only Tenant B's users
```

---

## Tenant Switching

### For API Consumers

Include `X-Tenant-ID` header with the target tenant UUID:

```bash
# Tenant A
curl -H "X-Tenant-ID: tenant-a-uuid" \
     -H "Authorization: Bearer $TOKEN" \
     $GW/api/v1/users

# Tenant B
curl -H "X-Tenant-ID: tenant-b-uuid" \
     -H "Authorization: Bearer $TOKEN" \
     $GW/api/v1/users
```

### JWT Tenant Binding

JWTs contain a `tenant_id` claim. The Gateway verifies that the request's
`X-Tenant-ID` matches the JWT's `tenant_id`:

```json
{
  "sub": "user-uuid",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "roles": ["admin"],
  "exp": 1720612860
}
```

Users can only access their own tenant (unless cross-tenant federation is
configured — Phase 15).

---

## Tables with RLS

| Table | RLS Column | Unique Constraint |
|-------|-----------|-------------------|
| `users` | `tenant_id` | `UNIQUE(tenant_id, username)` |
| `credentials` | `tenant_id` | `UNIQUE(tenant_id, user_id, type)` |
| `roles` | `tenant_id` | `UNIQUE(tenant_id, key)` |
| `role_permissions` | `tenant_id` | — |
| `user_roles` | `tenant_id` | `UNIQUE(tenant_id, user_id, role_id)` |
| `organizations` | `tenant_id` | — |
| `org_members` | `tenant_id` | `UNIQUE(tenant_id, org_id, user_id)` |
| `departments` | `tenant_id` | — |
| `teams` | `tenant_id` | — |
| `policies` | `tenant_id` | — |
| `audit_events` | `tenant_id` | — |
| `oauth_clients` | `tenant_id` | — |

---

## Index Strategy

All indexes include `tenant_id` as the first column:

```sql
CREATE INDEX idx_users_tenant_email ON users (tenant_id, email);
CREATE INDEX idx_audit_tenant_time ON audit_events (tenant_id, created_at DESC);
```

This ensures:
1. RLS filter (`WHERE tenant_id = ...`) hits the index
2. Tenant-scoped queries are fast regardless of total table size

---

## Production Considerations

### Non-Superuser Application Role

In production, create a database role that cannot bypass RLS:

```sql
CREATE ROLE ggid_app WITH LOGIN PASSWORD '...' NOBYPASSRLS;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES TO ggid_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ggid_app;
```

### Connection Pool Isolation

Each service sets `SET LOCAL app.tenant_id` per transaction. This is
transaction-scoped, so different transactions on the same pooled connection
can serve different tenants safely.

### Noisy Neighbor Prevention

To prevent one tenant from consuming all resources:

1. **Per-tenant rate limiting** (Gateway level)
2. **Query timeouts** (`statement_timeout` in PostgreSQL)
3. **Connection pool limits** per service

---

## Alternative: Database-Per-Tenant

For tenants requiring physical isolation (regulated industries):

```
Tenant A → Database A (dedicated PostgreSQL instance)
Tenant B → Database B (dedicated PostgreSQL instance)
```

The GGID architecture supports this by configuring different `DATABASE_URL`
per tenant. However, this is not the default deployment and requires
additional operational overhead.
