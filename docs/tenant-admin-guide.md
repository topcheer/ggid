# Tenant Admin Guide

Day-to-day operations guide for tenant administrators: user management, role
assignment, IdP configuration, MFA policy, API keys, audit log review, and
data export.

---

## Table of Contents

- [Admin Access](#admin-access)
- [User Management](#user-management)
- [Role Assignment](#role-assignment)
- [IdP Configuration](#idp-configuration)
- [MFA Policy](#mfa-policy)
- [API Key Management](#api-key-management)
- [Audit Log Review](#audit-log-review)
- [Data Export](#data-export)

---

## Admin Access

### Required Role

Tenant admin operations require one of:
- `admin` role (full tenant management)
- `security_admin` role (security + MFA + impersonation)

### API Authentication

```bash
export GW="https://iam.example.com"
export TENANT="00000000-0000-0000-0000-000000000001"
export TOKEN="<your-admin-jwt>"

# All requests need these headers
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Tenant-ID: $TENANT" \
     "$GW/api/v1/admin/users"
```

### Console Access

Admin Console: `https://console.example.com`

| Page | Role Required |
|------|---------------|
| Dashboard | viewer+ |
| Users | admin |
| Roles | admin |
| Organizations | admin |
| Audit Logs | admin |
| Settings | admin |
| Security | security_admin |

---

## User Management

### Create User

```bash
curl -X POST "$GW/api/v1/admin/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "jane.doe",
    "email": "jane.doe@example.com",
    "first_name": "Jane",
    "last_name": "Doe",
    "display_name": "Jane Doe",
    "status": "active",
    "send_welcome_email": true
  }'
```

### List Users (with filters)

```bash
# All active users
curl "$GW/api/v1/admin/users?status=active&page=1&page_size=50" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Search by name
curl "$GW/api/v1/admin/users?search=jane" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

### Suspend User

```bash
curl -X POST "$GW/api/v1/admin/users/{user_id}/deactivate" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -d '{"reason": "offboarding", "revoke_sessions": true}'
```

### Reset Password

```bash
curl -X POST "$GW/api/v1/admin/users/{user_id}/reset-password" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -d '{"temporary_password": "NewTemp123!", "require_change_on_login": true}'
```

### Unlock Account

```bash
curl -X POST "$GW/api/v1/admin/users/{user_id}/unlock" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

### Bulk Import (CSV)

```bash
curl -X POST "$GW/api/v1/admin/users/import" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -F "file=@users.csv" -F "send_welcome_email=true" -F "default_role=viewer"
```

---

## Role Assignment

### Assign Role

```bash
curl -X POST "$GW/api/v1/admin/users/{user_id}/roles" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -d '{"role_id": "role-uuid", "scope": "tenant"}'
```

### Create Custom Role

```bash
curl -X POST "$GW/api/v1/roles" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "Content Manager",
    "key": "content_manager",
    "permissions": ["content:read", "content:write", "content:publish"],
    "parent_role": "editor"
  }'
``n
### List All Roles

```bash
curl "$GW/api/v1/roles" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

---

## IdP Configuration

### Add SAML IdP

```bash
curl -X POST "$GW/api/v1/admin/saml/idp" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "Corporate Okta",
    "metadata_url": "https://corp.okta.com/federationmetadata",
    "name_id_format": "emailAddress"
  }'
```

### Add OIDC Provider

```bash
curl -X POST "$GW/api/v1/admin/oidc/providers" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "Auth0",
    "issuer": "https://login.auth0.com",
    "client_id": "xxx",
    "client_secret": "xxx",
    "scopes": ["openid", "email", "profile"]
  }'
```

### Configure Social Login

```bash
curl -X POST "$GW/api/v1/admin/social/providers" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -d '{
    "provider": "google",
    "client_id": "xxx.apps.googleusercontent.com",
    "client_secret": "xxx",
    "enabled": true
  }'
```

---

## MFA Policy

### Enforce MFA Tenant-Wide

```bash
curl -X PATCH "$GW/api/v1/admin/tenant/settings/mfa-policy" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -d '{
    "required": true,
    "allowed_methods": ["totp", "webauthn"],
    "enrollment_grace_period_days": 14
  }'
```

### Reset User MFA

```bash
curl -X DELETE "$GW/api/v1/admin/users/{user_id}/mfa" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -d '{"reason": "lost device"}'
```

---

## API Key Management

### Create API Key

```bash
curl -X POST "$GW/api/v1/admin/api-keys" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "CI/CD Pipeline",
    "scopes": ["users:read", "users:write"],
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

Response:
```json
{
  "id": "key-uuid",
  "key": "gkey_xxxxxxxxxxxxxxxxxxxxxxxx",
  "name": "CI/CD Pipeline",
  "scopes": ["users:read", "users:write"],
  "expires_at": "2025-12-31T23:59:59Z"
}
```

> The `key` value is only shown once at creation. Store it securely.

### List API Keys

```bash
curl "$GW/api/v1/admin/api-keys" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

### Revoke API Key

```bash
curl -X DELETE "$GW/api/v1/admin/api-keys/{key_id}" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

---

## Audit Log Review

### Query Events

```bash
curl "$GW/api/v1/audit/events?
event_type=user.login&
start=2024-01-01T00:00:00Z&
end=2024-01-31T23:59:59Z&
limit=50" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

### Review Failed Logins

```bash
curl "$GW/api/v1/audit/events?
event_type=user.login.failed&
start=2024-01-15T00:00:00Z&
limit=100" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

### Check Admin Actions

```bash
curl "$GW/api/v1/audit/events?
event_type=admin.config.changed&
limit=20" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

### Verify Audit Chain Integrity

```bash
curl -X POST "$GW/api/v1/audit/verify-chain" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -d '{"start": "2024-01-01T00:00:00Z", "end": "2024-01-31T23:59:59Z"}'
```

---

## Data Export

### Export Users

```bash
# CSV format
curl "$GW/api/v1/admin/users/export?format=csv&status=active" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -o users-export.csv

# JSON format
curl "$GW/api/v1/admin/users/export?format=json" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -o users-export.json
```

### Export Audit Logs

```bash
curl "$GW/api/v1/audit/events/export?
format=csv&
start=2024-01-01T00:00:00Z&
end=2024-01-31T23:59:59Z" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -o audit-january.csv
```

### SCIM Export (for IdP sync)

```bash
curl "$GW/scim/v2/Users" \
  -H "Authorization: Bearer $SCIM_TOKEN" \
  -H "Content-Type: application/scim+json" | jq '.Resources[]' > scim-users.json
```

---

## Common Workflows

### Onboard New Employee

```bash
# 1. Create user
USER_ID=$(curl -sX POST .../admin/users -d '{"username":"new.hire",...}' | jq -r .id)

# 2. Assign role
curl -X POST .../admin/users/$USER_ID/roles -d '{"role_id":"editor-role-id"}'

# 3. Add to organization
curl -X POST .../admin/users/$USER_ID/orgs -d '{"org_id":"eng-org-id"}'

# 4. User receives welcome email with temporary password
```

### Offboard Employee

```bash
# 1. Deactivate (revokes sessions)
curl -X POST .../admin/users/$USER_ID/deactivate -d '{"reason":"offboarding"}'

# 2. Revoke all roles
curl -X DELETE .../admin/users/$USER_ID/roles/editor-role-id

# 3. Remove from organizations
curl -X DELETE .../admin/orgs/eng-org-id/members/$USER_ID

# 4. Verify in audit log
curl ".../audit/events?user_id=$USER_ID&limit=10"

# 5. After grace period: soft delete
curl -X DELETE .../admin/users/$USER_ID
```
```
