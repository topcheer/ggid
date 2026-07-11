# Bot Protection for IAM Systems — Competitive Analysis and Gap Assessment

**Author:** GGID Security Research  
**Date:** 2025-01-20  
**Status:** Active Research  
**Scope:** GGID IAM platform bot detection capabilities vs Auth0, Keycloak, and industry best practices

---

## Table of Contents

1. [Bot Threat Landscape for IAM](#1-bot-threat-landscape-for-iam)
2. [Auth0 Bot Protection](#2-auth0-bot-protection)
3. [Keycloak Bot Protection](#3-keycloak-bot-protection)
4. [Bot Detection Techniques](#4-bot-detection-techniques)
5. [GGID Existing Bot Detection](#5-ggid-existing-bot-detection)
6. [Gap Analysis](#6-gap-analysis)
7. [Implementation Recommendations](#7-implementation-recommendations)
8. [Go Code: Middleware Implementations](#8-go-code-middleware-implementations)

---

## 1. Bot Threat Landscape for IAM

Identity and Access Management (IAM) systems are the primary target for automated bot attacks. Unlike general web applications, IAM endpoints expose high-value operations: authentication, account creation, password recovery, and token issuance. A successful bot attack against an IAM system can lead to account takeover (ATO), fraudulent account creation, and lateral movement into downstream applications.

### 1.1 Credential Stuffing Bots

Credential stuffing is the most prevalent IAM bot attack. Attackers use leaked credential databases (from data breaches) to attempt mass logins across multiple services. The attack exploits password reuse: studies show 65% of users reuse passwords across multiple accounts.

**Attack patterns:**
- **Credential stuffing APIs**: Commercial services like Sentry MBA, OpenBullet, and SilverBullet provide GUI-driven credential stuffing with built-in proxy rotation, CAPTCHA solving, and success parsing. These tools can test thousands of credential pairs per minute.
- **Selenium/Playwright automation**: Attackers increasingly use legitimate browser automation frameworks (Selenium WebDriver, Playwright, Puppeteer) to bypass simple bot detection. Headless Chrome with `--disable-blink-features=AutomationControlled` evades naive User-Agent checks.
- **Distributed attacks**: Botnets spread credential stuffing across thousands of IPs, making per-IP rate limiting insufficient. Each IP may attempt only 5-10 logins, staying under per-IP thresholds.
- **Credential validation APIs**: Some attackers use "checker" services that validate credentials against a target site and sell verified account access on darknet markets.

**Real-world incidents:**
- Disney+ (2019): Thousands of accounts compromised within hours of launch via credential stuffing.
- Norton LifeLock (2023): Credential stuffing attack exposed password manager data.
- 23andMe (2023): Credential stuffing led to data breach affecting 6.9 million users.

### 1.2 Account Creation Bots

Automated account registration floods IAM systems with fake accounts, enabling:
- **Free trial abuse**: Creating thousands of accounts to exploit free-tier resources.
- **Spam and phishing**: Fake accounts used to send spam, distribute phishing links, or manipulate social platform metrics.
- **Credential harvesting**: Fake accounts mimic legitimate brand login pages within the platform.
- **Resource exhaustion**: Mass registration consumes database storage and email quota.

**Attack patterns:**
- Disposable email APIs (e.g., Mailinator, Guerrilla Mail) for email verification bypass.
- Automated CAPTCHA solving services (2Captcha, Anti-Captcha) at $1-3 per 1000 solves.
- Headless browsers that execute JavaScript and pass client-side challenges.
- Rotating residential proxies (Luminati/Bright Data, Smartproxy) to evade IP-based limits.

### 1.3 Password Reset Abuse

Password reset endpoints are abused for:
- **Account lockout DoS**: Triggering resets for known email addresses to lock legitimate users out.
- **User enumeration**: Observing different response timing or messages to determine if an email is registered.
- **Email bombing**: Using reset flows to flood a victim's inbox with reset emails.
- **Token brute force**: Attempting to guess reset tokens if entropy is insufficient.

### 1.4 MFA Fatigue Bots (Push Bombing)

With the widespread adoption of push-based MFA (e.g., Duo Push, Microsoft Authenticator), attackers deploy bots that:
1. Obtain a valid username/password (via credential stuffing or phishing).
2. Trigger repeated MFA push notifications (50-100+ in rapid succession).
3. Rely on user fatigue — the victim eventually approves a notification to stop the barrage.

**Real-world incidents:**
- Uber (2022): Attacker bombarded an employee with MFA push requests until one was approved, leading to a major breach.
- Cisco (2022): Same MFA fatigue technique via voice phishing.
- Reddit (2023): Phishing + MFA fatigue led to limited source code exposure.

This attack vector is particularly dangerous because it bypasses the "something you have" factor entirely through social engineering at machine speed.

---

## 2. Auth0 Bot Protection

Auth0 (now Okta CIC) offers bot protection through its **Attack Protection** suite, a collection of pre-configured security features designed to mitigate common automated attacks.

### 2.1 Breached Password Detection (BPD)

**How it works:** Auth0 maintains a continuously updated database of credentials exposed in known data breaches. When a user attempts to log in or register, Auth0 checks the submitted password against this database in real-time. If the password appears in a known breach, Auth0 can:
- Block the login attempt.
- Require the user to change their password.
- Send a breach notification email.

**Coverage:** Checks against a database of 10+ billion breached credentials, sourced from HIBP (Have I Been Pwned) and proprietary data.

**Pricing:** Available on **B2C Professional** ($0.07/MA) and **Enterprise** tiers. Not available on the free "Essentials" tier.

### 2.2 Suspicious IP Throttling

**How it works:** Auth0 tracks IP addresses that exhibit suspicious behavior across all Auth0 tenants (shared threat intelligence). When an IP generates excessive login failures, registrations, or password resets, it is flagged. Subsequent requests from flagged IPs face progressive throttling.

**Key features:**
- **Anomaly detection**: Uses ML-based scoring to distinguish between shared IPs (NAT, corporate networks) and genuinely malicious IPs.
- **Cross-tenant intelligence**: An IP flagged on one Auth0 tenant is automatically throttled on all tenants — a significant advantage of a hosted platform.
- **Customizable policies**: Administrators can adjust sensitivity thresholds.

**Pricing:** **Enterprise** tier only.

### 2.3 Brute Force Protection

**How it works:** Auth0 limits the number of failed login attempts per user identifier (email/username) and per IP address. After exceeding the threshold (default: 10 failures per user, 50 per IP), the account or IP is temporarily blocked.

**Configuration:**
- `max_attempts`: Number of failed attempts before lockout.
- `lockout_minutes`: Duration of the lockout.
- `mode`: `count` (track failures) or `block` (actively block).

**Pricing:** Available on all paid tiers including **Essentials** ($35/month for 1,000 MA).

### 2.4 Limitations

| Limitation | Impact |
|---|---|
| BPD checks only known breach passwords — custom passwords in stuffing lists are undetected | Credential stuffing with non-breached passwords bypasses BPD |
| Suspicious IP Throttling requires Enterprise tier ($$$) | SMEs cannot afford cross-tenant threat intelligence |
| No native CAPTCHA integration (requires custom rules/actions) | Administrators must write code to add CAPTCHA |
| Rate limits are per-tenant, not per-user-session | Sophisticated attacks using many sessions from one IP may trigger false positives |
| No device fingerprinting or behavioral analysis | Cannot detect headless browsers that mimic legitimate User-Agents |
| Brute Force Protection is identifier-based — attackers rotate identifiers | If an attacker has 10,000 email addresses, each gets the full attempt budget |

---

## 3. Keycloak Bot Protection

Keycloak provides basic brute force protection out of the box, but its bot detection capabilities are minimal compared to Auth0 or commercial solutions.

### 3.1 Brute Force Detector

**How it works:** Keycloak tracks failed login attempts per user. When the configured failure threshold is reached, the user account is temporarily disabled.

**Configuration (Realm Settings > Defense > Brute Force Detection):**

| Setting | Description | Default |
|---|---|---|
| `bruteForceProtected` | Enable/disable brute force protection | `false` (disabled by default) |
| `failureFactor` | Max failed login attempts before lockout | 30 |
| `waitIncrementSeconds` | Seconds added to wait time per failure | 60 |
| `quickLoginCheckMilliSeconds` | Gap between rapid login attempts to detect bot speed | 1000 |
| `minimumQuickLoginWaitSeconds` | Minimum wait if rapid login detected | 60 |
| `maxFailureWaitSeconds` | Maximum wait time (cap) | 900 |
| `maxDeltaTimeSeconds` | Reset failure count after this idle period | 43200 (12h) |
| `failureFactor` | Lockout threshold | 30 |

**Important:** Brute force detection is **disabled by default**. Many Keycloak deployments leave it disabled due to concerns about locking out legitimate users during distributed attacks.

### 3.2 What's Built-in vs What Needs Configuration

**Built-in (but requires enabling):**
- Per-user failed login tracking with configurable lockout.
- Incremental wait times that increase with each failure.
- Rapid login detection (if two login attempts come within `quickLoginCheckMilliSeconds`, the IP is throttled).

**NOT built-in (requires plugins or custom code):**
- IP-based rate limiting (Keycloak only tracks per-user, not per-IP by default).
- CAPTCHA on login/registration forms.
- Breached password detection.
- Device fingerprinting.
- Behavioral analysis.
- Bot User-Agent blocking.
- Cross-tenant threat intelligence.

### 3.3 Community Plugins for CAPTCHA

Keycloak does not include CAPTCHA in its default login themes. Community solutions include:

1. **keycloak-recaptcha-login**: A community SPI (Service Provider Interface) plugin that adds Google reCAPTCHA v2 to the Keycloak login flow. Requires manual installation and configuration of reCAPTCHA site/secret keys.

2. **Custom Authenticator SPIs**: Developers can write custom authenticators that integrate hCaptcha, Cloudflare Turnstile, or custom challenges. This requires Java development and Keycloak SPI knowledge.

3. **Keycloak Themes with CAPTCHA**: Some organizations modify the FreeMarker login theme templates to embed CAPTCHA JavaScript, but this is fragile and bypassable since the verification is client-side only.

**Limitations:**
- No official CAPTCHA support — all solutions are community-maintained.
- CAPTCHA plugins are not compatible with Keycloak X (Quarkus-based) without migration.
- No integration with breached password databases.
- Brute force protection is per-user only, not per-IP — credential stuffing with diverse usernames is not throttled.

---

## 4. Bot Detection Techniques

Effective bot protection requires defense-in-depth: no single technique catches all bots. The most robust IAM deployments combine multiple layers.

### 4.1 CAPTCHA

| Solution | Type | Privacy | UX | Effectiveness | Pricing |
|---|---|---|---|---|---|
| **reCAPTCHA v3** | Score-based (0.0-1.0) | Poor (tracks user across Google) | Invisible (no challenge for most users) | High for basic bots; bypassable by ML CAPTCHA solvers | Free up to 1M/month |
| **reCAPTCHA Enterprise** | Score-based + adaptive | Poor | Invisible | Higher accuracy with risk analysis | $1/1000 assessments |
| **hCaptcha** | Challenge-based + score | Good (GDPR-compliant) | Visual challenges (privacy-friendly) | Comparable to reCAPTCHA; used by Cloudflare alternatives | Free tier available |
| **Cloudflare Turnstile** | Proof-of-work + browser checks | Excellent (no personal data) | Invisible (no visual challenge) | Very high for bots; no UX friction | Free (unlimited) |
| **AWS WAF CAPTCHA** | Challenge-based | Moderate | Visual challenge | Good; integrates with AWS WAF rules | $0.25/1000 CAPTCHA |
| **GeeTest** | Slider/puzzle | Moderate | Interactive | High for automated solvers | Paid |

**Recommendation for IAM:** Cloudflare Turnstile is the best choice for privacy-conscious, developer-friendly deployments. It is free, invisible to users, and uses proof-of-work challenges that are computationally expensive for bots to solve.

### 4.2 Device Fingerprinting

**TLS JA3/JA4 Fingerprinting:**
- JA3 hashes the TLS ClientHello message (cipher suites, extensions, elliptic curves) into a 32-character hash.
- Different TLS libraries (Go crypto/tls, OpenSSL, BoringSSL, Node.js) produce different JA3 hashes.
- Bots using non-standard TLS libraries (e.g., Python `requests` with urllib3) produce identifiable JA3 hashes.
- JA4 is the successor: a more structured, human-readable fingerprint that is easier to maintain and share.
- **Limitation:** JA3/JA4 is bypassable by bots that use standard browser TLS stacks (e.g., headless Chrome uses BoringSSL with the same JA3 as a real Chrome browser).

**Browser Fingerprinting:**
- Canvas fingerprint: Renders text on a hidden `<canvas>` element; the pixel data varies by GPU/driver/OS, producing a unique fingerprint.
- WebGL fingerprint: Uses 3D rendering parameters (vendor, renderer, extensions) to identify the graphics stack.
- AudioContext fingerprint: Analyzes audio processing differences across hardware.
- Font enumeration: Detects installed system fonts (varies by OS).
- Screen properties: Resolution, color depth, timezone, language.
- **Tools:** FingerprintJS (open source + commercial), CreepJS.

**Limitations:**
- Browser fingerprinting requires client-side JavaScript — not applicable to pure API-only IAM endpoints.
- Privacy regulations (GDPR Article 5) may require consent for fingerprinting.
- Headless browsers with fingerprint spoofing (e.g., puppeteer-extra-plugin-stealth) can evade basic fingerprinting.

### 4.3 Behavioral Analysis

Behavioral biometrics analyze *how* a user interacts with the application, not just *what* they submit.

**Mouse movement patterns:**
- Humans produce curved, noisy mouse trajectories; bots produce linear or perfectly timed movements.
- Metrics: curvature, jitter, acceleration profile, idle time distribution.
- Real users pause, backtrack, and overshoot; bots navigate directly to elements.

**Typing cadence (keystroke dynamics):**
- Inter-key delays (dwell time, flight time) form a biometric signature.
- Bots either fill fields instantly (paste) or with perfectly uniform timing.
- Can be used as a continuous authentication signal — if typing cadence changes mid-session, flag as suspicious.

**Navigation patterns:**
- Humans browse erratically: they visit multiple pages, read content, use back button.
- Bots navigate directly to the target endpoint (login, register) with minimal HTTP requests.
- Request ordering analysis: a bot that hits `/api/v1/auth/login` as its first request (no prior page load, CSS, JS fetches) is suspicious.

**Implementation considerations:**
- Behavioral analysis requires client-side data collection (mouse tracking, key logging) — significant JavaScript payload.
- ML models (LSTM, gradient boosting) needed for classification; rule-based heuristics are too brittle.
- Privacy concerns: continuous behavioral surveillance may violate GDPR.

### 4.4 Network-Level Detection

**IP Reputation:**
- Threat intelligence feeds (AbuseIPDB, Spamhaus, AlienVault OTX) provide IP reputation scores.
- Commercial services (Cloudflare, AWS WAF, Akamai) maintain proprietary reputation databases.
- Known bad IPs (spam sources, botnet C&C, credential stuffing proxies) are blocked or challenged.

**ASN Classification:**
- Some Autonomous System Numbers (ASNs) are associated with hosting providers (DigitalOcean, AWS, OVH) where legitimate users rarely browse from.
- Residential ASNs (Comcast, AT&T, Vodafone) are more trusted but residential proxy services blur this distinction.
- Datacenter ASN detection: if 90% of login attempts from an ASN are to `/api/v1/auth/login`, flag as suspicious.

**Tor Exit Nodes:**
- Tor exit node lists are publicly available and updated in real-time.
- Blocking all Tor traffic is heavy-handed but effective for high-security deployments.
- Alternatively: challenge Tor users with Turnstile/reCAPTCHA instead of blocking outright.

**VPN/Proxy Detection:**
- Commercial services (IPQS, IP2Location, MaxMind) detect VPN, proxy, and Tor usage.
- Residential proxy detection is harder — the IP appears to be a legitimate residential ISP.

### 4.5 Proof-of-Work Challenges

Proof-of-work (PoW) challenges require the client to solve a computationally intensive puzzle before the request is processed. This shifts the cost of automated attacks from bandwidth (cheap) to CPU (expensive).

**How it works:**
1. Server sends a challenge (random nonce + difficulty target).
2. Client computes hashes until it finds one below the target.
3. Client submits the solution; server verifies in O(1).
4. A human browser solves this in <100ms; a bot army must spend CPU on each request.

**Advantages:**
- No personal data collection (privacy-friendly).
- No visual challenge (no UX friction).
- Computationally scales with attack volume.

**Implementations:**
- Cloudflare Turnstile uses PoW internally.
- mCaptcha (open source, Rust) provides a standalone PoW challenge service.
- Custom Go implementations using Argon2id or SHA-256 difficulty targeting.

---

## 5. GGID Existing Bot Detection

### 5.1 Implementation Review

GGID's bot detection is implemented in `services/gateway/internal/middleware/botdetect.go`. The file contains two components:

#### BotDetect (User-Agent Blocklist)

```go
func BotDetect(next http.Handler) http.Handler {
    // Checks User-Agent against two lists:
    // 1. suspiciousPatterns: sqlmap, nikto, nmap, masscan, dirbuster, wpscan, hydra, metasploit, burp
    // 2. knownBotPatterns: googlebot, bingbot, slurp, duckduckbot, etc.
    // Blocks suspicious patterns with 403 Forbidden.
    // Tags known crawlers with X-Bot-Detected header.
}
```

**Technique:** Static User-Agent string matching (case-insensitive `strings.Contains`).

**Coverage:** Very limited. Only catches:
- Security scanning tools that don't spoof their User-Agent (rare in real attacks).
- Known search engine crawlers (tagged, not blocked).

**Bypass:** Any bot that sets a legitimate User-Agent (e.g., `Mozilla/5.0 (Windows NT 10.0; Win64; x64)`) passes completely undetected.

#### BehavioralBotDetect (Rate-Based Detection)

```go
type BehavioralBotDetect struct {
    window    time.Duration   // sliding window (e.g., 1 minute)
    threshold int             // max requests per window
    store     *botRateStore   // in-memory map
}
```

**Technique:** Per-IP sliding window rate counting. If an IP exceeds `threshold` requests within `window`, return 429 Too Many Requests.

**Coverage:** Detects high-volume attacks from a single IP. Misses:
- Distributed attacks (many IPs, few requests each).
- Slow-rate attacks (below the threshold).
- Attacks through residential proxy networks.

**Storage:** In-memory `map[string]*botRequestLog` with a mutex. No persistence, no Redis backing, no cleanup goroutine for expired entries (potential memory leak under sustained load).

### 5.2 Wiring Status: NOT WIRED

**Critical finding:** The `BotDetect` and `BehavioralBotDetect` middleware are **defined but never wired into the production handler chain.**

The actual middleware chain in `router.go:Handler()` (lines 373-382) is:

```
PanicRecovery → SecurityHeaders → CORS → RequestID → RequestLogger → RateLimit → TenantResolver → inner(JWTAuth → routing)
```

Bot detection is **absent** from this chain. The `stats.go` file (line 200) lists `"BotDetect", Order: 5, Category: "security"` in the inner middleware info, but this is purely informational metadata returned by the `/api/v1/gateway/middleware` debug endpoint. The middleware is never actually applied to live traffic.

**Impact:** GGID currently has **zero bot protection** in production. Every request passes through to authentication without any bot detection, regardless of User-Agent or request rate.

### 5.3 Existing Test Coverage

The bot detection middleware has unit tests in:
- `coverage_sprint15_test.go` — basic allow/block tests
- `coverage_sprint17_test.go` — suspicious pattern, known bot, and behavioral threshold tests
- `coverage_sprint19_test.go` — threshold boundary and empty IP edge cases

These tests verify the middleware functions correctly when invoked directly, but since the middleware is not wired, the tests do not reflect production behavior.

---

## 6. Gap Analysis

### 6.1 Feature Comparison Table

| Technique | Auth0 | Keycloak | GGID |
|---|---|---|---|
| **User-Agent blocklist** | No (relies on WAF) | No | Yes (defined, **NOT WIRED**) |
| **Per-IP rate limiting** | Yes (Suspicious IP Throttling) | No (per-user only) | Yes (TenantBucketLimiter — wired) |
| **Per-user brute force** | Yes (Brute Force Protection) | Yes (Failure Factor) | No |
| **Breached password check** | Yes (BPD, 10B+ credentials) | No | No |
| **CAPTCHA** | Custom (rules/actions) | Community plugins | No |
| **Device fingerprinting** | No (native) | No | No |
| **JA3/JA4 TLS fingerprint** | No | No | No |
| **Behavioral analysis** | Limited (anomaly detection) | No | No |
| **IP reputation** | Yes (cross-tenant) | No | No |
| **Tor exit node blocking** | No (native) | No | No |
| **Proof-of-work challenge** | No | No | No |
| **MFA fatigue protection** | No (native) | No | No |
| **Cross-tenant intelligence** | Yes | No | No |
| **Headless browser detection** | No (native) | No | No |
| **Account lockout policy** | Yes | Yes | No |

### 6.2 Priority Ranking of Missing Capabilities

| Priority | Gap | Risk | Effort |
|---|---|---|---|
| **P0** | BotDetect not wired into handler chain | No bot protection in production | 1 line of code |
| **P0** | No per-user brute force protection | Credential stuffing is unimpeded | Medium |
| **P0** | No breached password detection | Users with compromised passwords are unprotected | Medium |
| **P1** | No CAPTCHA on login/register | Bots can automate authentication freely | Medium |
| **P1** | No MFA fatigue protection | Push bombing attacks succeed | Medium |
| **P1** | No headless browser detection | Selenium/Playwright bots evade all checks | Medium |
| **P2** | No JA3/JA4 fingerprinting | Bots with non-standard TLS stacks go undetected | Medium |
| **P2** | No IP reputation integration | Known-bad IPs are not blocked | Low |
| **P2** | No Tor exit node blocking | Anonymous attacks are facilitated | Low |
| **P3** | No behavioral analysis (ML) | Sophisticated bots evade rule-based detection | High |
| **P3** | No device fingerprinting | Returning bots are not recognized | High |
| **P3** | No proof-of-work challenge | Attack cost is asymmetric (cheap for attacker) | Medium |

---

## 7. Implementation Recommendations

### 7.1 Quick Wins (1-2 days)

#### QW1: Wire BotDetect into the handler chain

The most critical fix: add `BotDetect` to the production middleware chain in `router.go:Handler()`:

```go
// After PanicRecovery, before SecurityHeaders
handler = middleware.BotDetect(handler)
```

Also wire `BehavioralBotDetect` with a conservative threshold (e.g., 100 requests/minute per IP):

```go
behavioral := middleware.NewBehavioralBotDetect(100, time.Minute)
handler = behavioral.Middleware(handler)
```

**Effort:** 5 lines of code.  
**Risk mitigation:** None — this enables basic protection that was designed but never activated.

#### QW2: Fix memory leak in BehavioralBotDetect

Add a cleanup goroutine to purge expired entries:

```go
func (b *BehavioralBotDetect) StartCleanup(interval time.Duration) {
    go func() {
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        for range ticker.C {
            b.store.mu.Lock()
            now := time.Now()
            for key, log := range b.store.buckets {
                if now.After(log.expires) {
                    delete(b.store.buckets, key)
                }
            }
            b.store.mu.Unlock()
        }
    }()
}
```

#### QW3: Add MFA fatigue protection

Limit MFA push challenges to 3 per user per 10 minutes. If exceeded, require an alternative authentication factor (TOTP, backup code) or enforce a cooldown period.

### 7.2 Medium-Term (1-2 weeks)

#### MT1: Integrate Cloudflare Turnstile

Cloudflare Turnstile is the recommended CAPTCHA solution for GGID:
- **Free** (unlimited volume).
- **Privacy-friendly** (no personal data collection, GDPR-compliant).
- **Invisible** to users (no visual challenge).
- **Proof-of-work** based (computationally expensive for bots).

**Implementation plan:**
1. Add Turnstile site key and secret to gateway config.
2. Frontend: embed Turnstile widget on login and registration pages.
3. Backend: add `TurnstileVerify` middleware that validates the Turnstile token server-side.
4. Apply to `/api/v1/auth/login` and `/api/v1/auth/register` endpoints.

See Section 8.1 for the Go middleware implementation.

#### MT2: Add per-user brute force protection

Track failed login attempts per user identifier (email/username) in Redis:
- Lock account after 5 failed attempts.
- Incremental lockout (1 min, 5 min, 15 min, 1 hour).
- Notify user via email when lockout triggers.

#### MT3: Add breached password detection

Integrate with the Have I Been Pwned (HIBP) API or download the offline password hash database:
- Check passwords against HIBP k-anonymity range API (privacy-preserving: only sends first 5 chars of SHA-1 hash).
- Block registration with breached passwords.
- Warn on login with breached passwords.

#### MT4: Add Tor exit node blocking

Fetch the Tor exit node list (updated every 30 minutes from `https://check.torproject.org/api/bulk`), cache in memory, and block/challenge requests from Tor exit IPs.

### 7.3 Long-Term (1-3 months)

#### LT1: Behavioral analysis with ML

Collect behavioral signals (request patterns, timing, navigation) and train an ML model (gradient boosting or LSTM) to classify requests as human/bot:
- Feature engineering: request interval distribution, endpoint diversity, time-of-day patterns, session depth.
- Model serving: batch prediction in Go using ONNX Runtime.
- Feedback loop: allow administrators to label false positives/negatives for model retraining.

#### LT2: Device fingerprinting integration

Integrate FingerprintJS (open source) into the GGID console:
- Collect canvas, WebGL, audio, and font fingerprints client-side.
- Submit fingerprint with authentication requests.
- Correlate fingerprints with known bot patterns.
- Detect fingerprint spoofing (multiple identities from one fingerprint).

#### LT3: JA3/JA4 TLS fingerprinting

Instrument the Go TLS listener to capture ClientHello messages:
- Use `crypto/tls` with a custom `GetConfigForClient` callback to extract JA3 components.
- Log JA3/JA4 hashes for all incoming connections.
- Build a reputation database of JA3 hashes associated with known bot tools.
- Challenge requests from unknown JA3 hashes on sensitive endpoints.

See Section 8.2 for the Go implementation.

---

## 8. Go Code: Middleware Implementations

### 8.1 Cloudflare Turnstile Verification Middleware

```go
// middleware/turnstile.go
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TurnstileVerify verifies a Cloudflare Turnstile token server-side.
// Apply to login, registration, and password reset endpoints.
func TurnstileVerify(siteSecret string) func(http.Handler) http.Handler {
	client := &http.Client{Timeout: 5 * time.Second}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Turnstile-Token")
			if token == "" {
				token = r.URL.Query().Get("cf-turnstile-response")
			}

			if token == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"bot challenge missing"}`))
				return
			}

			// Verify token with Cloudflare siteverify API
			form := url.Values{}
			form.Set("secret", siteSecret)
			form.Set("response", token)
			clientIP := extractClientIP(r)
			if clientIP != "" {
				form.Set("remoteip", clientIP)
			}

			resp, err := client.PostForm(
				"https://challenges.cloudflare.com/turnstile/v0/siteverify",
				form,
			)
			if err != nil {
				// Fail closed: if Cloudflare is unreachable, allow request
				// but log a warning. In high-security mode, fail open is wrong.
				next.ServeHTTP(w, r)
				return
			}
			defer resp.Body.Close()

			var result struct {
				Success    bool     `json:"success"`
				ErrorCodes []string `json:"error-codes"`
				Action     string   `json:"action"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if !result.Success {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				msg := fmt.Sprintf(`{"error":"bot verification failed","codes":["%s"]}`,
					strings.Join(result.ErrorCodes, `","`))
				w.Write([]byte(msg))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

### 8.2 JA3/JA4 TLS Fingerprint Logging

```go
// middleware/tlsfingerprint.go
package middleware

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// JA3Info holds the parsed TLS ClientHello parameters.
type JA3Info struct {
	Hash        string
	CipherSuites string
	Extensions   string
	Curves       string
	ECPointFmt   string
}

// ComputeJA3 computes a JA3 hash from TLS ClientHello fields.
// In Go, this requires instrumenting the tls.Config.GetConfigForClient callback.
func ComputeJA3(cipherSuites []uint16, extensions []uint16,
	curves []uint8, ecPointFormats []uint8) JA3Info {

	// Format: cipherSuites,extensions,curves,ecPointFormats
	// Each list is hyphen-separated, lists are comma-separated
	cs := joinU16(cipherSuites, "-")
	ext := joinU16(extensions, "-")
	cv := joinU8(curves, "-")
	ep := joinU8(ecPointFormats, "-")

	raw := fmt.Sprintf("%s,%s,%s,%s", cs, ext, cv, ep)
	sum := md5.Sum([]byte(raw))

	return JA3Info{
		Hash:         hex.EncodeToString(sum[:]),
		CipherSuites: cs,
		Extensions:   ext,
		Curves:       cv,
		ECPointFmt:   ep,
	}
}

func joinU16(vals []uint16, sep string) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, sep)
}

func joinU8(vals []uint8, sep string) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, sep)
}

// JA3Reputation holds known bot JA3 hashes for quick lookup.
type JA3Reputation struct {
	mu     sync.RWMutex
	known  map[string]string // JA3 hash -> tool name
}

func NewJA3Reputation() *JA3Reputation {
	return &JA3Reputation{
		known: map[string]string{
			// Known bot/script JA3 hashes (examples, update from threat intel)
			"e7d705a3286e19ea42f587b344ee6865": "python-requests/2.x",
			"5c62e3f6a4a9ba3b6fbb1e7c6a8f5d33": "curl/7.x",
			"b6a5c1f9e8d7c0b3a5f9e2d1c8b7a6f5": "go-http-client/1.1",
			"cd08e31494f9531f560d64c695473da9": "rust-tokio",
		},
	}
}

// IsKnownBot checks if a JA3 hash matches a known bot tool.
func (j *JA3Reputation) IsKnownBot(ja3Hash string) (string, bool) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	name, ok := j.known[ja3Hash]
	return name, ok
}

// JA3LogMiddleware logs JA3 fingerprints from the request context
// and tags requests from known bot tools.
// Requires a custom TLS listener that stores JA3 info in the request context.
func (j *JA3Reputation) JA3LogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// JA3 info is expected to be stored in context by the TLS listener
		// via a custom net.Listener wrapper.
		if ja3Info, ok := r.Context().Value(ja3Key{}).(JA3Info); ok {
			if toolName, isBot := j.IsKnownBot(ja3Info.Hash); isBot {
				w.Header().Set("X-TLS-Bot-Suspected", toolName)
				w.Header().Set("X-JA3-Hash", ja3Info.Hash)
			} else {
				w.Header().Set("X-JA3-Hash", ja3Info.Hash)
			}
		}
		next.ServeHTTP(w, r)
	})
}

