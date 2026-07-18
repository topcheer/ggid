# Password Strength Guide (KB-082)

## Overview

GGID integrates a zxcvbn-inspired password strength engine that estimates crack resistance using pattern detection, entropy analysis, and breach database lookups. The engine scores passwords 0-4 and provides actionable feedback for users.

## Scoring Model

| Score | Strength | Crack Time | Behavior |
|-------|----------|------------|----------|
| 0 | Very Weak | instant - seconds | Rejected at registration/password change |
| 1 | Weak | minutes - hours | Rejected |
| 2 | Fair | days - months | Accepted with warning |
| 3 | Strong | years | Accepted |
| 4 | Very Strong | centuries+ | Accepted |

## Pattern Detection (5 Types)

The engine detects five password weakness patterns, each reducing entropy estimates:

### 1. Dictionary Words
Checks against a built-in common password dictionary (top 100+ most-used passwords).

```
"password" → Pattern: dictionary → Score penalty: severe
"admin123" → Pattern: dictionary → Score penalty: severe
```

### 2. Keyboard Sequences
Detects sequential keys across QWERTY rows (`qwerty`, `asdfgh`, `zxcvbn`).

```
"qwerty"   → Pattern: keyboard_sequence
"1qaz2wsx" → Pattern: keyboard_sequence
```

### 3. Repeated Characters
Flags runs of 3+ identical characters.

```
"aaabbb"   → Pattern: repeats
"111222"   → Pattern: repeats
```

### 4. L33t Speak
Detects common character substitutions (`@` → `a`, `3` → `e`, `1` → `i`, `0` → `o`).

```
"p@ssw0rd" → Pattern: l33t (resolves to dictionary word)
"h3ll0"    → Pattern: l33t
```

### 5. All Digits / All Letters
Flags passwords with no character class diversity.

```
"12345678"  → Pattern: all_digits (zero entropy diversity)
"abcdefgh"  → Pattern: all_letters
```

## HIBP Breach Integration

The engine checks passwords against known data breach databases:

```go
// password_strength_api.go
func (h *Handler) isBreachedPassword(password string) bool {
    // Checks dictionary + hardcoded top-breached list
    // Production: calls HIBP k-anonymity API
}
```

### How It Works

1. Password is checked against the local breach dictionary
2. If matched, score is forced to **0** with warning: *"This password has been found in known data breaches"*
3. In production deployments, integrate the [HIBP k-anonymity API](https://haveibeenpwned.com/API/v3) for full coverage

### HIBP k-Anonymity Flow (Production)

```
1. SHA-1 hash the password
2. Send first 5 hex chars to HIBP API
3. Receive list of breached hash suffixes
4. Check if full hash suffix is in the response
```

This ensures the plaintext password never leaves the server.

## API Endpoint

### Evaluate Password Strength

```http
POST /api/v1/auth/password/strength
Content-Type: application/json

{
  "password": "MyStr0ng!Pass"
}
```

**Response:**
```json
{
  "score": 4,
  "crack_time": "centuries",
  "guesses": 1.2e15,
  "patterns": [],
  "suggestions": [],
  "warning": ""
}
```

### Weak Password Response
```json
{
  "score": 0,
  "crack_time": "instant",
  "guesses": 10,
  "patterns": ["dictionary"],
  "suggestions": ["Add another word or two", "Use a mix of letters, numbers, and symbols"],
  "warning": "This is a commonly used password"
}
```

## Password Strength Gate

The `checkPasswordStrengthGate` function enforces minimum strength at critical points:

### Registration Flow
```go
// http.go:782 — during user registration
if ok, _ := checkPasswordStrengthGate(req.Password); !ok {
    writeError(w, http.StatusBadRequest, "password too weak")
    return
}
```

### Password Change Flow
```go
// http.go:1000 — during password change
if ok, msg := checkPasswordStrengthGate(req.NewPassword); !ok {
    writeError(w, http.StatusBadRequest, msg)
    return
}
```

### Gate Threshold

Minimum accepted score: **2** (Fair). Passwords scoring 0 or 1 are rejected.

## Policy Configuration

Administrators can configure password policy via the console:

| Setting | Default | Description |
|---------|---------|-------------|
| Minimum score | 2 | Gate threshold (0-4) |
| Max length | 128 | Maximum password length |
| Min length | 8 | Minimum password length |
| Breach check | Enabled | HIBP / local breach dictionary |
| Pattern rules | All 5 | Which patterns to enforce |

## Best Practices

1. **Set minimum score to 3** for privileged accounts (admins, service owners)
2. **Enable HIBP** in production for real breach database coverage
3. **Display suggestions** in real-time during password entry to guide users
4. **Log score (not password)** in audit trail for compliance tracking
5. **Combine with MFA** — strong passwords alone are not sufficient
6. **Consider passkeys** — WebAuthn/passkeys eliminate password weakness entirely

## Test Coverage

The password strength engine has 15 tests covering:

| Test | Validates |
|------|-----------|
| DictionaryWord | Score 0 for common dictionary passwords |
| AllDigits | Pattern detection for digit-only passwords |
| StrongPassword | Score 3-4 for complex passwords |
| KeyboardSequence | Pattern detection for qwerty-type sequences |
| Repeats | Pattern detection for repeated characters |
| L33t | Pattern detection for leet speak substitutions |
| ScoreRange | Score stays within 0-4 bounds |
| Endpoint | POST /password/strength API responds correctly |
| BreachedPassword | Score 0 for known breached passwords |
| EmptyPassword | Score 0 + warning for empty input |
| WrongMethod | 405 for non-POST requests |
| GateRejectsWeak | Gate returns false for weak passwords |
| GateAcceptsStrong | Gate returns true for strong passwords |
| CrackTimeFormat | Crack time string is human-readable |
| Suggestions | Actionable suggestions provided for weak passwords |
