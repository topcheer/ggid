# Adaptive Authentication

Risk scoring engine with context signals (geo, device, time, behavior), step-up flow, trust elevation, session risk binding, continuous evaluation, and policy-driven thresholds.

## Risk Scoring Model

### Context Signals

| Signal | Source | Weight | Example |
|--------|--------|--------|---------|
| Geo-IP change | MaxMind DB | High | Login from new country |
| Impossible travel | Geo-IP + time | Critical | NY → Tokyo in 1h |
| New device | Device fingerprint | Medium | First login from device |
| Unusual time | User login history | Low | Login at 3am (never before) |
| Failed attempts | Auth attempts log | High | 5 failed logins in 10min |
| Velocity | Request rate | Medium | 100 logins/min from IP |
| Known breach | HIBP check | High | Password in breach DB |
| Bot detection | TLS fingerprint (JA3) | Medium | Headless browser pattern |

### Score Calculation

```go
type RiskEngine struct {
    weights map[string]int
}

func (e *RiskEngine) Score(signals RiskSignals) int {
    score := 0
    if signals.NewCountry { score += e.weights["new_country"] }     // 25
    if signals.ImpossibleTravel { score += e.weights["impossible"] } // 40
    if signals.NewDevice { score += e.weights["new_device"] }        // 15
    if signals.UnusualHour { score += e.weights["unusual_hour"] }    // 5
    if signals.FailedAttempts > 3 { score += e.weights["failed"] }   // 20
    if signals.BreachHit { score += e.weights["breach"] }            // 30
    if signals.BotPattern { score += e.weights["bot"] }              // 15
    return min(score, 100)
}
```

### Risk → Action Matrix

| Score | Level | Action |
|-------|-------|--------|
| 0-19 | Minimal | Proceed normally |
| 20-39 | Low | TOTP step-up (remember device 30d) |
| 40-59 | Medium | TOTP step-up (no remember) |
| 60-79 | High | WebAuthn step-up |
| 80-100 | Critical | Deny + alert security |

## Step-Up Flow

```go
func evaluateStepUp(signals RiskSignals) StepUpDecision {
    score := riskEngine.Score(signals)
    
    switch {
    case score < 20:
        return StepUpDecision{Required: false}
    case score < 60:
        if signals.DeviceTrusted && signals.RememberStepUp {
            return StepUpDecision{Required: false}
        }
        return StepUpDecision{Factor: "totp", TTL: 600}
    case score < 80:
        return StepUpDecision{Factor: "webauthn", TTL: 300}
    default:
        return StepUpDecision{
            Factor: "webauthn",
            RequireApproval: true,
            TTL: 300,
            Notify: "security@corp.com",
        }
    }
}
```

## Trust Elevation

```
Login (password) → Trust Level 1 (read access)
  ↓ TOTP verified → Trust Level 2 (write access)
  ↓ WebAuthn verified → Trust Level 3 (admin access)
  ↓ Dual approval → Trust Level 4 (break-glass)
```

Each level has a TTL and auto-expires. Downgrade is automatic.

## Session Risk Binding

```go
type Session struct {
    ID            string
    UserID        string
    RiskScore     int       // Set at login
    TrustLevel    int       // Elevated via step-up
    DeviceHash    string    // Bound to device
    GeoHash       string    // Bound to geo
    LastEvaluated time.Time
}

// Re-evaluate every 5 minutes
func (s *Session) ReEvaluate(currentSignals RiskSignals) {
    newScore := riskEngine.Score(currentSignals)
    
    // If risk increased significantly
    if newScore > s.RiskScore + 20 {
        // Require step-up
        s.TrustLevel = 1 // Downgrade
        notifyUser("Risk increased — re-verification required")
    }
    
    s.RiskScore = newScore
    s.LastEvaluated = time.Now()
}
```

## Continuous Evaluation

| Check | Frequency | Action on Failure |
|-------|-----------|------------------|
| Device fingerprint match | Every request | Step-up MFA |
| Geo-IP consistency | Every 5 min | Step-up if changed |
| Token binding (DPoP/mTLS) | Every request | Reject if mismatch |
| Policy still valid | Every request | Revoke if policy changed |
| User still active | Every request | Revoke if suspended |

## Policy-Driven Thresholds

```yaml
adaptive_policies:
  - name: "financial-strict"
    condition: |
      user.department == "Finance" ||
      request.path.startsWith("/api/v1/payments")
    thresholds:
      minimal: 0
      low: 10        # Stricter for finance
      medium: 30
      high: 50
      critical: 70
    
  - name: "standard"
    condition: "true"
    thresholds:
      minimal: 0
      low: 20
      medium: 40
      high: 60
      critical: 80
```

## ML vs Rule-Based

| Aspect | Rule-Based (Current) | ML-Based (Future) |
|--------|---------------------|-------------------|
| Transparency | High (explainable) | Low (black box) |
| Accuracy | Good for known patterns | Better for novel attacks |
| Tuning | Manual threshold | Auto-learning |
| Latency | <1ms | 5-20ms |
| Bias risk | Low | Higher (needs training data) |

GGID uses rule-based with hybrid option: rules for known patterns + anomaly detection for unknown.

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| Step-up completion rate | >90% | <80% → UX friction |
| False positive rate | <5% | >10% → tune thresholds |
| Critical risk denials | Track | Spike → attack pattern |
| Risk score distribution | Track | Shift → population change |

## See Also

- [Conditional Access](conditional-access.md)
- [Adaptive Authentication Design](adaptive-authentication-design.md)
- [Multi-Factor Step-Up Design](multi-factor-step-up-design.md)
- [Identity Threat Detection](identity-threat-detection.md)
- [Zero Trust Network Design](zero-trust-network-design.md)
