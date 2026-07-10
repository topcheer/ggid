# Multi-Factor Authentication Orchestration

> Focus: the **state machine** that coordinates which factor to challenge, in what order,
> how to fall back, how to remember devices, and how to prevent MFA fatigue.
> Risk scoring is covered in [adaptive-mfa-design.md](adaptive-mfa-design.md),
> step-up flows in [step-up-authentication-patterns.md](step-up-authentication-patterns.md),
> and push delivery in [mfa-push-notification-design.md](mfa-push-notification-design.md).

## 1. Overview

MFA orchestration is the decision layer that sits between "password verified" and "user
authenticated." It answers four questions on every login:

1. **Which factor?** — Of the factors a user has enrolled, which one to challenge.
2. **In what order?** — Primary first, then fallback in priority order.
3. **What if it fails?** — Retry, switch factor, or lock.
4. **How to prevent abuse?** — Rate-limit challenges, detect fatigue attacks.

The orchestration engine is a **state machine** that drives the authentication flow from
`auth_pending` through one or more factor challenges to `authenticated`. Each state
transition is validated, logged, and auditable.

NIST SP 800-63B establishes the Authenticator Assurance Level (AAL) requirements:
- **AAL1** — one factor (password). Acceptable for low-assurance services.
- **AAL2** — two distinct factors. Required for most enterprise applications.
- **AAL3** — hardware-backed factor (WebAuthn cross-platform, smartcard). For high-value transactions.

The orchestrator's job is to guarantee the session meets the **required AAL** with the best
user experience, choosing the strongest enrolled factor and degrading gracefully on failure.

## 2. Factor Catalog

| Factor | AAL | Cost | Latency | UX Friction | GGID Support |
|--------|-----|------|---------|-------------|--------------|
| Password | AAL1 | Free | <1 ms | Low | Yes |
| TOTP (RFC 6238) | AAL2 | Free | <1 ms verify | Medium (code entry) | Yes |
| WebAuthn platform | AAL2-3 | Free | <100 ms | Low (biometric) | Yes |
| WebAuthn cross-platform | AAL3 | $20-75 key | <100 ms | Low | Yes |
| Push notification | AAL2 | Infra cost | 5-60 s | Very low | Planned |
| SMS OTP | AAL2 | Per-SMS cost | 5-30 s | Medium | Not recommended |
| Email OTP | AAL2 | Free | 10-60 s | High (context switch) | Fallback only |
| Hardware token (HOTP) | AAL2-3 | $20-75 | <1 ms | Medium | Not implemented |

**Recommendation:** WebAuthn platform (Touch ID / Face ID) provides the best security-to-friction
ratio. TOTP is the reliable fallback when no biometric is available. SMS is deprecated by NIST
for new deployments. Email OTP is acceptable only as a last-resort fallback.

## 3. Factor Sequencing

### Primary -> Secondary Pattern

The default flow for password-based login:

```
Factor 1: password  (always required)
Factor 2: strongest enrolled factor
```

Preference order (strongest first):

1. WebAuthn platform (biometric, AAL3 if hardware-backed)
2. WebAuthn cross-platform (security key)
3. Push notification (when implemented)
4. TOTP (authenticator app)
5. Email OTP (worst UX, last resort)

### Adaptive Sequencing

The orchestrator consumes the risk assessment (from `AssessLoginRisk`) to select the factor:

| Risk Level | Factors Required | Example |
|------------|-----------------|---------|
| Low | password + remember-device skip | Returning from known IP |
| Medium | password + TOTP | Unknown IP, normal hours |
| High | password + WebAuthn (hardware-backed) | Failed attempts from same IP |
| Very High | password + WebAuthn + admin approval | Brute-force pattern detected |

### WebAuthn-First (Passwordless)

When a user has a WebAuthn platform authenticator with user verification enabled, the
passkey serves as both password replacement and second factor:

```
Factor 1: WebAuthn (user verification = AAL2+)
```

No separate password step. If WebAuthn fails (lost device, timeout), the orchestrator
falls back to `password -> TOTP`.

### Per-Tenant Factor Configuration

```go
type FactorPreference struct {
    FactorType string `json:"factor_type"`
    Priority   int    `json:"priority"`  // lower = preferred first
    MaxAttempts int   `json:"max_attempts"`
}

// TenantConfig defines per-tenant MFA orchestration.
type TenantMFAConfig struct {
    RequiredAAL        int                `json:"required_aal"`         // 1, 2, or 3
    FactorPreferences  []FactorPreference `json:"factor_preferences"`
    RememberDeviceDays int                `json:"remember_device_days"`  // 0 = disabled
    FatigueMaxPerWindow int               `json:"fatigue_max_per_window"` // default 3
}
```

