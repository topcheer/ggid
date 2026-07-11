# Password Policy Guide

This guide covers configuring password complexity, history, breach checking, pepper, and account lockout in GGID.

## Overview

GGID implements a multi-layer password security system:

1. **Argon2id hashing** — Memory-hard, GPU-resistant
2. **Password pepper** — Server-side secret appended before hashing
3. **Password history** — Prevents reuse of recent passwords
4. **Breach checking** — HIBP integration (optional)
5. **Account lockout** — Rate limiting failed attempts
6. **Complexity rules** — Configurable per tenant

## Password Hashing

### Argon2id Parameters

```yaml
PASSWORD_HASH_TIME: 1        # Iterations (seconds)
PASSWORD_HASH_MEMORY: 65536  # Memory in KB (64MB)
PASSWORD_HASH_THREADS: 2     # Parallelism
PASSWORD_HASH_KEY_LENGTH: 32 # Output length (bytes)
```
These parameters are tuned for ~100ms hash time on typical server hardware.

### Password Pepper

A pepper is a server-side secret appended to the password before hashing. Unlike a salt (stored per-user), the pepper is never stored in the database:

```go
// Hash: hash(password + salt + pepper)
hash := argon2id.Hash(password + salt + pepper)
```

**Configuration**:
```bash
PASSWORD_PEPPER=your-random-32-byte-pepper
```

**Critical**: If the pepper is lost, all stored password hashes become unverifiable. Store in a secrets manager (Vault, AWS Secrets Manager) with redundant backups.

## Password Complexity

### Default Policy

| Rule | Default | Configurable |
|------|---------|-------------|
| Minimum length | 8 | Yes (8-128) |
| Maximum length | 128 | Yes |
| Require uppercase | Yes | Yes |
| Require lowercase | Yes | Yes |
| Require digit | Yes | Yes |
| Require special char | Yes | Yes |
| Disallow common passwords | Yes | Yes |
| Disallow username in password | Yes | Yes |

### Per-Tenant Configuration

```bash
curl -X PUT https://api.ggid.example.com/api/v1/settings/password-policy \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "min_length": 12,
    "require_uppercase": true,
    "require_lowercase": true,
    "require_digit": true,
    "require_special": true,
    "min_special_chars": 1,
    "max_repeating_chars": 3,
    "disallow_username": true,
    "disallow_email": true
  }'
```

## Password History

Prevents users from reusing recent passwords:

```yaml
PASSWORD_HISTORY_COUNT: 5  # Remember last 5 passwords
```

When a user changes their password, GGID checks the new password against the history (re-hashes with each stored salt + pepper). If any match, the change is rejected.

## Password Expiry

```yaml
PASSWORD_MAX_AGE_DAYS: 90  # 0 = disabled (recommended: rely on breach detection instead)
```

> **Best practice**: NIST 800-63B recommends against forced password expiry. Use breach detection instead.

## Breach Checking (HIBP)

GGID integrates with Have I Been Pwned (HIBP) Passwords API to check if a password appears in known data breaches:

```yaml
HIBP_API_KEY: your-hibp-key   # Optional, enables breach checking
HIBP_ENABLED: true
```

### How It Works

1. Client sends password to GGID
2. GGID computes SHA-1 hash of password
3. GGID sends first 5 chars of hash to HIBP API (k-anonymity)
4. HIBP returns all breach hashes matching prefix
5. GGID checks if full hash is in the response
6. If breached: reject password with message

## Account Lockout

### Configuration

```yaml
MAX_LOGIN_ATTEMPTS: 5        # Lock after 5 failures
LOCKOUT_DURATION_MINUTES: 15 # Locked for 15 minutes
```

### How It Works

```
Failed attempt 1 → count = 1
Failed attempt 2 → count = 2
...
Failed attempt 5 → count = 5 → LOCK ACCOUNT
  → 429 Too Many Requests
  → Audit event: user.locked

After 15 minutes → account auto-unlocked
After successful login → counter reset
```

Admin can manually unlock:
```bash
curl -X POST https://api.ggid.example.com/api/v1/users/$USER_ID/unlock \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

## Password Reset Flow

```
1. User requests reset → POST /api/v1/auth/password-reset/request
   → GGID sends email with reset token (TTL: 15 min)

2. User clicks link → GET /reset?token=xxx
   → Console shows reset form

3. User submits new password → POST /api/v1/auth/password-reset/confirm
   → GGID verifies token
   → Validates complexity + history
   → Updates password hash
   → Revokes all sessions (jti blacklist)
   → Audit event: user.password_reset
```

## Security Checklist

- [ ] Argon2id with tuned parameters (~100ms hash time)
- [ ] Password pepper configured (32+ bytes, in secrets manager)
- [ ] Password history enabled (min 5)
- [ ] Breach checking enabled (HIBP)
- [ ] Account lockout configured (5 attempts, 15 min)
- [ ] Password reset tokens TTL < 15 min
- [ ] All sessions revoked on password change
- [ ] Password reset email sent with branded template

## See Also

- [MFA Setup](mfa-setup.md)
- [Security Audit Checklist](security-audit-checklist.md)
- [Session Management](../research/session-management.md)
- [Rate Limiting](rate-limiting-guide.md)
