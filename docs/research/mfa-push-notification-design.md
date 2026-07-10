# Push Notification MFA Design for GGID

> Research document: design for push-based multi-factor authentication using APNs/FCM.
> Status: Proposal. Current GGID supports TOTP-only MFA.

## 1. Overview

Push MFA delivers a verification prompt to the user's mobile device via a push
notification (separate channel from the login request). The user approves or
denies the sign-in directly from their phone — no code entry required.

**Out-of-band verification.** The second factor travels through a completely
different channel (push gateway + mobile OS) than the first factor (HTTP login),
making credential interception insufficient for compromise.

**Advantages over TOTP:**
- No manual code entry — better UX, fewer transcription errors
- Rich context: can display IP address, geo-location, timestamp, application name
- No shared secret to phish (TOTP secrets can be stolen via fake login pages)

**Advantages over SMS:**
- Not vulnerable to SIM-swap attacks
- Faster delivery (sub-second vs. 5-30s SMS)
- Encrypted end-to-end via APNs/FCM TLS

**Industry adoption:** Duo Push, Okta Verify, Microsoft Authenticator, Google
Prompt — all use push-based approval as their primary second factor.

## 2. Push Notification Architecture

### Components

| Component | Role |
|-----------|------|
| Auth service | Generates push challenge, dispatches push, polls or awaits callback |
| Push gateway | APNs (Apple) or FCM (Google) — delivers notification to device |
| Mobile app | Receives push, displays approval prompt, signs and sends response |
| Redis | Stores challenge state with TTL; auth service polls for result |

### Flow

```
Browser            Auth Service       APNs/FCM           Mobile App
  |-- POST /login -->|                  |                    |
  |  (password)      |-- verify passwd   |                    |
  |                  |-- create challenge|                    |
  |                  |-- send push ----->|                    |
  |<-- 202 Pending --|                   |-- push notif ------>|
  |                  |                   |                    |-- show prompt
  |                  |<-- POST /callback ----- (signed resp)--|
  |                  |-- verify sig      |                    |
  |-- poll /status ->|                   |                    |
  |<-- 200 + tokens --|                  |                    |
```

1. User logs in with password (first factor)
2. Auth service validates credentials, detects MFA requirement
3. Generates challenge: random nonce + context (IP, geo, timestamp)
4. Sends push via APNs/FCM to user's registered device
5. Mobile app shows: "Sign-in from {IP}, {location}. Approve?"
6. User taps Approve/Deny; app sends signed response to callback
7. Auth service verifies: correct nonce, within timeout, signed by device key
8. Approved: issue JWT/session. Denied: reject + security alert.

## 3. APNs/FCM Integration

### Apple Push Notification Service (APNs)

- **Prerequisites:** Apple Developer account, APNs auth key (`.p8`, Key ID, Team ID)
- **Device token:** per-device, obtained during iOS app registration
- **Payload:** `{ "aps": { "alert": "Sign-in from 1.2.3.4", "category": "MFA" }, "challenge_id": "uuid", "callback_url": "..." }`
- **Connection:** HTTP/2 to `api.push.apple.com`, TLS mutual auth with APNs key
- **Go library:** `github.com/sideshow/apns2`

```go
func (p *APNsProvider) SendPush(ctx context.Context, deviceToken string, payload []byte) error {
    resp, err := p.client.PushWithContext(ctx, &apns2.Notification{
        DeviceToken: deviceToken, Topic: p.bundleID, Payload: payload,
    })
    if err != nil { return fmt.Errorf("apns: %w", err) }
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("apns error: %d %s", resp.StatusCode, resp.Reason)
    }
    return nil
}
```

### Firebase Cloud Messaging (FCM)

- **Prerequisites:** Firebase project, service account JSON key
- **Device token:** FCM registration token (Android/iOS/web)
- **Payload:** `{ "notification": { "title": "MFA Request" }, "data": { "challenge_id": "uuid", "callback_url": "..." } }`
- **Connection:** HTTP POST to `fcm.googleapis.com/v1/projects/{project}/messages:send` with OAuth2 bearer token
- **Go library:** `firebase.google.com/go/v4/messaging`

### Device Registration

1. User installs the GGID companion mobile app, logs in (password or first-factor)
2. App requests push permission, obtains device token from APNs/FCM
3. App generates an ECDSA P-256 key pair — private key stored in device secure
   enclave/keystore, public key registered with GGID
4. App registers push token + public key via `POST /api/v1/mfa/push/register`
5. Stored in PostgreSQL table `push_devices`:
   - `id, tenant_id, user_id` (UUID)
   - `device_token` (TEXT — APNs/FCM token)
   - `platform` (TEXT — "ios" or "android")
   - `public_key` (BYTEA — ECDSA P-256)
   - `device_name, enabled, created_at, updated_at`
   - UNIQUE constraint on `(tenant_id, device_token)`

## 4. Push Challenge/Response Protocol

### Challenge

