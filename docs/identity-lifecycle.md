# Identity Lifecycle Management

Complete guide to user identity lifecycle in GGID — from registration to deletion.

---

## Lifecycle States

```
                 ┌──────────┐
                 │  (none)  │
                 └────┬─────┘
                      │ Register
                      ▼
               ┌────────────┐
          ┌───►│   Active   │◄─── Activate
          │    └──────┬─────┘
          │           │
    Unlock│     ┌─────┴──────┬──────────┐
          │     │            │          │
          │     ▼            ▼          ▼
          │ ┌────────┐ ┌──────────┐ ┌────────┐
          └─┤ Locked │ │ Inactive │ │Deleted │
            └────────┘ └────┬─────┘ └────────┘
                            │ Reactivate
                            ▼
                      ┌──────────┐
                      │  Active   │
                      └──────────┘
```

| State | Can Login | Visible in Lists | Data Retained | API Access |
|-------|:---------:|:----------------:|:-------------:|:----------:|
| Active | Yes | Yes | Yes | Yes |
| Locked | No | Yes | Yes | No |
| Inactive | No | Yes | Yes | No |
| Deleted | No | No (soft) | Audit trail only | No |

---

## Registration

### Self-Registration

```bash
POST /api/v1/auth/register
{
  "username": "john.doe",
  "email": "john@example.com",
  "password": "SecurePass@123"
}
# → Creates user (status: active, email_verified: false)
```

### Admin-Created

```bash
POST /api/v1/users
{
  "username": "jane.doe",
  "email": "jane@example.com",
  "display_name": "Jane Doe"
}
# → Creates user (status: active, email_verified: false)
```

### SCIM Provisioned

```bash
POST /scim/v2/Users
{
  "userName": "scott@example.com",
  "emails": [{"value": "scott@example.com"}],
  "active": true
}
# → Creates user (status: active, email_verified: true)
```

### LDAP Auto-Provision

When `LDAP_AUTO_PROVISION=true`, first LDAP login creates the user:
1. User authenticates via LDAP bind
2. GGID searches LDAP entry for attributes
3. Creates GGID user from LDAP attributes
4. Issues JWT

---

## Email Verification

After registration, users receive a verification email:

```bash
# Trigger verification email
POST /api/v1/auth/email/verify/request
{"email": "john@example.com"}

# User clicks link → verify
POST /api/v1/auth/email/verify
{"token": "token-from-email"}
# → Sets email_verified = true
```

### Unverified User Behavior

- Can log in (unless policy requires verified email)
- Cannot use password reset (email not verified)
- Audit event logs `email_verified: false`

### Forced Verification Policy

```bash
PUT /api/v1/settings/security
{"require_email_verification": true}
```

Blocks login for unverified users after grace period (default: 7 days).

---

## Activation / Deactivation

### Lock (Admin Action)

```bash
POST /api/v1/users/{user_id}/lock
{"reason": "Security incident"}
# → status: locked, JWT revoked, audit event published
```

Locked users:
- Cannot log in
- Existing tokens added to Redis blocklist
- Appear in user lists (for audit)

### Unlock

```bash
POST /api/v1/users/{user_id}/unlock
# → status: active
```

### Deactivate (Soft Disable)

```bash
PATCH /api/v1/users/{user_id}
{"status": "inactive"}
# → status: inactive (retained for reactivation)
```

Inactive users:
- Cannot log in
- Not counted as active users
- Can be reactivated

### Reactivate

```bash
PATCH /api/v1/users/{user_id}
{"status": "active"}
```

---

## Deletion

### Soft Delete (Default)

```bash
DELETE /api/v1/users/{user_id}
```

Cascade deletes:
- `users` row removed
- `credentials` deleted
- `user_roles` deleted
- `org_members` deleted

Audit events retain `actor_id` as NULL (compliance trail preserved).

### Hard Delete (GDPR Erasure)

For complete data removal (GDPR Article 17):

```bash
DELETE /api/v1/users/{user_id}?hard=true
```

Removes all references including:
- User row
- All associated data
- Audit events anonymized (actor_id → NULL, actor_name → "deleted user")

---

## Account Recovery

### Password Reset Flow

```
User requests reset
      │
      ▼
GGID generates token (Redis, TTL=30min)
      │
      ▼
Email sent with reset link
      │
      ▼
User clicks link → enters new password
      │
      ▼
Token verified, password updated
      │
      ▼
All sessions revoked (security)
      │
      ▼
Audit event: password.reset
```

```bash
# Step 1: Request
POST /api/v1/auth/password/forgot
{"email": "john@example.com"}

# Step 2: Reset
POST /api/v1/auth/password/reset
{"token": "reset-token", "new_password": "NewPass@456"}
```

### MFA Reset (Admin)

When user loses MFA device:

```bash
DELETE /api/v1/users/{user_id}/mfa
# Clears all MFA credentials
# User must set up MFA on next login (if required)
```

Audit event: `mfa.disable` (actor: admin, reason: "account recovery")

### Magic Link Recovery

```bash
POST /api/v1/auth/magic-link
{"email": "john@example.com"}
# Bypasses password — user authenticates via email link
# Then can set new password via settings
```

---

## Session Management

### Active Sessions

```bash
GET /api/v1/auth/sessions
# Lists all active sessions for current user
```

### Revoke Session

```bash
DELETE /api/v1/auth/sessions/{session_id}
```

### Logout All

```bash
POST /api/v1/auth/logout-all
# Revokes all sessions, adds all JWTs to blocklist
```

---

## LDAP Sync

When LDAP is configured, user attributes sync on each login:

1. User authenticates via LDAP
2. GGID queries LDAP for latest attributes
3. Updates GGID user record (department, title, name)
4. If LDAP user is disabled → GGID locks the user

```bash
# Enable LDAP sync
LDAP_AUTO_PROVISION=true
LDAP_USER_FILTER=(sAMAccountName=%s)
```

### Scheduled LDAP Sync (Planned)

Future enhancement: periodic background sync (every 4 hours) to detect
LDAP changes without requiring user login.

---

## Lifecycle Audit Events

| Event | Trigger | Actor |
|-------|---------|-------|
| `user.register` | New registration | Self or system |
| `user.email_verified` | Email confirmed | Self |
| `user.update` | Profile updated | Self or admin |
| `user.lock` | Account locked | Admin |
| `user.unlock` | Account unlocked | Admin |
| `user.delete` | Account deleted | Admin |
| `password.reset` | Password reset via token | Self |
| `password.change` | Password changed | Self |
| `mfa.enable` | MFA method added | Self |
| `mfa.disable` | MFA method removed | Self or admin |
| `session.revoke` | Session revoked | Self or admin |
| `user.scim.create` | Created via SCIM | System (IdP) |
| `user.scim.update` | Updated via SCIM | System (IdP) |
| `user.ldap.sync` | Attributes synced from LDAP | System |
