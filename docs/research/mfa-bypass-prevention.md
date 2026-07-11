# MFA Bypass Prevention for IAM Systems

> **Scope**: Attack vectors that bypass multi-factor authentication and their
> specific mitigations. This document complements — but does not duplicate —
> [`adaptive-mfa-design.md`](./adaptive-mfa-design.md) (risk scoring, step-up
> design) and [`mfa-push-notification-design.md`](./mfa-push-notification-design.md)
> (push architecture, push fatigue theory).
>
> **Audience**: GGID platform engineers, security reviewers, DevSecOps.
>
> **Threat model**: An attacker has obtained or can guess the user's primary
> credential (password) and now targets the second factor.

---

## Table of Contents

1. [Session Token Before MFA (Half-Authenticated State)](#1-session-token-before-mfa-half-authenticated-state)
2. [Push Bombing (MFA Fatigue)](#2-push-bombing-mfa-fatigue)
3. [SIM Swap Attacks](#3-sim-swap-attacks)
4. [Backup Code Rate Limiting](#4-backup-code-rate-limiting)
5. [MFA Bypass via Social Engineering](#5-mfa-bypass-via-social-engineering)
6. [Token Theft Post-MFA](#6-token-theft-post-mfa)
7. [Downgrade Attacks](#7-downgrade-attacks)
8. [MFA Status Manipulation](#8-mfa-status-manipulation)
9. [GGID MFA Bypass Surface Analysis](#9-ggid-mfa-bypass-surface-analysis)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. Session Token Before MFA (Half-Authenticated State)

### Attack Vector

The most dangerous MFA bypass occurs when the authentication system issues a
real session token (JWT, session cookie, refresh token) **before** the MFA
challenge is completed. If this happens, MFA becomes optional — an attacker
with the password can simply ignore the MFA challenge and use the session
token directly.

```
Client ── POST /login {username, password} ──▶ Server
Server ── 200 {access_token, refresh_token, mfa_required: true} ──▶ Client
                                                                 │
                                        Attacker ignores MFA prompt
                                        and uses access_token directly
```

Even if the token has `mfa_verified: false` in its claims, the server may not
enforce this flag on every request, especially for high-traffic API endpoints
where middleware only checks token validity and expiry.

### Mitigation: Pre-Auth Session State

The system must maintain an explicit **half-authenticated** state that:

1. **Issues no real session token** — only a short-lived, opaque challenge token.
2. **Stores the challenge server-side** (Redis) with the user ID and a strict TTL.
3. **Requires the challenge token + valid MFA code** to obtain the real token set.
4. **Rejects challenge tokens** at normal API endpoints.

```go
// PreMFASession represents the half-authenticated state between password
// verification and MFA completion. No real session token is issued.
type PreMFASession struct {
    ChallengeToken string    // opaque, random — NOT a JWT
    TenantID       uuid.UUID
    UserID         uuid.UUID
    DeviceID       uuid.UUID
    IssuedAt       time.Time
    ExpiresAt      time.Time // 5 minutes
    AttemptCount   int       // track MFA submission attempts
}

const preMFATTL = 5 * time.Minute

// IssueMFAChallenge is called after password verification succeeds but
// BEFORE any real session or token is created.
func (s *AuthService) IssueMFAChallenge(
    ctx context.Context, tenantID, userID, deviceID uuid.UUID,
) (*PreMFASession, error) {
    // Generate a cryptographically random challenge token.
    // This is NOT a JWT — it carries no claims usable at API endpoints.
    raw, err := crypto.GenerateRandomToken(32)
    if err != nil {
        return nil, fmt.Errorf("generate challenge: %w", err)
    }

    sess := &PreMFASession{
        ChallengeToken: raw,
        TenantID:       tenantID,
        UserID:         userID,
        DeviceID:       deviceID,
        IssuedAt:       time.Now(),
        ExpiresAt:      time.Now().Add(preMFATTL),
    }

    // Store server-side so the challenge can be verified and consumed.
    key := fmt.Sprintf("ggid:mfa_challenge:%s", hashToken(raw))
    val := fmt.Sprintf("%s:%s:%s", tenantID, userID, deviceID)
    if err := s.rdb.Set(ctx, key, val, preMFATTL).Err(); err != nil {
        return nil, fmt.Errorf("store mfa challenge: %w", err)
    }

    return sess, nil
}

// CompleteMFAChallenge verifies the MFA code and ONLY THEN issues real tokens.
func (s *AuthService) CompleteMFAChallenge(
    ctx context.Context, challengeToken, mfaCode, ip, userAgent string,
) (*domain.TokenSet, error) {
    // 1. Look up the challenge in Redis — server-side state.
    key := fmt.Sprintf("ggid:mfa_challenge:%s", hashToken(challengeToken))
    val, err := s.rdb.Get(ctx, key).Result()
    if err != nil {
        return nil, ErrInvalidMFACode // challenge expired or never issued
    }

    parts := strings.SplitN(val, ":", 3)
    if len(parts) != 3 {
        s.rdb.Del(ctx, key)
        return nil, ErrInvalidMFACode
    }
    tenantID, _ := uuid.Parse(parts[0])
    userID, _ := uuid.Parse(parts[1])
    deviceID, _ := uuid.Parse(parts[2])

    // 2. Rate-limit MFA attempts within the challenge window.
    attemptKey := fmt.Sprintf("ggid:mfa_attempts:%s:%s", tenantID, userID)
    count, _ := s.rdb.Incr(ctx, attemptKey).Result()
    if count == 1 {
        s.rdb.Expire(ctx, attemptKey, preMFATTL)
    }
    if count > 5 {
        s.rdb.Del(ctx, key) // invalidate challenge on brute-force
        return nil, ErrMFAAttemptsExceeded
    }

    // 3. Verify the TOTP code.
    if err := s.mfaService.VerifyUserCode(ctx, tenantID, userID, mfaCode); err != nil {
        return nil, ErrInvalidMFACode
    }

    // 4. Consume the challenge (single-use).
    s.rdb.Del(ctx, key)

    // 5. NOW issue the real session and tokens.
    _, session, err := s.sessionService.Create(ctx, CreateSessionParams{
        TenantID:  tenantID,
        UserID:    userID,
        IPAddress: ip,
        UserAgent: userAgent,
        TTL:       24 * time.Hour,
    })
    if err != nil {
        return nil, fmt.Errorf("create session: %w", err)
    }

    accessToken, expiresIn, err := s.tokenService.IssueAccessToken(tenantID, userID)
    if err != nil {
        return nil, err
    }

    refreshToken, err := s.tokenService.IssueRefreshToken(ctx, tenantID, userID, session.ID)
    if err != nil {
        return nil, err
    }

    return &domain.TokenSet{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        TokenType:    "Bearer",
        ExpiresIn:    expiresIn,
        SessionID:    session.ID.String(),
    }, nil
}
```

### Key Design Rules

| Rule | Rationale |
|------|-----------|
| No JWT/refresh token in the MFA-required response | Prevents direct API access |
| Challenge token is opaque, stored server-side | Cannot be decoded to extract user info |
| Challenge has 5-minute TTL | Limits replay window |
| Max 5 MFA attempts per challenge | Prevents TOTP brute-force (6 digits = 1M combos) |
| Challenge is single-use (deleted on success) | Prevents token replay |

---

## 2. Push Bombing (MFA Fatigue)

### Attack Vector

When the attacker has the user's password, they can repeatedly trigger MFA
push notifications. The user, overwhelmed or annoyed, may approve a push
just to stop the notifications — especially if pushes arrive at 3 AM.

```
Attacker ── login(user, known_password) ──▶ Server ── push ──▶ Victim phone
Attacker ── login(user, known_password) ──▶ Server ── push ──▶ Victim phone
Attacker ── login(user, known_password) ──▶ Server ── push ──▶ Victim phone
                                                                          │
                                              Victim approves to stop the noise
```

Real-world incidents: Uber breach (2022), Cisco breach (2022) — both via MFA
fatigue.

### Mitigation: Rate Limiting + Number Matching + Context Display

```go
// PushRateLimiter prevents MFA push bombing by enforcing:
//   - Max 3 pushes per user per hour
//   - Max 10 pushes per user per day
//   - Minimum 30-second gap between pushes
type PushRateLimiter struct {
    rdb *redis.Client
}

const (
    pushHourlyLimit  = 3
    pushDailyLimit   = 10
    pushCooldownSec  = 30
)

func (p *PushRateLimiter) CanSendPush(ctx context.Context, tenantID, userID uuid.UUID) error {
    // Cooldown check — prevent rapid-fire pushes.
    cooldownKey := fmt.Sprintf("mfa_push_cooldown:%s:%s", tenantID, userID)
    set, err := p.rdb.SetNX(ctx, cooldownKey, "1", pushCooldownSec*time.Second).Result()
    if err != nil {
        return fmt.Errorf("cooldown check: %w", err)
    }
    if !set {
        return ErrPushCooldown
    }

    // Hourly limit.
    hourlyKey := fmt.Sprintf("mfa_push_hourly:%s:%s", tenantID, userID)
    hourlyCount, err := p.rdb.Incr(ctx, hourlyKey).Result()
    if err != nil {
        return err
    }
    if hourlyCount == 1 {
        p.rdb.Expire(ctx, hourlyKey, time.Hour)
    }
    if hourlyCount > pushHourlyLimit {
        return ErrPushRateExceeded
    }

    // Daily limit.
    dailyKey := fmt.Sprintf("mfa_push_daily:%s:%s", tenantID, userID)
    dailyCount, err := p.rdb.Incr(ctx, dailyKey).Result()
    if err != nil {
        return err
    }
    if dailyCount == 1 {
        p.rdb.Expire(ctx, dailyKey, 24*time.Hour)
    }
    if dailyCount > pushDailyLimit {
        return ErrPushRateExceeded
    }

    return nil
}

// PushChallenge includes number matching — the user must select the matching
// number shown on their device, preventing blind approval.
type PushChallenge struct {
    ChallengeID    string
    NumberToMatch  int  // 2-digit number user must enter on their device
    ContextInfo    PushContext
    ExpiresAt      time.Time
}

type PushContext struct {
    AppName    string // "GGID Console"
    IPAddress  string
    Location   string // geocoded from IP
    DeviceOS   string // parsed from User-Agent
    Timestamp  time.Time
}

// IssuePushChallenge generates a number-matching push challenge.
func (s *AuthService) IssuePushChallenge(
    ctx context.Context, tenantID, userID uuid.UUID, ip, userAgent string,
) (*PushChallenge, error) {
    // Rate-limit push frequency.
    if err := s.pushLimiter.CanSendPush(ctx, tenantID, userID); err != nil {
        return nil, err
    }

    // Generate random 2-digit number for matching.
    var numBuf [1]byte
    if _, err := crypto_rand.Read(numBuf[:]); err != nil {
        return nil, err
    }
    matchNum := int(numBuf[0])%90 + 10 // 10–99

    challenge := &PushChallenge{
        ChallengeID:   crypto.GenerateRandomTokenSafe(32),
        NumberToMatch: matchNum,
        ContextInfo: PushContext{
            AppName:   "GGID Console",
            IPAddress: ip,
            DeviceOS:  parseUserAgent(userAgent),
            Timestamp: time.Now(),
        },
        ExpiresAt: time.Now().Add(90 * time.Second),
    }

    // Store server-side for verification.
    key := fmt.Sprintf("mfa_push_challenge:%s", challenge.ChallengeID)
    val := fmt.Sprintf("%s:%s:%d", tenantID, userID, matchNum)
    s.rdb.Set(ctx, key, val, 90*time.Second)

    // Deliver push to user's registered device (FCM/APNs).
    s.pushGateway.Send(ctx, userID, PushPayload{
        ChallengeID:   challenge.ChallengeID,
        NumberToMatch: matchNum,
        Context:       challenge.ContextInfo,
    })

    return challenge, nil
}

// VerifyPushResponse validates that the user selected the correct number.
func (s *AuthService) VerifyPushResponse(
    ctx context.Context, challengeID string, selectedNumber int,
) (uuid.UUID, uuid.UUID, error) {
    key := fmt.Sprintf("mfa_push_challenge:%s", challengeID)
    val, err := s.rdb.Get(ctx, key).Result()
    if err != nil {
        return uuid.Nil, uuid.Nil, ErrInvalidPushChallenge
    }

    parts := strings.SplitN(val, ":", 3)
    if len(parts) != 3 {
        s.rdb.Del(ctx, key)
        return uuid.Nil, uuid.Nil, ErrInvalidPushChallenge
    }

    tenantID, _ := uuid.Parse(parts[0])
    userID, _ := uuid.Parse(parts[1])
    expectedNum, _ := strconv.Atoi(parts[2])

    // Number must match — prevents blind "approve" taps.
    if selectedNumber != expectedNum {
        s.rdb.Del(ctx, key)
        return uuid.Nil, uuid.Nil, ErrPushNumberMismatch
    }

    s.rdb.Del(ctx, key) // single-use
    return tenantID, userID, nil
}
```

### Defense-in-Depth Layers

| Layer | Mechanism |
|-------|-----------|
| Rate limit | Max 3 pushes/hour, 10/day per user |
| Cooldown | 30-second minimum between pushes |
| Number matching | User must enter the number shown on screen |
| Context display | Show IP, location, OS in the push notification |
| Auto-disable | After 3 rejected pushes, force TOTP fallback |

---

## 3. SIM Swap Attacks

### Attack Vector

SIM swapping (also called SIM hijacking or port-out scam) is a social
engineering attack where the attacker convinces a mobile carrier to port the
victim's phone number to a SIM card the attacker controls. Once successful,
all SMS messages and phone calls — including OTP codes — are redirected to
the attacker.

```
Attacker ── social-engineers carrier ──▶ Carrier ports number
Attacker ── requests SMS OTP ──▶ Server sends SMS ──▶ Attacker's phone
Attacker ── submits OTP ──▶ Authentication complete
```

This attack is particularly dangerous because it is **invisible to the victim**
until they notice loss of cellular service.

### Detection Signals

| Signal | Description |
|--------|-------------|
| Recent port-out | Phone number was ported within last 72 hours |
| Carrier change | Number moved from one carrier to another |
| IMSI change | SIM serial number changed unexpectedly |
| Geo-mismatch | SMS OTP requested from IP in a different country than phone registration |

### Defense: Prefer TOTP/WebAuthn + Risk-Based SMS Gating

```go
// SMSRiskAssessor evaluates whether an SMS OTP request is safe.
type SMSRiskAssessor struct {
    rdb               *redis.Client
    carrierCheckAPI   CarrierCheckAPI // third-party or internal
}

// SMSSendDecision determines whether to send an SMS OTP or require a
// stronger factor.
type SMSSendDecision struct {
    Allow          bool
    Reason         string
    RequireFallback bool  // force TOTP/WebAuthn instead
}

func (a *SMSRiskAssessor) Assess(
    ctx context.Context, tenantID, userID uuid.UUID, phone, ip string,
) (*SMSSendDecision, error) {
    // 1. Check for recent SIM swap / port-out events.
    portInfo, err := a.carrierCheckAPI.CheckPortHistory(ctx, phone)
    if err == nil && portInfo.RecentlyPorted {
        // If ported within last 72 hours, treat as high-risk.
        if time.Since(portInfo.PortDate) < 72*time.Hour {
            return &SMSSendDecision{
                Allow:          false,
                Reason:         "SIM recently ported — use TOTP or WebAuthn",
                RequireFallback: true,
            }, nil
        }
    }

    // 2. Check for known SIM swap indicators in our own history.
    swapKey := fmt.Sprintf("sim_swap_flag:%s:%s", tenantID, userID)
    if flagged, _ := a.rdb.Get(ctx, swapKey).Bool(); flagged {
        return &SMSSendDecision{
            Allow:          false,
            Reason:         "Account flagged for SIM swap risk",
            RequireFallback: true,
        }, nil
    }

    // 3. Check geo-mismatch: is the requesting IP in a different country
    //    than the phone number's registered country?
    ipCountry := geoipLookup(ip)
    phoneCountry := parsePhoneCountryCode(phone)
    if ipCountry != "" && phoneCountry != "" && ipCountry != phoneCountry {
        // Log suspicious activity for risk analysis.
        a.flagAnomaly(ctx, tenantID, userID, "geo_mismatch",
            fmt.Sprintf("IP country %s != phone country %s", ipCountry, phoneCountry))
        // Still allow but require step-up — don't outright block.
        return &SMSSendDecision{
            Allow:          true,
            Reason:         "Geo-mismatch detected — step-up required",
            RequireFallback: true,
        }, nil
    }

    return &SMSSendDecision{Allow: true}, nil
}

// FlagSIMSwap is called by a webhook from the carrier fraud detection system
// or by an internal monitoring job that detects unusual SIM activity.
func (a *SMSRiskAssessor) FlagSIMSwap(
    ctx context.Context, tenantID, userID uuid.UUID,
) error {
    key := fmt.Sprintf("sim_swap_flag:%s:%s", tenantID, userID)
    // Flag for 7 days, then require re-verification.
    return a.rdb.Set(ctx, key, "1", 7*24*time.Hour).Err()
}

// EnforceFactorPreference ensures SMS is only used when the user has no
// stronger factor enrolled, or when explicitly chosen.
func (s *AuthService) SelectMFAMethod(
    ctx context.Context, tenantID, userID uuid.UUID,
) (string, error) {
    devices, err := s.mfaService.ListDevices(ctx, userID)
    if err != nil {
        return "", err
    }

    // Priority: WebAuthn > TOTP > SMS
    // If user has TOTP or WebAuthn, never fall back to SMS automatically.
    for _, d := range devices {
        if d.Type == "webauthn" {
            return "webauthn", nil
        }
        if d.Type == "totp" {
            return "totp", nil
        }
    }

    // Only SMS available — apply risk assessment.
    decision, err := s.smsAssessor.Assess(ctx, tenantID, userID,
        s.getPhoneForUser(ctx, userID), s.getClientIP(ctx))
    if err != nil || !decision.Allow {
        return "", ErrMFAUnavailable
    }

    return "sms", nil
}
```

### Policy Recommendations

- **Never use SMS as the sole MFA factor** for privileged accounts.
- **Deprecate SMS enrollment** for new users; offer only TOTP or WebAuthn.
- **Allow existing SMS users to migrate** to TOTP with a guided workflow.
- **Log all SMS OTP sends** with IP, timestamp, and carrier status for audit.

---

## 4. Backup Code Rate Limiting

### Attack Vector

Backup (recovery) codes are single-use codes generated during MFA enrollment
for use when the primary factor is unavailable. They are **weaker** than TOTP
because:

- They are static (don't change every 30 seconds).
- They are often stored insecurely (screenshots, plaintext files).
- They bypass device-based MFA, so device-based rate limiting doesn't apply.

An attacker who has stolen backup codes (via phishing, malware, or a leaked
screenshot) can use them to authenticate without the TOTP device.

### Mitigation: Strict Rate Limiting + Single-Use Enforcement + Lockout

```go
// BackupCodeService manages recovery codes with strict security controls.
type BackupCodeService struct {
    rdb  *redis.Client
    repo BackupCodeRepository
}

const (
    backupCodeMaxAttempts = 5       // lockout after 5 failed attempts
    backupCodeLockoutTTL  = 1 * time.Hour
    backupCodeSetTTL      = 90 * 24 * time.Hour // codes expire after 90 days
)

// ValidateBackupCode verifies a recovery code with full rate limiting.
func (s *BackupCodeService) ValidateBackupCode(
    ctx context.Context, tenantID, userID uuid.UUID, code string,
) error {
    // 1. Check if user is locked out from backup code attempts.
    lockoutKey := fmt.Sprintf("backup_lockout:%s:%s", tenantID, userID)
    locked, _ := s.rdb.Get(ctx, lockoutKey).Bool()
    if locked {
        return ErrBackupCodeLocked
    }

    // 2. Increment attempt counter.
    attemptKey := fmt.Sprintf("backup_attempts:%s:%s", tenantID, userID)
    attempts, err := s.rdb.Incr(ctx, attemptKey).Result()
    if err != nil {
        return fmt.Errorf("track attempts: %w", err)
    }
    if attempts == 1 {
        s.rdb.Expire(ctx, attemptKey, 15*time.Minute)
    }

    // 3. Check lockout threshold.
    if attempts > backupCodeMaxAttempts {
        s.rdb.Set(ctx, lockoutKey, "1", backupCodeLockoutTTL)
        s.rdb.Del(ctx, attemptKey)
        // Invalidate all backup codes on repeated failures.
        s.repo.InvalidateAllCodes(ctx, tenantID, userID)
        return ErrBackupCodeLocked
    }

    // 4. Look up the code by hash (codes are stored hashed, not plaintext).
    codeHash := hashToken(code)
    stored, err := s.repo.FindByHash(ctx, tenantID, userID, codeHash)
    if err != nil {
        return ErrInvalidBackupCode
    }
    if stored == nil {
        return ErrInvalidBackupCode
    }

    // 5. Verify the code has not been used (single-use enforcement).
    if stored.Used {
        // A used code was submitted — this is highly suspicious.
        // Invalidate ALL codes to protect the account.
        s.repo.InvalidateAllCodes(ctx, tenantID, userID)
        return ErrBackupCodeAlreadyUsed
    }

    // 6. Atomically mark the code as used. Use a conditional update to
    //    prevent race conditions where two requests use the same code.
    affected, err := s.repo.MarkUsed(ctx, tenantID, userID, stored.ID)
    if err != nil {
        return fmt.Errorf("mark code used: %w", err)
    }
    if affected == 0 {
        // Another request beat us to it — race condition prevented.
        return ErrBackupCodeAlreadyUsed
    }

    // 7. Clear the attempt counter on success.
    s.rdb.Del(ctx, attemptKey)

    // 8. Audit log: backup code usage is a significant security event.
    s.auditLog(ctx, tenantID, userID, "backup_code_used", map[string]any{
        "code_id":    stored.ID,
        "ip":         s.getClientIP(ctx),
        "user_agent": s.getUserAgent(ctx),
    })

    return nil
}

// GenerateBackupCodes creates a fresh set of recovery codes.
// All previously generated codes are invalidated.
func (s *BackupCodeService) GenerateBackupCodes(
    ctx context.Context, tenantID, userID uuid.UUID,
) ([]string, error) {
    // Invalidate old codes first.
    if err := s.repo.InvalidateAllCodes(ctx, tenantID, userID); err != nil {
        return nil, err
    }

    var codes []string
    var records []*BackupCodeRecord

    for i := 0; i < 10; i++ {
        // Generate human-readable code: XXXX-XXXX-XXXX format.
        code := generateHumanCode()

        record := &BackupCodeRecord{
            ID:        uuid.New(),
            TenantID:  tenantID,
            UserID:    userID,
            CodeHash:  hashToken(code),
            Used:      false,
            CreatedAt: time.Now(),
            ExpiresAt: time.Now().Add(backupCodeSetTTL),
        }
        records = append(records, record)
        codes = append(codes, code)
    }

    if err := s.repo.BatchCreate(ctx, records); err != nil {
        return nil, err
    }

    return codes, nil
}

func generateHumanCode() string {
    // Use crypto/rand for secure generation.
    var buf [6]byte
    if _, err := crypto_rand.Read(buf[:]); err != nil {
        panic(err)
    }
    // Encode to base32 (no ambiguous chars: 0, O, I, 1).
    const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
    b := make([]byte, 6)
    for i, v := range buf {
        b[i] = alphabet[int(v)%len(alphabet)]
    }
    return fmt.Sprintf("%s-%s-%s", b[0:2], b[2:4], b[4:6])
}
```

### Security Controls Summary

| Control | Value | Rationale |
|---------|-------|-----------|
| Max attempts | 5 per 15 min | Prevents brute-force (8-char code = 32^8 space) |
| Lockout duration | 1 hour | Slows automated attacks |
| Single-use | Atomic UPDATE...WHERE used=false | Prevents replay via race condition |
| Invalidation on reuse attempt | All codes invalidated | Reused code = compromise indicator |
| Code format | Base32, 6 chars, hyphenated | Human-readable, no ambiguous characters |
| Code lifetime | 90 days | Forces regeneration, limits exposure window |

---

## 5. MFA Bypass via Social Engineering

### Attack Vector

The weakest link in any MFA system is the **administrative override**. An
attacker calls the help desk, impersonates the user, and convinces the agent
to reset or disable MFA. Once MFA is removed, the attacker authenticates with
just the password.

```
Attacker ── calls help desk ──▶ "I lost my phone, please reset my MFA"
Help Desk ── disables MFA ──▶ Account now has single-factor auth
Attacker ── login(user, known_password) ──▶ Full access
```

This attack is particularly effective because help desk agents are trained
to be helpful, not suspicious.

### Mitigation: Verified Reset Workflow with Cooling-Off Period

```go
// MFAResetWorkflow implements a secure, multi-step MFA reset process.
type MFAResetWorkflow struct {
    rdb        *redis.Client
    mfaService *MFAService
    notifiers  NotificationService
}

const (
    resetVerificationDelay = 24 * time.Hour  // cooling-off period
    resetTokenTTL          = 48 * time.Hour
)

// InitiateMFAReset starts the MFA reset process. The actual reset does NOT
// happen immediately — it is delayed and verified.
func (w *MFAResetWorkflow) InitiateMFAReset(
    ctx context.Context, tenantID, userID uuid.UUID, requestedBy AdminID,
    verificationMethod string, // "email" or "manager_approval"
) (*MFAResetRequest, error) {
    // 1. Check for existing pending reset (prevent repeated requests).
    pendingKey := fmt.Sprintf("mfa_reset_pending:%s:%s", tenantID, userID)
    if existing, _ := w.rdb.Get(ctx, pendingKey).Result(); existing != "" {
        return nil, ErrMFAResetAlreadyPending
    }

    // 2. Create a reset request with a cooling-off period.
    resetToken, err := crypto.GenerateRandomToken(32)
    if err != nil {
        return nil, err
    }

    req := &MFAResetRequest{
        ID:                uuid.New(),
        TenantID:          tenantID,
        UserID:            userID,
        RequestedBy:       requestedBy,
        VerificationMethod: verificationMethod,
        RequestedAt:       time.Now(),
        EffectiveAt:       time.Now().Add(resetVerificationDelay),
        Status:            "pending_verification",
    }

    // 3. Store the reset request.
    reqKey := fmt.Sprintf("mfa_reset:%s", resetToken)
    data, _ := json.Marshal(req)
    w.rdb.Set(ctx, reqKey, data, resetTokenTTL)
    w.rdb.Set(ctx, pendingKey, resetToken, resetTokenTTL)

    // 4. Send notifications through MULTIPLE channels.
    //    Email: "An MFA reset has been requested. If you did not request
    //           this, contact security immediately."
    w.notifiers.SendSecurityAlert(ctx, userID, SecurityAlert{
        Type:    "mfa_reset_requested",
        Message: "An MFA reset has been requested for your account.",
        Action:  "If this was not you, click here to cancel.",
    })

    // 5. If manager approval required, notify the manager.
    if verificationMethod == "manager_approval" {
        manager := w.getManagerForUser(ctx, userID)
        w.notifiers.SendApprovalRequest(ctx, manager, ApprovalRequest{
            Type:      "mfa_reset",
            UserID:    userID,
            ExpiresAt: req.EffectiveAt,
        })
    }

    return req, nil
}

// CompleteMFAReset executes the MFA reset after the cooling-off period
// and verification.
func (w *MFAResetWorkflow) CompleteMFAReset(
    ctx context.Context, resetToken string, approver AdminID,
) error {
    reqKey := fmt.Sprintf("mfa_reset:%s", resetToken)
    data, err := w.rdb.Get(ctx, reqKey).Result()
    if err != nil {
        return ErrInvalidResetToken
    }

    var req MFAResetRequest
    if err := json.Unmarshal([]byte(data), &req); err != nil {
        return ErrInvalidResetToken
    }

    // 1. Verify the cooling-off period has elapsed.
    if time.Now().Before(req.EffectiveAt) {
        remaining := req.EffectiveAt.Sub(time.Now())
        return fmt.Errorf("cooling-off period not elapsed: %s remaining", remaining.Round(time.Hour))
    }

    // 2. Verify approval (if manager approval was required).
    if req.VerificationMethod == "manager_approval" {
        if !w.isApproved(ctx, req.ID, approver) {
            return ErrMFAResetNotApproved
        }
    }

    // 3. Revoke ALL active sessions before resetting MFA.
    //    This ensures the attacker can't maintain access.
    w.sessionService.RevokeAllForUser(ctx, req.TenantID, req.UserID, uuid.Nil)

    // 4. Disable MFA and force re-enrollment on next login.
    w.mfaService.ForceReEnrollment(ctx, req.TenantID, req.UserID)

    // 5. Clean up.
    w.rdb.Del(ctx, reqKey)
    pendingKey := fmt.Sprintf("mfa_reset_pending:%s:%s", req.TenantID, req.UserID)
    w.rdb.Del(ctx, pendingKey)

    // 6. Audit log.
    w.auditLog(ctx, req.TenantID, req.UserID, "mfa_reset_completed", map[string]any{
        "requested_by":  req.RequestedBy,
        "approved_by":   approver,
        "verification":  req.VerificationMethod,
    })

    return nil
}

// CancelMFAResset allows the user to cancel a pending reset.
// This is the user's defense against a fraudulent reset request.
func (w *MFAResetWorkflow) CancelMFAReset(
    ctx context.Context, tenantID, userID uuid.UUID, resetToken string,
) error {
    reqKey := fmt.Sprintf("mfa_reset:%s", resetToken)
    data, err := w.rdb.Get(ctx, reqKey).Result()
    if err != nil {
        return ErrInvalidResetToken
    }

    var req MFAResetRequest
    json.Unmarshal([]byte(data), &req)

    // Verify the cancellation is from the correct user.
    if req.TenantID != tenantID || req.UserID != userID {
        return ErrUnauthorized
    }

    w.rdb.Del(ctx, reqKey)
    pendingKey := fmt.Sprintf("mfa_reset_pending:%s:%s", tenantID, userID)
    w.rdb.Del(ctx, pendingKey)

    w.auditLog(ctx, tenantID, userID, "mfa_reset_cancelled_by_user", nil)
    return nil
}
```

### Administrative Controls

| Control | Implementation |
|---------|---------------|
| Cooling-off period | 24-hour delay between request and execution |
| Multi-channel notification | Email + in-app alert on reset request |
| Manager approval | Required for privileged accounts |
| Session revocation | All sessions revoked on MFA reset |
| User cancellation | User can self-cancel fraudulent requests |
| Force re-enrollment | User must re-enroll MFA before next login |
| Full audit trail | All reset actions logged with actor identity |

---

## 6. Token Theft Post-MFA

### Attack Vector

MFA protects the **authentication** step. It does NOT protect against token
theft after authentication. Once the user has completed MFA and obtained a
session token (JWT, cookie), an attacker who steals that token has full
access — no MFA required.

Common token theft vectors:
- **XSS** — JavaScript steals the token from localStorage or cookie.
- **Network interception** — Token captured over unencrypted connections.
- **Malware** — Info-stealing trojans exfiltrate browser cookies.
- **Reverse proxy** — Attacker sets up a relay to capture and replay tokens.

```
User ── MFA complete ──▶ Session token T
Malware ── steals T ──▶ Attacker uses T ──▶ Full access (no MFA challenge)
```

### Mitigation: Token Binding (DPoP / mTLS)

Token binding ties the access token to a cryptographic key held by the client.
A stolen token without the key is useless.

```go
// DPoPTokenBinder implements RFC 9449 Demonstrating Proof-of-Possession.
// The client generates an asymmetric key pair. The public key is embedded
// in the token (via header `jwk`). Each request includes a signed DPoP proof
// JWT that demonstrates possession of the private key.
type DPoPTokenBinder struct {
    keyStore KeyStore
}

// BindToken embeds the client's public key thumbprint into the JWT claims.
func (b *DPoPTokenBinder) BindToken(
    claims jwt.Claims, clientJWK json.RawMessage,
) (jwt.Claims, error) {
    // Compute the JWK thumbprint (RFC 7638).
    thumbprint, err := jwkThumbprint(clientJWK)
    if err != nil {
        return claims, err
    }

    // Add the `cnf` (confirmation) claim per RFC 7800.
    if claims == nil {
        claims = jwt.Claims{}
    }
    // Private claims map for custom JWT fields.
    private := jwt.PrivateClaims{
        "cnf": map[string]any{
            "jkt": thumbprint, // JWK SHA-256 thumbprint
        },
    }
    return jwt.Merge(claims, private), nil
}

// VerifyDPoPProof validates the DPoP proof JWT on each request.
func (b *DPoPTokenBinder) VerifyDPoPProof(
    ctx context.Context,
    dpopProof string,  // DPoP header value
    method string,     // HTTP method
    url string,        // full request URL
    accessToken string,
) error {
    // 1. Parse the DPoP JWT.
    tok, err := jwt.ParseSigned(dpopProof)
    if err != nil {
        return ErrInvalidDPoP
    }

    // 2. Extract the JWK from the header.
    var hdr jwt.Headers
    if err := tok.Headers(&hdr); err != nil {
        return ErrInvalidDPoP
    }
    if len(hdr) == 0 || hdr[0].KeyID == "" {
        return ErrInvalidDPoP
    }
    clientKey, err := jwkParseKey(hdr[0].JSON)
    if err != nil {
        return ErrInvalidDPoP
    }

    // 3. Verify the JWT signature with the client's public key.
    var claims struct {
        Htm string `json:"htm"` // HTTP method
        Htu string `json:"htu"` // HTTP URL
        Iat int64  `json:"iat"` // Issued at (timestamp)
        Jti string `json:"jti"` // Unique ID (prevent replay)
        Ath string `json:"ath"` // Access token hash
    }
    if err := tok.Verify(clientKey, &claims); err != nil {
        return ErrInvalidDPoPSignature
    }

    // 4. Verify HTTP method matches.
    if claims.Htm != method {
        return ErrDPoPMethodMismatch
    }

    // 5. Verify URL matches (without query string for some implementations).
    if claims.Htu != url {
        return ErrDPoPURLMismatch
    }

    // 6. Verify freshness — DPoP proof must be recent (within 60 seconds).
    if time.Since(time.Unix(claims.Iat, 0)) > 60*time.Second {
        return ErrDPoPExpired
    }

    // 7. Verify the access token hash matches.
    expectedAth := base64url(sha256(accessToken))
    if claims.Ath != expectedAth {
        return ErrDPoPTokenMismatch
    }

    // 8. Check replay — jti must not have been seen before.
    replayKey := fmt.Sprintf("dpop_jti:%s", claims.Jti)
    set, _ := b.keyStore.SetNX(ctx, replayKey, "1", 60*time.Second).Result()
    if !set {
        return ErrDPoPReplay
    }

    // 9. Verify the thumbprint matches the one in the access token's cnf claim.
    thumbprint, _ := jwkThumbprint(hdr[0].JSON)
    tokenThumbprint := b.getTokenConfirmation(ctx, accessToken)
    if thumbprint != tokenThumbprint {
        return ErrDPoPKeyMismatch
    }

    return nil
}

// MTLSSessionBinder provides TLS client certificate binding (RFC 8705).
// When mTLS is used, the access token is bound to the client certificate's
// thumbprint. A token presented with a different certificate is rejected.
func VerifyMTLSBinding(accessTokenCertHash, clientCertHash string) error {
    if accessTokenCertHash == "" || clientCertHash == "" {
        return ErrMTLSNotBound
    }
    if !constantTimeEqual(accessTokenCertHash, clientCertHash) {
        return ErrMTLSBindingMismatch
    }
    return nil
}

func constantTimeEqual(a, b string) bool {
    if len(a) != len(b) {
        return false
    }
    var result byte
    for i := 0; i < len(a); i++ {
        result |= a[i] ^ b[i]
    }
    return result == 0
}
```

### Comparison of Binding Methods

| Method | Stolen Token Usable? | Implementation Complexity | Client Requirement |
|--------|---------------------|--------------------------|-------------------|
| Bearer (no binding) | Yes | None | None |
| DPoP | No (needs private key) | Medium | JS Web Crypto API |
| mTLS | No (needs client cert) | High | Client certificate |
| Token Binding (RFC 8471) | No (needs TLS exporter) | Very High | Browser support |

---

## 7. Downgrade Attacks

### Attack Vector

An attacker forces the authentication system to use a weaker MFA factor than
what the user has enrolled. For example:

- User has TOTP + WebAuthn enrolled.
- Attacker somehow removes the WebAuthn enrollment via an admin API.
- System falls back to TOTP only.
- Attacker who has phished both password and TOTP can authenticate.

Alternatively, if the system supports factor selection at login time, an
attacker could choose the weakest available factor.

### Mitigation: Factor Strength Enforcement

```go
// FactorStrength defines the security level of each MFA method.
type FactorStrength int

const (
    StrengthNone     FactorStrength = 0
    StrengthSMS      FactorStrength = 10
    StrengthEmail    FactorStrength = 15
    StrengthTOTP     FactorStrength = 50
    StrengthPush     FactorStrength = 60
    StrengthWebAuthn FactorStrength = 100 // phishing-resistant
)

// FactorStrengthEnforcer prevents downgrading to a weaker factor.
type FactorStrengthEnforcer struct {
    repo     MFADeviceRepository
    policy   TenantMFAPolicy
}

// EnsureMinFactorStrength verifies that at least one enrolled factor meets
// the tenant's minimum strength requirement.
func (e *FactorStrengthEnforcer) EnsureMinFactorStrength(
    ctx context.Context, tenantID, userID uuid.UUID,
) error {
    devices, err := e.repo.ListDevicesByUser(ctx, tenantID, userID)
    if err != nil {
        return err
    }

    minStrength := e.policy.GetMinFactorStrength(ctx, tenantID)
    hasStrongEnough := false
    strongest := StrengthNone

    for _, d := range devices {
        s := factorStrengthForType(d.Type)
        if s > strongest {
            strongest = s
        }
        if s >= minStrength && d.Enabled {
            hasStrongEnough = true
        }
    }

    if !hasStrongEnough {
        return fmt.Errorf("no enrolled factor meets minimum strength %d (strongest: %d)",
            minStrength, strongest)
    }

    return nil
}

// ValidateFactorRemoval checks whether removing a factor would create a
// downgrade vulnerability.
func (e *FactorStrengthEnforcer) ValidateFactorRemoval(
    ctx context.Context, tenantID, userID uuid.UUID, removingDeviceID uuid.UUID,
) error {
    devices, err := e.repo.ListDevicesByUser(ctx, tenantID, userID)
    if err != nil {
        return err
    }

    // Simulate removal and check remaining factors.
    minStrength := e.policy.GetMinFactorStrength(ctx, tenantID)
    remainingStrongEnough := false

    for _, d := range devices {
        if d.ID == removingDeviceID {
            continue // skip the device being removed
        }
        if factorStrengthForType(d.Type) >= minStrength && d.Enabled {
            remainingStrongEnough = true
        }
    }

    if !remainingStrongEnough {
        return ErrFactorRemovalWouldDowngrade
    }

    return nil
}

// ValidateLoginFactor ensures the factor selected for authentication is not
// weaker than what the user has available.
func (e *FactorStrengthEnforcer) ValidateLoginFactor(
    ctx context.Context, tenantID, userID uuid.UUID, selectedFactorType string,
) error {
    devices, err := e.repo.ListDevicesByUser(ctx, tenantID, userID)
    if err != nil {
        return err
    }

    selectedStrength := factorStrengthForType(selectedFactorType)

    for _, d := range devices {
        if d.Enabled && factorStrengthForType(d.Type) > selectedStrength {
            // A stronger factor is available but not selected.
            // This is acceptable for convenience but should be logged.
            e.logDowngradeAttempt(ctx, tenantID, userID,
                fmt.Sprintf("Selected %s (strength %d) but %s available (strength %d)",
                    selectedFactorType, selectedStrength, d.Type,
                    factorStrengthForType(d.Type)))
        }
    }

    // Hard block: never allow SMS if WebAuthn is enrolled and enabled.
    for _, d := range devices {
        if d.Type == "webauthn" && d.Enabled && selectedFactorType == "sms" {
            return ErrDowngradeBlocked
        }
    }

    return nil
}

func factorStrengthForType(factorType string) FactorStrength {
    switch factorType {
    case "webauthn", "fido2":
        return StrengthWebAuthn
    case "push":
        return StrengthPush
    case "totp":
        return StrengthTOTP
    case "email":
        return StrengthEmail
    case "sms":
        return StrengthSMS
    default:
        return StrengthNone
    }
}
```

### Factor Strength Ranking

```
WebAuthn (100) > Push with number matching (60) > TOTP (50) > Email OTP (15) > SMS (10) > None (0)
```

The policy should enforce that **removal of a strong factor** is blocked if no
equivalent or stronger factor remains.

---

## 8. MFA Status Manipulation

### Attack Vector

If the MFA enrollment state can be manipulated via API, an attacker could:

1. **Start enrollment** — create a device record (Enabled=false).
2. **Exploit a race condition** — set Enabled=true before verification.
3. **Or** — mark an existing device as Enabled=false to weaken security.

Race conditions during enrollment are particularly dangerous: if the
"create device" and "enable device" steps are not atomic, a TOCTOU
(time-of-check-time-of-use) bug could allow an unverified device to become
active.

### Mitigation: Safe Enrollment State Machine

```go
// MFAEnrollmentState defines the lifecycle states for an MFA device.
type MFAEnrollmentState string

const (
    StateCreated    MFAEnrollmentState = "created"     // secret generated, not verified
    StateVerifying  MFAEnrollmentState = "verifying"   // first verification attempt in progress
    StateEnabled    MFAEnrollmentState = "enabled"     // verified and active
    StateDisabling  MFAEnrollmentState = "disabling"   // soft-delete in progress
    StateDisabled   MFAEnrollmentState = "disabled"    // soft-deleted
)

// EnrollmentStateMachine manages MFA device lifecycle transitions.
// Only valid transitions are allowed.
type EnrollmentStateMachine struct {
    repo MFADeviceRepository
}

var validTransitions = map[MFAEnrollmentState][]MFAEnrollmentState{
    StateCreated:   {StateVerifying, StateDisabled},
    StateVerifying: {StateEnabled, StateCreated, StateDisabled},
    StateEnabled:   {StateDisabling},
    StateDisabling: {StateDisabled},
    StateDisabled:  {}, // terminal state
}

// Transition moves a device to a new state if the transition is valid.
// This operation is atomic — uses optimistic concurrency control.
func (sm *EnrollmentStateMachine) Transition(
    ctx context.Context,
    tenantID, deviceID uuid.UUID,
    target MFAEnrollmentState,
    expectedCurrent MFAEnrollmentState,
) error {
    // 1. Validate the transition is allowed by the state machine.
    allowed := false
    for _, valid := range validTransitions[expectedCurrent] {
        if valid == target {
            allowed = true
            break
        }
    }
    if !allowed {
        return fmt.Errorf("invalid state transition: %s -> %s", expectedCurrent, target)
    }

    // 2. Atomic conditional update — only succeeds if the device is still
    //    in the expected state. This prevents TOCTOU race conditions.
    affected, err := sm.repo.ConditionalUpdateState(ctx,
        tenantID, deviceID, expectedCurrent, target)
    if err != nil {
        return fmt.Errorf("update state: %w", err)
    }
    if affected == 0 {
        // Another request changed the state between our read and write.
        return ErrConcurrentModification
    }

    return nil
}

// EnrollDevice is the safe enrollment workflow. Each step validates the
// state transition.
func (sm *EnrollmentStateMachine) EnrollDevice(
    ctx context.Context, tenantID, userID uuid.UUID, deviceName string,
) (*EnrollmentResult, error) {
    // 1. Check no existing enabled device.
    existing, _ := sm.repo.GetEnabledDevice(ctx, tenantID, userID)
    if existing != nil {
        return nil, fmt.Errorf("MFA already enabled — disable first")
    }

    // 2. Create device in "created" state.
    secret := generateTOTPSecret()
    device := &domain.MFADevice{
        ID:        uuid.New(),
        TenantID:  tenantID,
        UserID:    userID,
        Name:      deviceName,
        Secret:    secret,
        State:     StateCreated,
        Enabled:   false,
    }

    if err := sm.repo.CreateDevice(ctx, device); err != nil {
        return nil, err
    }

    // Transition to "verifying" — records intent to verify.
    if err := sm.Transition(ctx, tenantID, device.ID,
        StateVerifying, StateCreated); err != nil {
        sm.repo.DeleteDevice(ctx, tenantID, device.ID) // cleanup
        return nil, err
    }

    return &EnrollmentResult{
        DeviceID:  device.ID,
        Secret:    secret,
        QRCodeURI: buildOTPAuthURI("GGID", userID.String(), secret),
    }, nil
}

// CompleteEnrollment verifies the first TOTP code and enables the device.
func (sm *EnrollmentStateMachine) CompleteEnrollment(
    ctx context.Context, tenantID, deviceID uuid.UUID, code string,
) error {
    // 1. Get device in its current state.
    device, err := sm.repo.GetDeviceByID(ctx, tenantID, deviceID)
    if err != nil {
        return err
    }

    // 2. Verify the device is in "verifying" state.
    if device.State != StateVerifying {
        return fmt.Errorf("device not in verifying state: %s", device.State)
    }

    // 3. Validate the TOTP code.
    if !totp.Validate(code, device.Secret) {
        return ErrInvalidMFACode
    }

    // 4. Atomically transition to "enabled" with conditional update.
    if err := sm.Transition(ctx, tenantID, deviceID,
        StateEnabled, StateVerifying); err != nil {
        return err
    }

    // 5. Set Enabled flag and VerifiedAt.
    device.Enabled = true
    now := time.Now()
    device.VerifiedAt = &now
    return sm.repo.UpdateDevice(ctx, device)
}

// ConditionalUpdateState executes an atomic state transition in the database.
// This prevents race conditions where two concurrent requests try to change
// the device state simultaneously.
//
// SQL: UPDATE mfa_devices SET state = $4, updated_at = NOW()
//      WHERE tenant_id = $1 AND id = $2 AND state = $3
func (r *pgMFADeviceRepo) ConditionalUpdateState(
    ctx context.Context,
    tenantID, deviceID uuid.UUID,
    expectedState, newState MFAEnrollmentState,
) (int64, error) {
    tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return 0, err
    }
    defer tx.Rollback(ctx)

    mfaSetTenant(ctx, tx, tenantID)

    tag, err := tx.Exec(ctx, `
        UPDATE mfa_devices
        SET state = $4, updated_at = NOW()
        WHERE tenant_id = $1 AND id = $2 AND state = $3`,
        tenantID, deviceID, expectedState, newState)
    if err != nil {
        return 0, err
    }
    return tag.RowsAffected(), tx.Commit(ctx)
}
```

### State Transition Diagram

```
                ┌──────────┐
                │ created  │
                └────┬─────┘
                     │ EnrollDevice()
                     ▼
                ┌──────────┐    verify success     ┌─────────┐
                │ verifying├──────────────────────▶│ enabled │
                └────┬─────┘                       └────┬────┘
                     │ verify failed                    │ DisableMFA()
                     ▼                                  ▼
                ┌──────────┐                       ┌───────────┐
                │ created  │                       │ disabling │
                └──────────┘                       └─────┬─────┘
                                                         ▼
                                                   ┌──────────┐
                                                   │ disabled │ (terminal)
                                                   └──────────┘
```

The critical property: a device can only reach `enabled` from `verifying`,
and `verifying` can only be reached from `created`. No path allows jumping
directly from `created` to `enabled`.

---

## 9. GGID MFA Bypass Surface Analysis

This section reviews the actual GGID codebase for MFA bypass vulnerabilities.

### 9.1 Current Architecture

GGID's MFA implementation lives in `services/auth/internal/`:

| Component | File | Responsibility |
|-----------|------|----------------|
| MFAService | `service/mfa_service.go` | TOTP enrollment, verification, disable |
| MFADevice domain | `domain/mfa.go` | MFADevice + MFAChallenge structs |
| Auth login flow | `service/auth_service.go` | Login → MFA challenge → LoginMFA |
| HTTP handlers | `server/http.go` | REST endpoints for MFA setup/verify/login |
| Repository | `repository/mfa_repo.go` | PostgreSQL persistence |
| Step-up auth | `service/stepup.go` | Step-up challenges for sensitive ops |

### 9.2 Pre-MFA Session Token — SAFE

**Finding**: The login flow does **not** issue a real session token before MFA.

In `auth_service.go` `Login()` (lines 118-129):
```go
if s.mfaService != nil && s.mfaService.HasMFAEnabled(ctx, tc.TenantID, userID) {
    challenge, err := crypto.GenerateRandomToken(32)
    return &domain.TokenSet{
        MFARequired:  true,
        MFAChallenge: challenge,
    }, nil  // No AccessToken, RefreshToken, or SessionID
}
```

The `TokenSet` returned contains only `MFARequired: true` and an opaque
`MFAChallenge`. No `AccessToken`, `RefreshToken`, or `SessionID` is populated.
The real tokens are only issued in `LoginMFA()` after TOTP verification.

**Verdict**: No half-authenticated session token vulnerability.

### 9.3 MFA Challenge Not Server-Side Verified — RISK

**Finding**: The `MFAChallenge` token returned by `Login()` is **not stored in
Redis** and is **not verified** in `LoginMFA()`.

`LoginMFA()` (lines 337-394) re-authenticates the user from scratch:
```go
func (s *AuthService) LoginMFA(ctx context.Context, username, password, mfaCode, ip, userAgent string) ...
    // Re-authenticate via provider chain
    result, err := s.chain.Authenticate(ctx, ...)
```

The challenge token is never checked. The client simply calls `LoginMFA` with
`username + password + mfaCode`. The password is sent **twice** (once in
`Login`, once in `LoginMFA`).

**Risks**:
- Password is transmitted twice, increasing interception surface.
- No binding between the `Login` MFA challenge and the `LoginMFA` submission —
  an attacker could skip `Login` entirely and call `LoginMFA` directly.
- The `MFAChallenge` field is purely cosmetic; it serves no security purpose.

**Recommendation**: Store the challenge in Redis and require it in `LoginMFA`
(see Section 1 code). Stop re-sending the password.

### 9.4 No MFA Attempt Rate Limiting — RISK

**Finding**: `LoginMFA()` does not apply rate limiting on the MFA code itself.

The login endpoint has IP-based rate limiting (5/min), but `LoginMFA` re-uses
the same login rate limit key — it does not have a separate MFA attempt
counter. A 6-digit TOTP code has 1,000,000 possibilities. With 5
attempts/minute, an attacker needs ~138 days on average. But with multiple IPs
or a distributed attack, the effective rate is much higher.

**Recommendation**: Add per-user MFA attempt rate limiting (5 attempts per
5-minute window, lockout after 10 total).

### 9.5 Trusted Device Bypass — RISK

**Finding**: The `IsTrustedDevice()` / `RememberTrustedDevice()` mechanism
(auth_service.go lines 742-764) allows **complete MFA bypass** for 30 days
based on a device fingerprint stored in Redis.

```go
func (s *AuthService) IsTrustedDevice(ctx context.Context, tenantID, userID uuid.UUID, fingerprint string) bool
```

The fingerprint is client-supplied (typically a cookie or header value). If
an attacker obtains or forges the fingerprint value, they bypass MFA entirely
for 30 days. The fingerprint has no cryptographic binding to the device.

**Recommendation**: Bind the trusted device to a cryptographic key or
hardware attestation, not a plain string fingerprint. Consider shorter TTL
(7 days) and re-verify with MFA on IP/geo changes.

### 9.6 DisableMFA Without Verification — RISK

**Finding**: `DisableMFA()` (mfa_service.go line 137) requires only the
device ID — no additional verification (password, MFA code, or admin approval).

```go
func (s *MFAService) DisableMFA(ctx context.Context, deviceID uuid.UUID) error {
    return s.repo.DeleteDevice(ctx, tc.TenantID, deviceID)
}
```

If an attacker has a valid session token (post-auth) and knows or can guess
the device ID, they can disable MFA and then change the password to lock out
the legitimate user.

**Recommendation**: Require MFA code or password re-verification before
disabling MFA. Require a cooling-off period.

### 9.7 No Backup/Recovery Codes — GAP

**Finding**: There is **no backup code implementation** in the codebase. Users
who lose their TOTP device have no recovery path other than admin intervention.

**Recommendation**: Implement backup code generation and validation (see
Section 4 code).

### 9.8 TOTP Uses SHA-1 — LOW RISK

**Finding**: TOTP generation uses `otp.AlgorithmSHA1` (mfa_service.go line 53).
SHA-1 is used for compatibility with most authenticator apps. While SHA-1 has
known collision weaknesses, TOTP's security model does not depend on collision
resistance — it depends on the secret being unknown. This is acceptable but
should be noted.

### 9.9 No TOTP Code Replay Prevention — LOW RISK

**Finding**: There is no tracking of used TOTP codes. The same 6-digit code
can be submitted multiple times within its 30-second validity window. RFC 6238
Section 5.2 recommends that implementations reject reused codes.

**Recommendation**: Store the last-used TOTP code (or its hash) in Redis with
a 30-second TTL and reject duplicates.

### 9.10 Summary Table

| Finding | Severity | Status |
|---------|----------|--------|
| Pre-MFA session token | CRITICAL | SAFE — no token issued before MFA |
| MFA challenge not server-verified | MEDIUM | RISK — challenge is cosmetic |
| No MFA attempt rate limiting | MEDIUM | RISK — brute-force possible |
| Trusted device MFA bypass | HIGH | RISK — fingerprint not crypto-bound |
| DisableMFA without verification | HIGH | RISK — can disable with session only |
| No backup codes | MEDIUM | GAP — no recovery path |
| TOTP SHA-1 | LOW | ACCEPTABLE — compatibility requirement |
| No TOTP replay prevention | LOW | RISK — RFC 6238 violation |

---

## 10. Gap Analysis & Recommendations

### P0: Block DisableMFA Without Re-Verification (Effort: 2h)

**Problem**: Any authenticated session can disable MFA by device ID alone.

**Fix**: Add a `DisableMFAWithVerify()` method that requires either:
- A valid TOTP code from the device being disabled, or
- Password re-verification + admin approval for MFA reset.

Add a 24-hour cooling-off period where the old MFA device remains active
while the disable request is pending.

### P1: Server-Side MFA Challenge Verification (Effort: 3h)

**Problem**: `LoginMFA` re-sends the password and ignores the challenge token.

**Fix**: Store the challenge in Redis in `Login()`, require it in `LoginMFA`,
and stop re-sending the password. This:
- Eliminates double password transmission.
- Binds the MFA submission to the original login attempt.
- Enables per-challenge attempt limiting (5 tries per challenge token).

### P1: Per-User MFA Attempt Rate Limiting (Effort: 2h)

**Problem**: TOTP brute-force is only gated by IP-based login rate limiting.

**Fix**: Add a Redis-backed counter keyed by `tenantID:userID`:
- 5 attempts per 5-minute window.
- Account lockout after 10 total MFA failures (1-hour cooldown).
- Reset counter on successful MFA verification.

### P2: Cryptographic Trusted Device Binding (Effort: 8h)

**Problem**: Trusted device fingerprint is a plain string with no crypto binding.

**Fix**: Replace string fingerprint with a DPoP-style key pair:
- Client generates ECDSA key on first trusted-device enrollment.
- Public key stored in Redis.
- Each trusted-device login includes a signed proof.
- Without the private key, the trusted device bypass is unusable.

### P2: Backup Code Implementation (Effort: 6h)

**Problem**: No recovery path for lost TOTP devices.

**Fix**: Implement the `BackupCodeService` from Section 4:
- Generate 10 codes on enrollment.
- Hash-based storage (never plaintext).
- Single-use enforcement with atomic conditional updates.
- 5-attempt rate limiting with 1-hour lockout.

### P3: TOTP Replay Prevention (Effort: 1h)

**Problem**: Same TOTP code can be replayed within its 30-second window.

**Fix**: Store `sha256(tenantID:userID:code)` in Redis with 90-second TTL
(2 periods for clock drift). Reject on collision.

---

## References

- [RFC 6238](https://tools.ietf.org/html/rfc6238) — TOTP: Time-Based One-Time Password Algorithm
- [RFC 9449](https://tools.ietf.org/html/rfc9449) — DPoP: Demonstrating Proof-of-Possession
- [RFC 8705](https://tools.ietf.org/html/rfc8705) — OAuth 2.0 Mutual-TLS Client Authentication
- [NIST SP 800-63B](https://pages.nist.gov/800-63-3/sp800-63b.html) — Digital Identity Guidelines: Authentication
- [OWASP MFA Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html)
- Uber MFA Fatigue Breach (2022): [CSO Online](https://www.csoonline.com/article/3686354/uber-breach-2022.html)
- Cisco MFA Fatigue Breach (2022): [Cisco Advisory](https://sec.cloudapps.cisco.com/security/center/content/CiscoSecurityAdvisory/cisco-sa-asaftd-ra-vpn-exfiltration-jjSOdOke)

---

*Document version: 1.0 — covers GGID auth service as of current main branch.*
