# Multi-Tenant Architecture Guide

This guide covers GGID's multi-tenant architecture — RLS policies, tenant isolation, per-tenant configuration, onboarding, and cross-tenant security boundaries.

## Tenant Isolation Model

GGID uses **PostgreSQL Row-Level Security (RLS)** as the primary tenant isolation mechanism, enforced at the database level.

### How RLS Works

```sql
-- Enable RLS on all tenant tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

-- Policy: users can only see their tenant's rows
CREATE POLICY tenant_isolation ON users
  FOR ALL
  USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

**Enforcement**: Every query automatically filters by `tenant_id`. Even if the application has a bug, the database prevents cross-tenant access.

### Session Tenant Context

```sql
-- Set at connection checkout
SET LOCAL app.tenant_id = '00000000-0000-0000-0000-000000000001';
```

GGID's gateway extracts `tenant_id` from JWT claims and injects it into the database session.

### RLS Hierarchy

```
Gateway (extracts tenant_id from JWT)
  ↓ X-Tenant-ID header
Service (sets DB session variable)
  ↓ SET LOCAL app.tenant_id
PostgreSQL (enforces RLS on every query)
  ↓ Filtered result set
```

## Per-Tenant Configuration

### Branding

Each tenant can have unique branding (logo, colors, CSS). See [Branding Guide](branding-guide.md).

### Auth Providers

Per-tenant SSO configuration:

| Config | Scope | Tenant A | Tenant B |
|--------|-------|----------|----------|
| SAML IdP | Per-tenant | Okta | Azure AD |
| LDAP | Per-tenant | OpenLDAP | Active Directory |
| Social | Per-tenant | Google | GitHub |
| Password policy | Per-tenant | 12 chars | 8 chars |
| MFA requirement | Per-tenant | Required | Optional |

### Rate Limits

Rate limit tiers per tenant (Basic, Premium, Enterprise). See [Rate Limiting Guide](rate-limiting-guide.md).

## Tenant Onboarding Flow

```
1. Admin creates new tenant → UUID generated
2. Create schema/database role for tenant
3. Apply RLS policies
4. Configure default roles (admin, user)
5. Set up branding (default template)
6. Configure auth providers
7. Create first admin user
8. Tenant ready for use
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/tenants \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{
    "name": "Acme Corp",
    "plan": "enterprise",
    "admin_email": "admin@acme.com"
  }'
```

## Cross-Tenant Security Boundaries

| Boundary | Enforcement |
|----------|------------|
| Data access | PostgreSQL RLS (database-enforced) |
| JWT tokens | `tenant_id` claim embedded, verified on every request |
| Header spoofing | JWT claim takes priority over X-Tenant-ID header |
| API isolation | All queries include tenant_id filter (defense in depth) |
| Session isolation | Redis keys prefixed with tenant_id |
| NATS subjects | Tenant-prefixed subjects (tenant.{id}.events) |
| Audit isolation | Audit events scoped to tenant_id via RLS |

### Preventing Tenant Spoofing

```
Client sends: X-Tenant-ID: tenant-A
Client JWT:   tenant_id: tenant-B

GGID behavior: Uses tenant-B (JWT wins over header)
```

This prevents users from accessing other tenants' data by manipulating headers.

## Redis Namespacing

```
session:{tenant_id}:{session_id}
ratelimit:{tenant_id}:{ip}
jti:{tenant_id}:{jti}
oauth_state:{tenant_id}:{state}
consent:{tenant_id}:{user_id}:{client_id}
```

## NATS Subject Namespacing

```
audit.{tenant_id}.events
webhook.{tenant_id}.deliveries
```

This allows per-tenant consumers and isolation of event streams.

## Database Schema Strategy

### Option A: Shared Database, Shared Schema (Default)

All tenants share tables, separated by `tenant_id` column with RLS.

**Pros**: Simple, cost-effective, easy to manage
**Cons**: Noisy neighbor risk (one tenant's heavy queries affect others)

### Option B: Shared Database, Separate Schemas

Each tenant gets a separate PostgreSQL schema.

**Pros**: Better isolation, per-tenant backup/restore
**Cons**: More complex migrations

### Option C: Separate Databases

Each tenant gets a separate database.

**Pros**: Maximum isolation, compliance-friendly
**Cons**: High cost, operational overhead

GGID default: **Option A** (shared schema + RLS).

## Tenant Data Export/Import

```bash
# Export tenant data (admin only)
curl https://api.ggid.example.com/api/v1/tenants/$TENANT_ID/export \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -o tenant-export.json
```

## See Also

- RLS Performance
- [Branding Guide](branding-guide.md)
- [Session Management](session-management-guide.md)
- [Data Sovereignty](../research/data-sovereignty.md)
