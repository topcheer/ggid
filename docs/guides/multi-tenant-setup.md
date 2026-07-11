# Multi-Tenant Setup Guide

> How to create tenants, assign admins, configure isolation, and manage the tenant lifecycle.

---

## Table of Contents

1. [Tenant Model](#tenant-model)
2. [Creating a Tenant](#creating-a-tenant)
3. [Assigning Tenant Admin](#assigning-tenant-admin)
4. [Isolation Configuration](#isolation-configuration)
5. [Tenant Lifecycle](#tenant-lifecycle)
6. [Per-Tenant Settings](#per-tenant-settings)

---

## Tenant Model

GGID uses a shared-database model with PostgreSQL Row-Level Security (RLS):

```
┌──────────────────────────────────┐
│        Shared Database           │
│  ┌────────┐  ┌────────┐          │
│  │Tenant A│  │Tenant B│   RLS     │
│  │ rows   │  │ rows   │ enforces  │
│  │        │  │        │ isolation │
│  └────────┘  └────────┘          │
└──────────────────────────────────┘
```

### Three-Layer Isolation

1. **Application**: JWT `tenant_id` claim is authoritative
2. **Connection**: `SET LOCAL app.tenant_id` per transaction
3. **Database**: RLS policy filters rows automatically

---

## Creating a Tenant

### Prerequisites

- Super-admin JWT from default tenant

```bash
# Login as super-admin
export ADMIN_JWT=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","password":"W3lcome-2025!"}' | jq -r .access_token)
```

### Create Tenant

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corporation",
    "plan": "enterprise",
    "max_users": 5000
  }' | jq .

# Response:
# {
#   "id": "55000000-0000-0000-0000-000000000002",
#   "name": "Acme Corporation",
#   "plan": "enterprise",
#   "active": true
# }

export TENANT_ID="55000000-0000-0000-0000-000000000002"
```

---

## Assigning Tenant Admin

### Register First User

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "username": "acme-admin",
    "email": "admin@acme.com",
    "password": "AcmeSecure!2025",
    "first_name": "Jane",
    "last_name": "Doe"
  }' | jq .
```

### Create and Assign Admin Role

```bash
# Create tenant_admin role
ADMIN_ROLE_ID=$(curl -s -X POST http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{"name":"Tenant Admin","key":"tenant_admin","permissions":["read:*"]}' \
  | jq -r .id)

# Assign to user
USER_ID=$(curl -s http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: $TENANT_ID" | jq -r '.users[0].id')

curl -s -X POST "http://localhost:8080/api/v1/users/$USER_ID/roles" \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d "{\"role_id\":\"$ADMIN_ROLE_ID\"}" | jq .
```

---

## Isolation Configuration

### Verify RLS

```sql
-- Connect as tenant user, verify only tenant data visible
SET app.tenant_id = '55000000-0000-0000-0000-000000000002';
SELECT count(*) FROM users;  -- Only Acme users

SET app.tenant_id = '00000000-0000-0000-0000-000000000001';
SELECT count(*) FROM users;  -- Only default tenant users
```

### Per-Tenant Configuration

| Setting | Scope | Example |
|---------|-------|--------|
| Max users | Per tenant | `max_users: 5000` |
| Allowed auth methods | Per tenant | Password + SAML only |
| Rate limit override | Per tenant | Higher limits for enterprise |
| Custom branding | Per tenant | Logo, colors |
| SSO configuration | Per tenant | Each tenant has own IdP |

---

## Tenant Lifecycle

```
Created → Active → Suspended → Deleted
              ↑         │
              └─────────┘
              (reactivate)
```

### Suspend Tenant

```bash
curl -X POST "http://localhost:8080/api/v1/tenants/$TENANT_ID/suspend" \
  -H "Authorization: Bearer $ADMIN_JWT" | jq .
# All sessions revoked, users can't login, API returns 403
```

### Reactivate Tenant

```bash
curl -X POST "http://localhost:8080/api/v1/tenants/$TENANT_ID/activate" \
  -H "Authorization: Bearer $ADMIN_JWT" | jq .
# Users can login again, sessions re-created on next auth
```

### Delete Tenant

```bash
# Soft delete (30-day grace period)
curl -X DELETE "http://localhost:8080/api/v1/tenants/$TENANT_ID" \
  -H "Authorization: Bearer $ADMIN_JWT"

# Hard delete (immediate, GDPR Article 17)
curl -X DELETE "http://localhost:8080/api/v1/tenants/$TENANT_ID?hard=true" \
  -H "Authorization: Bearer $ADMIN_JWT"
```

---

## Per-Tenant SSO

Each tenant can have its own SSO configuration:

```bash
# Tenant A uses Okta (SAML)
curl -X POST http://localhost:8080/api/v1/saml/config \
  -H "X-Tenant-ID: $TENANT_A" \
  -d '{"idp_metadata_url":"https://acme.okta.com/metadata"}'

# Tenant B uses Azure AD (OIDC)
curl -X POST http://localhost:8080/api/v1/oauth/clients \
  -H "X-Tenant-ID: $TENANT_B" \
  -d '{"client_id":"azure-b","redirect_uris":["https://login.microsoftonline.com/..."]}'
```

---

*See: [Architecture Overview](../architecture/overview.md) | [ADR-0001: Database Choice](../architecture/decision-record/0001-jwt-rsa-shared-key.md) | [Multi-Tenancy Reference](../multi-tenancy.md)*

*Last updated: 2025-07-11*