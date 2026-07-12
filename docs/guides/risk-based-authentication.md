# Risk-Based Authentication

This guide covers risk score calculation, real-time evaluation, threshold-to-action mapping, and GGID's risk engine implementation.

## Overview

Risk-based authentication (RBA) dynamically adjusts authentication requirements based on the perceived risk of each login attempt. Instead of applying the same security checks to every request, RBA evaluates contextual signals and applies proportional challenges.

## Risk Score Calculation

### Score Range

Risk scores range from 0.0 (no risk) to 1.0 (maximum risk):

| Range | Level | Action |
|---|---|---|
| 0.0 - 0.2 | Minimal | Allow |
| 0.2 - 0.4 | Low | Allow, log |
| 0.4 - 0.6 | Medium | Require MFA |
| 0.6 - 0.8 | High | Require step-up + notify |
| 0.8 - 1.0 | Critical | Block + alert |

### Factor Weights

| Factor | Weight | Description |
|---|---|---|
| Device trust | 25% | Known vs unknown device |
| Geo-velocity | 20% | Impossible travel detection |
| IP reputation | 15% | Known bad IPs, datacenter, Tor |
| Time anomaly | 10% | Off-hours login |
| Behavioral | 10% | Deviation from patterns |
| Failed attempts | 10% | Recent failures |
| Network zone | 10% | Corporate vs public |

### Calculation

```go
type RiskFactors struct {
    DeviceTrust      float64  // 0 = unknown, 1 = managed
    GeoVelocity      float64  // 0 = impossible, 1 = normal
    IPReputation     float64  // 0 = bad, 1 = clean
    TimeAnomaly      float64  // 0 = off-hours, 1 = normal
    BehavioralMatch  float64  // 0 = anomalous, 1 = matches pattern
    FailedAttempts   float64  // 0 = many failures, 1 = none
    NetworkTrust     float64  // 0 = public, 1 = corporate
}

func CalculateRiskScore(f RiskFactors) float64 {
    // Each factor is 0-1 where 1 = safe, 0 = risky
    // Risk = 1 - weighted safety
    safety := f.DeviceTrust*0.25 +
        f.GeoVelocity*0.20 +
        f.IPReputation*0.15 +
        f.TimeAnomaly*0.10 +
        f.BehavioralMatch*0.10 +
        f.FailedAttempts*0.10 +
        f.NetworkTrust*0.10

    return 1.0 - safety  // Invert: high safety = low risk
}
```

## Real-Time Evaluation

### Evaluation Points

Risk is evaluated at multiple points:

1. **Pre-authentication** — Before credentials are checked
2. **Post-authentication** — After credentials validated, before token issued
3. **Per-request** — On every API request (lightweight)
4. **Periodic** — Background re-evaluation during active session

### Pre-Authentication Evaluation

```go
func PreAuthRiskEval(r *http.Request, username string) (float64, *RiskFactors) {
    factors := RiskFactors{
        DeviceTrust:     deviceTrustScore(r),
        GeoVelocity:     geoVelocityScore(username, clientIP(r)),
        IPReputation:    ipReputationScore(clientIP(r)),
        TimeAnomaly:     timeAnomalyScore(time.Now()),
        BehavioralMatch: behavioralMatchScore(username, r),
        FailedAttempts:  failedAttemptScore(username),
        NetworkTrust:    networkTrustScore(clientIP(r)),
    }
    return CalculateRiskScore(factors), &factors
}
```

### Per-Request Evaluation

Lightweight evaluation on every API request (only network + session factors):

```go
func PerRequestRiskEval(r *http.Request) float64 {
    session := getSession(r)
    risk := 0.0

    // Session age
    if time.Since(session.CreatedAt) > 8*time.Hour {
        risk += 0.1
    }

    // IP change mid-session
    if session.IP != clientIP(r) {
        risk += 0.3  // IP changed during session
    }

    // Network zone change
    if session.NetworkZone != currentNetworkZone(clientIP(r)) {
        risk += 0.2
    }

    return risk
}
```

