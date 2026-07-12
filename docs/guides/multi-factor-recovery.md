# Multi-Factor Authentication Recovery

This guide covers MFA recovery scenarios, methods, security considerations, and implementation details in GGID.

## Recovery Scenarios

| Scenario | Description | Recovery Method |
|---|---|---|
| Lost device | Phone with authenticator app lost | Backup codes / admin reset |
| Changed phone number | Phone number changed, SMS MFA fails | Admin reset / secondary email |
| App reset | Authenticator app reinstalled, seeds lost | Backup codes / admin reset |
| Damaged hardware key | FIDO2 security key broken | Backup key / admin reset |
| Biometric failure | Face/Touch ID permanently unavailable | Backup codes / PIN |
| Account takeover | Attacker enabled MFA, user locked out | Admin reset (with identity verification) |

## Recovery Methods

### 1. Backup Codes

One-time use codes generated at MFA enrollment. Each code is single-use.

```json
{
  "backup_codes": [
    "G7K9M2N4Q6R8",
    "B3D5F7H9J1L3",
    "X2Z4C6V8B0N1",
    "P5L7M9Q1R3S4",
    "T6U8W0Y2A4C5",
    "E7G9I1K3M5N6",
    "O8P0R2S4T6U7",
    "V9W1X3Y5Z7A8"
  ]
}
```

**Usage**: User enters a backup code instead of TOTP/HOTP during MFA challenge. Code is immediately consumed.

**Configuration**:
```yaml
mfa:
  backup_codes:
    enabled: true
    count: 8              # Number of codes generated
    length: 12            # Characters per code
    charset: "alphanumeric"
    single_use: true
    regenerate_allowed: false  # After initial enrollment
```

### 2. Recovery Key

A long passphrase that can reset MFA. Generated once at enrollment.

```
recovery_key: "correct-horse-battery-staple-mango-violet-piano-sunset"
```

**Usage**: User provides recovery key at recovery endpoint → MFA is reset → user must re-enroll.

**Configuration**:
```yaml
mfa:
  recovery_key:
    enabled: true
    format: "passphrase"  # or "hex"
    words: 8              # Number of words in passphrase
    min_entropy: 80       # bits
    storage: "argon2id"   # Hashing algorithm for storage
```

### 3. Admin Reset

Administrator manually resets MFA for a user after identity verification.

```yaml
mfa:
  admin_reset:
    enabled: true
    required_roles: ["security-admin", "helpdesk-admin"]
    require_identity_verification: true
    cooldown: 24h         # Min time between resets for same user
    notify_user: true     # Email user after reset
```

### 4. Secondary Email

Pre-registered alternate email for recovery verification.

```yaml
mfa:
  secondary_email:
    enabled: true
    require_verified: true
    verification_token_ttl: 15m
    code_length: 6
    code_ttl: 10m
```

## Recovery Flow Security

### Identity Verification Steps

```
1. User requests MFA recovery at /auth/mfa/recover
2. GGID presents identity verification challenge:
   a. Security questions (if configured)
   b. Email verification to registered address
   c. Secondary email verification (if configured)
   d. Admin approval (if required)
3. Optional: Mandatory waiting period (24h delay)
4. MFA is reset
5. User must re-enroll MFA within 24h
6. All existing sessions revoked
```

### Delay Period

```yaml
mfa:
  recovery:
    delay_period: 24h     # Time between request and reset
    allow_admin_override: true  # Admin can skip delay
    notify_pending: true  # Notify user of pending request
```

**Purpose**: Gives user time to find their device or notice suspicious activity. If an attacker is trying to reset MFA, the legitimate user gets notified.

### Session Revocation on Recovery

All active sessions and tokens are revoked immediately upon MFA reset:

```go
func ResetMFA(userID string) error {
    // Revoke all tokens
    revokeAllTokens(userID)

    // Destroy all sessions
    destroyAllSessions(userID)

    // Reset MFA enrollment
    clearMFAEnrollment(userID)

    // Require re-enrollment
    requireMFAEnrollment(userID)

    // Audit log
    audit.Log(AuditEvent{
        Type:   "mfa_reset",
        UserID: userID,
        Source: "recovery_flow",
        Time:   time.Now(),
    })

    return nil
}
```

## Recovery Code Generation

### Cryptographically Secure Generation

```go
import "crypto/rand"

func GenerateBackupCodes(count, length int) ([]string, error) {
    const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    codes := make([]string, count)
    for i := 0; i < count; i++ {
        code := make([]byte, length)
        _, err := rand.Read(code)
        if err != nil {
            return nil, err
        }
        for j, b := range code {
            codes[i] += string(charset[b%byte(len(charset))])
        }
    }
    return codes, nil
}
```

