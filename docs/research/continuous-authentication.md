# Continuous Authentication: Beyond Session-Based Trust

> **Scope:** Continuous authentication — evaluating trust on **every request**,
> not just at login. Complements `adaptive-mfa-design.md` (MFA scoring) and
> `step-up-authentication-patterns.md` (elevation flows) without repeating their
> algorithms. Focus: behavioral biometrics, device fingerprinting, per-request
> risk evaluation, and adaptive session management in GGID's gateway.

---

## 1. Overview

Traditional authentication is **verify-once-then-trust**: a user authenticates at
login, receives a session token, and that token is implicitly trusted for its
entire lifetime. If an attacker steals the token mid-session, they operate with
full privileges until expiry or revocation.

**Continuous authentication** replaces this binary gate with ongoing trust
evaluation: every request is scored against behavioral, device, and contextual
signals, and the session's risk profile updates in real time.

NIST SP 800-207 (Zero Trust Architecture) mandates this:

> *"Access to individual enterprise resources is granted on a per-session basis...
> authentication and authorization... are performed before a connection is
> established."* — SP 800-207, §3.1

| Layer | What It Measures | When Evaluated |
|-------|-----------------|----------------|
| Behavioral biometrics | Keystroke rhythm, mouse gait, touch pressure | Async, per-batch |
| Device posture | Fingerprint match, managed/unmanaged | Per-request |
| Request anomaly | IP geo-velocity, request rate, time-of-day | Per-request |
| Session risk | Cumulative score, age, step-up state | Per-request (cached 60s) |

**Goal:** detect account takeover *during* the session, not just at the login
gate, by observing that the current request no longer matches the legitimate
user's established baseline.

---

## 2. Behavioral Biometrics

Behavioral biometrics analyze **how** a user interacts — typing rhythm, mouse
movement, touch pressure — rather than **what** they know. This creates a unique
fingerprint that is continuously available throughout a session.

### Keystroke Dynamics

Two primary signals:
- **Flight time:** duration between releasing one key and pressing the next.
- **Dwell time:** duration a key is held down.

An individual's typing rhythm is consistent and difficult to mimic. The 2023
arXiv survey *"Keystroke Dynamics: Concepts, Techniques, and Applications"*
reports identification accuracy exceeding 99% with 200+ keystrokes using
statistical and distance-based classifiers.

**Algorithm:** (1) Train a baseline from the first 200+ keystrokes. (2) Extract
`(key, dwell_ms, flight_ms)` vectors. (3) Compare subsequent batches against the
baseline (Manhattan distance or SVM). (4) If distance exceeds threshold for N
consecutive batches, flag anomaly.

### Mouse Movement & Touch

Mouse trajectories exhibit gait-like signatures: curvature, speed, acceleration,
overshoot patterns. Collect `mousemove` events (throttled ~50ms), batch into 5-10s
windows, evaluate server-side. On mobile: swipe speed, touch pressure, gyroscope
angle provide additional signals unique per user.

### Implementation Considerations

| Concern | Mitigation |
|---------|-----------|
| Privacy | Biometric data under GDPR Art. 9 — requires explicit consent |
| Storage | Store only derived feature vectors, never raw event streams |
| Performance | Evaluate asynchronously — push to queue, don't block requests |
| False positives | Start conservative; tune per tenant; use as risk multiplier, not a gate |

---

## 3. Device Fingerprinting

### Techniques

| Category | Attributes | Stability |
|----------|-----------|-----------|
| Browser | UA, screen res, timezone, fonts, WebGL, canvas hash | Medium |
| HTTP | Accept headers, TLS JA3 hash, HTTP/2 settings | High |
| Hardware | CPU cores, memory, battery level | Low |
| Persistent | Cookies, localStorage, IndexedDB | High |

### Collection

A JavaScript library collects 30+ attributes and hashes them to a compact
fingerprint sent as `X-Device-Fingerprint` header:

```javascript
const attrs = [navigator.userAgent, navigator.language,
    `${screen.width}x${screen.height}`, new Date().getTimezoneOffset(),
    await canvasHash(), webglRenderer()].join('|');
const buf = await crypto.subtle.digest('SHA-256',
    new TextEncoder().encode(attrs));
// → hex string sent as X-Device-Fingerprint
```

### Privacy Concerns

Fingerprinting without consent violates GDPR Art. 5(1)(a) (transparency) and
CCPA. Requirements: disclose in privacy policy, offer opt-out, and prefer
WebAuthn device attestation — which proves device possession cryptographically
without revealing a trackable fingerprint.

### GGID Integration

- **Gateway:** extract `X-Device-Fingerprint` header in new middleware.
- **Auth service:** store alongside session metadata in existing Redis key
  `ggid:session:{sessionID}`.
- **Risk:** mismatched fingerprint increases device-mismatch risk component.

---

## 4. Per-Request Risk Evaluation

### Risk Signals