## Threshold to Action Mapping

### Configuration

```yaml
risk_engine:
  thresholds:
    allow: 0.3
    log: 0.4
    challenge_mfa: 0.6
    step_up: 0.8
    block: 0.85
  actions:
    log:
      level: "info"
      message: "elevated risk detected"
    challenge_mfa:
      method: "totp"  # or "webauthn"
      cooldown: 1h  # Don't re-challenge for 1h
    step_up:
      method: "webauthn"
      notify_admin: true
      notify_user: true
    block:
      alert_security_team: true
      lock_account: false  # Don't lock, just block
      message: "Access blocked due to security risk"
```

### Action Decision

```go
func DecideAction(riskScore float64, policy *RiskPolicy) RiskAction {
    switch {
    case riskScore >= policy.BlockThreshold:
        return ActionBlock
    case riskScore >= policy.StepUpThreshold:
        return ActionStepUp
    case riskScore >= policy.ChallengeThreshold:
        return ActionChallengeMFA
    case riskScore >= policy.LogThreshold:
        return ActionLog
    default:
        return ActionAllow
    }
}
```

## Device Trust

### Device Registration

```go
type Device struct {
    Fingerprint  string    // Hash of UA + screen + fonts
    UserID       string
    TenantID     string
    FirstSeen    time.Time
    LastSeen     time.Time
    TrustLevel   string    // "managed", "byod", "unknown"
    MDMEnrolled  bool
    Certificate  string    // Device cert for managed devices
}
```

### Trust Scoring

| Device State | Score | Rationale |
|---|---|---|
| Managed + MDM + cert | 1.0 | Highest trust |
| BYOD + registered + seen before | 0.7 | Known personal device |
| Registered but not seen in 90 days | 0.4 | Stale registration |
| Never seen before | 0.2 | New device |
| Known compromised | 0.0 | Blocked |

## Geo-Velocity

### Impossible Travel Detection

```go
func GeoVelocityScore(userID string, currentIP string) float64 {
    lastLogin := getLastSuccessfulLogin(userID)
    if lastLogin == nil {
        return 0.5  // No history, neutral
    }

    lastGeo := geoLocate(lastLogin.IP)
    currentGeo := geoLocate(currentIP)

    distance := haversineDistance(lastGeo, currentGeo)  // km
    timeDiff := time.Since(lastLogin.Timestamp).Hours()

    if timeDiff == 0 {
        return 0.0  // Can't determine
    }

    speed := distance / timeDiff  // km/h

    switch {
    case speed > 1000:  // > 1000 km/h (faster than commercial flight)
        return 0.0  // Impossible travel
    case speed > 500:
        return 0.3  // Very fast, suspicious
    case speed > 200:
        return 0.7  // Fast but possible (short flight)
    default:
        return 1.0  // Normal
    }
}
```

### Impossible Travel Example

```
Login 1: 10:00 AM, New York (40.71, -74.01)
Login 2: 10:30 AM, Tokyo (35.68, 139.69)
Distance: 10,838 km
Time: 0.5 hours
Speed: 21,676 km/h → Impossible!
Risk: Blocked
```

## New Device Risk

```go
func NewDeviceRisk(userID, fingerprint string) float64 {
    device := getDevice(userID, fingerprint)
    if device != nil {
        // Known device
        daysSinceSeen := time.Since(device.LastSeen).Hours() / 24
        if daysSinceSeen < 30 {
            return 0.1  // Recently seen, low risk
        }
        if daysSinceSeen < 90 {
            return 0.3  // Moderately stale
        }
        return 0.5  // Very stale
    }
    // Unknown device
    return 0.8  // High risk for new device
}
```

## Anomalous Time

```go
func TimeAnomalyScore(t time.Time, userPatterns *UserPatterns) float64 {
    if userPatterns == nil || len(userPatterns.LoginHours) == 0 {
        return 0.5  // No pattern data
    }

    hour := t.Hour()
    if contains(userPatterns.LoginHours, hour) {
        return 1.0  // Normal hour
    }

    // Check if near normal hours (±2)
    for _, h := range userPatterns.LoginHours {
        if abs(hour-h) <= 2 {
            return 0.7  // Near normal
        }
    }

    return 0.3  // Off-hours anomaly
}
```

