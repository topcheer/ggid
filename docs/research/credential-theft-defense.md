# Credential Theft Defense

> Research document for GGID IAM — credential attack detection, prevention, and automated response.
>
> **Scope:** Password-based credential attacks (stuffing, spraying, breach reuse) and defense layers.
> Token theft defense is covered in [token-lifecycle-security.md](./token-lifecycle-security.md).
> Session defense is covered in [session-management-design.md](./session-management-design.md).

---

## 1. Overview

Credential attacks are the most common initial access vector in data breaches. The Verizon DBIR consistently reports that over 80% of web-application breaches involve stolen or weak credentials. Identity and Access Management systems are the primary target — a single set of valid credentials unlocks account takeover, lateral movement, and data exfiltration.

### Attack Taxonomy

| Attack Type | Mechanism | Volume per Account | Volume Overall |
|---|---|---|---|
| **Credential Stuffing** | Breached username/password pairs tried at scale | Low (1-3) | Very high (10K+) |
| **Password Spraying** | Common passwords tried against many accounts | Very low (1-2) | Medium (hundreds) |
| **Phishing** | Social engineering to harvest credentials | N/A | Targeted |
| **Interception** | MITM, keylogger, or network sniffing | N/A | Targeted |
| **Brute Force** | Exhaustive password space enumeration | High | Low |

### Defense Layers

1. **Prevention** — stop the attack before it reaches password verification (rate limiting, CAPTCHA, IP blocking)
2. **Detection** — identify attack patterns in real time (correlation, anomaly scoring)
3. **Response** — automate containment (lockout, MFA step-up, alert)

---

## 2. Credential Stuffing Detection

### Attack Pattern

Automated tools (e.g., Sentry MBA, OpenBullet) feed username/password pairs harvested from breach databases against the login endpoint. Attackers rotate IPs via botnets and residential proxies to evade per-IP rate limits, keeping the per-account attempt count low to avoid triggering per-account lockout.

### Detection Signals

| Signal | Description | Threshold (example) |
|---|---|---|
| Failed-login flood from single IP | Many different usernames from one IP | >20 unique usernames / 5 min |
| Distributed low-volume pattern | Many IPs, each few attempts, correlated timing | >10 IPs hitting same endpoint within 1 min |
| User-agent anomaly | curl, python-requests, headless Chrome, Burp Suite | Non-browser UA on auth endpoints |
| Timing regularity | Exactly evenly spaced requests (bot signature) | stddev < 100ms between requests |
| Near-zero success rate | High volume, 0% success → probe/stuffing attack | <0.1% success, >100 attempts |
| Credential pair reuse | Same password tried with different usernames | Detect via hash correlation |

### GGID Mitigations

GGID already implements several stuffing-relevant controls:

- **Token-bucket rate limiting** (`gateway/middleware/token_bucket.go`): per-IP burst capacity with sustained refill rate. The `Allow()` method consumes one token per request; exhaustion returns 429.
- **Redis-backed fixed-window limiter** (`auth/service/ratelimit_service.go`): `CheckAndIncrement` provides distributed rate limiting across gateway replicas.
- **Bot detection** (`gateway/middleware/botdetect.go`): blocks suspicious user-agents (sqlmap, nikto, nmap, hydra, burp) and tags known crawlers.
- **Behavioral bot detection** (`gateway/middleware/botdetect.go`): sliding-window per-IP request counting with configurable threshold.
- **Login attempt logging** (`auth/service/login_attempt.go`): records every attempt (username, IP, UA, success, failure reason) to Redis sorted sets for forensic correlation.

### Proposed: StuffingDetector

A correlation engine that goes beyond per-IP rate limiting to detect distributed attacks:

```go
// StuffingDetector correlates login_failed patterns across IP ranges.
type StuffingDetector struct {
    rdb            *redis.Client
    window         time.Duration // 5 min
    ipThreshold    int           // unique usernames per IP
    rangeThreshold int           // failures per /24 to block range
}

func (d *StuffingDetector) RecordFailure(ctx context.Context, ip, username string) {
    // Per-IP unique username counter (HyperLogLog).
    d.rdb.PFAdd(ctx, fmt.Sprintf("ggid:stuffing:ip:%s", ip), username)
    // Per /24 aggregate.
    d.rdb.Incr(ctx, fmt.Sprintf("ggid:stuffing:range:%s", ipRange24(ip)))
}
```

---

## 3. Password Spraying Prevention

### Attack Pattern

The attacker selects a small set of common passwords (e.g., `Spring2024!`, `Welcome1#`, `P@ssw0rd`) and tries each against many accounts. Per-account attempt count stays at 1-2 — below the typical lockout threshold of 5. Requests are time-spaced over hours to avoid per-IP rate limits.

### Detection Signals

