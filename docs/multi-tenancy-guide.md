# GGID Multi-Tenancy Guide

Deep dive into GGID's multi-tenant architecture: Row-Level Security (RLS),
tenant_id propagation, per-tenant configuration, onboarding/offboarding, and
cross-tenant prevention patterns.

---

## Table of Contents

- [Overview](#overview)
- [Tenant ID Propagation Chain](#tenant-id-propagation-chain)
- [PostgreSQL Row-Level Security](#postgresql-row-level-security)
- [Tenant Context Middleware](#tenant-context-middleware)
- [Per-Tenant Configuration](#per-tenant-configuration)
- [Tenant Onboarding](#tenant-onboarding)
- [Tenant Offboarding](#tenant-offboarding)
- [Cross-Tenant Prevention Patterns](#cross-tenant-prevention-patterns)
- [Multi-Tenancy in Redis and NATS](#multi-tenancy-in-redis-and-nats)

---

## Overview

GGID uses a **shared-database with Row-Level Security** model. All tenants
share the same PostgreSQL database, but PostgreSQL RLS policies enforce
strict isolation at the database engine level — even if application code omits
a tenant filter, the database prevents cross-tenant access.

```
┌──────────────────────────────────────────────┐
│              Shared PostgreSQL               │
│  ┌────────────────────────────────────────┐  │
│  │           tenants table                │  │
│  │  ┌──────────┐  ┌──────────┐           │  │
│  │  │ Tenant A │  │ Tenant B │  ...       │  │
│  │  │ users    │  │ users    │           │  │
│  │  │ roles    │  │ roles    │           │  │
│  │  │ audits   │  │ audits   │           │  │
│  │  └──────────┘  └──────────┘           │  │
│  │     ↑ RLS         ↑ RLS                │  │
│  └────────────────────────────────────────┘  │
└──────────────────────────────────────────────┘
```

### Isolation Layers

| Layer | Mechanism | Enforcement Point |
|-------|-----------|-------------------|
| Gateway | X-Tenant-ID header extraction | HTTP middleware |
| Service | tenant_id in context | gRPC metadata / HTTP header |
| Database | `SET LOCAL app.tenant_id` + RLS policy | PostgreSQL engine |
| Redis | Key prefix `tid:{tenant_id}:` | Application code |
| NATS | Subject prefix `ggid.events.{tenant_id}.` | Application code |

---

## Tenant ID Propagation Chain

Tenant context flows through every layer of the system:

```
Client Request
    │
    ├── HTTP Header: X-Tenant-ID: 00000000-0000-0000-0000-000000000001
    │
    ▼
Gateway Middleware
    │
    ├── Extracts X-Tenant-ID header
    ├── Validates tenant exists and is active
    ├── Injects into gRPC metadata (if gRPC backend)
    ├── Injects into forwarded HTTP headers
    │
    ▼
Service Handler
    │
    ├── Reads tenant_id from context
    ├── Passes to repository layer
    │
    ▼
Repository Layer
    │
    ├── Begins transaction: tx.Begin(ctx)
    ├── Executes: SET LOCAL app.tenant_id = $1  (within tx)
    ├── All queries within tx are automatically scoped
    │
    ▼
PostgreSQL Engine
    │
    ├── RLS policy: USING (tenant_id = current_setting('app.tenant_id')::uuid)
    ├── Cross-tenant rows are invisible — zero-trust isolation
```

### Gateway: Header Extraction

```go
// gateway/internal/middleware/tenant.go
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := r.Header.Get("X-Tenant-ID")
        if tenantID == "" {
            writeError(w, 400, "X-Tenant-ID header required")
            return
        }

        // Validate tenant is active
        if !tenantStore.IsActive(r.Context(), tenantID) {
            writeError(w, 403, "tenant suspended or deleted")
            return
        }

        // Inject into context for downstream
        ctx := context.WithValue(r.Context(), tenantKey, tenantID)

        // For gRPC backends, inject into gRPC metadata
        ctx = metadata.AppendToOutgoingContext(ctx, "x-tenant-id", tenantID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Service: Context Extraction

```go
// services/identity/internal/repository/user_repo.go
func (r *UserRepo) ListUsers(ctx context.Context, limit int) ([]*User, error) {
    tenantID := tenant.FromContext(ctx)

    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback(ctx)

    // Set tenant context for this transaction
    _, err = tx.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID)
    if err != nil {
        return nil, fmt.Errorf("set tenant context: %w", err)
    }

    // Query is automatically scoped by RLS — no WHERE tenant_id = ... needed
    rows, err := tx.Query(ctx,
        "SELECT id, email, name, status FROM users ORDER BY created_at DESC LIMIT $1",
        limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    return scanUsers(rows)
}
```

---

## PostgreSQL Row-Level Security

### Enabling RLS

```sql
-- Enable RLS on all tenant-scoped tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE refresh_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE webauthn_credentials ENABLE ROW LEVEL SECURITY;

-- Create policies: users can only see rows matching their tenant
CREATE POLICY tenant_isolation ON users
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY tenant_isolation ON roles
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- ... repeat for each tenant-scoped table
```

### Force RLS Even for Table Owners

By default, table owners bypass RLS. To enforce RLS for everyone:

```sql
-- Force RLS for the application role (even though it owns the tables)
ALTER TABLE users FORCE ROW LEVEL SECURITY;
ALTER TABLE roles FORCE ROW LEVEL SECURITY;
```

### Superuser Bypass (for migrations only)

PostgreSQL superusers bypass RLS. Use this only for schema migrations:

```sql
-- Migration script (run as superuser)
SET ROLE postgres;  -- Bypasses RLS
ALTER TABLE users ADD COLUMN phone VARCHAR(20);
RESET ROLE;
```

### Verifying RLS is Active

```sql
-- Check which tables have RLS enabled
SELECT tablename, rowsecurity, forcerowsecurity
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY tablename;

-- Test: set tenant context and query
SET app.tenant_id = '00000000-0000-0000-0000-000000000001';
SELECT count(*) FROM users;  -- Only returns Tenant A's users

SET app.tenant_id = '00000000-0000-0000-0000-000000000002';
SELECT count(*) FROM users;  -- Only returns Tenant B's users
```

---

## Tenant Context Middleware

The tenant context is propagated through Go's `context.Context`:

```go
// pkg/tenant/context.go
package tenant

type contextKey struct{}

func WithTenantID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, contextKey{}, id)
}

func FromContext(ctx context.Context) string {
    if v, ok := ctx.Value(contextKey{}).(string); ok {
        return v
    }
    panic("tenant ID not found in context")
}

func FromContextSafe(ctx context.Context) (string, bool) {
    v, ok := ctx.Value(contextKey{}).(string)
    return v, ok
}
```

### gRPC Metadata Propagation

For service-to-service gRPC calls, tenant_id travels in gRPC metadata:

```go
// Sender: inject tenant into gRPC metadata
ctx = metadata.AppendToOutgoingContext(ctx, "x-tenant-id", tenantID)
resp, err := identityClient.GetUser(ctx, &pb.GetUserRequest{Id: userID})

// Receiver: extract tenant from gRPC metadata
func (s *IdentityServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    md, _ := metadata.FromIncomingContext(ctx)
    tenantID := md.Get("x-tenant-id")[0]
    ctx = tenant.WithTenantID(ctx, tenantID)

    return s.repo.GetByID(ctx, req.Id)
}
```

---

## Per-Tenant Configuration

Each tenant can have independent configuration for branding, features, and limits:

```bash
# Get tenant config
curl $API/api/v1/tenants/$TENANT_ID \
    -H "Authorization: Bearer $ADMIN_TOKEN"

# Update tenant config
curl -X PATCH $API/api/v1/tenants/$TENANT_ID \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "features": {
            "webauthn": true,
            "ldap": false,
            "saml": true,
            "scim": true
        },
        "rate_limits": {
            "login": { "limit": 20, "window": 60 },
            "api": { "limit": 200, "window": 60 }
        },
        "branding": {
            "primary_color": "#7c3aed",
            "logo_url": "https://cdn.example.com/logo.svg"
        }
    }'
```

### Config Resolution Priority

1. Per-tenant override (highest)
2. Tier default
3. Global default (lowest)

---

## Tenant Onboarding

### Automated Onboarding Flow

```
1. POST /api/v1/tenants (superadmin)
   ├── Create tenant record
   ├── Create default roles (admin, member, viewer)
   ├── Create default org
   └── Seed branding defaults

2. POST /api/v1/auth/register (first admin user)
   ├── Register with admin email
   ├── Assign admin role
   └── Send verification email

3. Configure tenant
   ├── Set branding (logo, colors)
   ├── Enable features (WebAuthn, LDAP, etc.)
   ├── Configure auth providers
   └── Set rate limits

4. Verify
   ├── Test login flow
   ├── Check health endpoints
   └── Verify RLS isolation
```

### Onboarding Script

```bash
#!/bin/bash
TENANT_NAME="$1"
ADMIN_EMAIL="$2"
API="https://iam.example.com"

# Create tenant
TENANT=$(curl -s -X POST "$API/api/v1/tenants" \
    -H "Authorization: Bearer $SUPERADMIN_TOKEN" \
    -d "{\"name\":\"$TENANT_NAME\",\"tier\":\"pro\"}")
TENANT_ID=$(echo "$TENANT" | jq -r '.id')
echo "Tenant created: $TENANT_ID"

# Register admin
curl -s -X POST "$API/api/v1/auth/register" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d "{\"username\":\"admin\",\"email\":\"$ADMIN_EMAIL\",\"password\":\"TempPass123!\"}"

# Assign admin role
JWT=$(curl -s -X POST "$API/api/v1/auth/login" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d '{"username":"admin","password":"TempPass123!"}' | jq -r '.access_token')

curl -s -X POST "$API/api/v1/users/admin/roles" \
    -H "Authorization: Bearer $JWT" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d '{"role":"admin"}'

echo "Done. Admin login: $ADMIN_EMAIL"
```

---

## Tenant Offboarding

### Soft Delete (Suspend)

```bash
# Suspend tenant — blocks all logins, keeps data
curl -X PATCH $API/api/v1/tenants/$TENANT_ID \
    -H "Authorization: Bearer $SUPERADMIN_TOKEN" \
    -d '{"status":"suspended"}'
```

Suspended tenants:
- All login attempts return 403
- No new sessions can be created
- Existing sessions are invalidated
- Data is fully retained
- Can be reactivated at any time

### Hard Delete (Purge)

```bash
# Step 1: Suspend
curl -X PATCH $API/api/v1/tenants/$TENANT_ID \
    -H "Authorization: Bearer $SUPERADMIN_TOKEN" \
    -d '{"status":"suspended"}'

# Step 2: Purge Redis keys
redis-cli --scan --pattern "tid:$TENANT_ID:*" | xargs redis-cli DEL

# Step 3: Delete from database (RLS policies cascade)
curl -X DELETE $API/api/v1/tenants/$TENANT_ID \
    -H "Authorization: Bearer $SUPERADMIN_TOKEN"

# Step 4: Verify deletion
curl $API/api/v1/tenants/$TENANT_ID \
    -H "Authorization: Bearer $SUPERADMIN_TOKEN"
# Expected: 404
```

### Data Export Before Deletion (GDPR)

```bash
# Export all tenant data as JSON
curl $API/api/v1/tenants/$TENANT_ID/export \
    -H "Authorization: Bearer $SUPERADMIN_TOKEN" \
    -o tenant-export.json
```

---

## Cross-Tenant Prevention Patterns

### Pattern 1: Always Use Context (Never Hardcode)

```go
// WRONG: hardcoded tenant_id
rows, _ := pool.Query(ctx, "SELECT * FROM users WHERE tenant_id = 'aaa-111'")

// RIGHT: from context, enforced by RLS
tenantID := tenant.FromContext(ctx)
tx, _ := pool.Begin(ctx)
tx.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID)
rows, _ := tx.Query(ctx, "SELECT * FROM users")  // RLS auto-filters
```

### Pattern 2: Validate Object Ownership

```go
// Before modifying a resource, verify it belongs to the caller's tenant
func (s *UserService) UpdateUser(ctx context.Context, userID uuid.UUID, updates *UserUpdate) error {
    user, err := s.repo.GetByID(ctx, userID)
    if err != nil {
        return ErrNotFound
    }

    // Double-check: RLS already enforces this, but belt-and-suspenders
    tenantID := tenant.FromContext(ctx)
    if user.TenantID != tenantID {
        return ErrNotFound  // Don't leak existence
    }

    return s.repo.Update(ctx, userID, updates)
}
```

### Pattern 3: Tenant-Scoped API Keys

```go
// API keys are validated and scoped to a specific tenant
func ValidateAPIKey(ctx context.Context, key string) (context.Context, error) {
    apiKey, err := apiKeys.Validate(ctx, key)
    if err != nil {
        return ctx, ErrInvalidAPIKey
    }

    // Inject the API key's tenant — not the client's claimed tenant
    ctx = tenant.WithTenantID(ctx, apiKey.TenantID)
    return ctx, nil
}
```

### Pattern 4: Audit All Cross-Tenant Denials

```sql
-- RLS denials are silent (rows just don't appear)
-- Application-level checks should log denied access attempts
INSERT INTO audit_events (tenant_id, event_type, data)
VALUES ($1, 'access.denied', '{"reason":"tenant_mismatch","resource":"user:550e8400"}');
```

---

## Multi-Tenancy in Redis and NATS

### Redis Key Namespacing

All Redis keys are prefixed with the tenant ID:

```
tid:{tenant_id}:{key_type}:{identifier}

Examples:
  tid:aaa-111:session:sess-abc123
  tid:aaa-111:rl:login:ip:192.168.1.1
  tid:aaa-111:refresh:rt-uuid-456
```

### NATS Subject Hierarchy

Audit events use tenant-scoped subjects:

```
ggid.events.{tenant_id}.{event_type}

Examples:
  ggid.events.aaa-111.user.created
  ggid.events.aaa-111.auth.login
```

Consumers filter by tenant:

```go
// Subscribe to single tenant only
js.Subscribe(fmt.Sprintf("ggid.events.%s.>", tenantID), handler)

// Subscribe to all tenants (superadmin only)
js.Subscribe("ggid.events.>", handler)
```

---

## Tenant Admin Role

Each tenant automatically gets an `admin` role with full management permissions
scoped to that tenant. Tenant admins cannot access other tenants' data or
manage platform-level settings.

### Tenant Admin Permissions

| Permission | Scope | Description |
|------------|-------|-------------|
| `users:read` | Tenant | List/view tenant users |
| `users:write` | Tenant | Create/update/suspend users |
| `users:delete` | Tenant | Delete users (soft-delete) |
| `roles:read` | Tenant | List tenant roles |
| `roles:write` | Tenant | Create/update roles |
| `orgs:read` | Tenant | List tenant organizations |
| `orgs:write` | Tenant | Create/update organizations |
| `audit:read` | Tenant | Query tenant audit events |
| `settings:read` | Tenant | View tenant config |
| `settings:write` | Tenant | Update tenant config (branding, features) |
| `api_keys:manage` | Tenant | Create/revoke API keys |
| `webhooks:manage` | Tenant | Configure webhooks |

### Super Admin (Platform Level)

A super admin role exists outside tenant scoping for platform operations:

```bash
# Super admin can manage tenants themselves
POST /api/v1/tenants          # Create tenant
PATCH /api/v1/tenants/{id}    # Suspend/activate tenant
DELETE /api/v1/tenants/{id}   # Delete tenant

# Super admin bypasses RLS via SET ROLE
SET ROLE postgres;  -- Platform operations only
```

### Assigning Tenant Admin

```bash
curl -X POST $API/api/v1/users/$USER_ID/roles \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d '{"role_key": "admin"}'
```

---

## Cross-Tenant Data Segregation Verification

### Automated Verification Script

```bash
#!/bin/bash
# verify-tenant-isolation.sh
# Verifies that RLS prevents cross-tenant data access

TENANT_A="00000000-0000-0000-0000-000000000001"
TENANT_B="00000000-0000-0000-0000-000000000002"

psql <<SQL
-- Test 1: Tenant A can only see own users
SET ROLE ggid_app;
SET LOCAL app.tenant_id = '$TENANT_A';
SELECT count(*) AS tenant_a_users FROM users;
-- Expected: Only Tenant A's users

-- Test 2: Tenant B cannot see Tenant A's data
RESET ROLE;
SET ROLE ggid_app;
SET LOCAL app.tenant_id = '$TENANT_B';
SELECT count(*) AS tenant_b_users FROM users;
-- Expected: Only Tenant B's users (different count)

-- Test 3: Cross-tenant query returns empty
SET LOCAL app.tenant_id = '$TENANT_B';
SELECT * FROM users WHERE tenant_id = '$TENANT_A';
-- Expected: 0 rows (RLS blocks this)

-- Test 4: Audit events are isolated
SET LOCAL app.tenant_id = '$TENANT_A';
SELECT count(*) FROM audit_events WHERE tenant_id != '$TENANT_A';
-- Expected: 0 rows
SQL
```

### Verification via API

```bash
# Login as Tenant A user
JWT_A=$(curl -s -X POST $API/api/v1/auth/login \
    -H "X-Tenant-ID: $TENANT_A" \
    -d '{"username":"alice","password":"pass"}' | jq -r '.access_token')

# Try to access Tenant B's users (should fail)
COUNT=$(curl -s $API/api/v1/users \
    -H "Authorization: Bearer $JWT_A" \
    -H "X-Tenant-ID: $TENANT_B" | jq '.data | length')

echo "Cross-tenant access returned: $COUNT users"
# Expected: 0 or 403 error
```

### RLS Audit Query

```sql
-- Verify all tenant-scoped tables have RLS enabled
SELECT
    relname AS table_name,
    relrowsecurity AS rls_enabled,
    relforcerowsecurity AS rls_forced
FROM pg_class
WHERE relname IN (
    'users', 'credentials', 'roles', 'user_roles',
    'sessions', 'refresh_tokens', 'api_keys',
    'webauthn_credentials', 'audit_events',
    'oauth_clients', 'oauth_consent', 'organizations'
)
ORDER BY relname;
```

---

## References

- [Multi-Tenant Architecture](./multi-tenant-architecture.md) — Design document
- [Security Hardening](./security-hardening.md) — Production security checklist
- [Configuration](./configuration.md) — All env vars and settings
- [Deployment Architecture](./deployment-architecture.md) — Production topology
