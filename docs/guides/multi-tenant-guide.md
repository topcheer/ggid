# Multi-Tenant Architecture Guide

> How GGID isolates tenants via PostgreSQL RLS, JWT claims, and gateway middleware.

---

## Tenant Isolation Layers

```
Request → Gateway (tenant resolution from JWT) → Backend Service → PostgreSQL (RLS policy)
              ↓                                        ↓                    ↓
         X-Tenant-ID header                    tenant_id in query           WHERE tenant_id = current_setting('app.current_tenant')
```

Three layers ensure tenant data never leaks:

1. **Gateway**: Resolves tenant from JWT claim (trusted) or API key
2. **Service**: Passes tenant_id to all database queries
3. **Database**: Row-Level Security (RLS) enforces isolation at the SQL level

---

## Tenant Resolution Priority

| Priority | Source | Trust Level |
|----------|--------|-------------|
| 1 | JWT `tenant_id` claim | High (signed) |
| 2 | API key → tenant mapping | Medium (DB lookup) |
| 3 | `X-Tenant-ID` header | Low (spoofable) |

JWT claim **always wins** to prevent header spoofing.

---

## PostgreSQL RLS Configuration

### Enable RLS

```sql
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON users
  USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

### Set Tenant Context Per Query

```go
// Before any query, set the tenant context
db.Exec(ctx, "SET LOCAL app.current_tenant = $1", tenantID)
// Now all queries are automatically scoped
rows, _ := db.Query(ctx, "SELECT * FROM users") // Only sees tenant's users
```

---

## Creating a New Tenant

```bash
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -d '{
    "id": "00000000-0000-0000-0000-000000000002",
    "name": "Acme Corp",
    "domain": "acme.com"
  }'
```

---

## JWT Tenant Claim

Every JWT includes `tenant_id`:

```json
{
  "sub": "usr_abc123",
  "tenant_id": "00000000-0000-0000-0000-000000000002",
  "scope": "read:users",
  "roles": ["admin"]
}
```

---

## Best Practices

1. **Always use `SET LOCAL`** before queries — never rely on application-level filtering alone
2. **Test isolation**: Query as tenant A, verify zero results from tenant B
3. **Per-tenant config**: Store branding, MFA policy, and rate limits per tenant
4. **Audit per tenant**: All audit events carry `tenant_id` for per-tenant compliance reports
5. **Tenant-aware caching**: Include `tenant_id` in cache keys to prevent cross-tenant data leaks

---

*See: [Security Overview](../architecture/security-overview.md) | [Multi-Tenant Setup](multi-tenant-setup.md) | [RBAC Guide](role-based-access.md)*

*Last updated: 2025-07-11*
