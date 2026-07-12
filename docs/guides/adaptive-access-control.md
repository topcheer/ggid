# Adaptive Access Control

This guide covers context-aware authorization, risk-based policy engine, dynamic trust scoring, and GGID's adaptive access control implementation.

## Overview

Adaptive access control evaluates contextual signals at authentication and authorization time to dynamically adjust security requirements. Instead of static rules, it uses risk scoring to determine whether to allow, challenge, or block access.

## Context-Aware Authorization

### Context Signals

| Signal | Source | Examples |
|---|---|---|
| Time | Request timestamp | Business hours, off-hours, weekend |
| Geolocation | IP geolocation | Country, city, distance from usual |
| Device | Fingerprint, UA | Managed, BYOD, unknown device |
| Network | IP, ASN, VPN | Corporate, VPN, public WiFi, cellular |
| Posture | Device agent | OS version, disk encryption, AV status |
| User behavior | Historical patterns | Login time, typical IP, velocity |
| Session | Current session | Age, step-up status, MFA verified |

### Signal Collection

```go
type AccessContext struct {
    UserID          string
    TenantID        string
    IP              string
    Country         string
    City            string
    DeviceFingerprint string
    DeviceType      string    // managed, byod, unknown
    UserAgent       string
    NetworkZone     string    // corporate, vpn, public
    TimeOfDay       time.Time
    SessionAge      time.Duration
    MFAVerified     bool
    StepUpAt        time.Time
    PostureScore    float64   // 0-1, from device agent
}
```

## Risk-Based Policy Engine

### Risk Scoring

Each context signal contributes to an overall risk score (0.0 = safe, 1.0 = high risk):

```go
func CalculateRiskScore(ctx *AccessContext) float64 {
    score := 0.0

    // Geolocation
    if ctx.Country != ctx.UserHomeCountry {
        score += 0.2
    }
    if isHighRiskCountry(ctx.Country) {
        score += 0.15
    }

    // Network
    switch ctx.NetworkZone {
    case "corporate":
        score -= 0.1  // Trusted network
    case "vpn":
        score += 0.05
    case "public":
        score += 0.2
    }

    // Device
    switch ctx.DeviceType {
    case "managed":
        score -= 0.05
    case "byod":
        score += 0.05
    case "unknown":
        score += 0.2
    }

    // Time
    hour := ctx.TimeOfDay.Hour()
    if hour < 6 || hour > 22 {
        score += 0.1
    }

    // Posture
    if ctx.PostureScore < 0.5 {
        score += 0.15
    }

    // Session age
    if ctx.SessionAge > 8*time.Hour {
        score += 0.1
    }

    // Clamp to [0, 1]
    if score < 0 {
        score = 0
    }
    if score > 1 {
        score = 1
    }

    return score
}
```

### Risk-Based Decisions

| Risk Score | Decision | Action |
|---|---|---|
| 0.0 - 0.3 | Allow | No additional challenge |
| 0.3 - 0.5 | Allow with monitoring | Log, no challenge |
| 0.5 - 0.7 | Challenge | Require MFA or step-up |
| 0.7 - 0.85 | Challenge + notify | MFA + admin notification |
| 0.85 - 1.0 | Block | Deny access, alert security team |

### Policy Configuration

```yaml
adaptive:
  enabled: true
  risk_thresholds:
    allow: 0.3
    challenge: 0.5
    block: 0.85
  signals:
    geolocation: true
    device: true
    network: true
    time: true
    posture: true
    behavior: true
  actions:
    challenge_method: "totp"  # or "webauthn"
    notify_admin: true
    notify_threshold: 0.7
```

## Dynamic Trust Scoring

### Trust Score Components

| Component | Weight | Source |
|---|---|---|
| Device trust | 25% | Managed = 1.0, BYOD = 0.5, Unknown = 0.0 |
| Network trust | 20% | Corporate = 1.0, VPN = 0.7, Public = 0.2 |
| Identity trust | 20% | MFA verified = 1.0, Password only = 0.5 |
| Behavioral trust | 15% | Matches patterns = 1.0, Anomalous = 0.3 |
| Posture trust | 10% | Compliant = 1.0, Non-compliant = 0.0 |
| Time trust | 10% | Business hours = 1.0, Off-hours = 0.5 |

### Trust Score Calculation

```go
func CalculateTrustScore(ctx *AccessContext) float64 {
    device := deviceTrustScore(ctx.DeviceType)      // 0-1
    network := networkTrustScore(ctx.NetworkZone)    // 0-1
    identity := identityTrustScore(ctx.MFAVerified)   // 0-1
    behavioral := behavioralTrustScore(ctx)           // 0-1
    posture := ctx.PostureScore                       // 0-1
    timeTrust := timeTrustScore(ctx.TimeOfDay)        // 0-1

    return device*0.25 + network*0.20 + identity*0.20 +
           behavioral*0.15 + posture*0.10 + timeTrust*0.10
}
```

### Trust-Based Session Lifetime

```yaml
adaptive_session:
  trust_to_lifetime:
    high:    # >0.8
      access_token: 30m
      refresh_token: 7d
      session: 8h
    medium:  # 0.5-0.8
      access_token: 15m
      refresh_token: 3d
      session: 4h
    low:     # <0.5
      access_token: 5m
      refresh_token: 1d
      session: 1h
```

## Environment Evaluation

### Device Classification

| Type | Criteria | Trust Level |
|---|---|---|
| Managed | Enrolled in MDM, corporate certificate | High |
| BYOD | Personal device, registered + verified | Medium |
| Unknown | No prior registration, no MDM | Low |

### Managed Device Verification

