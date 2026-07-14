# ADR-001: PostgreSQL with Row-Level Security for Multi-Tenancy

**Status:** Accepted
**Date:** 2024-Q1
**Deciders:** Architecture Team

---

## Context

GGID is a multi-tenant IAM platform. Multiple organizations (tenants) share the same infrastructure while requiring strict data isolation. The database choice is fundamental — it determines performance characteristics, operational complexity, isolation guarantees, and scaling limits.

### Requirements

1. **Strict tenant isolation**: One tenant must never see another tenant's data, even with application bugs
2. **Horizontal scale**: Must support thousands of tenants on shared infrastructure
3. **Operational simplicity**: Single backup/restore, single migration path
4. **Query performance**: Sub-100ms for typical user lookups
5. **ACID compliance**: Strong consistency for authentication and authorization
6. **Open source**: Apache 2.0 compatible, no proprietary lock-in

### Alternatives Considered

#### Option A: Separate Database per Tenant

```
Tenant A → Database A
Tenant B → Database B
Tenant C → Database C
```

**Pros:**
- Perfect physical isolation
- Per-tenant backup/restore possible
- No noisy neighbor problem

**Cons:**
- Does not scale: 10,000 tenants = 10,000 database connections
- Migration complexity: apply schema changes to all databases
- Operational nightmare: monitoring, backup, tuning for thousands of DBs
- Connection pool exhaustion at scale

#### Option B: Separate Schema per Tenant

```
Database
  ├── schema_tenant_a (users, roles, ...)
  ├── schema_tenant_b (users, roles, ...)
  └── schema_tenant_c (users, roles, ...)
```

**Pros:**
- Better than per-DB: one PostgreSQL instance
- Schema-level isolation
- Per-tenant backup via schema export

**Cons:**
- Still thousands of schemas for large deployments
- Connection setup overhead (search_path per connection)
- Complex migrations: apply to each schema
- Cross-tenant queries (admin dashboards) are difficult

#### Option C: Shared Database with Application-Level Filtering

```
Database
  └── public.users (tenant_id column on every row)
      WHERE tenant_id = ? -- application must remember this
```

**Pros:**
- Simplest implementation
- Single schema, single migration path
- Easy cross-tenant admin queries

**Cons:**
- **Dangerous**: one missed WHERE clause leaks data
- No defense in depth — relies entirely on application correctness
- Security audit nightmare

#### Option D: Shared Database with PostgreSQL RLS (Selected)

```
Database
  └── public.users (tenant_id column + RLS policy)
      -- RLS automatically filters: WHERE tenant_id = current_setting('app.tenant_id')
```

**Pros:**
- Defense in depth: DB enforces isolation, application is second layer
- Single schema, single migration path
- Easy cross-tenant admin queries (with superuser)
- Mature feature in PostgreSQL 16

**Cons:**
- Performance overhead from RLS policy evaluation
- Must set `app.tenant_id` on every connection
- `SET LOCAL` doesn't support parameterized queries

---

## Decision

Choose **PostgreSQL 16 with Row-Level Security (RLS)** as the multi-tenant data isolation mechanism.

### Implementation Details

1. Every tenant-scoped table has a `tenant_id UUID` column
2. RLS is ENABLED and FORCED on every tenant table:
   ```sql
   ALTER TABLE users ENABLE ROW LEVEL SECURITY;
   ALTER TABLE users FORCE ROW LEVEL SECURITY;
   ```
3. A policy filters all rows by the connection's tenant context:
   ```sql
   CREATE POLICY tenant_isolation ON users
     USING (tenant_id = current_setting('app.tenant_id')::uuid);
   ```
4. Every database connection sets tenant context before queries:
   ```sql
   SET LOCAL app.tenant_id = '00000000-0000-0000-0000-000000000001';
   ```
5. `SET LOCAL` uses `fmt.Sprintf` (not `$1` parameters, which PostgreSQL doesn't support for SET)

### Why PostgreSQL over alternatives

| Factor | PostgreSQL | MySQL | MongoDB | DynamoDB |
|--------|-----------|-------|--------|----------|
| RLS support | Mature (since 9.5) | No | No | No |
| ACID compliance | Full | Partial (InnoDB) | Multi-doc transactions | Single-item only |
| Open source | Yes | Yes | SSPL | Proprietary |
| JSON support | JSONB (indexed) | JSON | Native | Native |
| gRPC ecosystem | pgx (Go) | Good | Good | Limited |
| Managed service | RDS, Cloud SQL | RDS | Atlas | Native |

---

## Consequences

### Positive

- **Guaranteed isolation**: RLS provides defense in depth. Even with an application bug (missing WHERE clause), the database prevents cross-tenant data access.
- **Operational simplicity**: One database, one backup, one migration path for all tenants.
- **Audit-ready**: Security auditors love RLS — it's a verifiable, database-level control.
- **Flexible queries**: Superuser connections can query across tenants for admin dashboards.
- **Mature ecosystem**: PostgreSQL 16 is battle-tested with RLS since 2016.

### Negative

- **Performance overhead**: RLS adds a policy check to every query. Benchmark: ~5% overhead on indexed queries.
- **Connection discipline**: Must always set `app.tenant_id` before queries. Missing this causes silent empty results (not an error).
- **Parameter limitation**: `SET LOCAL` doesn't support `$1` bind parameters — must use `fmt.Sprintf` with careful escaping (UUID format makes injection unlikely).
- **No per-tenant backup**: Backup granularity is all-or-nothing. Per-tenant export requires application-level filtering.
- **Noisy neighbor risk**: One tenant's heavy queries can affect others. Mitigated by connection pooling and rate limiting.

### Mitigations

| Risk | Mitigation |
|------|----------|
| Performance overhead | Index `tenant_id` column on every table. RLS check becomes an index scan. |
| Connection discipline | Centralized `SetTenantContext(ctx, tenantID)` function used by all repositories. |
| Noisy neighbor | Per-tenant rate limiting at the gateway. Query timeout enforcement. |
| Per-tenant backup | Application-level export API (`GET /tenants/{id}/export`). |

---

## References

- [PostgreSQL RLS Documentation](https://www.postgresql.org/docs/16/ddl-rowsecurity.html)
- [GGID Multi-Tenancy Guide](../multi-tenancy.md)
- [GGID Database Schema](../database-schema.md)
- [GGID Security Architecture](../security-architecture.md)
- Related: [multi-tenant-rls.md](./multi-tenant-rls.md)

---

*Last updated: 2025-07-11*
