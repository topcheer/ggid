# ADR-004: PostgreSQL Row-Level Security (RLS) for Multi-Tenant Isolation

- **Status:** Accepted
- **Date:** 2024-02-01

## Context

GGID is a multi-tenant IAM platform. Multiple tenants share the same database,
but tenant data must be strictly isolated — a user in Tenant A must never see
Tenant B's users, roles, or audit events.

Options considered:

1. **Database-per-tenant** — each tenant gets a separate database
2. **Schema-per-tenant** — each tenant gets a separate PostgreSQL schema
3. **Row-Level Security (RLS)** — shared tables with a `tenant_id` column and
PostgreSQL RLS policies enforced at the database level
4. **Application-level filtering** — every query includes `WHERE tenant_id = $1`

### Forces

- Must support thousands of tenants efficiently
- Tenant isolation is a security-critical requirement (not just best-effort)
- Operational simplicity: managing thousands of databases is expensive
- Query performance: shared tables with proper indexing are faster
- Defense in depth: even if application code has a bug, the database should
prevent cross-tenant data leakage

## Decision

We chose **PostgreSQL Row-Level Security (RLS)** with a shared database.

### Design

- Every multi-tenant table has a `tenant_id UUID NOT NULL` column
- RLS policies are defined on each table:
  ```sql
  ALTER TABLE users ENABLE ROW LEVEL SECURITY;
  CREATE POLICY tenant_isolation ON users
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
  ```
- The application sets `SET LOCAL app.tenant_id = $1` at the start of each
transaction (extracted from the request context)
- An index on `(tenant_id, ...)` ensures queries are tenant-scoped and fast
- Unique constraints include `tenant_id` (e.g., `UNIQUE(tenant_id, key)` for roles)

## Consequences

### Positive

- **Defense in depth**: even if a service forgets to add `WHERE tenant_id =`,
the database rejects the query — cross-tenant data leakage is impossible
- **Scalability**: shared tables with proper indexing handle thousands of
tenants without per-tenant schema management
- **Simplicity**: one database, one connection pool, one migration path
- **Audit trail**: all tenant data is in one place for compliance queries

### Negative

- RLS adds a small per-query overhead (policy evaluation)
- Docker development uses a superuser role which bypasses RLS — must test
with a non-superuser role in staging
- `SET LOCAL` must be called in every transaction — easy to forget in new code
- Backup/restore affects all tenants (no per-tenant selective backup)

### Neutral

- For tenants requiring physical isolation (regulated industries), the
architecture supports database-per-tenant as a deployment variant
- pgx v5 is used for PostgreSQL access (native RLS support via transaction-scoped settings)
