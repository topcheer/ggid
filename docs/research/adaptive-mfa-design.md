# Adaptive MFA Design for GGID

> Risk-based step-up authentication architecture, integration with existing
> MFA/risk/anomaly infrastructure, and phased roadmap.

---

## 1. Adaptive MFA Overview

Traditional MFA applies the same factor requirement to every login regardless of context.
**Adaptive MFA** dynamically selects the required Authenticator Assurance Level (AAL) at
runtime based on real-time risk signals.

**Core principle:** only prompt for additional factors when the risk score warrants it.

**UX vs security tradeoff:** Adaptive MFA moves the curve outward — higher security at
lower average friction. Users on known devices from familiar locations get seamless
logins; unknown contexts trigger step-up challenges proportional to risk.

```
Security ┑          ╭─── Adaptive (optimal)
         ┑         ╱
         ┑        ╱ ─ ─ Always-MFA (high friction)
         ┑       ╱
         ┑      ╱ ─ ─ Never-MFA (low security)
         ┑___╱________________________________ Friction →
```

**GGID current state:** Already has `RiskAssessment` scoring (`risk_auth.go`), anomaly
detection (`anomaly_detection.go`), device tracking (`device_tracking.go`), step-up auth
(`stepup.go`), and MFA/TOTP (`mfa_service.go`). The pieces exist but are not yet unified
into a single adaptive policy engine with configurable thresholds.

---

## 2. Risk Signals

| Signal | Description | Max Pts | Status |
|--------|-------------|:-------:|:------:|
| IP reputation | Known/unknown/VPN/Tor/datacenter | 25 | Partial |
| New device fingerprint | UA+IP hash not seen for this user | 20 | Done |
| Geo anomaly | Impossible travel, new country (>500km) | 25 | Done |
| Time anomaly | Login 02:00–05:00 UTC | 10 | Done |
| Sensitive operation | Admin action, data export, money transfer | 30 | Not wired |
| Failed auth attempts | ≥2 fails: +20, ≥5: +40 | 40 | Done |
| Brute-force pattern | ≥3 distinct users from same IP | 30 | Done |
| User-agent change | Different UA since last login | 15 | Done |
| User role sensitivity | Admin/service-account baseline +15 | 15 | Not wired |
| Historical behavior | Deviation from 30-day pattern (ML) | 20 | Phase 3 |

Signals are additive, capped at 100. Categories:
- **Contextual (login):** IP rep, device, geo, time, failed attempts, brute-force, UA
- **Operational (step-up):** sensitive operation type, resource classification
- **Behavioral (baseline):** historical pattern deviation (Phase 3 ML)

---

## 3. Risk Scoring Algorithm

| Score | Tier | Action |
|:-----:|------|--------|
| 0–29 | Low | Allow (no extra factor) |
| 30–59 | Medium | Step-up MFA (TOTP or push) |
| 60–79 | High | Require hardware key (attested WebAuthn) |
| 80–100 | Critical | Block + admin alert |

### Go types

```go
// RiskEvaluator evaluates risk signals and produces a RiskAssessment.
type RiskEvaluator interface {
    Evaluate(ctx context.Context, req RiskRequest) (*RiskAssessment, error)
}

type RiskRequest struct {
    TenantID  uuid.UUID
    UserID    uuid.UUID
    IP        string
    UserAgent string
    Latitude  float64
    Longitude float64
    Operation string   // "login", "admin:role:grant", "data:export"
    UserRoles []string
}

// Per-tenant threshold overrides.
type RiskConfig struct {
    StepUpThreshold   int `json:"step_up_threshold"`    // default 30
    HardwareThreshold int `json:"hw_threshold"`          // default 60
    BlockThreshold    int `json:"block_threshold"`        // default 80
}
```

### Rule-based evaluator (recommended v1)

```go
func (e *ruleBasedEvaluator) Evaluate(ctx context.Context, req RiskRequest) (*RiskAssessment, error) {
    score := 0
    var reasons []string

    // Reuse existing AssessLoginRisk + anomaly signals
    loginRisk := e.assessLoginSignals(ctx, req)
    score += loginRisk.Score
    reasons = append(reasons, loginRisk.Reasons...)

    if isSensitiveOp(req.Operation) { score += 30 }
    if isPrivilegedRole(req.UserRoles) { score += 15 }

    if score > 100 { score = 100 }
    return &RiskAssessment{Score: score, Level: scoreToLevel(score, e.config), Reasons: reasons}, nil
}
```

**Why rule-based for v1:** Transparent, auditable, deterministic. Security teams can
explain every decision and tune weights without retraining. ML (Phase 3) layers on top
for anomaly detection beyond fixed thresholds.

---

## 4. NIST 800-63B AAL Requirements

