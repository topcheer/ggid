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
