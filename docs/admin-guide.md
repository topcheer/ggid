# Admin Operations Guide

Complete guide for GGID administrators. Covers user CRUD, bulk import/export,
role assignment, organization management, tenant configuration, audit log
review, security monitoring, MFA enforcement, and impersonation.

---

## Table of Contents

- [Admin API Overview](#admin-api-overview)
- [User Management](#user-management)
- [Bulk Import/Export](#bulk-importexport)
- [Role Assignment](#role-assignment)
- [Organization Management](#organization-management)
- [Tenant Configuration](#tenant-configuration)
- [Audit Log Review](#audit-log-review)
- [Security Monitoring](#security-monitoring)
- [MFA Enforcement](#mfa-enforcement)
- [Impersonation](#impersonation)

---

## Admin API Overview

All admin operations require an admin-scoped JWT:

```
Authorization: Bearer <admin-jwt>
X-Tenant-ID: <tenant-uuid>
```

### Admin Roles

| Role | Scope | Capabilities |
|------|-------|-------------|
| `super_admin` | All tenants | Full system access, tenant CRUD |
| `admin` | Single tenant | User/org/role management, audit review |
| `security_admin` | Single tenant | Security monitoring, MFA enforcement, impersonation |
| `viewer` | Single tenant | Read-only access to all resources |

### Base URLs

```
Admin REST API:  https://iam.example.com/api/v1/admin
Admin gRPC:       iam.example.com:50051
```

---

## User Management

### Create User

```bash
curl -X POST https://iam.example.com/api/v1/admin/users \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "X-Tenant-ID: <tenant-uuid>" \
  -d '{
    "username": "jane.doe",
    "email": "jane.doe@example.com",
    "display_name": "Jane Doe",
    "first_name": "Jane",
    "last_name": "Doe",
    "phone": "+1-555-123-4567",
    "status": "active",
    "send_welcome_email": true,
    "temporary_password": "TempPass123!"
  }'
```

### List Users

```bash
# Paginated with filters
curl "https://iam.example.com/api/v1/admin/users?page=1&page_size=50&status=active&search=jane" \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "X-Tenant-ID: <tenant-uuid>"
```

```json
{
  "users": [...],
  "total": 1523,
  "page": 1,
  "page_size": 50
}
```

### Update User

```bash
curl -X PATCH https://iam.example.com/api/v1/admin/users/{user_id} \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "X-Tenant-ID: <tenant-uuid>" \
  -d '{
    "display_name": "Jane Smith",
    "department": "Engineering"
  }'
```

### Deactivate / Activate

```bash
# Deactivate (revokes all sessions)
curl -X POST https://iam.example.com/api/v1/admin/users/{user_id}/deactivate \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "X-Tenant-ID: <tenant-uuid>" \
  -d '{ "reason": "Offboarding", "revoke_sessions": true }'

# Activate
curl -X POST https://iam.example.com/api/v1/admin/users/{user_id}/activate \
  -H "Authorization: Bearer <admin-jwt>"
```

### Delete User

```bash
curl -X DELETE https://iam.example.com/api/v1/admin/users/{user_id} \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "X-Tenant-ID: <tenant-uuid>"
```

> Deletion is soft-delete by default. Hard delete requires `?hard=true` and
> `super_admin` role.

### Reset Password

```bash
curl -X POST https://iam.example.com/api/v1/admin/users/{user_id}/reset-password \
  -H "Authorization: Bearer <admin-jwt>" \
  -d '{
    "temporary_password": "NewTemp123!",
    "require_change_on_login": true
  }'
```

### Unlock Account

```bash
curl -X POST https://iam.example.com/api/v1/admin/users/{user_id}/unlock \
  -H "Authorization: Bearer <admin-jwt>"
```

---

## Bulk Import/Export

### CSV Import

```bash
curl -X POST https://iam.example.com/api/v1/admin/users/import \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "X-Tenant-ID: <tenant-uuid>" \
  -F "file=@users.csv" \
  -F "send_welcome_email=true" \
  -F "default_role=viewer"
```

#### CSV Format

```csv
username,email,first_name,last_name,display_name,phone,department
john.smith,john.smith@example.com,John,Smith,John Smith,+15551234567,Engineering
jane.doe,jane.doe@example.com,Jane,Doe,Jane Doe,+15559876543,Marketing
```

#### Import Response

```json
{
  "total": 100,
  "created": 95,
  "skipped": 3,
  "errors": 2,
  "error_details": [
    { "row": 47, "email": "invalid-email", "error": "invalid email format" },
    { "row": 82, "username": "dup-user", "error": "username already exists" }
  ],
  "job_id": "import-abc123"
}
```

### Export Users

```bash
# Export as CSV
curl "https://iam.example.com/api/v1/admin/users/export?format=csv&status=active" \
  -H "Authorization: Bearer <admin-jwt>" \
  -o users-export.csv

# Export as JSON
curl "https://iam.example.com/api/v1/admin/users/export?format=json" \
  -H "Authorization: Bearer <admin-jwt>" \
  -o users-export.json
```

### SCIM Bulk Import

For large-scale provisioning from IdPs (Okta, Azure AD):

```bash
curl -X POST https://iam.example.com/scim/v2/Bulk \
  -H "Authorization: Bearer <scim-token>" \
  -d '{
    "schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
    "failOnErrors": 5,
    "Operations": [
      { "method": "POST", "path": "/Users", "data": { ... } },
      { "method": "POST", "path": "/Users", "data": { ... } }
    ]
  }'
```

---

## Role Assignment

### Assign Role

```bash
curl -X POST https://iam.example.com/api/v1/admin/users/{user_id}/roles \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "X-Tenant-ID: <tenant-uuid>" \
  -d '{
    "role_id": "role-uuid",
    "scope": "tenant",
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

### List User Roles

```bash
curl https://iam.example.com/api/v1/admin/users/{user_id}/roles \
  -H "Authorization: Bearer <admin-jwt>"
```

### Revoke Role

```bash
curl -X DELETE https://iam.example.com/api/v1/admin/users/{user_id}/roles/{role_id} \
  -H "Authorization: Bearer <admin-jwt>"
```

### Bulk Role Assignment

```bash
curl -X POST https://iam.example.com/api/v1/admin/roles/{role_id}/assign-bulk \
  -H "Authorization: Bearer <admin-jwt>" \
  -d '{
    "user_ids": ["user-1", "user-2", "user-3"],
    "scope": "tenant"
  }'
```

---

## Organization Management

### Create Organization

```bash
curl -X POST https://iam.example.com/api/v1/admin/orgs \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "X-Tenant-ID: <tenant-uuid>" \
  -d '{
    "name": "Engineering",
    "description": "Engineering Department",
    "parent_id": "parent-org-uuid",
    "metadata": { "cost_center": "CC-1001" }
  }'
```

### Organization Tree

```bash
curl https://iam.example.com/api/v1/admin/orgs/tree \
  -H "Authorization: Bearer <admin-jwt>"
```

```json
{
  "org": { "id": "root", "name": "Company", "children": [
    { "id": "eng", "name": "Engineering", "children": [
      { "id": "platform", "name": "Platform Team" },
      { "id": "frontend", "name": "Frontend Team" }
    ]},
    { "id": "sales", "name": "Sales" }
  ]}
}
```

### Move User to Organization

```bash
curl -X POST https://iam.example.com/api/v1/admin/users/{user_id}/orgs \
  -d '{ "org_id": "engineering-org-uuid" }'
```

---

## Tenant Configuration

### View Tenant Settings

```bash
curl https://iam.example.com/api/v1/admin/tenant \
  -H "Authorization: Bearer <admin-jwt>"
```

```json
{
  "id": "00000000-0000-0000-0000-000000000001",
  "name": "Acme Corp",
  "status": "active",
  "settings": {
    "password_policy": { ... },
    "mfa_policy": { ... },
    "session_policy": { ... },
    "branding": { ... }
  },
  "quotas": {
    "max_users": 10000,
    "max_admins": 50,
    "max_api_keys": 100
  }
}
```

### Update Password Policy

```bash
curl -X PATCH https://iam.example.com/api/v1/admin/tenant/settings/password-policy \
  -d '{
    "min_length": 12,
    "require_uppercase": true,
    "require_lowercase": true,
    "require_digit": true,
    "require_special": true,
    "max_age_days": 90,
    "history_count": 12,
    "lockout_threshold": 5,
    "lockout_duration_minutes": 30
  }'
```

### Update Session Policy

```bash
curl -X PATCH https://iam.example.com/api/v1/admin/tenant/settings/session-policy \
  -d '{
    "session_timeout_minutes": 480,
    "idle_timeout_minutes": 60,
    "concurrent_sessions": 3,
    "require_reauth_for_sensitive": true
  }'
```

---

## Audit Log Review

### Query Audit Events

```bash
curl "https://iam.example.com/api/v1/audit/events?\
event_type=user.login\
&user_id=user-uuid\
&start=2024-01-01T00:00:00Z\
&end=2024-01-31T23:59:59Z\
&page=1&page_size=50" \
  -H "Authorization: Bearer <admin-jwt>"
```

### Filter by Event Type

| Category | Event Types |
|----------|-------------|
| Authentication | `user.login`, `user.logout`, `user.login.failed`, `user.token.refresh` |
| User Management | `user.created`, `user.updated`, `user.deleted`, `user.activated`, `user.deactivated` |
| Role Management | `role.assigned`, `role.revoked`, `role.created`, `role.deleted` |
| Security | `user.locked`, `user.unlocked`, `mfa.enabled`, `mfa.disabled`, `password.reset`, `session.revoked` |
| Admin | `admin.impersonation.start`, `admin.impersonation.end`, `admin.config.change` |

### Export Audit Logs

```bash
curl "https://iam.example.com/api/v1/audit/events/export?\
start=2024-01-01T00:00:00Z&end=2024-01-31T23:59:59Z&format=csv" \
  -H "Authorization: Bearer <admin-jwt>" \
  -o audit-january.csv
```

### Real-Time Audit Stream (SSE)

```bash
curl -N https://iam.example.com/api/v1/audit/events/stream \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "Accept: text/event-stream"
```

---

## Security Monitoring

### Failed Login Dashboard

```bash
curl "https://iam.example.com/api/v1/admin/security/failed-logins?\
window=24h&threshold=5" \
  -H "Authorization: Bearer <admin-jwt>"
```

```json
{
  "users": [
    {
      "user_id": "user-uuid",
      "username": "suspicious-user",
      "failed_attempts": 12,
      "last_attempt": "2024-01-15T10:30:00Z",
      "source_ips": ["192.168.1.50", "10.0.0.15"],
      "locked": true
    }
  ]
}
```

### Active Sessions

```bash
curl https://iam.example.com/api/v1/admin/users/{user_id}/sessions \
  -H "Authorization: Bearer <admin-jwt>"
```

```json
{
  "sessions": [
    {
      "session_id": "sess-uuid",
      "ip_address": "192.168.1.50",
      "user_agent": "Mozilla/5.0...",
      "device_type": "desktop",
      "created_at": "2024-01-15T08:00:00Z",
      "last_activity": "2024-01-15T10:25:00Z",
      "expires_at": "2024-01-15T16:00:00Z"
    }
  ]
}
```

### Revoke Session

```bash
curl -X DELETE https://iam.example.com/api/v1/admin/sessions/{session_id} \
  -H "Authorization: Bearer <admin-jwt>"
```

### Revoke All Sessions for User

```bash
curl -X DELETE https://iam.example.com/api/v1/admin/users/{user_id}/sessions \
  -H "Authorization: Bearer <admin-jwt>"
```

---

## MFA Enforcement

### View MFA Status

```bash
curl https://iam.example.com/api/v1/admin/users/{user_id}/mfa \
  -H "Authorization: Bearer <admin-jwt>"
```

### Enforce MFA at Tenant Level

```bash
curl -X PATCH https://iam.example.com/api/v1/admin/tenant/settings/mfa-policy \
  -d '{
    "required": true,
    "allowed_methods": ["totp", "webauthn", "sms"],
    "enrollment_grace_period_days": 7,
    "excluded_roles": ["service-account"]
  }'
```

### Reset User MFA

```bash
curl -X DELETE https://iam.example.com/api/v1/admin/users/{user_id}/mfa \
  -H "Authorization: Bearer <admin-jwt>" \
  -d '{ "reason": "lost device" }'
```

This removes all MFA factors for the user and forces re-enrollment on next login.

---

## Impersonation

Admins can impersonate users for troubleshooting. All impersonation sessions are
heavily audited.

### Start Impersonation

```bash
curl -X POST https://iam.example.com/api/v1/admin/impersonate \
  -H "Authorization: Bearer <admin-jwt>" \
  -d '{
    "user_id": "target-user-uuid",
    "reason": "debugging login issue",
    "duration_minutes": 30
  }'
```

```json
{
  "impersonation_token": "imp-jwt-xxx",
  "expires_at": "2024-01-15T11:00:00Z",
  "audit_id": "audit-impersonation-uuid"
}
```

### Impersonation Constraints

| Constraint | Value |
|------------|-------|
| Max duration | 60 minutes |
| Requires role | `admin` or `security_admin` |
| Cannot impersonate | `super_admin` users |
| Concurrent impersonations | 1 per admin |
| Audit logged | Every action during impersonation |
| Notification | Target user emailed after session ends |

### End Impersonation

```bash
curl -X POST https://iam.example.com/api/v1/admin/impersonate/end \
  -H "Authorization: Bearer <imp-jwt>"
```

### Audit Trail

Every impersonation generates these audit events:

```
admin.impersonation.start   — admin started impersonating user
user.login                  — impersonation session login (flagged)
user.*                      — all actions during session
admin.impersonation.end     — session ended
```

All events include `actor: <admin-id>` and `impersonation: true` metadata.