| Signal | Description |
|---|---|
| Same password hash across many usernames | Hash incoming passwords and correlate |
| Login_failed events for distinct users from same IP | >50 unique usernames from one IP in 1 hour |
| Password in common-password list | Match against top-10K list |
| Evenly spaced attempts across different accounts | Bot timing signature |

### Mitigations

#### Per-Account Exponential Lockout

GGID already has two complementary lockout mechanisms:

1. **`AccountLockoutService`** (`email_lockout.go`): Redis-backed counter per `(tenantID, identifier)`. After `maxAttempts` (default 5) within the TTL window, `IsLocked()` returns true. TTL resets on each failure.

2. **`RecordFailedLoginAnomaly`** (`anomaly_detection.go`): Redis sorted-set tracker with a 15-minute sliding window. After 5 failures, the account is locked for 15 minutes.

**Gap:** Neither implements exponential backoff. A fixed 15-minute lock resets after expiry, allowing another 5 attempts. Proposed enhancement:

```go
// ExponentialBackoffLockout escalates lock duration on repeated failures.
func (l *ExponentialBackoffLockout) GetDelay(identifier string) time.Duration {
    count, _ := l.rdb.Get(ctx, "ggid:lockout:count:"+identifier).Int()
    delay := l.baseDelay * time.Duration(1<<count) // 1m, 2m, 4m, 8m...
    if delay > l.maxDelay { delay = l.maxDelay }
    return delay
}
```

#### Login Time Randomization

Add 200-500ms random delay to auth responses. This defeats timing-based enumeration and slows automated tools without noticeably affecting legitimate users:

```go
// In auth handler, before responding:
delay := 200 + rand.Intn(300) // 200-500ms
time.Sleep(time.Duration(delay) * time.Millisecond)
```

#### Password Blacklist

`PasswordService.Validate()` already checks against a configurable `policy.Blacklist` (`password_service.go:82`). This catches known-weak passwords but not the full breach corpus (see Section 4).

---

## 4. Breach Corpus Matching

### k-Anonymity HIBP API

GGID already implements the HaveIBeenPwned k-anonymity range query in `password_breach.go`:

```go
// CheckPasswordBreach queries HIBP using only first 5 chars of SHA-1 hash.
func (ps *PasswordService) CheckPasswordBreach(ctx context.Context, password string) error {
    h := sha1.Sum([]byte(password))
    hash := strings.ToUpper(hex.EncodeToString(h[:]))
    prefix := hash[:5]  // Only 5 chars sent to API
    suffix := hash[5:]

    resp, err := http.Get("https://api.pwnedpasswords.com/range/" + prefix)
    // ... parse "SUFFIX:COUNT" lines, match local suffix
}
```

**Design:** The password never leaves the server. Only the first 5 hex characters of its SHA-1 hash are sent. HIBP returns ~500 hash suffixes starting with that prefix. The full hash comparison happens locally. HIBP cannot determine which password is being checked.

### Integration Points

| Integration | Status | Notes |
|---|---|---|
| Registration (new password) | Available | Call `CheckPasswordBreach` in register handler |
| Password change | Available | Call before `SetPassword` |
| Periodic audit | Not implemented | Would require re-hashing stored passwords |
| Fail-open behavior | Implemented | API errors do not block registration |

### Gaps and Improvements

1. **No caching**: Each check hits the HIBP API. Add Redis cache (prefix -> response body, 24h TTL) to reduce API calls and latency.
2. **No offline fallback**: If HIBP is unreachable (network failure, DNS), the check silently passes. Consider downloading the full corpus (~30GB) for air-gapped deployments.
3. **Not wired into the register flow**: The method exists but is not called automatically. Must be invoked explicitly by the handler.
4. **Add `Add-Padding: true` header**: HIBP supports response padding to obscure result size. Prevents network-level enumeration.

```go
req.Header.Set("Add-Padding", "true")
```

---

## 5. Honeypot Accounts

### Concept

Honeypot accounts are fake accounts that look real to attackers but have no legitimate user. Any login attempt against a honeypot is by definition malicious — no real user would ever try to authenticate with those credentials.

### Detection Value

- **Immediate IP block**: honeypot login attempt = confirmed malicious IP
- **Breach corpus intelligence**: passwords tried against honeypots reveal the attacker's breach dictionary
- **Attack tool fingerprinting**: user-agent and timing patterns from honeypot hits identify the bot/tool
- **Early warning**: honeypot hits often precede targeted attacks on real accounts

### GGID Implementation Proposal

```go
// HoneypotDetector blocks IPs that attempt honeypot credentials.
type HoneypotDetector struct {
    rdb  *redis.Client
    nats *nats.Conn
}

func (h *HoneypotDetector) OnHoneypotHit(ctx context.Context, ip, identifier, password, ua string) {
    h.rdb.Set(ctx, "ggid:block:ip:"+ip, "honeypot", 24*time.Hour)        // block IP
    h.nats.Publish("security.alert", alertJSON(ip, identifier, ua))         // alert team
    h.rdb.SAdd(ctx, "ggid:honeypot:passwords", sha256hex(password))       // intel
}
```

