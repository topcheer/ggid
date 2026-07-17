# Multi-Tenant Architecture & Data Isolation: Deep Dive for GGID

> **Focus**: Verifying and hardening GGID's multi-tenant isolation — PostgreSQL RLS enforcement, tenant context propagation, cross-tenant attack prevention, tenant lifecycle management, and per-tenant resource quotas.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§7), DoD per backlog item (§8).

---

## 1. Executive Summary

Multi-tenancy is GGID's core architecture — one platform serving multiple organizations with guaranteed data isolation. A single cross-tenant data leak would be catastrophic for trust and compliance.

GGID has a **solid tenant isolation foundation**:
- `pkg/tenant/tenant.go:17` — `IsolationShared` enum + `Context` struct with `TenantID` ✅
- `pkg/tenant/rls_isolation_test.go` — 8 RLS verification tests ✅
- Context propagation via `WithContext()` / `FromContext()` / `MustFromContext()` ✅
- Tenant ID from JWT claims → context → all DB queries scoped ✅
- Per-tenant rate limiting (token bucket, `token_bucket.go:128`) ✅
- Cross-tenant spoof prevention (context can't be overwritten in derived context) ✅

**Gaps to harden:**
1. RLS is app-level (Go context), not PostgreSQL-level — DB compromise = data leak
2. No PostgreSQL `CREATE POLICY ... USING (tenant_id = current_setting('app.tenant_id'))`
3. No tenant quota enforcement (max users, max API calls, max storage)
4. No tenant onboarding/offboarding automation
5. No per-tenant audit log isolation verification at scale
6. No cross-tenant IDOR detection

**Recommendation**: Add PostgreSQL native RLS policies as defense-in-depth, tenant quota engine, automated tenant lifecycle (onboarding/offboarding), and cross-tenant access detection rules.

---

## 2. GGID Current State

### Tenant Infrastructure

| Component | File:Line | Status |
|-----------|-----------|--------|
| Tenant Context | `pkg/tenant/tenant.go:27` | ✅ `Context{TenantID, IsolationLevel}` |
| Isolation modes | `tenant.go:17` | ✅ `IsolationShared` (shared DB + RLS) |
| WithContext | `tenant.go` | ✅ Inject tenant into Go context |
| FromContext | `tenant.go` | ✅ Extract tenant from context |
| MustFromContext | `tenant.go` | ✅ Panic if missing (fail-safe) |
| RLS tests | `rls_isolation_test.go` | ✅ 8 tests: isolation, no-context reject, propagation, spoof prevention |
| Gateway tenant MW | `gateway/middleware/` | ✅ Extract tenant from JWT → inject into request context |
| Per-tenant rate limit | `token_bucket.go:128` | ✅ Redis token bucket per tenant |
| Per-tenant risk | `risk_score_handler.go:13` | ✅ Risk scored per tenant |

### Tenant Context Flow

```
Client request with JWT
  │
  ▼
Gateway JWT middleware: decode JWT → extract tenant_id claim
  │
  ▼
Gateway tenant middleware: WithContext(ctx, &Context{TenantID: claim.tenant_id})
  │
  ▼
Backend service receives context → FromContext(ctx) → tenant_id
  │
  ▼
All DB queries: WHERE tenant_id = $1 (using extracted tenant_id)
  │
  ▼
Result: tenant A can only see tenant A's data ✅
```

### What's Missing

| # | Gap | Risk Level |
|---|-----|-----------|
| 1 | App-level RLS only (no PostgreSQL RLS) | **High** — DB compromise = all tenants exposed |
| 2 | No tenant quota enforcement | Medium — resource exhaustion DoS |
| 3 | No tenant lifecycle automation | Medium — manual onboarding error-prone |
| 4 | No cross-tenant IDOR detection | Medium — subtle data leaks undetected |
| 5 | No per-tenant key isolation verification | Low — CMK designed but not implemented |
| 6 | No tenant data residency enforcement | Low — future GDPR requirement |

---

## 3. PostgreSQL Row-Level Security (RLS)

### Current vs Target

```
Current (app-level only):
  Application: SELECT * FROM users WHERE tenant_id = $1
  Problem: If app bug forgets WHERE clause → ALL tenants' data returned
  Or: Direct DB access bypasses app → no tenant filter

Target (defense-in-depth):
  PostgreSQL: ALTER TABLE users ENABLE ROW LEVEL SECURITY;
  CREATE POLICY tenant_isolation ON users
    USING (tenant_id::text = current_setting('app.tenant_id'));

  Application: SET app.tenant_id = 'tenant-uuid'; -- per connection
  SELECT * FROM users; -- RLS auto-filters to tenant
  
  Result: Even if app forgets WHERE clause → DB still filters
  Result: Direct DB access still filtered (if session has tenant_id set)
```

### RLS Implementation Plan

```sql
-- Enable RLS on all tenant-scoped tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE oauth_tokens ENABLE ROW LEVEL SECURITY;
-- ... (all tables with tenant_id column)

-- Create policy: only see rows matching current tenant
CREATE POLICY tenant_isolation ON users
  FOR ALL
  USING (tenant_id::text = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

-- Per-request: set tenant_id on connection
-- In Go: pool.Acquire(ctx) → conn.Exec("SET app.tenant_id = $1", tenantID)
```

### Connection Pool Strategy

```go
// Each acquired connection gets tenant_id set before use
type TenantAwarePool struct {
    pool *pgxpool.Pool
}

func (p *TenantAwarePool) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
    tc := tenant.MustFromContext(ctx)
    conn, err := p.pool.Acquire(ctx)
    if err != nil {
        return nil, err
    }
    // Set tenant for this connection's session
    _, err = conn.Exec(ctx, "SET LOCAL app.tenant_id = $1", tc.TenantID)
    if err != nil {
        conn.Release()
        return nil, fmt.Errorf("set tenant RLS: %w", err)
    }
    return conn, nil
}
```

---

## 4. Cross-Tenant Attack Vectors

| Attack | Vector | GGID Defense | Gap |
|--------|--------|-------------|-----|
| **IDOR** | User A accesses /api/v1/users/{tenant_B_user_id} | Tenant context in WHERE clause | ✅ Mitigated (app-level) |
| **Parameter tampering** | Attacker changes tenant_id in request | Tenant from JWT, not request body | ✅ Mitigated |
| **Token replay** | Tenant A token used for Tenant B | JWT includes tenant_id, validated at gateway | ✅ Mitigated |
| **Context spoofing** | Override tenant in Go context | Context derived tests verify | ✅ Mitigated |
| **DB direct access** | SQL injection or DB compromise | **App-level WHERE only** | ❌ **Gap — need PostgreSQL RLS** |
| **Forgotten WHERE** | Developer bug omits tenant filter | **App-level only** | ❌ **Gap — RLS auto-filters** |
| **Cache leak** | Redis key not tenant-scoped | Keys include tenant_id | ✅ Mitigated |

---

## 5. Tenant Lifecycle Management

### Onboarding Flow

```
1. Admin creates tenant: POST /api/v1/tenants
   ├── Create tenant record (name, plan, admin_email)
   ├── Generate tenant_id (UUID)
   ├── Create admin user
   ├── Generate tenant CMK (customer-managed key)
   ├── Seed default roles (admin, user, viewer)
   ├── Seed default policies
   ├── Configure rate limits (by plan tier)
   └── Send welcome email

2. Tenant activated: admin can log in
3. Tenant configured: SCIM, SSO, branding, etc.
```

### Offboarding Flow

```
1. Admin initiates offboarding: DELETE /api/v1/tenants/{id}
   ├── Verify: no active sessions
   ├── Revoke: all OAuth tokens, API keys, sessions
   ├── Disable: all users in tenant
   ├── Purge: tenant data (or crypto-shred if CMK)
   ├── Delete: CMK → encrypted data unreadable
   ├── Archive: audit log (retention compliance)
   ├── Release: rate limit quotas
   └── Confirm: tenant data inaccessible
```

### Tenant Quota Management

| Resource | Free | Pro | Enterprise |
|----------|------|-----|-----------|
| Max users | 100 | 5,000 | Unlimited |
| Max API calls/min | 100 | 1,000 | 10,000 |
| Max storage | 1GB | 50GB | 1TB |
| Max API keys | 5 | 50 | Unlimited |
| Max tenants per org | 1 | 5 | Unlimited |

---

## 6. Endpoint Precondition Check

### Existing

| Component | File:Line | Status |
|-----------|-----------|--------|
| Tenant context | `pkg/tenant/tenant.go:27` | ✅ |
| RLS tests | `rls_isolation_test.go` | ✅ |
| Gateway tenant MW | `gateway/middleware/` | ✅ |
| Per-tenant rate limit | `token_bucket.go:128` | ✅ |

### New Components

| Component | Priority |
|-----------|----------|
| PostgreSQL RLS policies | P0 |
| Tenant lifecycle API (onboarding/offboarding) | P0 |
| Tenant quota engine | P1 |
| Cross-tenant IDOR detection rule | P1 |
| Tenant data residency | P2 |

---

## 7. Implementation Backlog with DoD

### P0 — PostgreSQL RLS + Lifecycle (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | PostgreSQL RLS on all tenant tables | ✅ CREATE POLICY on 20+ tables ✅ Connection pool sets tenant_id ✅ Tests verify isolation ✅ ≥3 tests | 4d |
| 2 | Tenant lifecycle API | ✅ POST /tenants (onboard) ✅ DELETE /tenants/{id} (offboard) ✅ Cascade purge ✅ ≥3 tests | 4d |
| 3 | Tenant quota enforcement | ✅ Max users/API/storage per plan ✅ DB-backed ✅ ≥3 tests | 3d |

### P1 — Detection + Residency (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 4 | Cross-tenant IDOR detection rule | ✅ ITDR rule for cross-tenant access attempt ✅ ≥3 tests | 2d |
| 5 | Tenant isolation audit | ✅ Verify no cross-tenant queries ✅ Automated test suite ✅ ≥3 tests | 2d |
| 6 | Tenant data residency | ✅ Per-tenant region config ✅ Data stays in region ✅ ≥3 tests | 3d |

### P2 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 7 | Schema-per-tenant option | Isolation mode upgrade for high-security tenants |
| 8 | Tenant migration | Move tenant between regions |
| 9 | Tenant usage analytics | Per-tenant resource consumption dashboard |
| 10 | Tenant white-labeling | Custom branding, domain, emails |

---

## 8. Competitive Differentiation

| Feature | GGID (target) | Auth0 | AWS Cognito | Azure AD B2C | Keycloak |
|---------|---------------|-------|-------------|-------------|----------|
| **Tenant isolation** | **App + PostgreSQL RLS** | App-level | Pool isolation | App-level | Realm |
| **RLS defense-in-depth** | **PostgreSQL native** | No | No | No | No |
| **Quota management** | **Per-plan** | Yes (limits) | Yes (quotas) | Yes | No |
| **Lifecycle automation** | **API-driven** | Yes | Yes | Yes | Manual |
| **Context propagation** | **Go context + JWT** | Tenant tag | Pool ID | Tenant ID | Realm |
| **Cross-tenant detection** | **ITDR rule** | No | No | No | No |
| **Open source** | **Yes** | No | No | No | Yes |

**Key differentiator**: GGID would be the only open-source IAM with **PostgreSQL native RLS** as defense-in-depth — even if the application layer fails, the database enforces tenant isolation.

---

## References

- [PostgreSQL Row Level Security](https://www.postgresql.org/docs/current/ddl-rowsecurity.html) — RLS documentation
- [Auth0 Multi-Tenancy](https://auth0.com/docs/tenant/multi-tenancy) — Tenant patterns
- [AWS Cognito User Pools](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-identity-pools.html) — Pool isolation
- [Azure AD B2C Tenants](https://learn.microsoft.com/en-us/azure/active-directory-b2c/) — Tenant model
- [GGID Tenant Package](../pkg/tenant/tenant.go) — Context at line 27
- [GGID RLS Tests](../pkg/tenant/rls_isolation_test.go) — 8 verification tests
- [GGID Token Bucket](../services/gateway/internal/middleware/token_bucket.go) — Per-tenant at line 128
- [GGID CMK/KMS Research](./customer-managed-keys-kms.md) — Per-tenant encryption keys