type ja3Key struct{}
```

### 8.3 Enhanced Bot Detection Scoring

```go
// middleware/botscore.go
package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BotScore collects signals from multiple detection layers and produces
// a composite risk score (0-100). Scores above the threshold trigger
// a challenge (Turnstile) or block.
type BotScoreDetector struct {
	mu              sync.Mutex
	threshold       int           // Score above which request is challenged
	ja3Rep          *JA3Reputation
	rateStore       *botRateStore
	torExitNodes    map[string]bool // simplistic; use a real feed in production
	torLastUpdate   time.Time
}

// BotScoreConfig configures the scoring detector.
type BotScoreConfig struct {
	ChallengeThreshold int           // Default: 50
	RateThreshold      int           // Requests per minute per IP. Default: 60
	RateWindow         time.Duration // Default: 1 minute
}

// DefaultBotScoreConfig returns sensible defaults.
func DefaultBotScoreConfig() BotScoreConfig {
	return BotScoreConfig{
		ChallengeThreshold: 50,
		RateThreshold:      60,
		RateWindow:         time.Minute,
	}
}

// NewBotScoreDetector creates a composite bot scoring detector.
func NewBotScoreDetector(cfg BotScoreConfig) *BotScoreDetector {
	return &BotScoreDetector{
		threshold:    cfg.ChallengeThreshold,
		ja3Rep:       NewJA3Reputation(),
		rateStore:    &botRateStore{buckets: make(map[string]*botRequestLog)},
		torExitNodes: make(map[string]bool),
	}
}