**Never use `math/rand` for security-critical code generation.** Always use `crypto/rand`.

### Recovery Key (Passphrase)

```go
func GenerateRecoveryKey(wordCount int) string {
    words := loadWordList()  // EFF word list (7776 words)
    key := make([]string, wordCount)
    for i := 0; i < wordCount; i++ {
        idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(words))))
        key[i] = words[idx.Int64()]
    }
    return strings.Join(key, "-")
}
```

**Entropy**: 8 words from 7776-word list = log2(7776^8) = ~80 bits of entropy.

## Storage (Encrypted at Rest)

### Backup Codes Storage

```go
func StoreBackupCodes(userID string, codes []string) error {
    for _, code := range codes {
        hash := argon2id.Hash(code, userID)
        // Store hash, not plaintext
        db.Exec("INSERT INTO backup_codes (user_id, hash, used, created_at) VALUES (?, ?, false, ?)",
            userID, hash, time.Now())
    }
}
```

### Recovery Key Storage

```go
func StoreRecoveryKey(userID, key string) error {
    // Argon2id hash with user-specific salt
    hash := argon2id.Hash(key, userID)
    db.Exec("UPDATE users SET recovery_key_hash = ? WHERE id = ?", hash, userID)
}
```

**Never store recovery codes or keys in plaintext.** Always use Argon2id or bcrypt with appropriate parameters.

## Audit Trail

Every MFA recovery action is logged:

| Event | Logged Fields |
|---|---|
| Recovery requested | user_id, ip, user_agent, method_requested |
| Identity verification passed | user_id, verification_method, ip |
| Identity verification failed | user_id, ip, failure_reason |
| MFA reset completed | user_id, reset_method, admin_id (if admin), ip |
| Backup code used | user_id, code_hash (partial), ip |
| Recovery key used | user_id, ip |
| Admin override delay | user_id, admin_id, reason |

```go
audit.Log(AuditEvent{
    Type:       "mfa_recovery",
    UserID:     userID,
    Method:     "backup_code",
    IP:         clientIP,
    UserAgent:  userAgent,
    Success:    true,
    Timestamp:  time.Now(),
    Metadata: map[string]interface{}{
        "codes_remaining": remainingCodes,
        "sessions_revoked": true,
    },
})
```

## Rate Limiting Recovery Attempts

```yaml
mfa:
  recovery:
    rate_limit:
      per_user: 3/hour
      per_ip: 10/hour
      per_tenant: 50/hour
    lockout:
      threshold: 5      # Failed attempts before lockout
      duration: 1h      # Lockout period
      notify_admin: true
```

### Implementation

```go
func CheckRecoveryRateLimit(userID, ip string) error {
    userKey := "mfa:recovery:user:" + userID
    ipKey := "mfa:recovery:ip:" + ip

    userCount, _ := redis.Incr(ctx, userKey).Result()
    if userCount == 1 {
        redis.Expire(ctx, userKey, time.Hour)
    }
    if userCount > 3 {
        return ErrRateLimitExceeded
    }

    ipCount, _ := redis.Incr(ctx, ipKey).Result()
    if ipCount == 1 {
        redis.Expire(ctx, ipKey, time.Hour)
    }
    if ipCount > 10 {
        return ErrRateLimitExceeded
    }

    return nil
}
```

## Recovery Flow Diagram

```
User loses MFA device
        │
        ▼
  POST /auth/mfa/recover
        │
        ▼
  Identity verification
  (security questions + email)
        │
  ┌─────┴─────┐
  │  Pass?    │
  └─────┬─────┘
     Yes │ No → Rate limit + audit
        ▼
  Optional: 24h delay period
        │
        ▼
  MFA enrollment cleared
        │
        ▼
  All sessions revoked
        │
        ▼
  User must re-enroll MFA
  within 24h
        │
        ▼
  New MFA enrolled
  → New backup codes generated
  → New recovery key generated
```

## Best Practices

1. **Generate backup codes at enrollment** — user must save them immediately
2. **Single-use backup codes** — prevent replay attacks
3. **Encrypt at rest** — Argon2id hash all recovery credentials
4. **Rate limit aggressively** — recovery is a high-risk endpoint
5. **Audit everything** — full trail of all recovery actions
6. **Mandatory re-enrollment** — after recovery, user must set up MFA again
7. **Notify on recovery** — email user when MFA is reset
8. **Admin reset requires approval** — don't allow single-admin self-service
9. **Delay period for high-security tenants** — 24h waiting period
10. **Revoke all sessions** — recovery implies potential compromise