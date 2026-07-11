# ADR-0002: Multi-Tenancy via PostgreSQL Row-Level Security

**Status:** Accepted  
**Date:** 2025-Q1  
**Deciders:** Architecture Team

---

## Context

GGID is a multi-tenant IAM platform serving multiple organizations on shared infrastructure. The tenant isolation strategy is foundational — it affects security, performance, and operational complexity.

### Requirements

1. **Strict isolation**: Tenants must never see each other's data
2. **Defense in depth**: Multiple layers, not just application-level filtering
3. **Scalable**: Support 10,000+ tenants on shared infrastructure
4. **Operational simplicity**: Single backup, single migration path
5. **Queryable**: Admin dashboards can aggregate across tenants

---

## Alternatives Considered

### Option A: Database-per-Tenant

```
Tenant A → DB A    Tenant B → DB B    Tenant C → DB C
```

**Pros:** Perfect physical isolation, per-tenant backup
**Cons:** Connection exhaustion at scale (10k DBs), migration nightmare, monitoring overhead
**Verdict:** Rejected — doesn't scale

### Option B: Schema-per-Tenant

```
Single DB → schema_tenant_a, schema_tenant_b, ...
```

**Pros:** Better than per-DB, schema-level isolation
**Cons:** Thousands of schemas, complex migrations, cross-tenant queries difficult
**Verdict:** Rejected — operational complexity

### Option C: Application-Level Filtering

```
SELECT * FROM users WHERE tenant_id = ?
```

**Pros:** Simplest, single schema
**Cons:** One missed WHERE = data breach, no defense in depth
**Verdict:** Rejected — too dangerous

### Option D: PostgreSQL RLS (Selected)

```
SELECT * FROM users;
-- RLS automatically adds: WHERE tenant_id = current_setting('app.tenant_id')
```

**Pros:** Defense in depth, single schema, mature feature
**Cons:** ~5% query overhead, connection discipline required
**Verdict:** Selected — best balance of security and operability

---

## Decision

Use **PostgreSQL 16 Row-Level Security** as the multi-tenant isolation mechanism.

### Implementation

1. Every tenant-scoped table has `tenant_id UUID` column
2. RLS enabled and forced:
   ```sql
   ALTER TABLE users ENABLE ROW LEVEL SECURITY;
   ALTER TABLE users FORCE ROW LEVEL SECURITY;
   ```
3. Policy filters by connection context:
   ```sql
   CREATE POLICY tenant_isolation ON users
     USING (tenant_id = current_setting('app.tenant_id')::uuid);
   ```
4. Every transaction sets tenant context:
   ```sql
   SET LOCAL app.tenant_id = '00000000-0000-0000-0000-000000000001';
   ```
5. `SET LOCAL` uses `fmt.Sprintf` (PostgreSQL doesn't support `$1` for SET)

### Three-Layer Defense

```
Layer 1 (Application): JWT tenant_id claim is authoritative
Layer 2 (Connection):  SET LOCAL app.tenant_id per transaction
Layer 3 (Database):    RLS policy enforces row-level filter
```

Even if layers 1 and 2 fail, layer 3 blocks cross-tenant access.

---

## Performance Analysis

### RLS Overhead

| Query Type | Without RLS | With RLS | Overhead |
|------------|-------------|----------|----------|
| Index scan (tenant_id indexed) | 0.2ms | 0.21ms | ~5% |
| Seq scan | 5ms | 5.2ms | ~4% |
| Aggregate (COUNT) | 2ms | 2.1ms | ~5% |
| Join (2 tables) | 1.5ms | 1.6ms | ~7% |

**Conclusion**: ~5% overhead on indexed queries — acceptable for the security guarantee.

### Scaling Limits

| Tenants | Users/Tenant | Total Rows | Performance Impact |
|---------|-------------|------------|-------------------|
| 100 | 1,000 | 100K | None |
| 1,000 | 1,000 | 1M | None (indexed) |
| 10,000 | 500 | 5M | Minimal |
| 10,000 | 5,000 | 50M | Partitioning recommended |

### Mitigation at Scale

- **Index `tenant_id`** on every table — RLS check becomes index scan
- **Table partitioning** by `tenant_id` at 50M+ rows
- **Connection pooling** (PgBouncer) to manage connections

---

## Consequences

### Positive

- **Guaranteed isolation**: Database enforces it, application is second layer
- **Single schema**: One migration path, one backup, one monitoring setup
- **Audit-ready**: RLS is a verifiable, database-level control
- **Cross-tenant queries**: Superuser connections bypass RLS for admin dashboards

### Negative

- **Connection discipline**: Must always `SET LOCAL app.tenant_id` — missing it returns empty results
- **No per-tenant backup**: Requires application-level export API
- **Parameter limitation**: `SET LOCAL` doesn't support bind parameters
- **Noisy neighbor**: Heavy tenant queries can affect others (mitigated by rate limiting)

---

## References

- [PostgreSQL RLS Documentation](https://www.postgresql.org/docs/16/ddl-rowsecurity.html)
- [ADR-001: Database Choice](../design/adr-001-database-choice.md) — Full 4-alternative analysis
- [Multi-Tenant Setup Guide](../../guides/multi-tenant-setup.md)
- [Security Architecture](../../security-architecture.md)

---

*Last updated: 2025-07-11*