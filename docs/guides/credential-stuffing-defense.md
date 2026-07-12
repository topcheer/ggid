# Credential Stuffing Defense

This guide covers credential stuffing attack patterns, detection signals, defense measures, and GGID's multi-layer defense implementation.

## Attack Pattern Analysis

### What is Credential Stuffing?

Credential stuffing is the automated injection of breached username/password pairs into login forms. Unlike brute force (which tries random passwords), credential stuffing uses real credentials leaked from other breaches.

### Attack Characteristics

| Characteristic | Description |
|---|---|
| Scale | Thousands to millions of attempts |
| Credential source | Breached databases (Collection #1, HIBP, etc.) |
| Automation | Botnets, headless browsers, API calls |
| Success rate | 0.1-2% (enough to be profitable) |
| IP diversity | Hundreds/thousands of source IPs |
| Target | Login, password reset, OAuth token endpoints |

### Attack Vectors

1. **Direct API** — Attacker sends HTTP POST to `/api/v1/auth/login`
2. **Headless browser** — Selenium/Puppeteer simulates real user
3. **Botnet** — Distributed across many IPs to avoid rate limits
4. **Residential proxy** — Uses residential IPs to bypass IP blocking
5. **Low and slow** — 1-2 attempts per IP per hour to avoid detection

## Detection Signals

### Velocity Signals

| Signal | Threshold | Confidence |
|---|---|---|
| Failed logins per user | >10/hour | Medium |
| Failed logins per IP | >50/hour | High |
| Failed logins per tenant | >500/hour | High |
| Unique users per IP | >20/hour | High (bot behavior) |
| Unique IPs per user | >10/hour | High (account targeted) |
| Login success rate | <10% | High (brute force pattern) |

### IP Rotation Signals

| Signal | Description | Detection |
|---|---|---|
| Many IPs, same UA | Botnet with same tool | IP diversity + UA fingerprinting |
| Residential proxies | Legitimate-looking IPs | ASN reputation check |
| Geolocation spread | Logins from 5+ countries | Geo-velocity check |
| ASN concentration | 80%+ from one ASN | Unusual for organic traffic |

### Failed Attempt Signals

| Signal | Threshold | Action |
|---|---|---|
| Consecutive failures | 5 in a row | Trigger CAPTCHA |
| Failures with different passwords | 3 distinct passwords | Flag as stuffing |
| Failures with valid usernames | >50% valid usernames | Breached data in use |
| Failures across many users | 100+ users in 1 hour | Large-scale attack |

### User-Agent Signals

| Signal | Description |
|---|---|
| Missing UA | API-based attack |
| Known bot UA | Curl, python-requests, headless Chrome |
| Unusual UA | Rare browser versions |
| UA rotation | Different UA per request (evasion) |

### Behavioral Signals

| Signal | Description |
|---|---|
| No JS execution | Headless browser / curl |
| No mouse movement | Automated tool |
| Instant form submission | <100ms fill time |
| Identical timing | Robotic request intervals |

## Defense Measures

### Layer 1: Rate Limiting

```yaml
rate_limit:
  per_user:
    login: 5/minute
    login_failed: 10/hour
  per_ip:
    login: 50/minute
    login_failed: 100/hour
  per_tenant:
    login: 500/minute
    login_failed: 1000/hour
  per_ua:
    known_bot: 10/minute
```

### Layer 2: CAPTCHA

Trigger CAPTCHA after suspicious activity:

```yaml
captcha:
  enabled: true
  trigger:
    failed_attempts: 3
    ip_velocity: 20/hour
    user_velocity: 5/hour
  provider: "recaptcha"  # or "hcaptcha", "turnstile"
  action: "login"
  score_threshold: 0.5  # For reCAPTCHA v3
```

### Layer 3: Account Lockout

```yaml
account_lockout:
  enabled: true
  threshold: 10  # Failed attempts
  duration: 15m  # Lockout period
  escalate: true  # Longer lockout on repeat
  escalation:
    1st: 15m
    2nd: 1h
    3rd: 24h
  notify_user: true  # Email user about lockout
```

### Layer 4: Breached Password Check (HIBP)

```yaml
password:
  breach_check:
    enabled: true
    provider: "hibp"
    api_key: "<HIBP_API_KEY>"
    on_breach: "reject"  # or "warn"
    cache_ttl: 24h
```

### Layer 5: Adaptive MFA

Require MFA for suspicious logins even if not normally required:

```yaml
adaptive_mfa:
  enabled: true
  triggers:
    new_device: true
    new_ip: true
    new_geolocation: true
    high_risk_score: true
  risk_scoring:
    factors:
      ip_reputation: 0.3
      geo_velocity: 0.2
      device_familiarity: 0.2
      time_anomaly: 0.1
      user_agent: 0.1
      failed_attempts: 0.1
    high_risk_threshold: 0.7
    medium_risk_threshold: 0.4
```

### Layer 6: IP Reputation

```yaml
ip_reputation:
  enabled: true
  block:
    known_botnets: true
    tor_exit_nodes: true
    datacenter_ips: "challenge"  # Require CAPTCHA
  providers:
    - "internal-blocklist"
    - "external-threat-intel"
  refresh_interval: 1h
```

## GGID Multi-Layer Defense Implementation

### Defense in Depth

```
Request → IP Reputation → Rate Limit → CAPTCHA Check → Account Lockout Check
       → Breached Password Check → Adaptive MFA → Login
```

### Implementation

```go
func LoginHandler(w http.ResponseWriter, r *http.Request) {
    ip := clientIP(r)
    userID := r.FormValue("username")
    password := r.FormValue("password")

    // Layer 1: IP reputation
    if isBlockedIP(ip) {
        writeError(w, 403, "ip_blocked")
        return
    }

    // Layer 2: Rate limiting
    if !rateLimiter.Allow("login:ip:"+ip) {
        setRetryAfter(w, rateLimiter.RetryAfter("login:ip:"+ip))
        writeError(w, 429, "rate_limited")
        return
    }
    if !rateLimiter.Allow("login:user:"+userID) {
        writeError(w, 429, "rate_limited_user")
        return
    }

    // Layer 3: Account lockout check
    if isLocked(userID) {
        writeError(w, 423, "account_locked")
        return
    }

    // Layer 4: CAPTCHA (if triggered)
    if needsCaptcha(ip, userID) {
        if !verifyCaptcha(r) {
            writeError(w, 403, "captcha_required")
            return
        }
    }

    // Layer 5: Authenticate
    user, err := authenticate(userID, password)
    if err != nil {
        recordFailedAttempt(userID, ip)
        writeError(w, 401, "invalid_credentials")
        return
    }

    // Layer 6: Breached password check
    if isBreached(password) {
        // Force password change
        requirePasswordChange(user)
        writeError(w, 403, "password_breached")
        return
    }

    // Layer 7: Adaptive MFA
    riskScore := calculateRiskScore(ip, user, r)
    if riskScore > highRiskThreshold || user.MFARequired {
        if !verifyMFA(user, r) {
            writeError(w, 403, "mfa_required")
            return
        }
    }

    // Success
    issueToken(w, user)
    recordSuccessfulLogin(userID, ip)
}
```

### Risk Scoring

```go
func calculateRiskScore(ip string, user *User, r *http.Request) float64 {
    score := 0.0

    // IP reputation
    if isDatacenterIP(ip) {
        score += 0.3
    }
    if isTorExitNode(ip) {
        score += 0.5
    }

    // Geo-velocity
    lastLogin := user.LastSuccessfulLogin
    if lastLogin != nil {
        distance := geoDistance(lastLogin.IP, ip)
        timeDiff := time.Since(lastLogin.Timestamp)
        if distance/timeDiff.Hours() > 500 { // >500 km/h
            score += 0.2
        }
    }

    // New device
    deviceFP := deviceFingerprint(r)
    if !user.KnownDevices[deviceFP] {
        score += 0.2
    }

    // Time anomaly
    hour := time.Now().Hour()
    if hour < 6 || hour > 22 { // Outside normal hours
        score += 0.1
    }

    // Failed attempts
    failedCount := getFailedAttemptCount(user.ID)
    if failedCount > 3 {
        score += 0.1
    }

    return score
}
```

## HIBP API Integration

```go
func CheckBreachedPassword(password string) (bool, error) {
    // SHA-1 hash of password
    hash := sha1.Sum([]byte(password))
    hashHex := strings.ToUpper(hex.EncodeToString(hash[:]))

    // k-anonymity: send only first 5 chars
    prefix := hashHex[:5]
    suffix := hashHex[5:]

    // Query HIBP
    resp, err := http.Get(fmt.Sprintf("https://api.pwnedpasswords.com/range/%s", prefix))
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()

    // Check if our suffix is in the response
    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        parts := strings.Split(line, ":")
        if parts[0] == suffix {
            count, _ := strconv.Atoi(parts[1])
            return count > 0, nil
        }
    }
    return false, nil
}
```

### Caching

```go
var breachCache = cache.New(24*time.Hour, 1*time.Hour)

func isBreachedCached(password string) bool {
    hash := sha1Hex(password)
    if val, ok := breachCache.Get(hash); ok {
        return val.(bool)
    }
    breached, _ := CheckBreachedPassword(password)
    breachCache.SetDefault(hash, breached)
    return breached
}
```

## Response Playbook

### Detection Levels

| Level | Indicators | Response |
|---|---|---|
| Low | 10-50 failed logins/hour | Monitor, no action |
| Medium | 50-200 failed logins/hour | Enable CAPTCHA, increase rate limits |
| High | 200-1000 failed logins/hour | Block top offending IPs, adaptive MFA |
| Critical | >1000 failed logins/hour | Emergency mode: CAPTCHA for all, notify admins |

### Incident Response

1. **Detect** — monitoring alerts on velocity thresholds
2. **Classify** — determine attack scale (low/medium/high/critical)
3. **Respond** — apply response measures per level
4. **Communicate** — notify security team and affected users
5. **Recover** — after attack subsides, gradually relax controls
6. **Post-mortem** — analyze attack, improve defenses

### Emergency Mode

```yaml
emergency_mode:
  trigger: "critical"
  measures:
    captcha_all_logins: true
    rate_limit_multiplier: 0.1  # 10x stricter
    block_new_device_logins: true
    require_mfa_all_users: true
    notify_admins: true
  auto_disable_after: 2h  # After attack indicators subside
```

## Best Practices

1. **Defense in depth** — no single measure is sufficient, layer them
2. **Monitor velocity** — failed login rate is the best early indicator
3. **Use HIBP** — check passwords against breach database at registration and login
4. **Adaptive MFA** — require MFA for risky logins, not just all logins
5. **IP reputation** — block known bad IPs, challenge datacenter IPs
6. **CAPTCHA as escalation** — don't always show, only when triggered
7. **Notify users** — tell them when their account is targeted
8. **Don't reveal which field failed** — "invalid credentials" not "wrong password"
9. **Implement consistent timing** — don't let timing differences reveal valid usernames
10. **Log everything** — audit trail for forensic analysis