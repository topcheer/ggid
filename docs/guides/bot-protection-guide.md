# Bot Protection Guide

This guide covers bot detection methods, CAPTCHA selection, progressive challenges, false positive management, and GGID's bot protection implementation.

## Bot Types

| Bot Type | Purpose | Target Endpoints | Impact |
|---|---|---|---|
| Scraper | Content harvesting | GET APIs, pages | Data theft, bandwidth |
| Credential stuffer | Login automation | POST /auth/login | Account takeover |
| DDoS | Service disruption | All endpoints | Downtime |
| Content spam | Form submission | POST /users, /comments | Spam, abuse |
| Credential harvester | Phishing page mimics | All GET pages | User credential theft |
| Account creator | Mass registration | POST /auth/register | Resource abuse |
| API abuser | Rate limit evasion | All APIs | Service degradation |
| Sniper | Time-sensitive targeting | Token endpoints, limited resources | Unfair advantage |

## Detection Methods

### 1. Behavioral Analysis

Analyze user interaction patterns:

| Signal | Human | Bot |
|---|---|---|
| Mouse movement | Curved, varied | None or linear |
| Form fill time | 2-10 seconds | <100ms or instant |
| Click patterns | Varied intervals | Robotic timing |
| Page dwell time | Variable | Instant navigation |
| JavaScript execution | Full | None or minimal |
| Touch events | Present (mobile) | Absent |

```go
type BehavioralScore struct {
    MouseMovement   float64  // 0-1, 1 = human-like
    FormFillTime    float64
    ClickVariance   float64
    JSEnabled       bool
    DwellTime       float64
}

func isBot(score BehavioralScore) bool {
    total := score.MouseMovement*0.3 + score.FormFillTime*0.2 +
             score.ClickVariance*0.2 + score.DwellTime*0.15
    if !score.JSEnabled {
        total *= 0.5  // Big penalty for no JS
    }
    return total < 0.3  // Score below 0.3 = likely bot
}
```

### 2. CAPTCHA

Challenge-response test to verify human interaction.

### 3. Challenge Tokens

Server-issued challenges that require computation:

```go
// Issue challenge
challenge := generateChallenge()
// Client must compute: SHA256(challenge + nonce) must start with "0000"
// This takes ~100ms for a browser, but is expensive for bots at scale
```

### 4. TLS Fingerprinting

Identify clients by their TLS handshake:

| Fingerprint | Client | Trust |
|---|---|---|
| JA3: browser pattern | Chrome, Firefox, Safari | High |
| JA3: curl pattern | curl, wget | Low |
| JA3: python-requests | Python scripts | Low |
| JA3: go-http-client | Go programs | Low |
| JA3: headless Chrome | Puppeteer, Selenium | Medium |

```go
func classifyByTLS(fp string) string {
    switch {
    case isBrowserFP(fp):
        return "browser"
    case isCurlFP(fp):
        return "curl"
    case isPythonFP(fp):
        return "python"
    case isHeadlessBrowserFP(fp):
        return "headless"
    default:
        return "unknown"
    }
}
```

### 5. JavaScript Challenges

Server sends a JS challenge that the client must execute:

```javascript
// Server sends:
const challenge = "compute_this";
const result = await solveChallenge(challenge);
// Client must post result back
fetch('/api/v1/verify-challenge', {body: {challenge, result}});
```

### 6. Honeypot Fields

Hidden form fields that bots fill but humans don't:

```html
<input type="hidden" name="website" value="" style="display:none">
<!-- If 'website' is filled → bot -->
```

## CAPTCHA Selection

### Comparison

| Provider | Type | UX | Accessibility | Detection Rate | Cost |
|---|---|---|---|---|---|
| hCaptcha | Visual puzzle | Medium | Good (audio) | High | Free tier |
| Cloudflare Turnstile | Invisible | Excellent | Excellent | High | Free |
| reCAPTCHA v3 | Score-based | Excellent | Excellent | Medium | Free |
| reCAPTCHA v2 | Checkbox/puzzle | Medium | Good | High | Free |
| Arkose Labs | Gamified | Low (interactive) | Medium | Very high | Paid |

### Recommendation

```yaml
captcha:
  primary: "turnstile"     # Best UX, invisible
  fallback: "hcaptcha"     # When Turnstile fails
  aggressive: "arkose"     # For high-risk sessions
```

### Cloudflare Turnstile (Recommended)

```html
<div class="cf-turnstile" data-sitekey="YOUR_KEY" data-callback="onTurnstileSuccess"></div>
<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>
```

Server-side verification:
```go
func verifyTurnstile(token string, ip string) (bool, error) {
    resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify",
        url.Values{
            "secret":   {config.TurnstileSecret},
            "response": {token},
            "remoteip": {ip},
        })
    // Parse response → success: true/false
}
```

## Progressive Challenges

Escalate challenges based on risk:

```
Level 0: No challenge (trusted)
    ↓ (risk detected)
Level 1: Invisible challenge (Turnstile)
    ↓ (challenge failed or higher risk)
Level 2: Interactive challenge (hCaptcha)
    ↓ (challenge failed or high risk)
Level 3: Gamified challenge (Arkose)
    ↓ (challenge failed)
Level 4: Block + notify admin
```

### Implementation

```go
func getChallengeLevel(riskScore float64, failedChallenges int) int {
    if riskScore < 0.3 && failedChallenges == 0 {
        return 0  // No challenge
    }
    if riskScore < 0.5 || failedChallenges <= 1 {
        return 1  // Invisible
    }
    if riskScore < 0.7 || failedChallenges <= 2 {
        return 2  // Interactive
    }
    if riskScore < 0.85 || failedChallenges <= 3 {
        return 3  // Gamified
    }
    return 4  // Block
}
```