| Signal | Source | Weight | Anomaly Detection |
|--------|--------|--------|-------------------|
| IP geo-velocity | Request IP vs baseline | 0.25 | >500km since last, impossible travel |
| User-Agent change | UA vs baseline | 0.15 | String or parsed browser diff |
| Request velocity | Redis counter per session | 0.15 | >20 req/s → bot-like |
| Resource sensitivity | Endpoint classification | 0.10 | Admin endpoint adds base 15 |
| Time of day | Request timestamp | 0.05 | Off-hours adds 5-10 |
| Session age | Token `iat` claim | 0.10 | >4h adds 5; >12h adds 15 |
| Failed auth attempts | Redis counter | 0.10 | >3 failures/5min adds 10-20 |
| Device posture | Managed? Encrypted? EDR? | 0.10 | Unmanaged→admin adds 15 |

### Score Calculation

```
risk = Σ(weight_i × anomaly_i) × 100    // each anomaly_i ∈ [0, 1]
```

| Score | Action |
|-------|--------|
| 0-29 | Allow |
| 30-59 | Allow, require step-up for sensitive ops |
| 60-79 | Require re-authentication |
| 80-100 | Block, invalidate session, alert SOC |

Thresholds are **configurable per tenant** via the policy service. Scores cached
per session in Redis for 60s; sensitive endpoints force fresh evaluation.

### Go Implementation

```go
// pkg/risk/evaluator.go
package risk

type RiskSignal interface {
    Name() string
    Collect(ctx context.Context, r *http.Request) float64 // [0, 1]
}

type Score struct {
    Value      float64
    Action     Action
    Components map[string]float64
}

func (e *Engine) Evaluate(ctx context.Context, r *http.Request) Score {
    sessID := middleware.SessionIDFromContext(ctx)
    if cached := e.getCached(sessID); cached != nil {
        return *cached
    }
    sc := Score{Components: map[string]float64{}}
    for _, sig := range e.signals {
        a := sig.Collect(ctx, r)
        sc.Components[sig.Name()] = a * 100
        sc.Value += a * 100
    }
    sc.Action = e.thresholds.toAction(sc.Value)
    e.cacheScore(sessID, sc, 60*time.Second)
    return sc
}
```

---

## 5. Adaptive Session Timeout

### Dynamic TTL Based on Risk

| Risk | Refresh TTL | Rationale |
|------|------------|-----------|
| Low (0-29) | 7 days | User is legitimate — extend trust |
| Medium (30-59) | 24 hours | Some deviation — shorten window |
| High (60-79) | 1 hour | Significant risk — re-auth soon |
| Critical (80+) | Revoke now | Session compromised |

The **access token TTL stays fixed at 15 minutes** — short-lived tokens must not
be extended.

```go
func (asm *AdaptiveSessionManager) AdjustTTL(ctx context.Context, sessID string,
    score risk.Score) error {
    var ttl time.Duration
    switch {
    case score.Value >= 80:
        return asm.MarkSessionRevoked(ctx, sessID) // immediate
    case score.Value >= 60:
        ttl = time.Hour
    case score.Value >= 30:
        ttl = 24 * time.Hour
    default:
        ttl = 7 * 24 * time.Hour
    }
    return asm.rdb.Expire(ctx, "ggid:session:"+sessID+":refresh", ttl).Err()
}
```

This extends GGID's existing `SessionManager.touchSessionTTL` — the difference
is TTL is now risk-adaptive rather than fixed.

---

## 6. Step-Up Triggers (Continuous)

Continuous step-up triggers *during* an active session when risk crosses a
threshold on a sensitive endpoint:

1. User at AAL1 (password). Risk rises mid-session (IP change, device change).
2. Gateway detects score ≥ 30 on a sensitive endpoint (`/admin/*`, `DELETE`).
3. Returns **403** with step-up challenge:

```http
HTTP/1.1 403 Forbidden
WWW-Authenticate: step-up; realm="ggid"; methods="totp,webauthn"
X-Risk-Score: 47
```

4. Client prompts MFA (TOTP/WebAuthn) without full re-auth.
5. After step-up: session elevated to AAL2 for 15 minutes.

```go
func (g *Gateway) StepUpMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !isSensitive(r.URL.Path) { next.ServeHTTP(w, r); return }
        sc := g.riskEngine.Evaluate(r.Context(), r)
        if sc.Action >= risk.ActionStepUp {
            w.Header().Set("WWW-Authenticate",
                `step-up; realm="ggid"; methods="totp,webauthn"`)
            writeJSONError(w, 403, "step-up authentication required")
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

---

## 7. GGID Integration Architecture

### Components & Data Flow

```
┌──────────┐   ┌─────────────────────────────────────────────┐    ┌────────┐
│  Client  │──▶│                API Gateway                   │───▶│Backend │
│(Browser) │   │  JWTAuth → Session → RiskEval → RateLimit    │    │ Svcs   │
└──────────┘   └──────────────────────┬───────────────────────┘    └────────┘
                                      │
                           ┌──────────▼──────────┐
                           │  pkg/risk/ Engine    │
                           │  IPGeoSignal         │
                           │  DeviceSignal        │
                           │  VelocitySignal      │
                           │  TimeSignal          │
                           └──────────┬───────────┘
                                      │
                           ┌──────────▼──────────┐    ┌───────────┐
                           │       Redis          │    │    NATS   │
                           │ risk:{sess} (60s)    │    │ risk.events│──▶ Audit
                           │ velocity:{sess}      │    └───────────┘
                           │ device:{sess}        │
                           └─────────────────────┘
