# Passkey Recovery Strategy

Multi-device sync, backup authenticators, account recovery flow, admin-assisted recovery, re-enrollment security, and device loss scenarios.

## Overview

Passkeys eliminate passwords but introduce a new challenge: what happens when the user loses the device that holds their passkey? This guide covers all recovery scenarios.

## Defense in Depth

```
Layer 1: Multi-device sync (automatic)
Layer 2: Backup authenticator (user-set)
Layer 3: Recovery codes (user-generated)
Layer 4: Admin-assisted recovery (break-glass)
```

Each layer activates only if the previous fails.

## Multi-Device Sync

### Platform Sync (Automatic)

| Platform | Sync Mechanism | Cross-Platform |
|----------|---------------|----------------|
| Apple | iCloud Keychain | Apple devices only |
| Google | Google Password Manager | Android + Chrome |
| Microsoft | Windows Hello | Windows devices |

```
User registers passkey on iPhone
  → Automatically syncs to iPad, Mac (via iCloud Keychain)
  → If iPhone lost, passkey still available on Mac
```

**Limitation**: Sync is platform-specific. An Apple passkey doesn't sync to Android.

### Cross-Platform (Hybrid Transport)

```
Desktop (no passkey) → QR code → Phone (has passkey)
  → Phone authenticates via Bluetooth proximity
  → Assertion relayed to desktop
  → Desktop session authenticated (no passkey stored)
```

This is a one-time bridge, not a permanent sync.

## Backup Authenticators

### Enrollment

```bash
# Register a backup device during initial setup
POST /api/v1/auth/webauthn/register/begin
{"user_id": "uuid", "device_name": "Backup YubiKey"}

POST /api/v1/auth/webauthn/register/complete
{...}
# → 201 {credential_id: "uuid", device_name: "Backup YubiKey"}
```

### Backup Requirements

| Requirement | Rationale |
|-------------|-----------|
| Min 1 backup recommended | Protect against device loss |
| Max 10 credentials per user | Prevent abuse |
| Notify on new enrollment | Security awareness email |
| Backup should be different device | Not same phone |

## Recovery Codes

Generated at enrollment — single-use, bypass WebAuthn:

```bash
# Generate recovery codes (10 codes)
POST /api/v1/auth/webauthn/recovery-codes/generate
# → {
#   "codes": [
#     "ABCDE-FGHIJ-KLMNO-PQRST",
#     "UVWXY-ZABCD-EFGHI-JKLMN",
#     ... (10 total)
#   ],
#   "warning": "Store these securely. They will not be shown again."
# }

# Use a recovery code
POST /api/v1/auth/recovery
{"code": "ABCDE-FGHIJ-KLMNO-PQRST"}
# → 200 {access_token: "...", must_enroll_new_credential: true}
```

### Recovery Code Rules

| Rule | Value |
|------|-------|
| Code format | 4 groups × 5 chars (base32) |
| Total codes | 10 |
| Each code | Single-use |
| After use | Must enroll new credential |
| Storage | bcrypt-hashed in DB |
| Regeneration | Invalidates all previous codes |

## Account Recovery Flow

```
User lost device with passkey
    │
    ▼
Try multi-device sync (Layer 1)
    ├── Has synced device? → Login normally
    └── No synced device ↓

Try backup authenticator (Layer 2)
    ├── Has backup? → Login with backup
    └── No backup ↓

Try recovery codes (Layer 3)
    ├── Has codes? → Use code → Must enroll new passkey
    └── No codes ↓

Admin-assisted recovery (Layer 4)
    └── Identity verification → Admin re-enrollment
```

## Admin-Assisted Recovery

When all self-service options fail:

```bash
# User requests admin-assisted recovery
POST /api/v1/auth/recovery/request
{
  "user_id": "uuid",
  "verification_method": "email_otp",  // or "manager_approval"
  "reason": "lost device, no backup, no recovery codes"
}
# → 202 {recovery_request_id: "rec-abc", status: "pending_verification"}
```

### Verification Chain

```
1. Email OTP to registered email
2. Manager or peer approval (if configured)
3. Security team review (high-risk accounts)
4. Admin enrolls new credential on user's behalf
```

```bash
# Admin completes recovery (requires admin scope)
POST /api/v1/admin/recovery/{recovery_request_id}/complete
{
  "verification_passed": true,
  "approved_by": "admin-uuid",
  "new_credential": {...}  // WebAuthn registration
}
```

### Break-Glass Rules

| Rule | Enforcement |
|------|-------------|
| Dual control | Admin + manager approval |
| Full audit trail | Every step logged |
| Time-boxed | Recovery session expires in 15 min |
| Rate limited | Max 3 recovery events per year per user |
| Notify security | Automatic SIEM alert |
| Cool-down | 24h before user can change credentials again |

## Device Loss Scenarios

### Scenario 1: Lost Phone (with synced passkey)

```
1. User gets new phone
2. Signs in with Apple ID / Google account
3. Passkey syncs automatically
4. User can authenticate immediately
→ No recovery needed
```

### Scenario 2: Lost Phone (no sync, has backup)

```
1. User has backup YubiKey
2. Login with backup
3. Enroll new passkey on new phone
4. Revoke lost phone's credential
→ Self-service, no admin needed
```

### Scenario 3: Lost All Devices (has recovery codes)

```
1. User enters recovery code
2. Granted temporary access
3. Must enroll new passkey immediately
4. Recovery code consumed
→ Self-service, codes depleted by 1
```

### Scenario 4: Lost Everything (no codes, no backup)

```
1. Admin-assisted recovery
2. Identity verification (email + manager approval)
3. Admin enrolls new credential
4. Previous credentials revoked
5. 24h cool-down
→ Requires admin intervention
```

## Re-Enrollment Security

After recovery, the system forces re-enrollment:

```bash
# Recovery grants temporary access with flag
POST /api/v1/auth/recovery
{"code": "..."}
# → 200 {
#   "access_token": "...",
#   "must_enroll_credential": true,  // UI forces enrollment
#   "credential_enrollment_deadline": "2025-01-15T11:00:00Z"  // 1 hour
# }

# If user doesn't enroll within deadline → session expires
```

### Credential Revocation After Recovery

```bash
# Revoke all previous credentials
POST /api/v1/auth/webauthn/revoke-all
{"user_id": "uuid", "reason": "device_lost"}
# → All existing WebAuthn credentials revoked
# → User starts fresh
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Recovery code usage | Any → investigate |
| Admin-assisted recovery | Any → page security |
| Multiple recovery attempts | >3/year/user → social engineering risk |
| Credential enrollment spike | >5 in 1h → possible account takeover |
| Stale credentials (>180 days) | Flag for user review |

## See Also

- [Passwordless Auth Architecture](passwordless-auth-architecture.md)
- [WebAuthn Server Implementation](webauthn-server-implementation.md)
- [WebAuthn Recovery](webauthn-recovery.md)
- [Multi-Factor Step-Up Design](multi-factor-step-up-design.md)