### Deployment

- Seed 5-10 honeypot accounts per tenant with realistic-looking usernames
- Flag accounts with `is_honeypot = true` in the users table (hidden from admin UI)
- Distribute honeypot credentials in fake breach dumps to attract attackers
- Rotate honeypot accounts quarterly to prevent fingerprinting

---

## 6. Credential Guard Patterns

### Password Storage Defense

| Defense | Implemented | Location |
|---|---|---|
| Argon2id (64MB memory, 3 iterations) | Yes | `pkg/crypto/crypto.go:HashPassword` |
| Unique per-user salt (16 bytes) | Yes | `crypto/rand` in `HashPassword` |
| Constant-time comparison | Yes | `constantTimeCompare` in `VerifyPassword` |
| Password history (no reuse) | Yes | `PasswordService.CheckHistory` |
| Password policy (min length, complexity) | Yes | `PasswordService.Validate` |
| Password blacklist (weak passwords) | Yes | `policy.Blacklist` |
| **Pepper** (server-side secret) | **No** | Not implemented |

**Pepper gap:** A pepper is an additional secret mixed into the password hash that is stored outside the database (e.g., in a KMS or environment variable). If an attacker steals only the database, peppered hashes cannot be cracked. Implementation:

```go
func HashPasswordWithPepper(password, pepper string) (string, error) {
    hash := argon2.IDKey([]byte(password+pepper), salt, iter, mem, par, keyLen)
    // ... same encoding as HashPassword
}
```

### Token Theft Defense

- Short-lived access tokens (15 min) limit the theft window
- DPoP / mTLS binding makes stolen tokens unusable without the client's private key
- Refresh token rotation detects theft via reuse detection
- See [token-lifecycle-security.md](./token-lifecycle-security.md)

### Session Defense

- Session ID regeneration on authentication (prevents fixation)
- Device fingerprint binding for session continuity
- Concurrent session limits detect sharing
- See [session-management-design.md](./session-management-design.md)

### Transport Defense

- TLS 1.3 for all credential transmission
- HSTS header forces HTTPS
- Certificate pinning for mobile SDK clients

---

## 7. Response Automation

### Automated Response Matrix

| Trigger | Response | Latency |
|---|---|---|
| >5 failed logins from one account | Lock account 15 min (existing) | Immediate |
| >20 unique usernames from one IP | Block IP 1 hour | Immediate |
| Honeypot login attempt | Block IP 24h + alert | Immediate |
| Breach password detected at registration | Reject with error (existing) | Immediate |
| High-risk login score (>60) | Force MFA step-up (existing) | Immediate |
| /24 range >100 failures | Block entire range | 5 min window |

### Proposed: ThreatResponder Playbook Engine

```go
// ThreatResponder evaluates audit events and triggers automated responses.
type ThreatResponder struct {
    rdb        *redis.Client
    nats       *nats.Conn
    thresholds ThreatThresholds
}

type ThreatThresholds struct {
    IPLockFailures      int           // 20
    RangeBlockFailures  int           // 100
    MFAScoreThreshold   int           // 60
    LockDuration        time.Duration // 15 min
}

func (t *ThreatResponder) HandleEvent(ctx context.Context, event AuditEvent) {
    switch event.Type {
    case "login_failed":
        count, _ := t.rdb.Incr(ctx, "ggid:threat:ip:"+event.IP).Result()
        t.rdb.Expire(ctx, "ggid:threat:ip:"+event.IP, 5*time.Minute)
        if count >= int64(t.thresholds.IPLockFailures) {
            t.blockIP(ctx, event.IP, t.thresholds.LockDuration)
            t.publishAlert(ctx, "ip_blocked", event.IP)
        }
    case "honeypot_hit":
        t.blockIP(ctx, event.IP, 24*time.Hour)
        t.publishAlert(ctx, "honeypot_hit", event.IP)
    case "high_risk_login":
        if event.RiskScore >= t.thresholds.MFAScoreThreshold {
            t.publishAlert(ctx, "mfa_stepup_required", event.UserID)
        }
    }
}
```

### NATS Alert Topic

All automated responses publish to `security.alert` on NATS JetStream for downstream consumption by the security operations dashboard, SIEM integration, and paging systems.

---

## 8. GGID Current Defense Audit

