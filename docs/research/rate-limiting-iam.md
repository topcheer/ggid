# Rate Limiting Strategies Specific to IAM Systems

> **Scope**: This document covers IAM-specific rate limiting dimensions, threat
> models, adaptive policies, CAPTCHA escalation, distributed Redis patterns, and
> a full audit of GGID's gateway rate limiting implementation. It does NOT
> duplicate the generic algorithm survey in `rate-limiting-strategies.md`
> (token bucket vs. sliding window vs. fixed window). Read that doc first for
> algorithm fundamentals and Redis Lua basics.

---

## Table of Contents

1. [IAM-Specific Rate Limiting Dimensions](#1-iam-specific-rate-limiting-dimensions)
2. [Per-IP vs Per-User vs Per-Tenant](#2-per-ip-vs-per-user-vs-per-tenant)
3. [Adaptive Rate Limiting](#3-adaptive-rate-limiting)
4. [CAPTCHA Escalation](#4-captcha-escalation)
5. [Distributed Rate Limiting with Redis](#5-distributed-rate-limiting-with-redis)
6. [429 Response Best Practices for IAM](#6-429-response-best-practices-for-iam)
7. [GGID Gateway Rate Limit Audit](#7-ggid-gateway-rate-limit-audit)
8. [Gap Analysis & Recommendations](#8-gap-analysis--recommendations)

---

## 1. IAM-Specific Rate Limiting Dimensions

A generic API rate limiter treats every endpoint identically. IAM systems
cannot. Each authentication endpoint is an attack surface with distinct threat
models, traffic profiles, and acceptable failure costs. The table below maps
each IAM endpoint to its threat model and recommended limit strategy.

### 1.1 Login Rate Limiting (Credential Stuffing Defense)

Credential stuffing uses breached username/password pairs at scale. Unlike
classic brute force (which targets one account with many passwords), stuffing
tries many accounts with few passwords each. This means a per-account limit
alone is insufficient — the attacker distributes attempts across thousands of
accounts.

**Required dimensions**: per-IP AND per-username AND per-tenant.

| Limit Key         | Threshold           | Rationale                                    |
|-------------------|---------------------|----------------------------------------------|
| `ip:login`        | 10 attempts / min   | Single-source stuffing detection             |
| `username:login`  | 5 failures / 15 min | Targeted brute force on one account          |
| `tenant:login`    | 100 attempts / min  | Distributed stuffing across one organization |
| `global:login`    | 10,000 attempts/min | Platform-wide attack cap                     |

```go
// LoginAttempt tracks failed login attempts across multiple dimensions.
type LoginAttemptTracker struct {
    store RateLimitStore // Redis or in-memory
}

// CheckLoginAllowed evaluates all rate limit dimensions for a login attempt.
// Returns the reason if denied, empty string if allowed.
func (t *LoginAttemptTracker) CheckLoginAllowed(
    ctx context.Context,
    ip, username, tenantID string,
) (denied bool, reason string, retryAfter time.Duration) {
    checks := []struct {
        key    string
        limit  int
        window time.Duration
    }{
        {fmt.Sprintf("login:ip:%s", ip), 10, time.Minute},
        {fmt.Sprintf("login:user:%s", username), 5, 15 * time.Minute},
        {fmt.Sprintf("login:tenant:%s", tenantID), 100, time.Minute},
    }

    for _, c := range checks {
        count, _ := t.store.CountAndAdd(ctx, c.key, time.Now().Add(-c.window), time.Now())
        if int(count) >= c.limit {
            return true, fmt.Sprintf("rate_limit_exceeded:%s", c.key), c.window
        }
    }
    return false, "", 0
}

// RecordLoginFailure increments failure counters on authentication failure.
// Call this AFTER CheckLoginAllowed returns allowed but authentication fails.
func (t *LoginAttemptTracker) RecordLoginFailure(
    ctx context.Context,
    ip, username, tenantID string,
) {
    // The CountAndAdd in CheckLoginAllowed already incremented the counter.
    // This method exists to record metadata (timestamp, IP) for forensic
    // analysis and to trigger account lockout after N failures.
    _ = t.store.Increment(ctx, fmt.Sprintf("login:fail:user:%s", username))
}
```

**Critical design note**: Only count *failed* attempts toward the
per-username limit. Counting successful logins penalizes legitimate users who
log in from multiple devices. The per-IP limit should count *all* attempts
(success or failure) because a botnet endpoint making rapid successful logins
is also suspicious.

### 1.2 Password Reset Throttling

Password reset endpoints are abused for two attacks: **email bombing**
(sending thousands of reset emails to harass a user) and **account enumeration**
(probing which email addresses exist by observing response timing or content).

```go
// PasswordResetLimiter limits reset requests by multiple dimensions.
type PasswordResetLimiter struct {
    store RateLimitStore
}

func (l *PasswordResetLimiter) CheckResetAllowed(
    ctx context.Context,
    ip, email, tenantID string,
) (allowed bool, retryAfter time.Duration) {
    checks := []struct {
        key    string
        limit  int
        window time.Duration
    }{
        // Per-email: prevent email bombing of a single user
        {fmt.Sprintf("reset:email:%s", email), 3, time.Hour},
        // Per-IP: prevent enumeration across many emails
        {fmt.Sprintf("reset:ip:%s", ip), 10, time.Hour},
        // Per-tenant: distributed reset flooding
        {fmt.Sprintf("reset:tenant:%s", tenantID), 50, time.Hour},
    }

    for _, c := range checks {
        count, _ := l.store.CountAndAdd(ctx, c.key, time.Now().Add(-c.window), time.Now())
        if int(count) >= c.limit {
            return false, c.window
        }
    }
    return true, 0
}
```

**Anti-enumeration**: Always return the same response body and timing
regardless of whether the email exists. A 429 should NOT reveal whether the
email is registered:

```json
// GOOD — same response for existing and non-existing emails
{"message":If this email is registered, a reset link has been sent."}

// BAD — reveals account existence via rate limit timing
{"message":"User not found"}  // fast response
{"message":"Reset link sent"} // slow response (email sent)
```

### 1.3 Registration Rate Limiting (Account Creation Abuse)

Automated account creation is used for spam, free-tier abuse, and symlink
attacks. Registration limits must be strict per-IP (bots share IPs) but
generous per-tenant (legitimate organizations onboard users in bulk).

```go
// RegistrationLimit defines multi-dimensional registration throttling.
type RegistrationLimit struct {
    PerIPLimit     int           // e.g., 5 per hour per IP
    PerTenantLimit int           // e.g., 100 per hour per tenant
    GlobalLimit    int           // e.g., 5000 per hour globally
    Window         time.Duration // e.g., 1 hour
}
```

| Dimension      | Limit       | Threat                                              |
|----------------|-------------|-----------------------------------------------------|
| Per-IP         | 5 / hour    | Bot-driven mass registration from single source     |
| Per-Tenant     | 100 / hour  | Configurable per plan (free: strict, enterprise: high) |
| Per-domain     | 10 / hour   | Disposable email domain abuse                       |
| Global         | 5000 / hour | Platform-level abuse cap                            |

### 1.4 MFA Verification Rate Limiting

MFA brute force is the most under-protected IAM surface. A 6-digit TOTP code
has only 1,000,000 possibilities — at 100 requests/sec without rate limiting,
an attacker can exhaust the space in under 3 hours. With rate limiting at
5 attempts / 5 minutes, exhaustive search takes 9.5 years.

```go
// MFALimiter enforces strict limits on MFA code verification.
type MFALimiter struct {
    store RateLimitStore
}

func (l *MFALimiter) CheckMFAAllowed(
    ctx context.Context,
    userID string,
) (allowed bool, remaining int, lockoutUntil time.Time) {
    key := fmt.Sprintf("mfa:user:%s", userID)

    // Count attempts in the last 5 minutes
    count, _ := l.store.CountAndAdd(ctx, key,
        time.Now().Add(-5*time.Minute), time.Now())

    if int(count) >= 5 {
        // Lock account from MFA for 15 minutes after 5 failures
        lockoutKey := fmt.Sprintf("mfa:lockout:%s", userID)
        _ = l.store.SetTTL(ctx, lockoutKey, "locked", 15*time.Minute)
        return false, 0, time.Now().Add(15 * time.Minute)
    }

    return true, 5 - int(count) - 1, time.Time{}
}
```

**MFA rate limiting must be per-user, never per-IP**, because the attacker
controls their IP (botnet, VPN) but cannot change the target user ID.

### 1.5 API Token Issuance Limits

OAuth token issuance (`POST /oauth/token`) is abused for: token flooding
(exhausting rate limits on downstream APIs), authorization code replay, and
refresh token rotation attacks.

| Token Endpoint         | Limit          | Threat                                |
|------------------------|----------------|---------------------------------------|
| `/oauth/token`         | 30 / min / IP  | Token flooding                        |
| `/oauth/token` per app | 100 / min      | Misbehaving OAuth client              |
| Refresh token rotation | 10 / min / user| Token rotation DoS                    |
| Device code flow       | 5 / hour / IP  | Device code phishing                  |

### 1.6 Why IAM Needs Different Rules Per Endpoint

The fundamental difference between a general API rate limiter and an IAM rate
limiter is **threat-model specificity**:

- **General API**: the goal is fair resource allocation. One limit fits all.
- **IAM**: each endpoint has a unique attacker economic model. An attacker
  brute-forcing MFA tolerates 5 req/min (it still cracks codes in months).
  An attacker credential stuffing needs thousands of IPs to be effective.
  Applying a uniform 100 req/min limit means: MFA is unprotected (100
  guesses/min cracks TOTP in 2.7 hours), and legitimate API users are
  throttled unnecessarily.

---

## 2. Per-IP vs Per-User vs Per-Tenant

### 2.1 When Each Dimension Matters

| Dimension     | Threat Addressed              | Key Source         | Failure Mode                          |
|---------------|-------------------------------|--------------------|---------------------------------------|
| **Per-IP**    | Botnets, single-source abuse  | `X-Forwarded-For`  | NAT/proxies (shared IP, false positives) |
| **Per-User**  | Account takeover, brute force | JWT `sub` or body  | Requires authentication context       |
| **Per-Tenant**| Multi-tenant abuse, noisy neighbor | `X-Tenant-ID` | One user's attack affects whole tenant |

**Credential stuffing** requires per-IP + per-username simultaneously:
- Per-IP alone: attacker rotates IPs (botnet, residential proxies).
- Per-username alone: attacker rotates usernames (tries 1 password per account).
- Both combined: attacker must rotate BOTH IPs and usernames, raising cost
  exponentially.

**Brute force** requires per-account limiting: the attacker targets one
specific account with many password guesses. Per-IP limits are useless if the
attacker uses a distributed network.

**Multi-tenant abuse** requires per-tenant quotas: a free-tier tenant should
not be able to consume API calls at enterprise volume, even across many users
and IPs.

### 2.2 Multi-Dimensional Rate Limiter (Go)

```go
// MultiDimRateLimiter evaluates rate limits across IP, user, and tenant
// dimensions simultaneously. The most restrictive dimension wins.
type MultiDimRateLimiter struct {
    store    RateLimitStore
    policies map[string]LimitPolicy // keyed by endpoint pattern
}

// LimitPolicy defines per-dimension thresholds for one endpoint.
type LimitPolicy struct {
    IPLimit     int
    UserLimit   int
    TenantLimit int
    Window      time.Duration
}

// CheckResult describes which dimension triggered the limit (if any).
type CheckResult struct {
    Allowed     bool
    DeniedBy    string // "ip", "user", "tenant", or ""
    RetryAfter  time.Duration
    Remaining   map[string]int // remaining budget per dimension
}

// AuthContext carries the identity dimensions extracted from the request.
type AuthContext struct {
    IP       string
    UserID   string // empty for unauthenticated requests
    TenantID string
}

func (m *MultiDimRateLimiter) Check(
    ctx context.Context,
    endpoint string,
    auth AuthContext,
) CheckResult {
    policy, ok := m.policies[endpoint]
    if !ok {
        return CheckResult{Allowed: true}
    }

    now := time.Now()
    windowStart := now.Add(-policy.Window)
    result := CheckResult{Remaining: make(map[string]int)}

    dimensions := []struct {
        name   string
        key    string
        limit  int
    }{
        {"ip", fmt.Sprintf("%s:ip:%s", endpoint, auth.IP), policy.IPLimit},
        {"user", fmt.Sprintf("%s:user:%s", endpoint, auth.UserID), policy.UserLimit},
        {"tenant", fmt.Sprintf("%s:tenant:%s", endpoint, auth.TenantID), policy.TenantLimit},
    }

    for _, d := range dimensions {
        if d.limit == 0 {
            continue // skip disabled dimensions
        }
        count, err := m.store.CountAndAdd(ctx, d.key, windowStart, now)
        if err != nil {
            // Fail-open on store errors (fail-closed for auth endpoints)
            result.Remaining[d.name] = d.limit
            continue
        }
        remaining := d.limit - int(count) - 1
        if remaining < 0 {
            result.Allowed = false
            result.DeniedBy = d.name
            result.RetryAfter = policy.Window
            return result
        }
        result.Remaining[d.name] = remaining
    }

    result.Allowed = true
    return result
}
```

### 2.3 Key Construction Best Practices

Rate limit keys must be namespaced to avoid collisions across endpoints and
dimensions:

```
rate:{endpoint}:{dimension}:{id}

Examples:
  rate:login:ip:203.0.113.5
  rate:login:user:alice@example.com
  rate:login:tenant:acme-corp
  rate:reset:ip:203.0.113.5
  rate:mfa:user:user-uuid-123
```

Use a consistent prefix (`rate:`) for easy Redis `SCAN` and cleanup. Include
the endpoint in the key so that login limits do not affect reset limits.

---

## 3. Adaptive Rate Limiting

### 3.1 Escalating Limits Based on Threat Signals

Static rate limits are either too strict for trusted users or too lenient for
attackers. Adaptive rate limiting adjusts thresholds based on real-time threat
signals:

| Signal                | Source                    | Adaptation                          |
|-----------------------|---------------------------|-------------------------------------|
| Known-good user       | JWT with established trust history | Generous limits (2x baseline)  |
| Anonymous request     | No JWT                    | Strict limits (baseline)            |
| Flagged IP            | Threat intel feed, abuse history | Exponential backoff (0.5x, halving) |
| Geolocation anomaly   | GeoIP vs user's usual location | CAPTCHA challenge trigger       |
| Failed login velocity | Redis failure counter     | Progressive lockout (5→15→60 min)   |
| Device fingerprint    | Cookie/header hash        | New device = stricter limits        |

### 3.2 Adaptive Rate Limit Policy (Go)

```go
// ThreatLevel represents the assessed risk of a request.
type ThreatLevel int

const (
    ThreatTrusted   ThreatLevel = iota // Known-good user, established device
    ThreatNormal                       // Authenticated user, normal pattern
    ThreatSuspicious                   // Flagged IP, new device, high velocity
    ThreatHostile                      // Known attack patterns, banned IP
)

// AdaptiveLimit defines per-threat-level rate limit parameters.
type AdaptiveLimit struct {
    Multiplier  float64       // applied to baseline limit
    Window      time.Duration // window for this level
    Lockout     time.Duration // progressive lockout duration
    RequireCAPTCHA bool       // force CAPTCHA challenge
}

// AdaptivePolicy maps threat levels to rate limit behavior.
type AdaptivePolicy struct {
    BaselineLimit int
    Limits        map[ThreatLevel]AdaptiveLimit
}

func DefaultAdaptivePolicy() *AdaptivePolicy {
    return &AdaptivePolicy{
        BaselineLimit: 10, // login attempts per minute
        Limits: map[ThreatLevel]AdaptiveLimit{
            ThreatTrusted:    {Multiplier: 2.0, Window: time.Minute, Lockout: 0, RequireCAPTCHA: false},
            ThreatNormal:     {Multiplier: 1.0, Window: time.Minute, Lockout: 5 * time.Minute, RequireCAPTCHA: false},
            ThreatSuspicious: {Multiplier: 0.3, Window: time.Minute, Lockout: 15 * time.Minute, RequireCAPTCHA: true},
            ThreatHostile:    {Multiplier: 0.0, Window: time.Minute, Lockout: time.Hour, RequireCAPTCHA: true},
        },
    }
}

// AdaptiveRateLimiter combines threat assessment with dynamic rate limiting.
type AdaptiveRateLimiter struct {
    policy  *AdaptivePolicy
    store   RateLimitStore
    threat  ThreatAssessor
}

// ThreatAssessor evaluates request context to determine threat level.
type ThreatAssessor interface {
    Assess(ctx context.Context, ip, userID string, headers http.Header) ThreatLevel
}

// Check evaluates the adaptive rate limit for a login attempt.
func (a *AdaptiveRateLimiter) Check(
    ctx context.Context,
    ip, userID string,
    r *http.Request,
) (allowed bool, captchaRequired bool, retryAfter time.Duration) {
    level := a.threat.Assess(ctx, ip, userID, r.Header)
    limit, ok := a.policy.Limits[level]
    if !ok {
        limit = a.policy.Limits[ThreatNormal]
    }

    effectiveLimit := int(float64(a.policy.BaselineLimit) * limit.Multiplier)
    if effectiveLimit == 0 {
        // Hostile: deny outright
        return false, true, limit.Lockout
    }

    key := fmt.Sprintf("adaptive:login:%s", ip)
    windowStart := time.Now().Add(-limit.Window)
    count, _ := a.store.CountAndAdd(ctx, key, windowStart, time.Now())

    if int(count) >= effectiveLimit {
        // Escalate: if already at suspicious level and limit hit, escalate to lockout
        return false, limit.RequireCAPTCHA, limit.Lockout
    }

    // Require CAPTCHA even when under limit, if threat level demands it
    return true, limit.RequireCAPTCHA, 0
}
```

### 3.3 Geolocation Anomaly Detection

```go
// GeoThreatAssessor checks if the request IP is geographically implausible
// relative to the user's recent login locations.
type GeoThreatAssessor struct {
    store   RateLimitStore
    geoip   GeoIPResolver
    maxKmH  float64 // max plausible travel speed (e.g., 900 km/h for flights)
}

func (g *GeoThreatAssessor) Assess(
    ctx context.Context,
    ip, userID string,
    headers http.Header,
) ThreatLevel {
    if userID == "" {
        return ThreatNormal // anonymous — no history to compare
    }

    currentLoc, err := g.geoip.Lookup(ip)
    if err != nil {
        return ThreatNormal // can't assess, default to normal
    }

    // Get the user's last known location and timestamp
    lastLocKey := fmt.Sprintf("geo:last:%s", userID)
    lastLocStr, err := g.store.Get(ctx, lastLocKey)
    if err != nil || lastLocStr == "" {
        // First login — record location, no anomaly
        g.store.Set(ctx, lastLocKey, fmt.Sprintf("%f,%f,%d", currentLoc.Lat, currentLoc.Lon, time.Now().Unix()))
        return ThreatNormal
    }

    // Parse last location and check travel feasibility
    // ... distance/time calculation ...
    // If impossible travel detected: return ThreatSuspicious
    // If flagged IP list match: return ThreatHostile

    return ThreatNormal
}
```

**Impossible travel example**: User logs in from New York at 10:00 UTC, then
from Tokyo at 10:15 UTC. The 10,880 km distance in 15 minutes implies
43,520 km/h — impossible. This should trigger ThreatSuspicious and require
CAPTCHA or MFA step-up.

---

## 4. CAPTCHA Escalation

### 4.1 When to Trigger CAPTCHA

CAPTCHA should not be the first line of defense — it degrades UX and
accessibility. Use it as a graduated response:

| Failed Attempts | Response                           |
|-----------------|------------------------------------|
| 0-2             | Normal flow                        |
| 3-4             | Sliding-window rate limit (429)    |
| 5+              | CAPTCHA required before next attempt |
| 10+             | Account lockout + email notification |

```go
// CAPTCHAPolicy defines when CAPTCHA is required based on failure count.
type CAPTCHAPolicy struct {
    Threshold    int           // failures before CAPTCHA required
    LockoutAfter int           // failures before account lockout
    Window       time.Duration // failure counting window
}

// RequiresCAPTCHA checks if the given identity must solve a CAPTCHA.
func (p *CAPTCHAPolicy) RequiresCAPTCHA(
    ctx context.Context,
    store RateLimitStore,
    identity string,
) bool {
    key := fmt.Sprintf("captcha:fail:%s", identity)
    count, _ := store.Count(ctx, key, p.Window)
    return int(count) >= p.Threshold
}
```

### 4.2 reCAPTCHA v3 vs hCaptcha

| Feature               | reCAPTCHA v3                     | hCaptcha                      |
|-----------------------|----------------------------------|-------------------------------|
| User interaction      | None (score-based)               | None (enterprise) or visual   |
| Score                 | 0.0-1.0 per request              | Binary (pass/fail)            |
| Privacy               | Google ToS, data sharing concerns | Privacy-focused, GDPR aligned |
| Threshold tuning      | Configurable per action           | Less granular                 |
| Cost                  | Free (enterprise tier available)  | Free (enterprise tier available) |
| Accessibility         | No user challenge (best)          | Visual challenge (worse)      |

**reCAPTCHA v3** is preferred for IAM because it never interrupts legitimate
users — it returns a score that the server evaluates against a threshold.
For login: `score >= 0.5 → allow`, `0.1-0.5 → require MFA step-up`,
`< 0.1 → deny`.

### 4.3 Server-Side CAPTCHA Verification in Go

```go
// CAPTCHAVerifier validates CAPTCHA tokens server-side.
type CAPTCHAVerifier interface {
    Verify(ctx context.Context, token, remoteIP string) (score float64, err error)
}

// reCAPTCHAV3 verifies Google reCAPTCHA v3 tokens.
type reCAPTCHAV3 struct {
    secretKey string
    client    *http.Client
    minScore  float64
}

type recaptchaResponse struct {
    Success    bool     `json:"success"`
    Score      float64  `json:"score"`
    Action     string   `json:"action"`
    ErrorCodes []string `json:"error-codes"`
    Hostname   string   `json:"hostname"`
}

func (r *reCAPTCHAV3) Verify(ctx context.Context, token, remoteIP string) (float64, error) {
    if token == "" {
        return 0, fmt.Errorf("missing captcha token")
    }

    form := url.Values{
        "secret":   {r.secretKey},
        "response": {token},
        "remoteip": {remoteIP},
    }

    req, _ := http.NewRequestWithContext(ctx, "POST",
        "https://www.google.com/recaptcha/api/siteverify",
        strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := r.client.Do(req)
    if err != nil {
        return 0, fmt.Errorf("recaptcha verify request failed: %w", err)
    }
    defer resp.Body.Close()

    var result recaptchaResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return 0, fmt.Errorf("recaptcha response decode failed: %w", err)
    }

    if !result.Success {
        return 0, fmt.Errorf("recaptcha verification failed: %v", result.ErrorCodes)
    }

    return result.Score, nil
}

// VerifyWithThreshold returns nil if score meets threshold, error otherwise.
func (r *reCAPTCHAV3) VerifyWithThreshold(ctx context.Context, token, remoteIP string, action string) error {
    score, err := r.Verify(ctx, token, remoteIP)
    if err != nil {
        return err
    }
    if score < r.minScore {
        return fmt.Errorf("recaptcha score %.2f below threshold %.2f", score, r.minScore)
    }
    return nil
}
```

### 4.4 Integration with Rate Limiter — CAPTCHA Bypass

After a user solves a CAPTCHA, they should receive a temporary bypass token
that suppresses CAPTCHA challenges for a short window. This prevents
re-challenging on every request.

```go
// CAPTCHABypassStore manages CAPTCHA bypass tokens.
type CAPTCHABypassStore struct {
    store RateLimitStore
    ttl   time.Duration // e.g., 30 minutes
}

// IssueBypass creates a bypass token after successful CAPTCHA verification.
// The token is stored in Redis with a TTL and returned as a signed cookie.
func (c *CAPTCHABypassStore) IssueBypass(ctx context.Context, ip string) (string, error) {
    token := uuid.NewString()
    key := fmt.Sprintf("captcha:bypass:%s", token)
    if err := c.store.Set(ctx, key, ip, c.ttl); err != nil {
        return "", err
    }
    // Also record per-IP bypass to prevent token sharing across IPs
    ipKey := fmt.Sprintf("captcha:bypass:ip:%s", ip)
    c.store.Set(ctx, ipKey, token, c.ttl)
    return token, nil
}

// HasValidBypass checks if the request has a valid CAPTCHA bypass token
// that matches the requesting IP.
func (c *CAPTCHABypassStore) HasValidBypass(ctx context.Context, token, ip string) bool {
    if token == "" {
        return false
    }
    key := fmt.Sprintf("captcha:bypass:%s", token)
    storedIP, err := c.store.Get(ctx, key)
    if err != nil || storedIP != ip {
        return false // token not found or IP mismatch
    }
    return true
}
```

**Middleware integration**:

```go
func CAPTCHAMiddleware(
    verifier CAPTCHAVerifier,
    bypass   *CAPTCHABypassStore,
    policy   *CAPTCHAPolicy,
    store    RateLimitStore,
) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := ClientIP(r)

            // Check if CAPTCHA bypass is active for this IP
            bypassToken, _ := r.Cookie("captcha_bypass")
            if bypassToken != nil && bypass.HasValidBypass(r.Context(), bypassToken.Value, ip) {
                next.ServeHTTP(w, r)
                return
            }

            // Check if CAPTCHA is required based on failure count
            identity := ip // or userID if authenticated
            if !policy.RequiresCAPTCHA(r.Context(), store, identity) {
                next.ServeHTTP(w, r)
                return
            }

            // CAPTCHA required — verify token from request header
            captchaToken := r.Header.Get("X-Captcha-Token")
            score, err := verifier.Verify(r.Context(), captchaToken, ip)
            if err != nil || score < 0.5 {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusTooManyRequests)
                w.Header().Set("X-Captcha-Required", "true")
                json.NewEncoder(w).Encode(map[string]string{
                    "error":   "captcha_required",
                    "message": "Please complete the CAPTCHA challenge",
                })
                return
            }

            // CAPTCHA passed — issue bypass token
            token, _ := bypass.IssueBypass(r.Context(), ip)
            http.SetCookie(w, &http.Cookie{
                Name:     "captcha_bypass",
                Value:    token,
                MaxAge:   int((30 * time.Minute).Seconds()),
                HttpOnly: true,
                Secure:   true,
                SameSite: http.SameSiteStrictMode,
            })

            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 5. Distributed Rate Limiting with Redis

> The generic sliding-window Lua script is documented in
> `rate-limiting-strategies.md`. This section focuses on IAM-specific concerns:
> atomic multi-key operations, fail-open vs fail-closed decisions, and Redis
> outage resilience.

### 5.1 Atomic Multi-Key Rate Check for Login

A credential-stuffing defense requires checking IP + username + tenant counters
atomically. If the checks are not atomic, a race condition allows two concurrent
requests to both pass the limit check before either increments.

```lua
-- iam_login_check.lua
-- Atomically checks IP, username, and tenant rate limits for login.
-- Returns: 1 if allowed, 0 if denied (with which key denied in ARGV).
-- KEYS[1] = ip key       KEYS[2] = username key    KEYS[3] = tenant key
-- ARGV[1] = ip limit     ARGV[2] = username limit   ARGV[3] = tenant limit
-- ARGV[4] = now (unix ms) ARGV[5] = window ms       ARGV[6] = request id

local now = tonumber(ARGV[4])
local window = tonumber(ARGV[5])
local window_start = now - window

-- Check each dimension
for i = 1, 3 do
    local key = KEYS[i]
    local limit = tonumber(ARGV[i])

    -- Remove expired entries
    redis.call('ZREMRANGEBYSCORE', key, '-inf', '(' .. window_start)

    local count = redis.call('ZCARD', key)
    if count >= limit then
        -- Denied by this dimension
        local retry = redis.call('ZSCORE', key, redis.call('ZRANGE', key, 0, 0)[1])
        if retry then
            return {0, i, math.ceil((tonumber(retry) + window - now) / 1000)}
        end
        return {0, i, math.ceil(window / 1000)}
    end
end

-- All dimensions passed — add entry to each
local req_id = ARGV[6]
for i = 1, 3 do
    local key = KEYS[i]
    local member = req_id .. ':' .. i
    redis.call('ZADD', key, now, member)
    redis.call('EXPIRE', key, math.ceil(window / 1000) + 10)
end

return {1, 0, 0}
```

```go
// DistributedLoginLimiter uses Redis Lua for atomic multi-dimension login rate limiting.
type DistributedLoginLimiter struct {
    rdb        *redis.Client
    luaScript  *redis.Script
    ipLimit    int
    userLimit  int
    tenantLimit int
    window     time.Duration
}

func NewDistributedLoginLimiter(rdb *redis.Client) *DistributedLoginLimiter {
    return &DistributedLoginLimiter{
        rdb:         rdb,
        luaScript:   redis.NewScript(iamLoginCheckLua),
        ipLimit:     10,
        userLimit:   5,
        tenantLimit: 100,
        window:      time.Minute,
    }
}

func (l *DistributedLoginLimiter) Check(
    ctx context.Context,
    ip, username, tenantID string,
) (allowed bool, deniedBy string, retryAfterSec int) {
    keys := []string{
        fmt.Sprintf("login:ip:%s", ip),
        fmt.Sprintf("login:user:%s", username),
        fmt.Sprintf("login:tenant:%s", tenantID),
    }

    result, err := l.luaScript.Run(ctx, l.rdb, keys,
        l.ipLimit, l.userLimit, l.tenantLimit,
        time.Now().UnixMilli(),
        l.window.Milliseconds(),
        uuid.NewString(),
   ).Slice()

    if err != nil {
        // FAIL-OPEN on Redis errors for general endpoints
        // FAIL-CLOSED for critical auth endpoints (configurable)
        return true, "", 0
    }

    allowed = result[0].(int64) == 1
    if !allowed {
        dimIdx := int(result[1].(int64))
        dims := []string{"ip", "user", "tenant"}
        deniedBy = dims[dimIdx-1]
        retryAfterSec = int(result[2].(int64))
    }
    return
}
```

### 5.2 Fail-Open vs Fail-Closed

When Redis is unavailable, the rate limiter must decide: allow all traffic
(fail-open) or deny all traffic (fail-closed). The right choice depends on the
endpoint's risk profile:

| Endpoint           | Fail Mode    | Rationale                                                    |
|--------------------|-------------|--------------------------------------------------------------|
| Login              | Fail-closed  | An auth bypass during Redis outage enables brute force       |
| Password reset     | Fail-closed  | Email bombing during outage is high-impact                   |
| API read endpoints | Fail-open    | Availability > security for read operations                  |
| Registration       | Fail-open    | Blocking registration during outage blocks all new users     |
| MFA verification   | Fail-closed  | MFA bypass is critical security failure                      |

```go
// FailMode controls behavior when the rate limit store is unavailable.
type FailMode int

const (
    FailOpen   FailMode = iota // allow on store error
    FailClosed                 // deny on store error
)

func (l *DistributedLoginLimiter) CheckWithFailMode(
    ctx context.Context,
    ip, username, tenantID string,
    mode FailMode,
) (allowed bool) {
    allowed, _, _ = l.Check(ctx, ip, username, tenantID)
    // Check() already fails-open internally on Redis errors.
    // For fail-closed, override:
    if mode == FailClosed {
        // Use a circuit breaker pattern to detect Redis outage
        if l.redisHealthy(ctx) {
            return allowed
        }
        return false // Redis down → deny
    }
    return allowed
}

func (l *DistributedLoginLimiter) redisHealthy(ctx context.Context) bool {
    _, err := l.rdb.Ping(ctx).Result()
    return err == nil
}
```

### 5.3 Redis Outage Graceful Degradation

During a Redis outage, the system should degrade to in-memory rate limiting
rather than either allowing everything or denying everything:

```go
// HybridRateLimiter uses Redis as primary and falls back to in-memory
// when Redis is unavailable. This provides defense-in-depth: even during
// a Redis outage, rate limiting continues at the local-instance level.
type HybridRateLimiter struct {
    primary   RateLimitStore // Redis
    fallback  RateLimitStore // in-memory
    failMode  FailMode
}

func (h *HybridRateLimiter) CountAndAdd(
    ctx context.Context,
    key string,
    windowStart, now time.Time,
) (int64, error) {
    count, err := h.primary.CountAndAdd(ctx, key, windowStart, now)
    if err == nil {
        // Sync to fallback for continuity if Redis recovers
        h.fallback.CountAndAdd(ctx, key, windowStart, now)
        return count, nil
    }

    // Redis error — use fallback
    return h.fallback.CountAndAdd(ctx, key, windowStart, now)
}
```

---

## 6. 429 Response Best Practices for IAM

### 6.1 RFC 6585 and Draft RateLimit Headers

RFC 6585 Section 4 defines HTTP 429 "Too Many Requests". The `Retry-After`
header is mandatory. The `RateLimit-*` headers follow
[draft-ietf-httpapi-ratelimit-headers](https://datatracker.ietf.org/doc/draft-ietf-httpapi-ratelimit-headers/).

```
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 60
X-RateLimit-Limit: 5
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1700000060
RateLimit-Policy: 5;w=60
RateLimit: limit=5, remaining=0, reset=60
```

### 6.2 Distinguishing Rate Limit Reasons

IAM systems must distinguish between different 429 causes so that clients can
respond appropriately:

| Cause                | `error_code`              | Client Action                    |
|----------------------|---------------------------|----------------------------------|
| Rate limited (retry) | `rate_limited`            | Wait `Retry-After`, then retry   |
| Too many sessions    | `too_many_sessions`       | Log out from another device      |
| Suspicious activity  | `suspicious_activity`     | Contact admin, verify identity   |
| CAPTCHA required     | `captcha_required`        | Render CAPTCHA, retry with token |
| Account locked       | `account_locked`          | Wait lockout period or reset     |

```go
// WriteIAM429 writes a structured 429 response with IAM-specific error codes.
func WriteIAM429(
    w http.ResponseWriter,
    errorCode string,
    message string,
    retryAfter int,
    details map[string]any,
) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
    w.WriteHeader(http.StatusTooManyRequests)

    body := map[string]any{
        "error":       "too_many_requests",
        "error_code":  errorCode,
        "message":     message,
        "retry_after": retryAfter,
    }
    for k, v := range details {
        body[k] = v
    }
    json.NewEncoder(w).Encode(body)
}

// Example usage in login handler:
// WriteIAM429(w, "captcha_required",
//     "Too many failed attempts. Please complete CAPTCHA.",
//     60,
//     map[string]any{"captcha_action": "login"})
```

### 6.3 Response Body Format

```json
{
  "error": "too_many_requests",
  "error_code": "rate_limited",
  "message": "Too many login attempts from this IP address. Please try again later.",
  "retry_after": 60,
  "details": {
    "limit": 10,
    "window_seconds": 60,
    "remaining": 0,
    "reset_at": "2024-01-01T00:01:00Z"
  }
}
```

### 6.4 Security Considerations for 429 Responses

1. **Never reveal the exact limit in the error body** for auth endpoints.
   Attackers use this to calibrate their attack rate just under the threshold.
   Return `Retry-After` but omit `X-RateLimit-Limit` for login/reset endpoints.
2. **Return identical response for existing/non-existing accounts** to prevent
   enumeration.
3. **Log all 429 responses** with full context (IP, user-agent, path) for
   attack analysis.
4. **Consider jitter on `Retry-After`** to prevent synchronized retry storms.
   If 1000 clients all get `Retry-After: 60` and retry simultaneously, the
   server gets a thundering herd. Add `Retry-After: 60` + random(0, 15).

---

## 7. GGID Gateway Rate Limit Audit

### 7.1 Existing Rate Limiting Implementations

GGID's gateway middleware package contains **five distinct rate limiting
implementations**, but a critical finding is that **none are wired into the
production request pipeline**.

#### 7.1.1 `ratelimit.go` — Fixed-Window Per-Endpoint Limiter

**Status**: Implemented but NOT in the `Handler()` middleware chain.

- Algorithm: Fixed window (count + expire)
- Key: `{path}:{ip}` — per-endpoint + per-IP
- Endpoint-specific limits:
  - `/api/v1/auth/login`: 5 req/min
  - `/api/v1/auth/register`: 3 req/min
  - `/api/v1/api/v1/*`: 100 req/min
- Skips: `/healthz`, `/docs`, `/api-docs`, `/login`, `/register`
- 429 headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`, `Retry-After`

**Gaps**:
- `bucketKey()` uses `r.RemoteAddr` directly (with port) instead of `ClientIP()` —
  the key includes the port number, meaning the same IP with different ephemeral
  ports gets separate buckets. This makes per-IP limiting ineffective.
- Only 3 endpoints are configured. Missing: `/api/v1/auth/password/forgot`,
  `/api/v1/auth/password/reset`, `/oauth/token`, `/api/v1/auth/mfa/*`, SCIM endpoints.
- No per-username or per-tenant dimension.
- In-memory only — no distributed coordination.

#### 7.1.2 `token_bucket.go` — Token Bucket Per-Tenant Limiter

**Status**: Implemented but NOT in the `Handler()` middleware chain.

- Algorithm: Token bucket
- Key: `{tenantID}:{ip}` — per-tenant + per-IP
- Tier-based overrides: free (20 burst, 2/s), pro (100 burst, 10/s), enterprise (1000 burst, 100/s)
- Uses `ClientIP()` correctly (extracts real IP from headers)
- Background cleanup goroutine
- 429 headers: `Retry-After`, `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Tier`

**Gaps**:
- Does NOT set `X-RateLimit-Reset` — clients cannot compute when the window resets.
- No `RateLimit-Policy` header.
- No per-username dimension.
- In-memory only.

#### 7.1.3 `sliding_ratelimit.go` — Sliding Window with Redis Backend

**Status**: Implemented with Redis support but NOT in the `Handler()` middleware chain.

- Algorithm: Sliding window (sorted set)
- Key: `ratelimit:{tenantID}:{tier}` — per-tenant + per-tier
- Redis-backed (`RedisRateLimitStore`) with in-memory fallback
- Tier limits: free (100/min), starter (500/min), pro (5000/min), enterprise (unlimited)
- Lua script for atomic `ZREMRANGEBYSCORE` + `ZCARD` + `ZADD`
- Fail-open on Redis errors
- Full rate limit headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`, `X-RateLimit-Tier`, `Retry-After`

**Gaps**:
- No per-IP or per-username dimension — only per-tenant.
- No endpoint-specific limits — all endpoints share the same tenant-tier quota.
- Enterprise tier is unlimited (0 requests = skip) — no upper bound.

#### 7.1.4 `tier_ratelimit.go` — Fixed-Window Per-Tenant Limiter

**Status**: Implemented but NOT in the `Handler()` middleware chain.

- Algorithm: Fixed window
- Key: `{tenantID}` — per-tenant only (no IP dimension)
- Tier-based: free (100/min), pro (1000/min), enterprise (unlimited)
- 429 headers: `Retry-After`, `X-RateLimit-Limit`, `X-RateLimit-Tier` (missing `Remaining` and `Reset`)

**Gaps**:
- No IP dimension — all users in a tenant share one counter.
- No endpoint-specific limits.
- Missing `X-RateLimit-Remaining` and `X-RateLimit-Reset` headers.
- In-memory only.

#### 7.1.5 `botdetect.go` — Behavioral Bot Detection

**Status**: Implemented but NOT in the `Handler()` middleware chain.

- `BotDetect`: Static User-Agent pattern matching (blocks sqlmap, nikto, etc.)
- `BehavioralBotDetect`: Per-IP request rate threshold
- 429 response with `Retry-After`

**Gaps**:
- Static pattern list is trivially bypassed.
- No CAPTCHA integration — just blocks.
- No learning/adaptation.
- In-memory only.

### 7.2 Middleware Chain Analysis

The production request pipeline is defined in `router.go:Handler()`:

```
PanicRecovery → CORS → RequestID → RequestLogger → TenantResolver → JWTAuth → ServeHTTP
```

**None of the five rate limiters appear in this chain.** This means:

1. **All auth endpoints (login, register, password reset, OAuth, SAML) have
   ZERO rate limiting in production.** They are public paths that skip JWT
   verification, and no rate limiter runs before them.
2. **All API endpoints have ZERO rate limiting in production.** Even the
   token bucket and sliding window implementations are never applied.
3. **Bot detection is never applied.** The `BotDetect` and
   `BehavioralBotDetect` middleware are not in the chain.

The limiters exist as standalone middleware with full test coverage, but they
are never composed into the actual handler pipeline. This is the single most
critical security gap in the gateway.

### 7.3 Auth Endpoint Protection Status

| Endpoint                          | Rate Limited? | Per-IP? | Per-User? | Per-Tenant? |
|-----------------------------------|:------------:|:-------:|:---------:|:-----------:|
| `POST /api/v1/auth/login`         | No           | —       | —         | —           |
| `POST /api/v1/auth/register`      | No           | —       | —         | —           |
| `POST /api/v1/auth/refresh`       | No           | —       | —         | —           |
| `POST /api/v1/auth/password/forgot` | No         | —       | —         | —           |
| `POST /api/v1/auth/password/reset`| No           | —       | —         | —           |
| `POST /oauth/token`               | No           | —       | —         | —           |
| `POST /api/v1/auth/mfa/*`         | No           | —       | —         | —           |
| `GET /api/v1/*` (authenticated)   | No           | —       | —         | —           |

**Note**: The `ratelimit.go` middleware is designed to protect login and
register endpoints with 5/min and 3/min limits respectively. But since it is
not in the `Handler()` chain, these limits are never enforced.

### 7.4 Session Rate Limiting

`session.go` implements session validation via Redis, but it does not enforce
session count limits. There is no "maximum concurrent sessions per user"
control, meaning session fixation attacks can create unlimited sessions.

---

## 8. Gap Analysis & Recommendations

### 8.1 Critical Gaps

| # | Gap | Severity | Impact |
|---|-----|----------|--------|
| 1 | **No rate limiter in production pipeline** | Critical | All endpoints exposed to brute force, credential stuffing, and flooding |
| 2 | **No per-username rate limiting** | Critical | Brute force against individual accounts is uncontrolled |
| 3 | **No MFA rate limiting** | Critical | 6-digit TOTP can be brute-forced without throttling |
| 4 | **No CAPTCHA escalation** | High | No graduated challenge for suspicious behavior |
| 5 | **No adaptive rate limiting** | High | Static limits cannot respond to attack patterns |
| 6 | **`ratelimit.go` bucket key includes port** | High | Per-IP limits are bypassable via ephemeral port rotation |
| 7 | **Missing endpoint coverage** | High | Password reset, OAuth token, MFA, SCIM endpoints unprotected |
| 8 | **No distributed state for auth endpoints** | Medium | Multi-instance deployment multiplies effective limits |
| 9 | **Inconsistent 429 response format** | Medium | Each limiter returns different headers and body format |
| 10 | **No fail-closed mode for auth endpoints** | Medium | Redis outage = no rate limiting at all |

### 8.2 Implementation Roadmap

#### Action 1: Wire Rate Limiting into Production Pipeline (Effort: 1 day, Priority: P0)

Insert the existing `RateLimiter` (fixed-window) and `TenantBucketLimiter`
(token bucket) into the `Handler()` middleware chain in `router.go`, positioned
BEFORE JWTAuth so that public endpoints are also protected:

```go
// In router.go Handler():
handler := middleware.JWTAuth(...)(gw)  // inner
handler = rl.Middleware(handler)         // rate limit (before JWT so public paths covered)
handler = middleware.TenantResolver(...)(handler)
handler = middleware.RequestLogger(logger)(handler)
handler = middleware.RequestID(handler)
handler = middleware.CORS(handler)
handler = middleware.PanicRecovery(logger)(handler)
```

**Also fix**: Replace `r.RemoteAddr` with `ClientIP(r)` in `ratelimit.go:bucketKey()`.

#### Action 2: Add Per-Username and Per-Tenant Dimensions (Effort: 2-3 days, Priority: P0)

Implement the `MultiDimRateLimiter` from Section 2.2. For authenticated
endpoints, extract the JWT `sub` claim; for login, extract the `username` from
the request body (pre-authentication). Add MFA-specific limits (5 attempts /
5 minutes per user).

#### Action 3: Expand Endpoint Coverage (Effort: 1 day, Priority: P0)

Add rate limit policies for all auth endpoints:

```go
func (rl *RateLimiter) getLimit(path string) int {
    switch {
    case strings.HasPrefix(path, "/api/v1/auth/login"):
        return 5       // brute force defense
    case strings.HasPrefix(path, "/api/v1/auth/register"):
        return 3       // account creation abuse
    case strings.HasPrefix(path, "/api/v1/auth/password/"):
        return 3       // reset flooding
    case strings.HasPrefix(path, "/api/v1/auth/mfa/"):
        return 5       // MFA brute force
    case strings.HasPrefix(path, "/oauth/token"):
        return 30      // token flooding
    case strings.HasPrefix(path, "/api/v1/scim/"):
        return 100     // SCIM provisioning
    case strings.HasPrefix(path, "/api/v1/"):
        return 100     // general API
    default:
        return 0
    }
}
```

#### Action 4: Implement Distributed Auth Rate Limiting with Redis (Effort: 3-4 days, Priority: P1)

Use the `SlidingWindowLimiter` with `RedisRateLimitStore` (already implemented)
for the auth dimension. Implement the multi-key Lua script from Section 5.1 for
atomic IP + username + tenant checks. Configure fail-closed for login and MFA
endpoints, fail-open for read endpoints.

#### Action 5: Add CAPTCHA Escalation for Login (Effort: 2 days, Priority: P1)

Integrate reCAPTCHA v3 server-side verification (Section 4.3). Trigger after
3 failed login attempts per username. Issue bypass cookie after successful
CAPTCHA. Add `X-Captcha-Required: true` header to 429 responses to signal
the frontend.

### 8.3 Summary

GGID has five rate limiting implementations with good algorithmic foundations
and test coverage, but **none are wired into the production request pipeline**.
The highest-impact action is to connect the existing limiters to the `Handler()`
chain and fix the per-port bucket key bug. This takes one day and immediately
closes the critical gap of unprotected auth endpoints. Subsequent work should
add per-username dimensions (for brute force defense), MFA-specific limits,
and Redis-backed distributed enforcement for multi-instance deployments.

The sliding window limiter with Redis backend (`sliding_ratelimit.go`) is the
most production-ready implementation — it already has fail-open Redis handling,
Lua atomicity, and complete rate limit headers. It should be the primary
limiter once wired in, supplemented by endpoint-specific fixed-window limits
for auth endpoints.