```go
func isManagedDevice(fingerprint string, tenantID string) bool {
    // Check MDM enrollment
    enrolled := checkMDMEnrollment(fingerprint, tenantID)
    if !enrolled {
        return false
    }
    // Check corporate certificate
    hasCert := checkDeviceCertificate(fingerprint)
    return hasCert
}
```

### Posture Check

```yaml
posture:
  checks:
    - name: "os_version"
      requirement: ">= 14.0"
      platform: "macos"
    - name: "disk_encryption"
      requirement: "enabled"
    - name: "antivirus"
      requirement: "active"
    - name: "screen_lock"
      requirement: "enabled"
      timeout: "5m"
  non_compliant_action: "challenge"  # or "block"
```

## Network Trust Zones

| Zone | Description | Trust Level |
|---|---|---|
| Corporate | Office network, known IP ranges | High |
| VPN | Authenticated VPN connection | Medium-High |
| Cloud | Cloud provider IP (AWS/GCP/Azure) | Medium |
| Public | Public WiFi, coffee shop, airport | Low |
| Cellular | Mobile carrier IP | Medium |
| Tor | Tor exit node | Blocked |

### Zone Configuration

```yaml
network_zones:
  corporate:
    cidrs: ["10.0.0.0/8", "192.168.0.0/16"]
    trust: 1.0
  vpn:
    cidrs: ["172.16.0.0/12"]
    trust: 0.7
    require_mfa: true
  public:
    cidrs: ["0.0.0.0/0"]
    trust: 0.2
    exclude: ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
  blocked:
    cidrs: ["TOR_EXIT_NODES"]
    trust: 0.0
    action: "block"
```

## Time-Based Access Windows

```yaml
time_policy:
  business_hours:
    start: "09:00"
    end: "18:00"
    timezone: "America/New_York"
    days: ["Mon", "Tue", "Wed", "Thu", "Fri"]
  after_hours:
    action: "challenge"  # Require step-up
    notify_admin: true
  weekend:
    action: "block"  # For non-admin roles
    exceptions: ["on-call"]
```

### Implementation

```go
func isWithinAccessWindow(t time.Time, policy TimePolicy) bool {
    loc, _ := time.LoadLocation(policy.Timezone)
    localTime := t.In(loc)

    // Check day
    day := localTime.Weekday().String()
    if !contains(policy.Days, day) {
        return false
    }

    // Check time
    hour, min := localTime.Clock()
    currentMin := hour*60 + min
    startMin := parseTimeToMinutes(policy.Start)
    endMin := parseTimeToMinutes(policy.End)

    return currentMin >= startMin && currentMin <= endMin
}
```

## Adaptive Session Lifetime

Sessions adjust lifetime based on ongoing risk evaluation:

```go
func AdjustSessionLifetime(session *Session, ctx *AccessContext) time.Duration {
    trustScore := CalculateTrustScore(ctx)

    switch {
    case trustScore > 0.8:
        return 8 * time.Hour   // High trust
    case trustScore > 0.5:
        return 4 * time.Hour   // Medium trust
    default:
        return 1 * time.Hour   // Low trust
    }
}
```

### Session Re-evaluation

GGID periodically re-evaluates session risk:

```yaml
session:
  reeval_interval: 5m  # Re-evaluate every 5 minutes
  on_risk_increase:
    threshold: 0.2  # If risk increases by 0.2+
    action: "require_step_up"
  on_critical_risk:
    threshold: 0.85
    action: "terminate_session"
```

## GGID Adaptive Policy Implementation

### Middleware

```go
func AdaptiveAccessMiddleware(policy *AdaptivePolicy) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := buildAccessContext(r)

            riskScore := CalculateRiskScore(ctx)
            trustScore := CalculateTrustScore(ctx)

            // Block
            if riskScore >= policy.BlockThreshold {
                audit.Log(AuditEvent{Type: "adaptive_block", RiskScore: riskScore})
                writeError(w, 403, "access_blocked_risk")
                return
            }

            // Challenge
            if riskScore >= policy.ChallengeThreshold {
                if !ctx.MFAVerified {
                    audit.Log(AuditEvent{Type: "adaptive_challenge", RiskScore: riskScore})
                    writeError(w, 403, "step_up_required")
                    return
                }
            }

            // Notify admin on high risk
            if riskScore >= policy.NotifyThreshold {
                notifyAdmin(ctx, riskScore)
            }

            // Adjust session
            session := getSession(r)
            session.AdjustedLifetime = AdjustSessionLifetime(session, ctx)

            next.ServeHTTP(w, r)
        })
    }
}
```

### Policy Engine API

```bash
# Get current risk assessment
GET /api/v1/auth/risk-assessment
Authorization: Bearer <token>

Response:
{
  "risk_score": 0.35,
  "trust_score": 0.72,
  "factors": {
    "geolocation": "normal",
    "device": "managed",
    "network": "corporate",
    "time": "business_hours",
    "posture": "compliant"
  },
  "decision": "allow",
  "session_lifetime": "8h"
}
```

## Best Practices

1. **Start conservative** — Begin with observation-only mode, log risk scores without enforcing
2. **Tune thresholds gradually** — Lower thresholds over time as you collect data
3. **Monitor false positives** — Track legitimate users being blocked/challenged
4. **Provide bypass for admins** — Emergency access should not be blocked by risk scoring
5. **Combine with step-up** — Use risk score to trigger step-up authentication
6. **Audit all decisions** — Log risk score, trust score, and decision for every request
7. **Update threat intelligence** — Keep IP reputation and geo-risk data current
8. **Consider user experience** — Don't challenge users on every request, use cooldowns
9. **Test with real scenarios** — Verify risk scoring with actual user patterns
10. **Document policy** — Clear documentation of what triggers challenges and blocks