| AAL | Description | Authenticator Examples | Risk Tier |
|:---:|-------------|----------------------|:---------:|
| **AAL1** | Single-factor. Confidence that claimant controls one authenticator. | Password, PIN | Low |
| **AAL2** | Two-factor. Proof of possession of two distinct factors. | Password + TOTP, Password + WebAuthn | Medium |
| **AAL3** | Hardware-based. FIPS 140 L1+ validated crypto, phishing-resistant. | WebAuthn with attestation (YubiKey), PIV/CAC | High |

**Key AAL3 requirements beyond AAL2:**
1. Hardware-based authenticator — keys in isolated hardware
2. Verifier impersonation resistance — phishing-resistant (WebAuthn origin binding)
3. Verifier compromise resistance — no replayable secrets transmitted
4. Authentication intent — user must consciously actuate authenticator

### Adaptive MFA → AAL mapping

| Score | Action | AAL |
|:-----:|--------|:---:|
| 0–29 | Password only | AAL1 |
| 30–59 | Password + TOTP/WebAuthn | AAL2 |
| 60–79 | Password + hardware key (attested) | AAL3 |
| 80–100 | Block | — |

GGID's existing `acrLevel()` maps ACR strings to numeric levels (silver=1, gold=2). The
adaptive engine extends this to AAL3 for gold with hardware attestation requirement.

---

## 5. Competitor Comparison

### Auth0 / Okta Adaptive MFA

- **Signals:** Geovelocity, device recognition, IP reputation (threat intel), impossible
  travel, breached password detection
- **ML:** Auth0 Anomaly Detection (ML-based), Okta proprietary risk engine
- **Step-up:** Push (Okta Verify), OTP, or WebAuthn based on tier
- **Pricing:** Auth0 ~$0.023/MAU add-on; Okta included in Workforce Identity Advanced ($8+/user/mo)

### Keycloak

- **Signals:** Step-up via `acr_values` in OIDC, role-based auth level, conditional flows
- **ML:** None built-in; risk steps require custom SPI
- **Pricing:** Open source (free), Red Hat enterprise support

### Azure AD (Entra ID) Conditional Access

- **Signals:** User risk (Identity Protection ML), sign-in risk, device compliance, location,
  client app type
- **ML:** Identity Protection uses supervised models on global threat telemetry
- **Pricing:** P1 ($6/user/mo), Identity Protection P2 ($9/user/mo)

### GGID adoption plan

| Feature | Decision | Rationale |
|---------|:--------:|-----------|
| Rule-based risk scoring | **Now** | Transparent, tunable, zero infra |
| Step-up via ACR values | **Now** | Already partially implemented |
| IP reputation (threat intel) | Phase 2 | MaxMind/IPinfo for geo + Tor/datacenter |
| Device fingerprinting | Phase 2 | Enhance SHA-256(UA+IP) with TLS/canvas |
| ML-based anomaly detection | Phase 3 | Valuable but needs data pipeline |
| Conditional Access UI | Phase 2 | Per-tenant configurable rules in console |
| Push notification MFA | **Skip v1** | Mobile app infra not justified yet |

---

## 6. GGID Implementation Design

### Architecture

```
Login → Gateway (extract IP/UA/geo)
          → RiskEvaluator (assessLoginRisk + anomaly + ops)
            → Adaptive Policy (score → tier → action)
                ├─ Allow (AAL1)
                ├─ Step-up TOTP/WebAuthn (AAL2)
                ├─ Require hardware key (AAL3)
                └─ Block + alert
```

### RiskEvaluator interface

```go
type RiskEvaluator interface {
    Evaluate(ctx context.Context, req RiskRequest) (*RiskAssessment, error)
}

type RiskRequest struct {
    TenantID  uuid.UUID
    UserID    uuid.UUID
    IP, UserAgent string
    Latitude, Longitude float64
    Operation string
    UserRoles []string
}
```

### Gateway middleware

```go
func AdaptiveMFAMiddleware(eval service.RiskEvaluator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            req := buildRiskRequest(r)
            assessment, err := eval.Evaluate(r.Context(), req)
            if err != nil { next.ServeHTTP(w, r); return } // fail-open

            switch {
            case assessment.Score >= 80:
                http.Error(w, "blocked: high risk", http.StatusForbidden)
            case assessment.Score >= 60:
                redirectStepUp(w, r, "webauthn-hardware", assessment)
            case assessment.Score >= 30:
                redirectStepUp(w, r, "mfa", assessment)
            default:
                next.ServeHTTP(w, r)
            }
        })
    }
}
```

### Policy service integration (ABAC with risk score)

The Policy ABAC engine treats `risk_score` and `risk_level` as attributes. The gateway
passes `X-Risk-Score` / `X-Risk-Level` headers to downstream services.

```yaml
effect: deny
condition:
  any_of:
    - risk_score >= 70
    - risk_level == "high" AND operation IN ["admin:*", "data:export"]
```

### Per-tenant config + risk events table