```

1. **Client request** → Gateway with `Authorization: Bearer <jwt>` +
   `X-Device-Fingerprint`.
2. **JWTAuth** (existing): validates signature, extracts `JWTCClaims` to context.
3. **RiskEvaluation** (new): extract signals → compute score → cache in Redis.
   - Block → 403 + revoke + publish event. StepUp on sensitive → 403 challenge.
4. **Gateway** forwards with `X-Risk-Score` header.
5. **Backend** can use risk score for additional policy checks.

### Middleware Chain

```go
chain := middleware.Chain(
    middleware.RequestID,            // existing
    middleware.Logging,              // existing
    middleware.JWTAuth,              // existing — validates signature
    middleware.JWTClaimExtraction,   // existing — claims to context
    middleware.SessionMiddleware,    // existing — validates session in Redis
    riskEval.Middleware,             // NEW — per-request risk score
    middleware.StepUpMiddleware,     // NEW — continuous step-up
    rateLimiter.Middleware,          // existing
    proxyHandler,                    // existing — reverse proxy
)
```

### Signal Implementations

```go
// IPGeoSignal — geo-velocity anomaly
func (s *IPGeoSignal) Collect(ctx context.Context, r *http.Request) float64 {
    dist := s.geoIP.Distance(clientIP(r), sessionBaselineIP(ctx))
    switch { case dist > 5000: return 1.0; case dist > 500: return 0.5 }
    return 0
}

// VelocitySignal — bot detection via Redis counter
func (s *VelocitySignal) Collect(ctx context.Context, r *http.Request) float64 {
    sess, _ := middleware.SessionIDFromContext(ctx)
    key := "velocity:" + sess
    n, _ := s.rdb.Incr(ctx, key).Result()
    if n == 1 { s.rdb.Expire(ctx, key, time.Second) }
    switch { case n > 20: return 1.0; case n > 10: return 0.5 }
    return 0
}
```

---

## 8. Privacy and Performance

### Privacy

- Behavioral biometrics = **GDPR Art. 9 special category** data. Requires
  **explicit consent** (not legitimate interest). Right to erasure must remove
  all model artifacts.
- Store only **derived feature vectors** — never raw keystroke/mouse events.
  Pipeline discards raw events after feature extraction.
- Device fingerprinting must be disclosed with an **opt-out**. Prefer WebAuthn
  attestation for privacy-first deployments.

### Performance

| Operation | Target | Method |
|-----------|--------|--------|
| Risk eval (cache hit) | < 1ms | Redis GET |
| Risk eval (cache miss) | < 5ms | 5 signals, weighted sum |
| Device fingerprint parse | < 0.1ms | Header parse |
| Behavioral analysis | Async | NATS queue worker |

Target: risk evaluation adds **< 5ms** to p99. Behavioral analysis never blocks
requests — it runs async and updates the score for *subsequent* requests. Redis
cache hit rate should exceed 95% under normal traffic.

---

## 9. Roadmap

| Phase | Scope | Effort | Priority |
|-------|-------|--------|----------|
| **1. Per-request risk** | IP geo-velocity, device mismatch, velocity, time-of-day. Rule-based scoring with configurable thresholds. | 1-2 weeks | P1 |
| **2. Adaptive session timeout** | Risk-adaptive refresh TTL. Integrate with `SessionManager`. | 3-5 days | P1 |
| **3. Continuous step-up** | Gateway middleware + client SDK for challenge response. | 1 week | P1 |
| **4. Device fingerprinting** | JS library + gateway extraction + baseline comparison + consent UI. | 1-2 weeks | P2 |
| **5. Behavioral biometrics** | Keystroke/mouse dynamics. Consent flow, async pipeline, baseline training. | 4-6 weeks | P2 |
| **6. ML anomaly detection** | Replace rule-based weights with isolation forest / autoencoder. Needs Phase 1-5 training data. | 4+ weeks | P3 |

Phases 1-3 deliver core zero-trust value without behavioral data and can ship
within a sprint. Phases 4-6 require consent infrastructure, client SDKs, and ML
pipelines. **Continuous authentication is a key differentiator** for zero-trust
positioning — it moves GGID from session-based trust to NIST SP 800-207-compliant
continuous verification.

---

## References

- NIST SP 800-207, *Zero Trust Architecture* (2020) — §3.1: per-session access
- *Keystroke Dynamics: Concepts, Techniques, and Applications*, arXiv:2303.04605
  (2023) — behavioral biometric accuracy survey
- FIDO Alliance, *Continuous Authentication Best Practices* — attestation as
  privacy-preserving fingerprinting alternative
- OWASP, *Authentication Cheat Sheet* — session management and step-up patterns

---

*Part of GGID's IAM research series. Related: `adaptive-mfa-design.md`,
`step-up-authentication-patterns.md`.*