## 4. MFA State Machine

### States

| State | Description |
|-------|-------------|
| `auth_pending` | Initial state, password verification in progress |
| `password_verified` | First factor completed, evaluating MFA requirement |
| `mfa_challenge` | MFA challenge issued, awaiting user response |
| `mfa_verified` | MFA completed successfully |
| `mfa_failed` | MFA verification failed (wrong code / rejected push) |
| `mfa_timeout` | Challenge expired without response |
| `step_up_required` | Elevated AAL needed for sensitive operation |
| `authenticated` | Full authentication complete, tokens issued |
| `locked` | Account locked after max retries |

### Transition Summary

```
auth_pending --password_ok--> password_verified
password_verified --mfa_required--> mfa_challenge
password_verified --no_mfa--> authenticated
mfa_challenge --success--> mfa_verified --> authenticated
mfa_challenge --fail(retry_left)--> mfa_failed --> mfa_challenge
mfa_challenge --fail(max_retries)--> locked (via fallback chain exhaustion)
mfa_challenge --timeout--> mfa_timeout --> mfa_challenge (retry)
authenticated --step_up_needed--> step_up_required --> mfa_challenge
```

### Go Implementation

```go
type MFAState string // auth_pending, password_verified, mfa_challenge, mfa_verified,
                     // mfa_failed, mfa_timeout, step_up_required, authenticated, locked
type MFAEvent string // password_ok, mfa_required, mfa_success, mfa_failed, mfa_timeout, retry

type MFAStateMachine struct {
    state         MFAState
    currentFactor FactorType
    fallback      *FallbackChain
    attempts      int
    maxRetry      int
}

func (sm *MFAStateMachine) Transition(event MFAEvent) error {
    switch sm.state {
    case StateAuthPending:
        if event == EventPasswordOK { sm.state = StatePasswordVerified }
    case StatePasswordVerified:
        if event == EventMFARequired { sm.state = StateMFAChallenge } else { sm.state = StateAuthenticated }
    case StateMFAChallenge:
        switch event {
        case EventMFASuccess:
            sm.state = StateMFAVerified
        case EventMFAFailed:
            sm.attempts++
            if sm.attempts >= sm.maxRetry {
                next, err := sm.fallback.NextFactor(sm.currentFactor)
                if err != nil { sm.state = StateLocked; break }
                sm.currentFactor = next; sm.attempts = 0
            }
        case EventMFATimeout:
            sm.state = StateMFATimeout
        }
    case StateMFAFailed, StateMFATimeout:
        if event == EventRetry { sm.state = StateMFAChallenge }
    case StateMFAVerified:
        sm.state = StateAuthenticated
    case StateAuthenticated:
        if event == EventStepUpNeeded { sm.state = StateStepUpRequired }
    case StateStepUpRequired:
        sm.state = StateMFAChallenge
    }
    return nil
}
```

## 5. Fallback Chains

### Concept

When the primary MFA factor fails or is unavailable (lost device, authenticator app not
installed), the orchestrator escalates to the next factor in the chain rather than locking
the user out immediately.

```
webauthn → totp → email → admin_recovery → locked
```

### Configuration

```json
{
  "mfa_chain": [
    {"factor": "webauthn", "max_attempts": 3},
    {"factor": "totp", "max_attempts": 3},
    {"factor": "email", "max_attempts": 1},
    {"factor": "admin_recovery", "max_attempts": 1}
  ]
}
```

### Security Rules

1. **Never downgrade AAL silently.** Falling back from WebAuthn (AAL3) to TOTP (AAL2) is
   acceptable. Falling back from any MFA factor to password-only is NOT — it drops the
   session from AAL2 to AAL1.
2. **Same-AAL fallback is safe.** TOTP -> Email is acceptable (both AAL2).
3. **Higher-AAL escalation never falls back.** If AAL3 is required and WebAuthn fails,
   the user must use another WebAuthn device or admin-issued temporary credential.
4. **Admin recovery is the terminal fallback.** After all automated factors are exhausted,
   the account is locked and requires administrator intervention.

### Go Implementation