| Defense | Implemented | Location | Gap | Priority |
|---|---|---|---|---|
| Per-IP rate limiting (token bucket) | Yes | `gateway/middleware/token_bucket.go` | In-memory only, not distributed | P1 |
| Per-IP rate limiting (Redis fixed window) | Yes | `auth/service/ratelimit_service.go` | 60s fixed window, no sliding | P2 |
| Per-account lockout (5 attempts) | Yes | `auth/service/email_lockout.go` | Fixed 15min, no exponential backoff | P1 |
| Anomaly lockout (sliding window) | Yes | `auth/service/anomaly_detection.go` | Overlaps with AccountLockout | P2 |
| Breach password check (HIBP) | Yes | `auth/service/password_breach.go` | Not wired into register flow | P0 |
| Risk-based login assessment | Yes | `auth/service/risk_auth.go` | Score model is basic (IP + time) | P2 |
| Bot detection (UA blocklist) | Yes | `gateway/middleware/botdetect.go` | Does not block curl/python | P1 |
| Behavioral bot detection | Yes | `gateway/middleware/botdetect.go` | In-memory only | P2 |
| IP allowlist / denylist | Yes | `gateway/middleware/ip_filter.go` | Per-tenant manual config | P3 |
| Login attempt logging | Yes | `auth/service/login_attempt.go` | Per-user only, not per-IP | P2 |
| Geo-anomaly detection | Yes | `auth/service/anomaly_detection.go` | Requires GeoIP DB | P2 |
| Device fingerprint tracking | Yes | `auth/service/anomaly_detection.go` | Basic set membership | P3 |
| Exponential backout lockout | **No** | — | — | P0 |
| Stuffing correlation detector | **No** | — | Rate limiting helps but no cross-IP correlation | P1 |
| Spraying correlation detector | **No** | — | — | P1 |
| Honeypot accounts | **No** | — | — | P2 |
| CAPTCHA / progressive friction | **No** | — | — | P2 |
| Password pepper | **No** | — | DB-only compromise exposes hashes | P1 |
| User-agent filtering (non-browser block) | **No** | — | Bot patterns blocked but not curl/python | P1 |
| Automated response playbook | **No** | — | Manual IP blocking only | P2 |

---

## 9. Roadmap

### Phase 1 — Critical Gaps (P0, ~5 days)

| Task | Effort | Files |
|---|---|---|
| Wire `CheckPasswordBreach` into register handler | 0.5 day | `auth/service/auth_service.go` |
| Add exponential backoff to `AccountLockoutService` | 1.5 days | `auth/service/email_lockout.go` |
| Cache HIBP responses in Redis (24h TTL) | 0.5 day | `auth/service/password_breach.go` |
| Add `Add-Padding` header to HIBP requests | 0.1 day | `auth/service/password_breach.go` |
| Block non-browser UAs on auth endpoints | 1 day | `gateway/middleware/botdetect.go` |
| Wire risk assessment to enforce MFA step-up | 1.5 days | `auth/service/risk_auth.go` |

### Phase 2 — Correlation Detection (P1, ~2 weeks)

| Task | Effort |
|---|---|
| Implement `StuffingDetector` (HyperLogLog per-IP, /24 range tracking) | 3 days |
| Implement `SprayingDetector` (distinct usernames per IP, password hash correlation) | 3 days |
| Distribute token-bucket limiter via Redis Lua script | 2 days |
| Add password pepper to `HashPassword` / `VerifyPassword` | 2 days |
| Implement login time randomization (200-500ms jitter) | 0.5 day |

### Phase 3 — Advanced Defenses (P2, ~3 weeks)

| Task | Effort |
|---|---|
| Honeypot account seeding and `HoneypotDetector` middleware | 3 days |
| CAPTCHA integration (hCaptcha / Cloudflare Turnstile) after N failures | 3 days |
| `ThreatResponder` playbook engine with NATS alerts | 5 days |
| SIEM integration via CEF/LEEF event export | 3 days |
| Offline HIBP corpus for air-gapped deployments | 2 days |

### Effort Summary

| Phase | Duration | Impact |
|---|---|---|
| Phase 1 (P0) | ~5 days | Closes the most critical gaps: breach check, exponential lockout |
| Phase 2 (P1) | ~2 weeks | Stuffing/spraying detection, distributed rate limiting, pepper |
| Phase 3 (P2) | ~3 weeks | Honeypots, CAPTCHA, automated response, SIEM |

---

## References

- NIST SP 800-63B: Digital Identity Guidelines — Authentication and Lifecycle Management
- Verizon Data Breach Investigations Report (DBIR) — annual credential attack statistics
- HaveIBeenPwned Password API — k-anonymity range query model (api.pwnedpasswords.com)
- OWASP Authentication Cheat Sheet — credential attack prevention guidance
- RFC 9126: Pushed Authorization Requests (PAR) — credential interception defense
- [token-lifecycle-security.md](./token-lifecycle-security.md) — token theft and refresh rotation
- [session-management-design.md](./session-management-design.md) — session fixation and binding
