# Row-Level Security (RLS) Tenant Isolation — Technical Guide

> Feature: PostgreSQL RLS for Multi-Tenant Isolation
> Location: `services/oauth/migrations/000001_initial_schema.up.sql`, `services/oauth/internal/repository/pg_repo.go`

## What It Does

GGID uses PostgreSQL Row-Level Security (RLS) to enforce tenant isolation at the database level. Even if application logic has a bug, RLS prevents one tenant from accessing another tenant's data.

## How It Works

### 1. RLS Policy Definition

Each table has an RLS policy that filters rows by `tenant_id`:

```sql
ALTER TABLE oauth_clients ENABLE ROW LEVEL SECURITY;
ALTER TABLE oauth_clients FORCE ROW LEVEL SECURITY;

CREATE POLICY oauth_clients_tenant_isolation ON oauth_clients
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id'))
    WITH CHECK (tenant_id = current_setting('app.tenant_id'));
```

- **USING**: Filters existing rows on SELECT/UPDATE/DELETE.
- **WITH CHECK**: Validates new rows on INSERT/UPDATE.
- **FORCE**: Applies RLS even to table owners.

### 2. Tenant Context Setting

Before any query, the application sets the tenant context within the transaction:

```go
func setTenantRLS(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
    _, err := tx.Exec(ctx,
        fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID.String()))
    return err
}
```

`SET LOCAL` ensures the setting is scoped to the current transaction — it automatically resets when the transaction commits or rolls back.

### 3. Query Execution

All subsequent queries within the transaction are automatically filtered:

```sql
-- This query only returns rows where tenant_id matches app.tenant_id
SELECT * FROM oauth_clients WHERE client_name LIKE '%test%';
```

No `WHERE tenant_id = ?` clause is needed in application code — RLS handles it transparently.

## Tables with RLS

| Table | Service | Policy |
|-------|---------|--------|
| `oauth_clients` | OAuth | tenant_isolation |
| `oauth_authorization_codes` | OAuth | tenant_isolation |
| `oidc_id_tokens` | OAuth | tenant_isolation |
| `oauth_refresh_tokens` | OAuth | tenant_isolation |
| `consent_records` | Identity | tenant_isolation |
| `audit_events` | Audit | tenant_isolation |

## BYPASSRLS

The `BYPASSRLS` attribute is granted only to specific roles:

| Role | BYPASSRLS | Use Case |
|------|-----------|----------|
| `ggid_app` | No | Normal application queries |
| `ggid_admin` | Yes | Migration scripts, cross-tenant admin operations |
| `ggid_migrate` | Yes | Schema migrations |

> **Security rule**: Application service accounts must NEVER have BYPASSRLS.

## Isolation Testing

### Manual Test

```sql
-- Set tenant A context
SET app.tenant_id = 'tenant-a-uuid';
SELECT count(*) FROM oauth_clients; -- Returns only tenant A's clients

-- Switch to tenant B
SET app.tenant_id = 'tenant-b-uuid';
SELECT count(*) FROM oauth_clients; -- Returns only tenant B's clients
```

### Automated Test

```go
func TestRLSTenantIsolation(t *testing.T) {
    // Insert client for tenant A
    ctx := context.Background()
    tx, _ := pool.Begin(ctx)
    defer tx.Rollback(ctx)

    setTenantRLS(ctx, tx, tenantA)
    tx.Exec(ctx, "INSERT INTO oauth_clients (id, tenant_id, ...) VALUES (...)", ...)
    tx.Commit(ctx)

    // Query as tenant B — should not see tenant A's data
    tx2, _ := pool.Begin(ctx)
    defer tx2.Rollback(ctx)
    setTenantRLS(ctx, tx2, tenantB)
    var count int
    tx2.QueryRow(ctx, "SELECT count(*) FROM oauth_clients").Scan(&count)
    assert.Equal(t, 0, count, "tenant B should not see tenant A data")
}
```

## API Endpoints

RLS is transparent — no dedicated API. All endpoints automatically respect tenant isolation.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Empty query results | `app.tenant_id` not set | Verify `setTenantRLS()` is called before queries |
| Cross-tenant data leak | BYPASSRLS granted to app role | Revoke BYPASSRLS from app role immediately |
| Permission denied on SET LOCAL | Insufficient privileges | Grant SET on `app.tenant_id` to app role |
| RLS policy not enforced | Table not FORCE'd | Run `ALTER TABLE ... FORCE ROW LEVEL SECURITY` |

## Best Practices

- **Always use SET LOCAL**: Never use global SET — it persists across connections.
- **Test isolation**: Include RLS isolation tests in CI pipeline.
- **Monitor BYPASSRLS usage**: Audit which roles have BYPASSRLS access.
- **Add RLS to all tenant tables**: Every table with tenant_id must have RLS policy.
- **Use FORCE**: Prevents even table owners from bypassing RLS.
- **Never log tenant_id bypass**: If BYPASSRLS is used, log the reason for audit.