### Configuration

```yaml
bot_protection:
  progressive:
    enabled: true
    levels:
      0:
        action: "allow"
      1:
        action: "challenge_invisible"
        provider: "turnstile"
      2:
        action: "challenge_interactive"
        provider: "hcaptcha"
      3:
        action: "challenge_gamified"
        provider: "arkose"
      4:
        action: "block"
        notify_admin: true
```

## False Positive Management

### Common False Positives

| Scenario | Why Flagged | Mitigation |
|---|---|---|
| Screen reader users | No mouse movement | Accessibility bypass token |
| Keyboard-only navigation | No mouse | Detect keyboard events |
| Slow typers | Long form fill time | Increase threshold |
| VPN users | Datacenter IP | VPN allowlist |
| API clients (legitimate) | No JS, curl UA | API key bypass |
| Mobile apps | Different TLS fingerprint | App certificate |

### Bypass Mechanisms

```yaml
bot_protection:
  bypass:
    api_clients:
      enabled: true
      auth: "api_key"  # Valid API key bypasses challenges
    accessibility:
      enabled: true
      token: "accessibility_bypass_token"
      max_uses_per_hour: 100
    allowlisted_ips:
      - "192.168.1.0/24"  # Corporate network
    allowlisted_uas:
      - "GGID-SDK-Go/1.0"
      - "GGID-SDK-Node/1.0"
```

### Monitoring False Positives

```yaml
monitoring:
  challenge_success_rate:
    alert_below: 0.85  # If <85% pass, may be too aggressive
  challenge_abandonment:
    alert_above: 0.15  # If >15% abandon, too aggressive
  false_positive_reports:
    track: true
    feedback_url: "/feedback/challenge"
```

## Bot Allowlist

### Search Engine Bots

```yaml
allowlist:
  search_engines:
    - name: "Googlebot"
      ua_pattern: "Googlebot"
      verify_dns: "*.googlebot.com"
      verify_txt: "google.com"
    - name: "Bingbot"
      ua_pattern: "bingbot"
      verify_dns: "*.search.msn.com"
    - name: "DuckDuckBot"
      ua_pattern: "DuckDuckBot"
      verify_dns: "*.duckduckgo.com"
```

### Verification

```go
func verifySearchBot(ua, ip string) bool {
    // Step 1: Match UA pattern
    if !matchesPattern(ua, "Googlebot") {
        return false
    }
    // Step 2: Reverse DNS lookup
    names, err := net.LookupAddr(ip)
    if err != nil {
        return false
    }
    // Step 3: Verify DNS name matches
    for _, name := range names {
        if strings.HasSuffix(name, "googlebot.com.") {
            // Step 4: Forward DNS verification
            resolvedIPs, err := net.LookupHost(strings.TrimSuffix(name, "."))
            if err == nil && contains(resolvedIPs, ip) {
                return true
            }
        }
    }
    return false
}
```

### API Client Allowlist

```yaml
allowlist:
  api_clients:
    - ua_pattern: "GGID-SDK-*"
      auth_required: true
      rate_limit_override: "api_tier"
    - ua_pattern: "Prometheus/*"
      paths: ["/metrics"]
      auth_required: false
```

## GGID Bot Protection Implementation

### Defense Layers

```
Request → IP Reputation → TLS Fingerprint → Rate Limit
       → Behavioral Analysis → CAPTCHA (progressive) → Allow/Block
```

### Middleware

```go
func BotProtectionMiddleware(config *BotConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Skip for allowlisted clients
            if isAllowlisted(r, config) {
                next.ServeHTTP(w, r)
                return
            }

            // Layer 1: IP reputation
            if isBlockedIP(clientIP(r)) {
                writeError(w, 403, "ip_blocked")
                return
            }

            // Layer 2: TLS fingerprinting
            tlsClass := classifyByTLS(getTLSFingerprint(r))
            if tlsClass == "curl" || tlsClass == "python" {
                // Require API key for non-browser clients
                if !hasValidAPIKey(r) {
                    writeError(w, 403, "api_key_required")
                    return
                }
            }

            // Layer 3: Rate limiting (already in middleware chain)

            // Layer 4: Behavioral analysis
            riskScore := calculateBotRiskScore(r)
            failedChallenges := getFailedChallengeCount(r)

            // Layer 5: Progressive challenge
            level := getChallengeLevel(riskScore, failedChallenges)
            if level == 4 {
                writeError(w, 403, "bot_detected")
                notifyAdmin(r)
                return
            }
            if level > 0 {
                challenge := generateChallenge(level, config)
                writeChallengeResponse(w, challenge)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### Configuration

```yaml
bot_protection:
  enabled: true
  detection:
    behavioral: true
    tls_fingerprinting: true
    ip_reputation: true
    js_challenge: true
    honeypot: true
  captcha:
    primary: "turnstile"
    fallback: "hcaptcha"
  progressive:
    enabled: true
  allowlist:
    search_engines: true
    api_clients: true
  monitoring:
    challenge_success_rate: true
    false_positive_tracking: true
```

## Best Practices

1. **Layer defenses** — No single method catches all bots
2. **Start invisible** — Use invisible challenges first, escalate only when needed
3. **Allow legitimate bots** — Search engines and API clients need access
4. **Monitor false positives** — Track challenge pass rates and abandonment
5. **Verify bot claims** — Don't trust UA alone, verify via DNS
6. **Use progressive challenges** — Don't show hard CAPTCHA immediately
7. **Respect accessibility** — Provide bypass for screen reader users
8. **Log everything** — Challenge events, pass/fail, risk scores
9. **Update detection** — Bot techniques evolve, update detection patterns
10. **Consider UX impact** — Aggressive bot protection hurts real users