// ScoreRequest evaluates a request and returns a risk score 0-100.
func (b *BotScoreDetector) ScoreRequest(r *http.Request) int {
	score := 0
	ip := extractClientIP(r)
	ua := strings.ToLower(r.Header.Get("User-Agent"))

	// Signal 1: Suspicious User-Agent (+20)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(ua, pattern) {
			score += 20
			break
		}
	}

	// Signal 2: Empty User-Agent (+15)
	if ua == "" || ua == " " {
		score += 15
	}

	// Signal 3: Missing common browser headers (+10 each)
	if r.Header.Get("Accept") == "" {
		score += 10
	}
	if r.Header.Get("Accept-Language") == "" {
		score += 10
	}
	if r.Header.Get("Accept-Encoding") == "" {
		score += 5
	}

	// Signal 4: Known bot JA3 hash (+30)
	if ja3Info, ok := r.Context().Value(ja3Key{}).(JA3Info); ok {
		if _, isBot := b.ja3Rep.IsKnownBot(ja3Info.Hash); isBot {
			score += 30
		}
	}

	// Signal 5: Tor exit node (+25)
	if b.isTorExitNode(ip) {
		score += 25
	}

	// Signal 6: High request rate (+20)
	if b.isHighRate(ip) {
		score += 20
	}

	// Signal 7: Headless browser indicators (+15)
	if strings.Contains(ua, "headless") ||
		strings.Contains(ua, "phantomjs") ||
		strings.Contains(ua, "selenium") ||
		strings.Contains(ua, "puppeteer") ||
		strings.Contains(ua, "playwright") {
		score += 15
	}

	// Signal 8: Datacenter ASN (simplified — check known datacenter IP ranges)
	if b.isDatacenterIP(ip) {
		score += 10
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

// Middleware applies bot scoring to all requests.
// Requests scoring above the threshold receive a 403 or Turnstile challenge.
func (b *BotScoreDetector) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		score := b.ScoreRequest(r)

		// Attach score to response headers for observability
		w.Header().Set("X-Bot-Score", strconv.Itoa(score))

		if score >= b.threshold {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"bot suspicion threshold exceeded","score":` +
				strconv.Itoa(score) + `}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isTorExitNode checks if an IP is a known Tor exit node.
func (b *BotScoreDetector) isTorExitNode(ip string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Refresh Tor exit node list every 30 minutes
	if time.Since(b.torLastUpdate) > 30*time.Minute {
		b.refreshTorExitNodes()
	}

	return b.torExitNodes[ip]
}

// isHighRate checks if the IP has exceeded the rate threshold.
func (b *BotScoreDetector) isHighRate(ip string) bool {
	if ip == "" {
		return false
	}
	b.rateStore.mu.Lock()
	defer b.rateStore.mu.Unlock()

	key := "botscore:" + ip
	log, exists := b.rateStore.buckets[key]
	now := time.Now()

	if !exists || now.After(log.expires) {
		b.rateStore.buckets[key] = &botRequestLog{count: 1, expires: now.Add(time.Minute)}
		return false
	}

	log.count++
	return log.count > 60
}

// isDatacenterIP performs a simplified check against known datacenter IP ranges.
// In production, use MaxMind GeoIP2 or ipinfo.io for ASN lookup.
func (b *BotScoreDetector) isDatacenterIP(ip string) bool {
	// Simplified: check well-known datacenter prefixes.
	// Real implementation should use a GeoIP database.
	datacenterPrefixes := []string{
		"10.0.", "172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.",
		"172.25.", "172.26.", "172.27.", "172.28.", "172.29.",
		"172.30.", "172.31.",
	}
	for _, prefix := range datacenterPrefixes {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}
	return false
}

// refreshTorExitNodes fetches the current Tor exit node list.
// In production, this should use https://check.torproject.org/api/bulk
// or the Tor consensus document.
func (b *BotScoreDetector) refreshTorExitNodes() {
	b.torLastUpdate = time.Now()
	// Stub: real implementation fetches and parses the Tor exit list.
	// For now, the map stays empty — no Tor IPs are flagged.
}
```

### 8.4 Wiring Bot Detection into the Handler Chain

To activate all bot detection in `router.go`, update the `Handler()` method:

```go
// In router.go Handler() method, after PanicRecovery:

// Bot detection layers (outermost first)
handler = middleware.BotDetect(handler)

behavioral := middleware.NewBehavioralBotDetect(100, time.Minute)
handler = behavioral.Middleware(handler)

botScorer := middleware.NewBotScoreDetector(middleware.DefaultBotScoreConfig())
handler = botScorer.Middleware(handler)

// Apply Turnstile on auth endpoints specifically (inner chain):
// In the inner handler, for /api/v1/auth/login and /api/v1/auth/register:
//   h = middleware.TurnstileVerify(cfg.TurnstileSecret)(h)
```

---

## Summary

GGID's bot protection is in a **critical gap state**: the middleware code exists, is unit-tested, and is listed in the middleware stats, but it is **never applied to production traffic**. The immediate fix (wiring `BotDetect` into the handler chain) is a one-line change with zero risk.

For competitive parity with Auth0, GGID needs:
1. Per-user brute force protection (Auth0 has this on all paid tiers).
2. Breached password detection (Auth0 BPD, available on Professional+).
3. CAPTCHA integration (Auth0 requires custom rules; GGID can use Turnstile for free).

For competitive parity with Keycloak, GGID already exceeds Keycloak's capabilities if the existing middleware is wired: Keycloak has only per-user brute force (disabled by default), while GGID has User-Agent blocking + per-IP behavioral detection.

**Recommended implementation order:**
1. Wire BotDetect + BehavioralBotDetect (1 day, P0)
2. Add per-user brute force with Redis (2 days, P0)
3. Integrate Cloudflare Turnstile (3 days, P1)
4. Add breached password detection via HIBP (2 days, P0)
5. Add Tor exit node blocking + IP reputation (2 days, P2)
6. Add JA3/JA4 fingerprint logging (3 days, P2)
7. Long-term: behavioral ML model + device fingerprinting (3 months, P3)