## Adaptive MFA Integration

### MFA Trigger Based on Risk

```go
func ShouldRequireMFA(riskScore float64, user *User) bool {
    // Always require MFA for admin accounts
    if user.HasRole("admin") || user.HasRole("security-admin") {
        return true
    }

    // Require MFA when risk is elevated
    if riskScore >= 0.4 {
        return true
    }

    // Require MFA if user's policy mandates it
    if user.MFARequired {
        return true
    }

    return false
}
```

### MFA Method Selection Based on Risk

| Risk Level | MFA Method | Rationale |
|---|---|---|
| Low (0.4-0.5) | TOTP | Convenient, sufficient |
| Medium (0.5-0.7) | TOTP or push | User choice |
| High (0.7-0.85) | WebAuthn | Hardware-backed |
| Critical (0.85+) | WebAuthn + admin approval | Strongest |

## GGID Risk Engine

### Architecture

```
Login Request → Risk Engine → Score Calculation
                           → Action Decision
                           → Allow / Challenge / Block
                           → Audit Log
```

### API Endpoint

```bash
GET /api/v1/auth/risk-assessment
Authorization: Bearer <token>

Response:
{
  "risk_score": 0.45,
  "factors": {
    "device_trust": 0.7,
    "geo_velocity": 1.0,
    "ip_reputation": 0.8,
    "time_anomaly": 0.3,
    "behavioral_match": 0.6,
    "failed_attempts": 1.0,
    "network_trust": 0.2
  },
  "decision": "challenge_mfa",
  "recommended_mfa_method": "totp",
  "session_lifetime": "4h"
}
```

### Configuration

```yaml
risk_engine:
  enabled: true
  evaluation_points:
    pre_auth: true
    post_auth: true
    per_request: true
    periodic: true
    periodic_interval: 5m
  thresholds:
    allow: 0.3
    log: 0.4
    challenge_mfa: 0.6
    step_up: 0.8
    block: 0.85
  geo:
    database: "maxmind"  # or "ip2location"
    update_interval: 24h
  ip_reputation:
    providers: ["internal", "external"]
    refresh_interval: 1h
  cooldown:
    challenge: 1h  # Don't re-challenge within 1h
    step_up: 30m
  audit:
    log_all_evaluations: true
    log_level: "info"
```

### Implementation

```go
type RiskEngine struct {
    geoDB       GeoDatabase
    ipReput     IPReputationService
    deviceStore DeviceStore
    userPatterns UserPatternStore
    config      RiskConfig
}

func (e *RiskEngine) Evaluate(r *http.Request, username string) *RiskAssessment {
    factors := e.collectFactors(r, username)
    score := CalculateRiskScore(factors)
    action := DecideAction(score, e.config)

    assessment := &RiskAssessment{
        Score:    score,
        Factors:  factors,
        Action:   action,
        Time:     time.Now(),
    }

    // Audit
    audit.Log(AuditEvent{
        Type:       "risk_evaluation",
        UserID:     username,
        RiskScore:  score,
        Action:     action.String(),
        Factors:    factors,
        IP:         clientIP(r),
    })

    return assessment
}
```

## Best Practices

1. **Start with logging only** — Observe risk scores before enforcing actions
2. **Use cooldowns** — Don't challenge users repeatedly within short periods
3. **Provide feedback** — Tell users why additional verification was required
4. **Monitor false positives** — Track legitimate users being blocked
5. **Combine factors** — No single factor should trigger a block
6. **Update geo databases** — IP geolocation data changes frequently
7. **Cache results** — Don't recalculate risk on every single request
8. **Allow admin bypass** — Emergency access should not be blocked
9. **Audit everything** — Full trail of risk evaluations for forensics
10. **Tune gradually** — Start with conservative thresholds, tighten over time