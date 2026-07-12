# Adaptive Authentication Design

Risk scoring model, signal collection, threshold tuning, ML vs rule-based, A/B testing, and user experience.

## Overview

Adaptive authentication dynamically adjusts security requirements based on real-time risk signals. High-risk sessions get step-up MFA; low-risk sessions proceed frictionless.

## Risk Scoring Model

### Score Calculation

```
risk_score = weighted_sum(
    device_risk * 0.20,
    location_risk * 0.20,
    behavior_risk * 0.20,
    threat_risk * 0.20,
    app_risk * 0.10,
    time_risk * 0.10
)
```

Each component returns 0-100. Final score is 0-100.

### Score → Action Mapping

| Score | Level | Action |
|-------|-------|--------|
| 0-19 | Minimal | Allow |
| 20-39 | Low | TOTP step-up (remember 30d) |
| 40-59 | Medium | TOTP step-up (no remember) |
| 60-79 | High | WebAuthn step-up |
| 80-100 | Critical | Deny + alert |

## Signal Collection

### Device Signals

```go
type DeviceSignals struct {
    IsKnownDevice    bool    // Seen before
    DeviceAge        int     // Days since first enrollment
    IsManaged        bool    // MDM enrolled
    OSVersion        string  // Current OS
    IsOSOutdated     bool    // Security patches
    HasScreenLock    bool    // Device policy
    BrowserVersion   string
    IsTorBrowser     bool
}
```

### Location Signals

```go
type LocationSignals struct {
    Country          string
    IsKnownCountry   bool    // Previously logged in from
    IsHighRiskGeo    bool    // Sanctioned/high-fraud country
    IsVPN            bool    // Known VPN exit
    IsTor            bool    // TOR exit node
    ImpossibleTravel bool    // Faster than possible from last login
    NewASN           bool    // First time from this ASN
}
```

### Behavioral Signals

```go
type BehavioralSignals struct {
    LoginTimeAnomaly    bool   // Unusual time of day
    LoginFrequencyAnomaly bool  // Unusual frequency
    TypingPatternMatch  float64 // Keystroke dynamics (0-1)
    SessionDurationNorm bool    // Normal session length
    NewUserAgent        bool    // First time UA
}
```

### Threat Signals

```go
type ThreatSignals struct {
    IPInBlacklist       bool    // Known bad IP
    CredentialLeaked    bool    // Found in breach database (HIBP)
    BruteForceFromIP    int     // Failed attempts from this IP
    AccountTakeoverRisk bool    // ATO indicator
}
```

## ML vs Rule-Based

### Rule-Based (Current)

```yaml
rules:
  - if: impossible_travel
    add_score: 40
  - if: ip_in_blacklist
    add_score: 60
  - if: new_device && !managed
    add_score: 20
  - if: credential_leaked
    add_score: 50
  - if: hour < 6 || hour > 23
    add_score: 10
```

| Pros | Cons |
|------|------|
| Explainable | Can't detect novel attacks |
| Easy to tune | Static thresholds |
| Fast (<1ms) | May produce false positives |
| Auditable | Doesn't learn patterns |

### ML-Based (Roadmap)

```python
model = GradientBoostingClassifier(features=[
    'device_known', 'geo_risk', 'time_anomaly',
    'failed_attempts', 'session_pattern', 'typing_speed',
    'vpn_detected', 'breach_found'
])
score = model.predict(request_features)  # 0.0 - 1.0
```

| Pros | Cons |
|------|------|
| Detects novel patterns | Black box (hard to explain) |
| Adapts over time | Needs training data |
| Lower false positive rate | Slower inference (10-50ms) |
| Correlates signals | Requires ML ops |

### Hybrid (Recommended)

```
Rule-based (fast path) → If score 20-80 → ML refinement
  → Final score = blend(rule_score * 0.6, ml_score * 0.4)
```

Rules catch known threats fast; ML catches subtle patterns in the uncertain zone.

## Threshold Tuning

### Baseline Calibration

```bash
# Analyze historical logins to calibrate thresholds
POST /api/v1/admin/risk/calibrate
{
  "analysis_period": "90d",
  "target_false_positive_rate": 0.02  # Max 2% legit users get step-up
}
# → {
#   "suggested_thresholds": {
#     "step_up_totp": 25,    # Was 20
#     "step_up_webauthn": 55, # Was 60
#     "deny": 85             # Was 80
#   },
#   "expected_impact": {"step_up_rate": "4.1%", "deny_rate": "0.3%"}
# }
```

### Threshold Adjustment Rules

| Principle | Guideline |
|-----------|-----------|
| Start conservative | Lower thresholds, accept more step-ups |
| Monitor user feedback | Complaints about excessive MFA → raise threshold |
| Attack spike | Temporarily lower thresholds |
| Seasonal patterns | Adjust for holidays/outages |

## A/B Testing

```bash
# Test new risk model on 10% of traffic
POST /api/v1/admin/risk/ab-test
{
  "experiment_name": "ml-model-v2",
  "control": {"model": "rule-based-v1"},
  "treatment": {"model": "hybrid-v2"},
  "traffic_split": 0.10,
  "metrics": ["step_up_rate", "deny_rate", "login_success_rate", "ato_detected"]
}
```

### Metrics to Track

| Metric | Control Target | Action |
|--------|---------------|--------|
| Step-up rate | <5% | If treatment > control → investigate |
| Login success rate | >95% | Drop in treatment → too aggressive |
| ATO detection | Baseline | If treatment catches more → promote |
| False positive rate | <2% | If higher → tune thresholds |

## User Experience

### Friction Budget

```
Low risk: 0 seconds (no step-up)
Medium risk: 10 seconds (TOTP)
High risk: 5 seconds (WebAuthn biometric)
Critical: 30+ seconds (WebAuthn + approval)
```

### Communicating Step-Up

```
"Additional verification required"
"We detected unusual activity. Please verify it's you."
"Your security is important. Please enter your authenticator code."
```

**Never** say: "You have been flagged as suspicious" or "Your risk score is high."

### Remember Device

```
Low-risk step-up → "Remember this device for 30 days" checkbox
High-risk step-up → No remember option
Critical → Always require full verification
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Average risk score | Drift >10% → model degradation |
| Step-up rate | >5% → investigate false positives |
| Deny rate | >1% → investigate |
| ML model latency | >50ms → optimize |
| ATO events caught | Track for model validation |

## See Also

- [Conditional Access](conditional-access.md)
- [MFA Architecture](mfa-architecture.md)
- [Multi-Factor Step-Up Design](multi-factor-step-up-design.md)
- [Identity Threat Detection](identity-threat-detection.md)
