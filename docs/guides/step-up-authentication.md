# Step-Up Authentication

This guide covers step-up authentication in GGID — when it's triggered, methods available, implementation patterns, and UX best practices.

## Overview

Step-up authentication requires users to provide additional verification beyond their initial login when performing sensitive operations or in high-risk contexts. This is distinct from initial MFA enrollment and is triggered dynamically.

## Trigger Conditions

### Sensitive Operation

Operations that require elevated assurance:

| Operation | Step-Up Required | Rationale |
|---|---|---|
| Change password | Yes | Account takeover prevention |
| Add/remove MFA | Yes | Security config change |
| Generate API key | Yes | Credential creation |
| Modify OAuth client | Yes | Security config |
| Tenant config change | Yes | Affects all users |
| Admin role assignment | Yes | Privilege escalation |
| Export audit data | Yes | Data exfiltration risk |
| Delete user/account | Yes | Irreversible action |
| View PII (bulk) | Yes | Privacy compliance |
| SCIM config change | Yes | Provisioning control |

### High-Risk Context

Contextual risk factors that trigger step-up:

| Factor | Threshold | Step-Up |
|---|---|---|
| New device | Never seen before | Yes |
| New IP geolocation | Different country | Yes |
| Unusual time | Outside normal hours | Optional |
| High-velocity requests | >100/min | Optional |
| Post-suspicious activity | After security alert | Yes |

### Policy Requirement

Administrators can define step-up policies:

```yaml
step_up:
  policies:
    - name: "sensitive-operations"
      operations: ["password_change", "mfa_modify", "api_key_create", "role_assign"]
      require: "mfa"
    - name: "admin-actions"
      roles: ["tenant-admin", "security-admin"]
      operations: ["tenant_config", "scim_config", "oauth_manage"]
      require: "webauthn"
    - name: "new-device"
      condition: "device_fingerprint != known"
      require: "mfa"
      cooldown: 1h  # Don't re-challenge for 1 hour
```

## Step-Up Methods

### MFA Challenge

Re-prompt for the user's registered MFA factor (TOTP, SMS, push):

```
POST /api/v1/auth/step-up
{
  "method": "totp",
  "code": "123456"
}
```

### WebAuthn Re-Authentication

Require the user to tap their security key or passkey:

```javascript
// Browser invokes WebAuthn
const assertion = await navigator.credentials.get({
  publicKey: {
    challenge: serverChallenge,
    allowCredentials: registeredCredentials,
    userVerification: "required"
  }
});
```

### Biometric

Use device biometrics (Touch ID, Face ID, Windows Hello) via WebAuthn platform authenticator.

## Implementation Patterns

### Pattern 1: Session Flag

Set a flag in the session indicating step-up completed:

```go
type Session struct {
    UserID        string
    StepUpAt      time.Time  // When step-up was completed
    StepUpMethod  string     // "totp", "webauthn", "biometric"
    StepUpExpiry  time.Time  // When step-up expires
}

func RequireStepUp(next http.HandlerFunc, maxAge time.Duration) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        session := getSession(r)
        if session.StepUpAt.IsZero() || time.Since(session.StepUpAt) > maxAge {
            // Return 403 with step-up required indicator
            writeError(w, 403, "step_up_required", map[string]interface{}{
                "available_methods": getAvailableMethods(session.UserID),
                "challenge_url": "/api/v1/auth/step-up",
            })
            return
        }
        next.ServeHTTP(w, r)
    }
}
```

### Pattern 2: Token Claim

Include step-up status in the JWT:

```json
{
  "sub": "user-uuid",
  "acr": "urn:oasis:names:tc:SAML:2.0:ac:classes:TimeSyncToken",
  "amr": ["totp", "otp"],
  "step_up_at": 1700000000,
  "step_up_exp": 1700000600
}
```

- `acr` (Authentication Context Class Reference): Indicates authentication strength
- `amr` (Authentication Methods References): Lists methods used

### Pattern 3: Challenge-Response

Issue a one-time challenge that must be signed/verified:

```
1. Server generates challenge → returns to client
2. Client proves identity (TOTP code, WebAuthn assertion)
3. Server validates → issues elevated token
4. Client uses elevated token for sensitive operation
5. Elevated token expires after use or short TTL
```

## GGID Step-Up API

### Initiate Step-Up

