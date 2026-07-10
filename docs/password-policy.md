# Password Policy Guide

Configure and enforce password security policies in GGID.

---

## Default Policy

| Rule | Default | Configurable |
|------|---------|:------------:|
| Minimum length | 8 characters | Yes |
| Maximum length | 128 characters | Yes |
| Uppercase letter | Required | Yes |
| Lowercase letter | Required | Yes |
| Digit | Required | Yes |
| Special character | Required | Yes |
| History check | Last 5 passwords | Yes |
| Expiration | Never | Yes |
| Breach check (HIBP) | Disabled | Yes |

---

## Configure Policy

### Via Console

**Settings** → **Security** → **Password Policy**

### Via API

```bash
PUT /api/v1/settings/password-policy
{
  "min_length": 12,
  "max_length": 128,
  "require_uppercase": true,
  "require_lowercase": true,
  "require_digit": true,
  "require_special": true,
  "special_chars": "!@#$%^&*()-_=+[]{}|;:,.<>?",
  "history_count": 10,
  "expiration_days": 90,
  "breach_check": true,
  "breach_check_api": "https://api.pwnedpasswords.com/range/"
}
```

---

## Complexity Rules

### Minimum Length

```json
{"min_length": 12}
```

NIST recommends minimum 8 characters. For sensitive systems, use 12+.

### Character Classes

```json
{
  "require_uppercase": true,
  "require_lowercase": true,
  "require_digit": true,
  "require_special": true
}
```

A valid password must contain at least one character from each enabled class.

### Custom Special Characters

```json
{"special_chars": "!@#$%^&*"}
```

Define which characters count as "special". Default includes common punctuation.

---

## Password History

Prevent password reuse:

```json
{"history_count": 10}
```

- Stores hashes of the last N passwords per user
- When user changes password, new password is checked against history
- If match found → rejected with `PASSWORD_IN_HISTORY` error
- History is checked using Argon2id verification (not plaintext comparison)

### How It Works

```go
// PasswordService.CheckHistory
func (ps *PasswordService) CheckHistory(ctx context.Context, tenantID, userID uuid.UUID, newPassword string) error {
    history, _ := ps.repo.GetPasswordHistory(ctx, tenantID, userID, 10)
    for _, oldHash := range history {
        if argon2id.Verify(newPassword, oldHash) {
            return ErrPasswordInHistory
        }
    }
    return nil
}
```

---

## Password Expiration

```json
{"expiration_days": 90}
```

- Passwords expire after N days
- On login, if expired → user forced to change password
- Audit event: `password.expired`
- Grace period configurable (default: 0 days — hard expiry)

### Configure Grace Period

```json
{
  "expiration_days": 90,
  "expiration_warning_days": 7
}
```

Users get a warning N days before expiration. During warning window, login succeeds but UI shows "Your password expires in X days."

---

## Breach Detection (HIBP)

Check passwords against the [Have I Been Pwned](https://haveibeenpwned.com/API/v3) database:

```json
{
  "breach_check": true,
  "breach_check_api": "https://api.pwnedpasswords.com/range/"
}
```

### How It Works (k-Anonymity)

1. Hash the password with SHA-1
2. Send first 5 hex characters to HIBP API
3. HIBP returns all breach hashes starting with those 5 chars
4. Check locally if the full hash is in the response

```
Password → SHA1("password") → 5BAA61E4C9B93F3F0682250B6CF8331B7EE68FD8
Send to HIBP: GET https://api.pwnedpasswords.com/range/5BAA6
Receive: 1E4C9B93F3F0682250B6CF8331B7EE68FD8:5  ← found 5 times
Result: PASSWORD_FOUND_IN_BREACH
```

### Response on Breach

If the password is found in a breach database:

```json
{
  "error": {
    "code": "PASSWORD_TOO_WEAK",
    "message": "This password has been found in known data breaches. Please choose a different password.",
    "details": {"breach_count": 4523}
  }
}
```

### Self-Hosted HIBP

For air-gapped deployments, download the HIBP database locally:

```bash
# Download HIBP password list (7-Zip compressed)
wget https://downloads.pwnedpasswords.com/passwords/pwned-passwords-sha1-ordered-v8.txt.7z

# Configure GGID to use local file
{"breach_check_api": "file:///data/hibp/pwned-passwords.txt"}
```

---

## Password Hashing

GGID uses **Argon2id** (RFC 9106) for all password storage:

| Parameter | Value | Description |
|-----------|-------|-------------|
| `time` | 1 | Iterations |
| `memory` | 64 MB | Memory per hash |
| `parallelism` | 2 | Threads |
| `salt length` | 16 bytes | Random per-user |
| `key length` | 32 bytes | Hash output |

### Why Argon2id?

- **Memory-hard** — Resistant to GPU/ASIC brute-force
- **Side-channel resistant** — Data-independent memory access
- **PHC winner** — Recommended by Password Hashing Competition

---

## Password Validation API

Validate a password against policy without changing it:

```bash
POST /api/v1/auth/password/validate
{"password": "TestPass@123"}
```

Response:

```json
{
  "valid": true,
  "score": 4,
  "checks": {
    "length": true,
    "uppercase": true,
    "lowercase": true,
    "digit": true,
    "special": true,
    "not_in_history": true,
    "not_in_breach": true
  },
  "suggestions": []
}
```

### Invalid Response

```json
{
  "valid": false,
  "score": 1,
  "checks": {
    "length": true,
    "uppercase": false,
    "lowercase": true,
    "digit": false,
    "special": false,
    "not_in_history": true,
    "not_in_breach": true
  },
  "suggestions": [
    "Add at least one uppercase letter",
    "Add at least one digit",
    "Add at least one special character"
  ]
}
```

---

## Best Practices

1. **Use HIBP breach check** — Catches common passwords users try to set
2. **Set min_length to 12+** — NIST SP 800-63B recommends 8 minimum, but 12+ is better
3. **Don't require periodic rotation** — NIST no longer recommends forced rotation (unless breach suspected)
4. **Allow paste in password fields** — Password managers improve security, not reduce it
5. **Show strength meter** — Real-time feedback helps users choose strong passwords
6. **Rate limit password changes** — Prevent rapid cycling to bypass history
7. **Log password policy changes** — Audit trail for compliance