```go
type FactorType string // webauthn_platform, webauthn_cross_platform, totp, push, email, admin_recovery

type FallbackChain struct {
    chain  []ChainEntry  // [{Factor, MaxAttempts}, ...]
    aalMap map[FactorType]int
}

// NextFactor returns the next factor after current, or error if chain exhausted.
func (fc *FallbackChain) NextFactor(current FactorType) (FactorType, error) {
    for i, e := range fc.chain {
        if e.Factor == current && i+1 < len(fc.chain) {
            return fc.chain[i+1].Factor, nil
        }
    }
    return "", fmt.Errorf("no more factors in fallback chain")
}
```

## 6. Remember This Device

### Pattern

After successful MFA, the orchestrator can issue a **device token** that lets the user
skip MFA on future logins from the same device:

1. User completes password + MFA.
2. Orchestrator computes a device fingerprint (UA hash + IP class C + user ID).
3. Stores the fingerprint in Redis with a configurable TTL (default: 30 days).
4. On next login, if fingerprint matches and risk is low, skip MFA.

### Security Considerations

- **Risk-gated.** Don't offer "remember" for high-risk sessions (new IP, off-hours, impossible travel).
- **Revocable.** Users can view/revoke remembered devices. GGID already has `ListDevices` / `RemoveDevice`.
- **Break-glass.** Sensitive operations **always** require step-up MFA regardless of remember status.
- **Storage.** Redis key (server-side, instantly revocable) preferred over signed cookie for enterprise.

### Go Implementation

```go
type RememberDeviceManager struct {
    rdb *redis.Client
    ttl time.Duration // default 30 days
}

func (m *RememberDeviceManager) Remember(ctx context.Context, userID, fp string) error {
    return m.rdb.Set(ctx, fmt.Sprintf("mfa:remember:%s:%s", userID, fp), time.Now().Unix(), m.ttl).Err()
}

func (m *RememberDeviceManager) IsRemembered(ctx context.Context, userID, fp string) bool {
    _, err := m.rdb.Get(ctx, fmt.Sprintf("mfa:remember:%s:%s", userID, fp)).Result()
    return err == nil
}

func (m *RememberDeviceManager) Revoke(ctx context.Context, userID, fp string) error {
    return m.rdb.Del(ctx, fmt.Sprintf("mfa:remember:%s:%s", userID, fp)).Err()
}
```

GGID's existing `deviceFingerprint()` in `device_tracking.go` (SHA-256 of UA + IP) is the fingerprint input.

## 7. MFA Fatigue Prevention

### The Attack (MFA Bombing / Push Fatigue)

An attacker who has stolen a password repeatedly triggers push MFA challenges. The victim
is bombarded with notifications and eventually approves one out of frustration or to stop
the noise. This attack was used in the 2022 Uber breach.

### Prevention Measures

| Measure | Description | Implementation |
|---------|-------------|----------------|
| Rate limiting | Max 3 MFA challenges per user per 5 min | Redis sliding window |
| Number matching | Display a random number; user must enter the matching number | Challenge includes nonce |
| Location context | "Sign-in from San Francisco, CA" in the push payload | IP geolocation |
| Cooldown | After 3 challenges, 15-minute lockout before next | Redis TTL key |
| Admin alert | >5 challenges in 10 min triggers security event | Audit log + SIEM webhook |
| Factor switch | After 2 push failures, offer TOTP instead | Fallback chain |

### Go Implementation

```go
type FatiguePreventer struct {
    rdb          *redis.Client
    maxPerWindow int           // default 3
    window       time.Duration // default 5 min
    cooldown     time.Duration // default 15 min
}

func (fp *FatiguePreventer) ShouldAllow(ctx context.Context, userID string) bool {
    // Check cooldown.
    if _, err := fp.rdb.Get(ctx, fmt.Sprintf("mfa:fatigue:cooldown:%s", userID)).Result(); err == nil {
        return false
    }
    // Sliding window.
    key := fmt.Sprintf("mfa:fatigue:%s", userID)
    count, _ := fp.rdb.Incr(ctx, key).Result()
    if count == 1 { fp.rdb.Expire(ctx, key, fp.window) }
    if count > int64(fp.maxPerWindow) {
        fp.rdb.Set(ctx, fmt.Sprintf("mfa:fatigue:cooldown:%s", userID), "1", fp.cooldown)
        return false
    }
    return true
}
```

## 8. GGID Current State