```bash
POST /api/v1/auth/step-up/init
Authorization: Bearer <access_token>

Response:
{
  "challenge_id": "challenge-uuid",
  "available_methods": ["totp", "webauthn"],
  "expires_in": 120
}
```

### Complete Step-Up

```bash
POST /api/v1/auth/step-up/complete
Authorization: Bearer <access_token>

{
  "challenge_id": "challenge-uuid",
  "method": "totp",
  "code": "123456"
}

Response:
{
  "step_up_token": "eyJ...",  // Short-lived elevated token
  "step_up_method": "totp",
  "expires_in": 300  // 5 minutes
}
```

### Use Elevated Token

```bash
POST /api/v1/users/me/password
Authorization: Bearer <step_up_token>

{
  "current_password": "...",
  "new_password": "..."
}
```

## Post-Step-Up Token Upgrade

After successful step-up, GGID can upgrade the existing access token:

```go
func CompleteStepUp(userID string, method string) (string, error) {
    // Verify step-up challenge
    if err := verifyChallenge(userID, method); err != nil {
        return "", err
    }

    // Upgrade token with elevated claims
    claims := getCurrentClaims(userID)
    claims["acr"] = getACRForMethod(method)
    claims["amr"] = append(claims["amr"], method)
    claims["step_up_at"] = time.Now().Unix()
    claims["step_up_exp"] = time.Now().Add(5 * time.Minute).Unix()

    // Issue new token
    return signToken(claims)
}
```

### ACR Values

| Method | ACR Value |
|---|---|
| Password only | `urn:oasis:names:tc:SAML:2.0:ac:classes:Password` |
| TOTP | `urn:oasis:names:tc:SAML:2.0:ac:classes:TimeSyncToken` |
| WebAuthn (UV) | `urn:oasis:names:tc:SAML:2.0:ac:classes:Smartcard` |
| WebAuthn (PIN) | `urn:oasis:names:tc:SAML:2.0:ac:classes:MobileOneFactorUnregistered` |
| WebAuthn (UV+PIN) | `urn:oasis:names:tc:SAML:2.0:ac:classes:MobileTwoFactorUnregistered` |

## Timeout and Retry

### Step-Up Token Lifetime

| Context | TTL | Rationale |
|---|---|---|
| Sensitive operation | 5 minutes | Short window, must re-auth for next op |
| Admin session | 15 minutes | Longer window for admin work |
| New device trust | 1 hour | Don't re-challenge immediately |

### Retry Handling

```yaml
step_up:
  max_retries: 3
  lockout_after: 5
  lockout_duration: 15m
  retry_methods: true  # Allow switching methods on retry
```

### Cooldown

After successful step-up, don't re-challenge for the same context:

```go
func NeedsStepUp(session *Session, operation string) bool {
    // Check if operation requires step-up
    if !requiresStepUp(operation) {
        return false
    }
    // Check cooldown
    if session.StepUpAt.IsZero() {
        return true
    }
    cooldown := getCooldownForOp(operation)
    return time.Since(session.StepUpAt) > cooldown
}
```

## UX Best Practices

### 1. Clear Communication

```
"This action requires additional verification.
 Please enter your authenticator code to continue."
```

### 2. Method Selection

If user has multiple MFA factors, let them choose:

```json
{
  "available_methods": [
    {"method": "totp", "name": "Authenticator App"},
    {"method": "webauthn", "name": "Security Key"},
    {"method": "sms", "name": "SMS Code"}
  ]
}
```

### 3. Minimal Disruption

- Don't step-up for read operations unless policy requires
- Cache step-up status for reasonable period
- Don't force re-auth after every single action
- Show step-up as a modal/overlay, not a full page redirect

### 4. Graceful Fallback

If the user can't complete step-up (lost device):
- Offer recovery flow
- Allow admin-assisted step-up override
- Provide clear instructions

### 5. Accessibility

- Step-up modal must be keyboard navigable
- Screen reader compatible
- Don't auto-dismiss step-up challenge
- Provide alternative methods for users with disabilities

## Security Considerations

1. **Step-up tokens are short-lived** — 5 minutes max
2. **Single-use for critical operations** — one token per operation
3. **Rate limit step-up attempts** — prevent brute force
4. **Audit all step-up events** — method, operation, success/failure
5. **Don't downgrade ACR** — once elevated, don't allow operations requiring lower ACR
6. **Bind step-up to session** — prevent token theft replay
7. **Clear step-up on logout** — don't persist across sessions