| Field | Type | Description |
|-------|------|-------------|
| `challenge_id` | UUID | Unique per auth attempt |
| `nonce` | 32 bytes | Cryptographically random |
| `context` | JSON | `{ ip, location, timestamp, user_agent }` |
| `match_number` | int | Random 2-digit number (for number matching) |
| `expires_at` | time | 60 seconds from issuance |
| `status` | enum | `pending` / `approved` / `denied` / `expired` |

Stored in Redis under key `ggid:pushmfa:{challenge_id}` with 60s TTL.

### Response (from mobile app)

| Field | Type | Description |
|-------|------|-------------|
| `challenge_id` | UUID | Echo back the challenge ID |
| `decision` | string | `"approve"` or `"deny"` |
| `match_number` | int | Number displayed on login screen |
| `signature` | bytes | `ECDSA_Sign(device_private_key, challenge_id + decision + nonce)` |
| `timestamp` | time | When user responded |

### Verification

1. Look up challenge in Redis (must exist, not expired)
2. Verify ECDSA signature against device's registered public key
3. If number matching enabled: verify `match_number` matches challenge
4. Check `decision`: `approve` → proceed, `deny` → block
5. Delete challenge from Redis (single-use, prevents replay)

### Go Code Sketch

```go
type PushChallenge struct {
    ChallengeID string    `json:"challenge_id"`
    Nonce       string    `json:"nonce"`
    MatchNumber int       `json:"match_number"`
    Context     Context   `json:"context"`
    ExpiresAt   time.Time `json:"expires_at"`
}

// SignResponse signs with the device's ECDSA private key.
func SignResponse(priv *ecdsa.PrivateKey, challengeID, decision, nonce string) ([]byte, error) {
    hash := sha256.Sum256([]byte(challengeID + decision + nonce))
    return ecdsa.SignASN1(rand.Reader, priv, hash[:])
}

// VerifyResponse validates the signed response from the mobile app.
func VerifyResponse(c *PushChallenge, decision string, matchNum int, sig []byte, pub *ecdsa.PublicKey) error {
    if time.Now().After(c.ExpiresAt)         { return ErrChallengeExpired }
    if matchNum != c.MatchNumber             { return ErrNumberMismatch }
    hash := sha256.Sum256([]byte(c.ChallengeID + decision + c.Nonce))
    if !ecdsa.VerifyASN1(pub, hash[:], sig)  { return ErrInvalidSignature }
    return nil
}
```

## 5. Security Considerations

### Push Fatigue Attack

**Attack:** Repeatedly trigger MFA prompts until the user approves out of
frustration or muscle memory.

**Mitigation:** Rate-limit push challenges (max 3 per 5-minute window per user),
display a running count ("This is your 4th request today"), require long-press
instead of tap. CISA recommends number matching as the primary defense.

### Response Spoofing

**Attack:** Craft a fake approval response and POST it to the callback endpoint.

**Prevention:** ECDSA device-key signing binds the response to the specific
challenge (challenge_id + decision + nonce). Without the device's private key
(stored in secure enclave), an attacker cannot forge a valid signature.

### Device Compromise

**Attack:** Attacker has physical possession of the user's phone.

**Mitigation:** Require biometric unlock (Face ID / Touch ID / fingerprint)
before displaying the approval prompt. Number matching adds a second interactive
step. Optional: geofencing (reject if device is far from login IP location).

### Number Matching (2024 Best Practice)

CISA and Microsoft Entra ID now recommend number matching as a mandatory
push MFA feature:

1. Auth service generates a random 2-digit number (e.g., `47`)
2. Login page displays the number to the user
3. Push notification asks user to enter the matching number (not a simple tap)
4. Prevents push fatigue: cannot blindly approve without seeing the number

**Adopted by:** Microsoft Authenticator (mandatory since 2023), Duo (2023),
Okta Verify (2023). GGID should implement this from day one.

### Notification Interception

- Push payload must NOT include sensitive data — only challenge reference (UUID)
- Callback endpoint requires signature verification (no unsigned callbacks accepted)
- Optional: mutual TLS between mobile app and auth service callback

## 6. Timeout and Fallback

| Scenario | Action |
|----------|--------|
| Push timeout (60s) | Offer fallback: TOTP, SMS OTP, backup codes |
| User denies | Immediate rejection + security alert event |
| No response | Never auto-approve (user may be asleep or phone off) |
| Device unreachable | Retry once after 10s, then fall back |

### Go: Timeout Handling

```go
func (s *PushMFAService) WaitForApproval(ctx context.Context, challengeID string, timeout time.Duration) (*domain.TokenSet, error) {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return nil, ErrMFATimedOut
        case <-ticker.C:
            status := s.getChallengeStatus(challengeID)
            switch status {
            case "approved":
                return s.issueTokens(challengeID)
            case "denied":
                return nil, ErrMFADenied
            }
        }
    }
}
```

Alternatively, use a Redis pub/sub channel or WebSocket so the auth handler
receives a push event immediately instead of polling.