```sql
CREATE TABLE risk_config (
    tenant_id          UUID PRIMARY KEY,
    step_up_threshold  SMALLINT DEFAULT 30,
    hardware_threshold SMALLINT DEFAULT 60,
    block_threshold    SMALLINT DEFAULT 80,
    enabled_signals    JSONB DEFAULT '{}'
);

CREATE TABLE risk_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    user_id      UUID NOT NULL,
    event_type   TEXT NOT NULL,
    ip_address   TEXT,
    risk_score   SMALLINT NOT NULL,
    risk_level   TEXT NOT NULL,
    reasons      TEXT[],
    action_taken TEXT NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_risk_events_tenant_user ON risk_events(tenant_id, user_id, created_at DESC);
```

### Step-up auth flow

```
Client                Gateway              AuthService
  │── GET /admin ───►│── EvaluateRisk ───►│
  │                  │◄── score=45 ───────│
  │◄── 401 + step-up │                    │
  │── POST /stepup ─►│── InitStepUp ─────►│
  │◄── challenge ────│◄── token ──────────│
  │── POST verify ──►│── VerifyStepUp ───►│
  │◄── step_up_tok ──│◄── token ──────────│
  │── GET /admin ───►│ (X-Step-Up-Token)  │
  │                  │── ValidateStepUp ─►│
  │◄── 200 (data) ───│                    │
```

The existing `stepup.go` already implements `InitStepUp`, `VerifyStepUp`, and
`ValidateStepUpToken`. The adaptive engine wraps these and selects the method
(`"mfa"` or `"webauthn"`) based on the risk tier.

### Adaptive login wrapper

```go
func (s *AdaptiveAuthService) Login(ctx context.Context, creds Credentials) (*LoginResult, *RiskAssessment, error) {
    // ... validate password ...
    assessment, _ := s.evaluator.Evaluate(ctx, buildRiskReq(creds))
    s.recordRiskEvent(ctx, creds, assessment) // audit trail

    switch {
    case assessment.Score >= cfg.BlockThreshold:
        return nil, assessment, ErrRiskBlocked
    case assessment.Score >= cfg.HardwareThreshold:
        ch, _ := s.InitStepUp(ctx, user.ID, "webauthn")
        return &LoginResult{StepUpRequired: true, Challenge: ch}, assessment, nil
    case assessment.Score >= cfg.StepUpThreshold:
        if s.mfaService.HasMFAEnabled(ctx, creds.TenantID, user.ID) {
            ch, _ := s.InitStepUp(ctx, user.ID, "mfa")
            return &LoginResult{StepUpRequired: true, Challenge: ch}, assessment, nil
        }
        return nil, assessment, ErrMFAMandatory
    }
    return s.issueTokens(ctx, user), assessment, nil // low risk
}
```

---

## 7. Roadmap

### Phase 1: Rule-based risk scoring + step-up TOTP (4–6 weeks)

- Unify `RiskAssessment` + `AnomalyResult` into `RiskEvaluator` (3 days)
- Per-tenant `RiskConfig` table + Redis cache (2 days)
- Wire `RiskEvaluator` into login flow before token issuance (2 days)
- Sensitive-operation classification map (1 day)
- Gateway `AdaptiveMFAMiddleware` (3 days)
- Risk events table + audit publisher (2 days)
- Console risk config page (3 days)
- Test coverage: threshold boundaries, edge cases (3 days)

### Phase 2: IP reputation + device fingerprinting (4–5 weeks)

- MaxMind GeoIP2 / IPinfo integration for geo + threat classification (3 days)
- Tor exit node + datacenter detection via public feeds (2 days)
- Enhanced device fingerprint (TLS JA3 + canvas hash) (5 days)
- Conditional Access policy UI in Console (5 days)
- Impossible-travel detection with timestamp history (3 days)
- Policy service ABAC: `risk_score` attribute (3 days)

### Phase 3: ML-based anomaly detection (6–8 weeks)

- Historical login data pipeline (30-day rolling window) (5 days)
- Anomaly model: Isolation Forest or statistical baseline (7 days)
- Model serving: in-process Go scoring or gRPC sidecar (5 days)
- Admin dashboard: risk trends, false-positive tuning (5 days)
- Shadow mode: ML alongside rules, compare before activation (2 weeks)

| Phase | Duration | Risk Reduction | Friction |
|-------|----------|---------------|----------|
| P1 (rules + TOTP) | 4–6 wk | Medium | Low |
| P2 (IP rep + device) | 4–5 wk | High | Low |
| P3 (ML anomaly) | 6–8 wk | Very High | Minimal |

Total: 14–19 weeks. Phase 1 is independently production-usable.

---

*References: NIST SP 800-63B / 800-63-4 draft, Auth0 Adaptive MFA, Okta Adaptive MFA,
Azure AD Conditional Access, Keycloak Authentication Flows.*
