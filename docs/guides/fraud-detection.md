# Fraud Detection Implementation Guide

## Overview

GGID's fraud detection system identifies and blocks suspicious identity activities including synthetic identities, credential abuse, velocity attacks, and automated bot activity.

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                  Fraud Detection Engine                    │
│                                                            │
│  ┌───────────┐   ┌───────────┐   ┌───────────────────┐   │
│  │ Velocity   │   │ Synthetic │   │ Device            │   │
│  │ Rules      │   │ Identity  │   │ Fingerprinting    │   │
│  │ Engine     │   │ Detector  │   │ Service           │   │
│  └─────┬─────┘   └─────┬─────┘   └────────┬──────────┘   │
│        │                │                   │              │
│  ┌─────┴─────────────────┴───────────────────┴──────┐     │
│  │              Risk Scoring Aggregator              │     │
│  └───────────────────────┬──────────────────────────┘     │
│                          │                                 │
│  ┌───────────────────────┴──────────────────────────┐     │
│  │              Action Executor                       │     │
│  │  (block / challenge / flag / throttle)            │     │
│  └───────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────┘
```

## Velocity Rules Engine

### Rule Configuration

```yaml
velocity_rules:
  - name: "Registration spam from IP"
    description: "More than 10 registrations from same IP in 1 hour"
    metric: registration_count
    dimension: ip_address
    threshold: 10
    window: 1h
    action: block
    duration: 24h

  - name: "Login brute force"
    description: "More than 20 failed logins per user in 5 minutes"
    metric: failed_login_count
    dimension: user_id
    threshold: 20
    window: 5m
    action: lock_account
    duration: 30m

  - name: "OTP flooding"
    description: "More than 5 OTP requests per phone in 10 minutes"
    metric: otp_request_count
    dimension: phone_number
    threshold: 5
    window: 10m
    action: throttle
    cooldown: 1h
```

### Implementation
- **Storage**: Redis sliding window counters (Lua atomic operations)
- **Evaluation**: Real-time on every auth event
- **Bypass**: Trusted IP allowlist per tenant

## Synthetic Identity Detection

### Indicators
1. **Disposable email**: Check against known disposable email providers (10minutemail, guerrillamail, etc.)
2. **Phone validation**: VOIP number detection, carrier lookup
3. **Name patterns**: Generated name patterns (first+last from common lists)
4. **Timing anomalies**: Registration + immediate high-value action within seconds
5. **Profile completeness**: Missing optional fields that real users typically fill

### Scoring Model
```python
synthetic_score = (
    0.30 * disposable_email +
    0.20 * voip_phone +
    0.15 * name_pattern_match +
    0.15 * timing_anomaly +
    0.10 * profile_completeness +
    0.10 * velocity_correlation
)
# score > 0.7 → block
# score 0.4-0.7 → manual review
# score < 0.4 → allow
```

## Device Fingerprinting

### Fingerprint Components
| Component | Collection Method | Stability |
|-----------|------------------|-----------|
| User-Agent | HTTP header | Low (easily spoofed) |
| Screen resolution | JavaScript | Medium |
| Timezone | JavaScript | Medium |
| Canvas fingerprint | Canvas API hash | High |
| WebGL fingerprint | WebGL renderer | High |
| Audio fingerprint | AudioContext | High |
| Font list | JavaScript detection | Medium |
| Plugin list | Navigator API | Low |

### Fingerprint Lifecycle
1. **Collection**: Client-side JavaScript collects 8+ signals
2. **Hashing**: SHA-256 of normalized signal concatenation
3. **Storage**: `device_fingerprints` table with first_seen, last_seen, trust_level
4. **Correlation**: Link fingerprints to user accounts
5. **Alerting**: Same fingerprint across many accounts → fraud indicator

## TOR/VPN/Proxy Detection

### Detection Layers
1. **IP reputation**: Real-time check against TOR exit nodes, known VPN providers
2. **ASN analysis**: Datacenter ASN vs residential ASN
3. **Port scan signals**: Open SOCKS/HTTP proxy ports
4. **Behavioral**: Multiple users from same datacenter IP

### Action Matrix
| Source Type | Risk Level | Default Action |
|-------------|-----------|----------------|
| Residential ISP | Low | Allow |
| Corporate VPN | Medium | Allow + flag |
| Known VPN provider | Medium | Challenge (CAPTCHA) |
| TOR exit node | High | Block |
| Datacenter IP | High | Challenge + flag |
| Known proxy | Critical | Block immediately |

## Integration Points

### API Endpoints
```http
GET  /api/v1/identity/fraud/indicators?severity=high
GET  /api/v1/identity/fraud/velocity-rules
PUT  /api/v1/identity/fraud/velocity-rules/{id}
GET  /api/v1/identity/fraud/blocked-entities
POST /api/v1/identity/fraud/block-entity
DELETE /api/v1/identity/fraud/block-entity/{id}
GET  /api/v1/identity/fraud/false-positives?status=pending
```

### Event Stream
- All fraud events published to NATS `fraud.events` stream
- SIEM forwarding via `/api/v1/audit/siem/forwarder-config`
- Real-time dashboard via WebSocket subscription

## Configuration

### Per-Tenant Settings
```json
{
  "fraud_detection": {
    "enabled": true,
    "sensitivity": "balanced",
    "velocity_rules": "default",
    "synthetic_detection": true,
    "device_fingerprinting": true,
    "tor_vpn_blocking": "challenge",
    "trusted_ips": ["10.0.0.0/8"],
    "action_on_detect": "challenge",
    "notify_on_block": true
  }
}
```

### Sensitivity Presets
| Preset | Velocity Threshold | Synthetic Cutoff | TOR Action |
|--------|-------------------|-----------------|------------|
| Relaxed | 2x default | 0.8 | Log |
| Balanced | Default | 0.7 | Challenge |
| Strict | 0.5x default | 0.5 | Block |
| Paranoia | 0.25x default | 0.3 | Block + Alert |

## False Positive Management

### Review Queue
1. Flagged events enter FP review queue
2. Analyst reviews: user history, device, geo, behavior
3. Decision: confirm block / unblock + whitelist
4. Feedback loop: update scoring model weights

### Whitelist Management
- IP allowlist: bypass velocity rules for trusted networks
- User allowlist: exempt trusted users from synthetic detection
- Device allowlist: known-good device fingerprints

## Best Practices

1. **Layer defenses**: Don't rely on single signal — combine velocity + synthetic + device
2. **Tune continuously**: Review false positive rate weekly, adjust thresholds
3. **Preserve UX**: Prefer CAPTCHA challenge over outright block for medium-risk
4. **Log everything**: Every fraud decision is auditable with full context
5. **Privacy first**: Device fingerprints are hashed, never store raw browser data
6. **Monitor coverage**: Track which rules fire most, retire stale rules