## 7. GGID Architecture

### Current State

GGID's `MFAService` (`mfa_service.go`) is TOTP-only — methods: `SetupMFA`,
`VerifyMFA`, `VerifyUserCode`, `DisableMFA`, `HasMFAEnabled`, `ListDevices`.
The login flow checks `HasMFAEnabled()` and returns `TokenSet{MFARequired: true}`.
The client calls `LoginMFA()` with a TOTP code.

| Feature | Current | Needed |
|---------|---------|--------|
| MFA method | TOTP | + Push |
| Device storage | `mfa_devices` | + `push_devices` |
| Challenge storage | Redis | Redis (same pattern) |
| Mobile app | None | Companion app |
| Push gateway | None | APNs + FCM |

### Proposed Architecture

New package: `pkg/pushmfa/`

```go
// PushProvider is the interface for platform-specific push delivery.
type PushProvider interface {
    SendPush(ctx context.Context, deviceToken string, payload []byte) error
    Name() string // "apns" or "fcm"
}

// PushMFAService manages the push MFA lifecycle.
type PushMFAService struct {
    rdb        *redis.Client
    providers  map[string]PushProvider // keyed by platform: "ios", "android"
    deviceRepo PushDeviceRepository
    timeout    time.Duration
}

// InitiatePush creates a challenge and dispatches push to the user's device.
func (s *PushMFAService) InitiatePush(ctx context.Context, tenantID, userID uuid.UUID, loginCtx Context) (*PushChallenge, error)

// HandleCallback processes the signed response from the mobile app.
func (s *PushMFAService) HandleCallback(ctx context.Context, resp PushResponse) error

// CheckStatus returns the current status of a push challenge.
func (s *PushMFAService) CheckStatus(ctx context.Context, challengeID string) (string, error)
```

**Integration with existing auth flow:** In `AuthService.Login()`, after
password validation, check for push devices before falling back to TOTP:

```go
if s.pushMFA.HasPushDevice(ctx, tc.TenantID, userID) {
    ch, _ := s.pushMFA.InitiatePush(ctx, tc.TenantID, userID, Context{IP: ip, UserAgent: userAgent})
    return &domain.TokenSet{MFARequired: true, MFAChallenge: ch.ChallengeID, MFAMethod: "push"}, nil
}
// Fall back to TOTP
return &domain.TokenSet{MFARequired: true, MFAChallenge: totpChallenge}, nil
```

**New API endpoints:**

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/api/v1/mfa/push/register` | Register device token + public key |
| POST | `/api/v1/mfa/push/callback` | Mobile app submits signed approval/denial |
| GET | `/api/v1/mfa/push/status/{challenge_id}` | Client polls for approval status |
| DELETE | `/api/v1/mfa/push/devices/{id}` | Unregister a device |

**Gateway:** No changes needed. Push MFA is async — the auth handler manages
the challenge lifecycle. The callback endpoint is registered on the auth service
directly.

**Config (per-tenant):**

```yaml
push_mfa:
  enabled: false              # per-tenant toggle
  timeout: 60s                # challenge TTL
  fallback_chain: [push, totp, sms]  # if push fails, try next
  number_matching: true       # enable number matching
  rate_limit: 3/5m            # max 3 pushes per 5 minutes
```

## 8. Comparison with Other Push MFA Solutions

| Feature | Duo Push | Okta Verify | MS Authenticator | GGID (proposed) |
|---------|----------|-------------|------------------|-----------------|
| Platform | iOS + Android | iOS + Android | iOS + Android | iOS + Android |
| Biometric unlock | Yes | Yes | Yes | Yes (app-enforced) |
| Number matching | Yes | Yes | Yes (mandatory) | Yes (Phase 4) |
| Context display | IP + app + geo | IP + app | IP + app | IP + geo + UA |
| Offline fallback | Passcode | TOTP | TOTP | TOTP (existing) |
| SDK for integration | Yes | Yes | MS Graph API | REST API |
| Open source | No | No | No | Yes (Apache 2.0) |

## 9. Roadmap

| Phase | Scope | Effort |
|-------|-------|--------|
| 1 | Push challenge/response protocol + device registration API + Redis storage | ~1 week |
| 2 | FCM integration (Android push delivery) | 3-5 days |
| 3 | APNs integration (iOS push delivery) | 3-5 days |
| 4 | Number matching + push fatigue rate limiting + biometric enforcement | 2-3 days |
| 5 | GGID companion mobile app (React Native or native — separate project) | 4-6 weeks |

**Critical note:** Push MFA requires a companion mobile app — it cannot work
with third-party authenticator apps (Google Authenticator, Authy, etc.) because
those apps do not receive push notifications or have callback capability. The
mobile app is the largest work item and should be started in parallel with
Phase 1.

**Dependency chain:** Phases 1-3 can proceed in parallel with mobile app
development. Phase 4 depends on Phase 1. Phase 5 is required for end-to-end
functionality.
