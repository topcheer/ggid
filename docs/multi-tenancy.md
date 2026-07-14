# Multi-Tenant Architecture

GGID's multi-tenant model: tenant lifecycle, RLS policies, isolation
verification, per-tenant configuration, and cross-tenant data leakage
prevention.

> **See also**: [Multi-Tenancy Guide](multi-tenancy-guide.md) for tenant
> admin operations, [Multi-Tenant Architecture](multi-tenant-architecture.md)
> for resource quotas and tier mapping.

---

## Table of Contents

- [Tenant Isolation Model](#tenant-isolation-model)
- [Tenant Lifecycle](#tenant-lifecycle)
- [RLS Policy Enforcement](#rls-policy-enforcement)
- [Isolation Verification](#isolation-verification)
- [Per-Tenant Configuration](#per-tenant-configuration)
- [Cross-Tenant Data Leakage Prevention](#cross-tenant-data-leakage-prevention)

---

## Tenant Isolation Model

GGID uses PostgreSQL Row-Level Security (RLS) as the primary isolation
mechanism. Every tenant-scoped query sets `app.tenant_id` via `SET LOCAL`
before execution.

```
┌──────────────────────────────────────────────────────┐
│                    PostgreSQL                         │
│                                                       │
│  ┌─────────────────────────────────────────────┐     │
│  │     SET LOCAL app.tenant_id = 'tenant-A'    │     │
│  │                                              │     │
│  │  Policy: USING (tenant_id = current_setting) │     │
│  │                                              │     │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐     │     │
│  │  │Tenant A  │ │Tenant B  │ │Tenant C  │     │     │
│  │  │ users    │ │ users    │ │ users    │     │     │
│  │  │ roles    │ │ roles    │ │ roles    │     │     │
│  │  └──────────┘ └──────────┘ └──────────┘     │     │
│  └─────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────┘
```

### Defense in Depth

| Layer | Mechanism | Purpose |
|-------|-----------|---------|
| Database | RLS policies | Prevent SQL-level cross-tenant reads |
| Application | Tenant context middleware | Inject tenant_id into every request |
| Gateway | X-Tenant-ID validation | Reject requests without tenant header |
| JWT | `tenant_id` claim | Bind token to issuing tenant |
| Policy | Per-tenant role scoping | Prevent privilege escalation |

---

## Tenant Lifecycle

### Create Tenant

```bash
curl -X POST https://iam.example.com/api/v1/admin/tenants \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{
    "name": "Acme Corp",
    "plan": "enterprise",
    "max_users": 10000,
    "settings": {
      "mfa_required": true,
      "password_policy": "strict"
    }
  }'
```

```json
{
  "id": "00000000-0000-0000-0000-000000000002",
  "name": "Acme Corp",
  "status": "active",
  "created_at": "2024-01-15T10:00:00Z"
}
```

### Configure Tenant

```bash
# Update password policy
curl -X PATCH https://iam.example.com/api/v1/admin/tenants/{id}/settings/password-policy \
  -d '{ "min_length": 14, "require_special": true }'

# Update MFA policy
curl -X PATCH https://iam.example.com/api/v1/admin/tenants/{id}/settings/mfa-policy \
  -d '{ "required": true, "allowed_methods": ["totp", "webauthn"] }'

# Set branding
curl -X PATCH https://iam.example.com/api/v1/admin/tenants/{id}/settings/branding \
  -d '{ "logo_url": "https://acme.com/logo.png", "primary_color": "#1a73e8" }'
```

### Suspend / Delete Tenant

```bash
# Suspend (blocks all auth, preserves data)
curl -X POST https://iam.example.com/api/v1/admin/tenants/{id}/suspend \
  -d '{ "reason": "non-payment" }'

# Delete (soft-delete, 30-day grace period)
curl -X DELETE https://iam.example.com/api/v1/admin/tenants/{id} \
  -d '{ "confirm": "DELETE", "grace_days": 30 }'
```

---

## RLS Policy Enforcement

### Policy Template

Every tenant-scoped table follows this pattern:

```sql
-- Enable RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Policy: tenant isolation
CREATE POLICY tenant_isolation ON users
    USING (tenant_id::text = current_setting('app.tenant_id', true));

-- Force RLS (even for table owner)
ALTER TABLE users FORCE ROW LEVEL SECURITY;
```

### Application-Level Enforcement

```go
func (s *Service) GetUsers(ctx context.Context) ([]User, error) {
    tenantID, ok := tenant.FromContext(ctx)
    if !ok {
        return nil, ErrMissingTenant
    }

    // SET LOCAL app.tenant_id before every query
    _, err := s.db.ExecContext(ctx,
        fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID))
    if err != nil {
        return nil, err
    }

    return s.db.QueryUsers(ctx)  // RLS filters automatically
}
```

### Tables with RLS

| Table | Service | Tenant Column |
|-------|---------|---------------|
| users | Identity | tenant_id |
| groups | Identity | tenant_id |
| credentials | Auth | tenant_id |
| sessions | Auth | tenant_id |
| mfa_factors | Auth | tenant_id |
| webauthn_credentials | Auth | tenant_id |
| oauth_clients | OAuth | tenant_id |
| oauth_refresh_tokens | OAuth | tenant_id |
| roles | Policy | tenant_id |
| role_assignments | Policy | tenant_id |
| policies | Policy | tenant_id |
| orgs | Org | tenant_id |
| org_members | Org | tenant_id |
| audit_events | Audit | tenant_id |

---

## Isolation Verification

### Automated Test

```go
func TestTenantIsolation(t *testing.T) {
    // Create user in Tenant A
    ctxA := tenant.WithContext(context.Background(), tenantA)
    userA, _ := svc.Create(ctxA, "alice@tenant-a.com")

    // Try to read from Tenant B
    ctxB := tenant.WithContext(context.Background(), tenantB)
    _, err := svc.Get(ctxB, userA.ID)

    // Must return not found
    require.ErrorIs(t, err, ErrNotFound)
}
```

### SQL Verification

```sql
-- Verify RLS is active
SELECT relname, relrowsecurity, relforcerowsecurity
FROM pg_class
WHERE relname IN ('users', 'roles', 'audit_events');

-- Expected: relrowsecurity = true, relforcerowsecurity = true
```

### Penetration Test Checklist

- [ ] Token from Tenant A cannot read Tenant B's users
- [ ] Token from Tenant A cannot modify Tenant B's roles
- [ ] SCIM endpoint scoped to JWT tenant only
- [ ] Audit query cannot cross tenant boundary
- [ ] X-Tenant-ID header cannot override JWT tenant_id claim
- [ ] Direct database query without SET LOCAL returns no rows

---

## Per-Tenant Configuration

### Feature Flags

```yaml
tenant_config:
  features:
    webauthn: true
    saml_sso: false
    scim_provisioning: true
    custom_claims: true
```

### Rate Limits

```yaml
tenant_config:
  rate_limits:
    login_attempts_per_minute: 10
    api_requests_per_minute: 1000
    token_requests_per_minute: 100
```

### Token Policy

```yaml
tenant_config:
  token:
    access_token_ttl: "15m"
    refresh_token_ttl: "24h"
    max_session_duration: "8h"
    concurrent_sessions: 3
```

### Branding

```yaml
tenant_config:
  branding:
    logo_url: "https://acme.com/logo.png"
    primary_color: "#1a73e8"
    login_page_title: "Acme Corp Login"
    email_from: "noreply@acme.com"
```

---

## Cross-Tenant Data Leakage Prevention

### Threat Model

| Attack Vector | Prevention |
|--------------|------------|
| SQL injection bypassing WHERE | RLS enforced at DB level (not just app) |
| JWT tampering (change tenant_id) | JWT signed by issuer; tenant_id immutable |
| X-Tenant-ID spoofing | Gateway compares header to JWT tenant_id claim |
| Direct DB access | `FORCE ROW LEVEL SECURITY` applies even to owner |
| Cache poisoning | Redis keys prefixed with tenant_id |
| NATS message leak | Subjects include tenant_id; consumer filters |

### Gateway Enforcement

```go
func TenantResolver(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        headerTenant := r.Header.Get("X-Tenant-ID")
        jwtTenant := r.Context().Value(tenantKey)

        // If JWT present, its tenant_id wins (prevents spoofing)
        if jwtTenant != nil && jwtTenant != headerTenant {
            respondError(w, 403, "gateway.cross_tenant_denied",
                "JWT tenant does not match X-Tenant-ID header")
            return
        }

        if headerTenant == "" {
            respondError(w, 412, "gateway.missing_tenant",
                "X-Tenant-ID header required")
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### Redis Key Isolation

```
# Every Redis key is prefixed with tenant_id
session:{tenant_id}:{session_token}
rate_limit:{tenant_id}:{ip}:{endpoint}
jwks_cache:{tenant_id}
```

### NATS Subject Namespacing

```
audit.events.{tenant_id}      # Per-tenant audit subject
user.events.{tenant_id}       # Per-tenant user events
```

Consumers filter by tenant_id in subject pattern, preventing cross-tenant
message delivery.

---

## Tenant CRUD API

### Create Tenant

```bash
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer <super-admin-JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corporation",
    "plan": "enterprise",
    "max_users": 10000,
    "features": ["sso", "scim", "audit_export"]
  }'
```

Response: `201 Created`
```json
{
  "id": "55000000-0000-0000-0000-000000000002",
  "name": "Acme Corporation",
  "plan": "enterprise",
  "max_users": 10000,
  "active": true,
  "created_at": "2025-07-11T12:00:00Z"
}
```

### Get Tenant

```bash
curl http://localhost:8080/api/v1/tenants/{tenant_id} \
  -H "Authorization: Bearer <JWT>"
```

### List Tenants

```bash
curl http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer <super-admin-JWT>"
```

### Update Tenant

```bash
curl -X PUT http://localhost:8080/api/v1/tenants/{tenant_id} \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corp (Updated)",
    "max_users": 50000,
    "features": ["sso", "scim", "audit_export", "webhooks"]
  }'
```

### Suspend Tenant

```bash
curl -X POST http://localhost:8080/api/v1/tenants/{tenant_id}/suspend \
  -H "Authorization: Bearer <super-admin-JWT>" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Non-payment"}'
```

Suspended tenants: all API calls return `403 Forbidden` with `tenant_suspended` error.

### Delete Tenant

```bash
curl -X DELETE http://localhost:8080/api/v1/tenants/{tenant_id} \
  -H "Authorization: Bearer <super-admin-JWT>"
```

**Warning**: This cascades to all tenant data (users, roles, orgs, audit events). This operation is irreversible.

---

## Default Tenant

The default tenant is pre-seeded during database migration:

| Property | Value |
|----------|-------|
| **UUID** | `00000000-0000-0000-0000-000000000001` |
| **Name** | `Default` |
| **Plan** | `unlimited` |
| **Active** | `true` |

### Purpose

- Single-tenant deployments use this as the only tenant
- Development and testing default to this tenant
- Gateway requires `X-Tenant-ID` header — use this UUID for single-tenant mode

### Gateway Configuration

The gateway always requires a tenant identifier. In single-tenant mode, all requests use the default tenant UUID:

```bash
# All API calls must include X-Tenant-ID header
curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <JWT>" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
```

### Gateway Tenant Forwarding

The gateway forwards tenant context to backend services via two mechanisms:

1. **Query parameter**: `tenant_id` appended to URL for GET requests
2. **JSON body**: `tenant_id` injected into POST/PUT/PATCH body

This is because Policy/Org/Audit services expect `tenant_id` as a query parameter or body field (not just a header).

---

## Tenant Context Flow (Complete)

### Three Sources of Tenant Identity

```
┌─────────────────────────────────────────────────────┐
│ Source 1: JWT Claim (AUTHORITATIVE)                  │
│                                                      │
│ "tenant_id" claim in JWT payload                     │
│ → Takes PRIORITY over all other sources              │
│ → Prevents tenant spoofing via header injection       │
│ → Set at token issuance time by auth service          │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│ Source 2: X-Tenant-ID Header (SERVICE-TO-SERVICE)    │
│                                                      │
│ HTTP header set by gateway                            │
│ → Used when no JWT is present (health checks)         │
│ → IGNORED if JWT is present (prevents spoofing)       │
│ → For logging and audit only when JWT exists          │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│ Source 3: Database Session (ENFORCEMENT)              │
│                                                      │
│ SET LOCAL app.tenant_id = '<uuid>'                   │
│ → Set on every database connection                    │
│ → Enforced by RLS policies                            │
│ → Cannot be bypassed by application bugs              │
└─────────────────────────────────────────────────────┘
```

### Tenant Spoofing Prevention

**Threat**: Malicious user tries to access another tenant's data by sending a different `X-Tenant-ID` header.

**Defense**: The gateway extracts `tenant_id` from the JWT claim and ignores the header:

```go
// In gateway middleware
tenantID := jwtClaims.TenantID  // From JWT — authoritative
// NOT: tenantID := r.Header.Get("X-Tenant-ID")  // SPOOFABLE!
```

---

## Tenant Migration Patterns

### Pattern 1: Tenant Onboarding

When a new tenant signs up:

```
1. Create tenant record in tenants table
2. Create default roles for tenant (super_admin, end_user)
3. Create first admin user with super_admin role
4. Send onboarding email
5. Emit tenant.created webhook event
```

### Pattern 2: Tenant Data Export

```bash
# Export all tenant data for backup or migration
curl http://localhost:8080/api/v1/tenants/{tenant_id}/export \
  -H "Authorization: Bearer <JWT>" \
  -o tenant_export.json
```

Export includes:
- All users (passwords excluded)
- All roles and permissions
- Organization hierarchy
- Audit events (configurable date range)
- Webhook configurations

### Pattern 3: Tenant Data Import

```bash
curl -X POST http://localhost:8080/api/v1/tenants/{tenant_id}/import \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d @tenant_export.json
```

### Pattern 4: Tenant Splitting

When a large tenant needs to be split into multiple tenants:

1. Export source tenant data
2. Create new tenant(s)
3. Import users into new tenant(s)
4. Reassign roles in new tenant
5. Verify data isolation
6. Suspend original tenant

### Pattern 5: Schema Migration Per Tenant

For schema changes that need per-tenant validation:

```sql
-- Step 1: Migrate default tenant
SET LOCAL app.tenant_id = '00000000-0000-0000-0000-000000000001';
ALTER TABLE users ADD COLUMN IF NOT EXISTS department VARCHAR(255);

-- Step 2: Verify
SELECT COUNT(*) FROM users WHERE department IS NOT NULL;

-- Step 3: Migrate remaining tenants (application handles this)
```

**Note**: DDL changes apply to the table (all tenants). DML changes are tenant-scoped via RLS.

---

## Per-Tenant Rate Limiting

Rate limits are applied per-tenant, not just per-IP:

```
Rate Limit Key: rate_limit:{tenant_id}:{ip}:{endpoint}

Default Limits:
  - 1000 req/min per tenant (enterprise)
  - 100 req/min per tenant (starter)
  - 10 req/min per IP (unauthenticated)
```

This prevents one tenant's heavy usage from affecting others.

---

## Tenant Monitoring

### Key Metrics Per Tenant

| Metric | Description | Alert Threshold |
|--------|-------------|----------------|
| `tenant.user_count` | Active users | >90% of max_users |
| `tenant.api_calls` | API calls per minute | >80% of rate limit |
| `tenant.failed_logins` | Failed login attempts | >50 per hour |
| `tenant.audit_events` | Audit events per day | Anomaly detection |
| `tenant.storage_mb` | Database storage used | >80% of quota |

### Querying Tenant Metrics

```bash
curl "http://localhost:8080/api/v1/tenants/{tenant_id}/metrics" \
  -H "Authorization: Bearer <JWT>"
```

---

*Last updated: 2025-07-11*
