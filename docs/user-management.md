# User Lifecycle Management

Complete user lifecycle: create, import, activate, suspend, delete, profile
management, password reset, account locking, bulk operations, and dormant
account cleanup.

---

## Table of Contents

- [User States](#user-states)
- [Create User](#create-user)
- [Bulk Import](#bulk-import)
- [Activate/Suspend/Delete](#activatesuspenddelete)
- [Profile Management](#profile-management)
- [Password Reset Flow](#password-reset-flow)
- [Account Locking](#account-locking)
- [Bulk Operations](#bulk-operations)
- [Dormant Account Cleanup](#dormant-account-cleanup)

---

## User States

```
                 ┌──────────┐
         ┌──────►│  active   │◄──────┐
         │       └──────┬───┘       │
         │              │           │
    activate        suspend     reset_password
         │              │           │
         │       ┌──────▼───┐       │
         │       │ suspended │       │
         │       └──────┬───┘       │
         │              │           │
    create      delete/reactivate   │
         │              │           │
  ┌──────▼───┐   ┌──────▼───┐  ┌───┴──────┐
  │  pending │   │ deleted  │  │ locked   │
  └──────────┘   └──────────┘  └──────────┘
```

| State | Login | API Access | Sessions |
|-------|:-----:|:----------:|:--------:|
| `pending` | No | No | None |
| `active` | Yes | Yes | Active |
| `suspended` | No | No | Revoked |
| `locked` | No | No | Preserved |
| `deleted` | No | No | Revoked |

---

## Create User

### Admin Create

```bash
curl -X POST https://iam.example.com/api/v1/admin/users \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "username": "jane.doe",
    "email": "jane.doe@example.com",
    "first_name": "Jane",
    "last_name": "Doe",
    "display_name": "Jane Doe",
    "phone": "+1-555-123-4567",
    "status": "active",
    "send_welcome_email": true,
    "temporary_password": "TempPass123!"
  }'
```

### Self-Registration

```bash
curl -X POST https://iam.example.com/api/v1/auth/register \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "username": "jane.doe",
    "email": "jane.doe@example.com",
    "password": "SecurePass123!"
  }'
```

### JIT Provisioning (via IdP)

When a user authenticates via SAML/OIDC/Social and doesn't exist, GGID can
auto-provision:

```yaml
federation:
  jit_provisioning: true
  default_role: "viewer"
  default_status: "active"
```

---

## Bulk Import

### CSV Import

```bash
curl -X POST .../admin/users/import \
  -F "file=@users.csv" \
  -F "send_welcome_email=true" \
  -F "default_role=viewer"
```

CSV format:
```csv
username,email,first_name,last_name,phone,department
john.smith,john@example.com,John,Smith,+15551234567,Engineering
jane.doe,jane@example.com,Jane,Doe,+15559876543,Marketing
```

Response:
```json
{
  "total": 100,
  "created": 95,
  "skipped": 3,
  "errors": 2,
  "error_details": [...]
}
```

### SCIM Bulk

```bash
curl -X POST .../scim/v2/Bulk \
  -d '{
    "Operations": [
      {"method": "POST", "path": "/Users", "data": {"userName": "alice@example.com", ...}},
      {"method": "POST", "path": "/Users", "data": {"userName": "bob@example.com", ...}}
    ]
  }'
```

---

## Activate/Suspend/Delete

### Suspend User

```bash
curl -X POST .../admin/users/{id}/deactivate \
  -d '{ "reason": "offboarding", "revoke_sessions": true }'
```

- Revokes all active sessions and tokens
- User cannot authenticate
- Data preserved (soft state)

### Reactivate

```bash
curl -X POST .../admin/users/{id}/activate
```

### Delete User

```bash
# Soft delete (default — 30-day grace)
curl -X DELETE .../admin/users/{id}

# Hard delete (permanent, super_admin only)
curl -X DELETE ".../admin/users/{id}?hard=true"
```

---

## Profile Management

### Update Profile

```bash
curl -X PATCH .../admin/users/{id} \
  -d '{
    "display_name": "Jane Smith",
    "department": "Engineering",
    "phone": "+1-555-999-8888"
  }'
```

### Self-Service Profile Update

```bash
curl -X PATCH .../me \
  -H "Authorization: Bearer $USER_TOKEN" \
  -d '{ "display_name": "Jane Smith" }'
```

---

## Password Reset Flow

### Admin-Initiated Reset

```bash
curl -X POST .../admin/users/{id}/reset-password \
  -d '{
    "temporary_password": "NewTemp123!",
    "require_change_on_login": true
  }'
```

### Self-Service Reset (via email)

```
1. User requests reset: POST /api/v1/auth/password/reset-request {email}
2. GGID sends email with reset link (token valid 30 min)
3. User clicks link → reset page
4. User submits new password: POST /api/v1/auth/password/reset {token, password}
5. GGID validates token, updates password, revokes all sessions
```

### Password Change (authenticated)

```bash
curl -X POST .../me/password \
  -H "Authorization: Bearer $TOKEN" \
  -d '{ "current_password": "old", "new_password": "new" }'
```

---

## Account Locking

### Automatic Lockout

| Trigger | Action |
|---------|--------|
| 5 failed login attempts | Lock for 30 minutes |
| 10 failed attempts | Lock indefinitely (admin unlock required) |

```yaml
security:
  lockout:
    threshold: 5
    duration_minutes: 30
    auto_unlock: true
    max_attempts_before_admin: 10
```

### Admin Unlock

```bash
curl -X POST .../admin/users/{id}/unlock \
  -d '{ "reset_attempts": true }'
```

---

## Bulk Operations

### Bulk Status Change

```bash
curl -X POST .../admin/users/bulk \
  -d '{
    "action": "deactivate",
    "user_ids": ["user-1", "user-2", "user-3"],
    "reason": "department_closure"
  }'
``n
### Bulk Export

```bash
curl ".../admin/users/export?format=csv&status=active" \
  -o users.csv
```

---

## Dormant Account Cleanup

### Dormancy Detection

```yaml
security:
  dormant:
    enabled: true
    inactive_days: 90           # Flag after 90 days inactivity
    suspend_after_days: 180     # Suspend after 180 days
    delete_after_days: 365      # Delete after 365 days
    notify_before_days: 14      # Warn 14 days before action
    exclude_roles: ["service_account"]
```

### Dormancy Report

```bash
curl ".../admin/users/dormant?threshold_days=90" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```json
{
  "dormant_users": [
    {
      "user_id": "550e8400-...",
      "username": "old.user",
      "last_login": "2023-10-01T10:00:00Z",
      "days_inactive": 106,
      "action": "suspend",
      "action_date": "2024-04-01T00:00:00Z"
    }
  ],
  "total_dormant": 23
}
```

### Automated Cleanup Cron

```bash
# Runs daily via cron
GGID_DORMANT_CLEANUP=true ./bin/ggid-cleanup

# Actions:
# 1. Send warnings (14 days before action)
# 2. Suspend users past 180 days
# 3. Delete users past 365 days (soft-delete, 30-day grace)
```