| Capability | Status | Details |
|-----------|--------|---------|
| TOTP verification | **Implemented** | `MFAService.VerifyUserCode()` — single device per user, RFC 6238 |
| WebAuthn verification | **Implemented** | Separate WebAuthn service in `pkg/webauthn` |
| Phone OTP | **Implemented** | `SendPhoneOTP` / `VerifyPhoneOTP` — passwordless SMS login |
| Risk assessment | **Implemented** | `AssessLoginRisk()` — Low/Medium/High with step-up flag |
| Step-up auth | **Implemented** | `InitStepUp` / `VerifyStepUp` — password or MFA, ACR-based |
| Device tracking | **Implemented** | `TrackDevice` / `ListDevices` / `RemoveDevice` — Redis |
| Per-tenant force MFA | **Implemented** | `IsForceMFA()` — boolean flag, blocks login if no MFA set up |
| Factor sequencing | **Gap** | Login flow is binary: `HasMFAEnabled` -> challenge. No multi-factor priority ordering. |
| Formal state machine | **Gap** | Login is linear (`password -> MFA -> tokens`). No retry counting, no timeout state, no event-driven FSM. |
| Fallback chain | **Gap** | Single MFA factor (TOTP only). If TOTP fails 3x, account is just locked — no TOTP -> email escalation. |
| Remember this device | **Gap** | Device tracking records devices but is **not** used to skip MFA. Every login with MFA enabled always challenges. |
| MFA fatigue prevention | **Gap** | No rate limiting on MFA challenges. Phone OTP has its own rate limit (5 per 5 min), but TOTP challenges are unlimited. |
| Per-tenant factor config | **Gap** | Only `force_mfa` boolean. No factor preference ordering or per-tenant AAL requirement. |

### Current Login Flow (Linear)

```
Login(username, password)
  → chain.Authenticate()          // verify password
  → HasMFAEnabled?                // binary check
    → YES: return MFAChallenge    // single TOTP challenge
    → NO:  issue tokens           // authenticated
```

### Target Flow (State Machine)

```
Login(username, password)
  → chain.Authenticate()          // StateAuthPending → StatePasswordVerified
  → AssessLoginRisk()             // risk assessment
  → SelectFactor(risk, enrolled)  // choose factor from priority
  → Remembered? & risk=low?       // skip MFA if remembered
    → YES: → StateAuthenticated
    → NO:  → StateMFAChallenge
      → FatiguePreventer.ShouldAllow()  // rate-limit
      → Verify factor (TOTP/WebAuthn/push)
        → success: → StateAuthenticated
        → fail:   fallback.NextFactor() → StateMFAChallenge (retry)
        → exhausted: → StateLocked
```

## 9. Implementation Roadmap

| Phase | Description | Priority | Effort | Dependencies |
|-------|-------------|----------|--------|-------------|
| 1 | **MFA state machine** — Formalize `MFAStateMachine` with states, events, transition validation, retry counting, and timeout handling. Replace the linear login flow. | P1 | ~3 days | None |
| 2 | **Configurable factor sequencing** — Per-tenant `TenantMFAConfig` with factor priorities, required AAL, and max attempts. Store in tenant config table. | P1 | ~2 days | Phase 1 |
| 3 | **Remember-this-device** — `RememberDeviceManager` using Redis. Integrate into login flow with risk-gating. Wire to existing `deviceFingerprint()`. | P2 | ~2 days | Phase 1 |
| 4 | **Fatigue prevention** — `FatiguePreventer` with rate limiting, cooldown, and admin alerts. Applies to push and TOTP challenges. | P2 | ~1 day | Phase 1 |
| 5 | **Fallback chains** — `FallbackChain` with auto-escalation (WebAuthn -> TOTP -> Email). Security-rule validation (no AAL downgrade). | P2 | ~3 days | Phases 1, 2 |
| 6 | **Push MFA integration** — Wire push factor from [mfa-push-notification-design.md](mfa-push-notification-design.md) into the state machine with number matching and location context. | P2 | ~1 week | Phases 1, 4 |

### Total Effort

- **P1 (Phases 1-2):** ~5 days — state machine + per-tenant config
- **P2 (Phases 3-6):** ~8 days — remember-device, fatigue, fallback, push
- **Total:** ~13 developer-days

### Testing Strategy

- Unit tests for `MFAStateMachine.Transition()` — all state/event combinations
- Integration tests for `FallbackChain.NextFactor()` — verify no AAL downgrade
- E2E test: password -> TOTP fails -> email fallback -> success
- E2E test: fatigue prevention triggers cooldown after 3 challenges
- E2E test: remembered device skips MFA on second login
