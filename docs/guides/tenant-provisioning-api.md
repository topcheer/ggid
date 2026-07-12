# Tenant Provisioning API Guide

Create, update, suspend, and delete tenants; admin user setup, resource allocation, quota configuration, DNS/branding.

## Overview

GGID is a multi-tenant IAM platform. Each tenant is an isolated identity domain with its own users, roles, and configuration. Row-Level Security (RLS) ensures complete data isolation at the PostgreSQL level.

## Tenant Lifecycle

```
Created → Provisioned → Active → Suspended → Deleted (soft) → Purged (hard)
```

## Create Tenant

```bash
POST /api/v1/admin/tenants
{
  "name": "Acme Corporation",
  "slug": "acme",
  "plan": "enterprise",
  "admin": {
    "email": "admin@acme.com",
    "display_name": "Acme Admin",
    "password": "temp-secure-password"
  },
  "branding": {
    "logo_url": "https://acme.com/logo.png",
    "primary_color": "#0052CC"
  }
}
# → 201 Created
# {
#   "id": "uuid",
#   "name": "Acme Corporation",
#   "slug": "acme",
#   "plan": "enterprise",
#   "status": "active",
#   "admin_user_id": "uuid",
#   "created_at": "2025-01-15T10:00:00Z"
# }
```

### What Happens on Creation

1. PostgreSQL: Create tenant record with RLS policy
2. Auto-create admin user for the tenant
3. Provision default roles (tenant_admin, user)
4. Create default groups (All Users, Administrators)
5. Set up audit event stream for tenant
6. Send admin activation email
7. Record provisioning in audit log

## Update Tenant

```bash
PATCH /api/v1/admin/tenants/{tenant_id}
{
  "name": "Acme Corp International",
  "plan": "enterprise-plus"
}
```

### Updatable Fields

| Field | Description |
|-------|-------------|
| name | Display name |
| plan | Subscription tier (free/starter/business/enterprise) |
| status | active/suspended |
| branding | Logo, colors, custom CSS |
| features | Feature flags per tenant |
| dns_config | Custom domain mapping |

## Suspend Tenant

```bash
POST /api/v1/admin/tenants/{tenant_id}/suspend
{
  "reason": "Non-payment",
  "effective_at": "2025-02-01T00:00:00Z"
}
```

Suspension effects:
- All user sessions revoked immediately
- New logins blocked (error: "tenant suspended")
- OAuth tokens invalidated
- Data retained (not deleted)
- Admin can still access Console (read-only)

## Reactivate Tenant

```bash
POST /api/v1/admin/tenants/{tenant_id}/activate
```

Users must re-authenticate. Previous sessions are not restored.

## Delete Tenant (Soft Delete)

```bash
DELETE /api/v1/admin/tenants/{tenant_id}
{
  "confirmation": "DELETE-ACME-PERMANENTLY",
  "reason": "Customer offboarding"
}
```

Soft delete effects:
- Tenant status → `deleted`
- All users anonymized (email → hashed)
- OAuth clients revoked
- API keys invalidated
- Audit logs retained for 7 years (compliance)
- Data purge scheduled after 90 days

## Purge Tenant (Hard Delete)

```bash
POST /api/v1/admin/tenants/{tenant_id}/purge
{
  "confirmation": "PURGE-CONFIRMED"
}
```

This is irreversible. Only available after soft delete + 90-day grace period.

## Resource Allocation & Quotas

```bash
GET /api/v1/admin/tenants/{tenant_id}/quotas
# → {
#   "max_users": 10000,
#   "max_roles": 500,
#   "max_orgs": 100,
#   "max_oauth_clients": 50,
#   "max_api_keys": 100,
#   "storage_mb": 5000,
#   "audit_retention_days": 2555,
#   "api_rate_limit": 10000
# }

PUT /api/v1/admin/tenants/{tenant_id}/quotas
{
  "max_users": 50000,
  "storage_mb": 20000
}
```

### Plan-Based Defaults

| Plan | Max Users | Storage | Rate Limit | Audit Retention |
|------|----------|---------|-----------|----------------|
| Free | 100 | 100MB | 1K/min | 90 days |
| Starter | 1,000 | 1GB | 5K/min | 1 year |
| Business | 10,000 | 5GB | 10K/min | 3 years |
| Enterprise | 100,000 | 20GB | 50K/min | 7 years |

Quota violations return `413 Payload Too Large` or `429 Too Many Requests`.

## Admin User Setup

Each tenant gets a default admin. Additional admins can be created:

```bash
POST /api/v1/admin/tenants/{tenant_id}/admins
{
  "email": "admin2@acme.com",
  "display_name": "Secondary Admin",
  "send_activation": true
}
```

The admin user is assigned `tenant_admin` role with full tenant-scoped permissions.

## DNS & Branding

### Custom Domain

```bash
PUT /api/v1/admin/tenants/{tenant_id}/dns
{
  "custom_domain": "auth.acme.com",
  "verify_method": "CNAME",
  "cname_target": "tenants.ggid.dev"
}
# → Returns DNS verification record to add

# After DNS configured:
POST /api/v1/admin/tenants/{tenant_id}/dns/verify
# → 200 {"verified": true, "ssl_provisioned": true}
```

### White-Label Branding

```bash
PUT /api/v1/admin/tenants/{tenant_id}/branding
{
  "logo_url": "https://acme.com/logo.png",
  "primary_color": "#0052CC",
  "secondary_color": "#F4F5F7",
  "login_page_title": "Acme Sign In",
  "email_from": "noreply@acme.com",
  "custom_css": "/* tenant-specific overrides */"
}
```

Branded elements: login page, consent screen, email templates, password reset page.

## Monitoring

| Metric | Alert |
|--------|-------|
| Tenant user count >80% of quota | Notify tenant admin |
| Tenant API rate limit hits >5% | Notify tenant admin |
| Suspended tenant still receiving traffic | Possible misconfiguration |
| Quota exceeded | Block + log |

## See Also

- [RBAC Design Patterns](rbac-design-patterns.md)
- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
- [Compliance Framework Mapping](compliance-framework-mapping.md)
