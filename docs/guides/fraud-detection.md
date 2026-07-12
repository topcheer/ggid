# Fraud Detection Engine Guide

This guide covers GGID's fraud detection capabilities — device fingerprinting, velocity rules, synthetic identity detection, and Tor/VPN detection.

## Overview

Fraud detection in GGID analyzes real-time signals to identify suspicious activities that don't match legitimate user behavior patterns.

## Device Fingerprinting

### Signal Collection

GGID collects browser/device signals during authentication:

| Signal | Source | Stability |
|--------|--------|-----------|
| User agent | HTTP header | Low (changes with updates) |
| Screen resolution | Client-side JS | Medium |
| Timezone | Client-side JS | High |
| Language | HTTP header | High |
| Platform | navigator.platform | High |
| Canvas fingerprint | Canvas API | High |
| WebGL fingerprint | WebGL renderer | High |
| Font list | Client-side JS | Medium |
| Audio context | Web Audio API | High |

### Fingerprint Hash

```go
func ComputeFingerprint(signals DeviceSignals) string {
    data := fmt.Sprintf("%s|%s|%s|%s|%s",
        signals.UserAgent, signals.ScreenResolution,
        signals.Timezone, signals.CanvasHash, signals.WebGLHash)
    return sha256Hex(data)
}
```

### Device Registry

```sql
CREATE TABLE device_fingerprints (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    fingerprint_hash VARCHAR(64) NOT NULL,
    user_agent TEXT,
    first_seen TIMESTAMPTZ DEFAULT NOW(),
    last_seen TIMESTAMPTZ DEFAULT NOW(),
    trusted BOOLEAN DEFAULT FALSE,
    UNIQUE(tenant_id, user_id, fingerprint_hash)
);
```

### New Device Alert

When a login comes from an unknown fingerprint:
1. Flag session as "new device"
2. Require step-up MFA
3. Send notification email
4. Log audit event: `security.new_device_login`

## Velocity Rules

Velocity rules detect unusually rapid activity patterns:

| Rule | Condition | Action |
|------|-----------|--------|
| Login velocity | > 5 logins from different IPs in 10m | Block + alert |
| Account creation | > 10 accounts from same IP in 1h | Block IP |
| Password reset | > 3 resets in 10m | Rate limit |
| Failed login velocity | > 20 failures in 5m | Block IP |
| Token refresh | > 10 refreshes in 1m | Rate limit |
| API key creation | > 5 keys in 1h | Alert |

### Configuration

```bash
curl -X PUT https://api.ggid.example.com/api/v1/settings/fraud/velocity \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "rules": [
      {
        "name": "Login velocity",
        "metric": "user.login",
        "field": "ip",
        "threshold": 5,
        "window": "10m",
        "action": "block"
      }
    ]
  }'
```

## Synthetic Identity Detection

Detect accounts created with fake or stolen identities:

| Signal | Detection Method | Confidence |
|--------|-----------------|------------|
| Disposable email | Check against known disposable domains | High |
| Username pattern | Regex for random strings (e.g., `user12345678`) | Medium |
| Phone number VOIP | Check carrier type | Medium |
| Registration velocity | Multiple accounts from same device | High |
| Email/phone mismatch | Area code doesn't match IP geo | Low |
| Stolen credentials | HIBP breach check | High |

### Implementation

```go
type FraudScore struct {
    Score       int      // 0-100
    Signals     []string // Triggered signals
    Recommended string   // allow | review | deny
}

func EvaluateRegistration(req RegisterRequest, ip string) FraudScore {
    score := 0
    var signals []string

    if isDisposableEmail(req.Email) { score += 30; signals = append(signals, "disposable_email") }
    if isRandomUsername(req.Username) { score += 15; signals = append(signals, "random_username") }
    if isVOIPPhone(req.Phone) { score += 20; signals = append(signals, "voip_phone") }
    if checkBreach(req.Password) { score += 40; signals = append(signals, "breached_password") }

    switch {
    case score >= 60: return FraudScore{score, signals, "deny"}
    case score >= 30: return FraudScore{score, signals, "review"}
    default: return FraudScore{score, signals, "allow"}
    }
}
```

## Tor / VPN Detection

### Threat Intelligence Feed

GGID checks client IPs against known threat intelligence:

| Source | Type | Update Frequency |
|--------|------|-----------------|
| Tor exit nodes | Public list | Hourly |
| Known VPN providers | Commercial feed | Daily |
| Botnet C&C | Threat intel feed | Hourly |
| Proxy services | Commercial feed | Daily |
| Datacenter IPs | ASN lookup | Monthly |

### Configuration

```yaml
fraud_detection:
  tor_detection: true
  vpn_detection: true
  proxy_detection: true
  threat_intel_feed: "https://feeds.example.com/blocklists.json"
  update_interval: "1h"
```

### Action on Tor/VPN Login

```bash
curl -X PUT https://api.ggid.example.com/api/v1/settings/fraud/ip-policy \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "tor": "step_up_mfa",
    "vpn": "allow_with_warning",
    "proxy": "step_up_mfa",
    "datacenter": "log_only"
  }'
```

## Risk Scoring Integration

All fraud signals feed into a composite risk score (0-100):

```
risk_score = 0
  + device_fingerprint_risk     (0-30)
  + velocity_risk               (0-25)
  + synthetic_identity_risk     (0-20)
  + ip_reputation_risk          (0-15)
  + behavioral_anomaly_risk     (0-10)

decision:
  < 20  → ALLOW
  20-50 → STEP_UP (MFA)
  > 50  → DENY
```

## See Also

- [ITDR Implementation](itdr-implementation.md)
- [Adaptive Authentication](../research/adaptive-authentication.md)
- [Risk Scoring Model](../research/risk-scoring-adaptive-access.md)
- [Security Audit Checklist](security-audit-checklist.md)
