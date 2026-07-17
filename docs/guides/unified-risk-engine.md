# Unified Risk Engine (URE) — Technical Guide

> Feature: Unified Risk Engine with 26 signals, composite scoring, UEBA enhancement
> Location: `services/policy/internal/server/unified_risk_engine.go`
> Endpoints: `/api/v1/risk/evaluate`, `/api/v1/risk/scores/`, `/api/v1/risk/policies`

## What It Does

The Unified Risk Engine evaluates real-time risk for every authentication and authorization request. It aggregates 26 signals across 5 categories, applies configurable weights, and produces a composite score (0-100) with a decision: allow, step_up, step_up_strong, or block.

## Signal Registry (26 Signals, 5 Categories)

### Device (6 signals)

| Signal ID | Name | Default Weight |
|-----------|------|---------------|
| `device_posture` | Device Posture Score | 0.15 |
| `device_managed` | Managed Device | 0.10 |
| `device_encrypted` | Disk Encryption | 0.08 |
| `device_jailbreak` | Jailbreak/Root Detected | 0.20 |
| `device_compliant_os` | OS Compliance | 0.07 |
| `device_trust_score` | Device Trust Score | 0.10 |

### Geo (5 signals)

| Signal ID | Name | Default Weight |
|-----------|------|---------------|
| `geo_impossible_travel` | Impossible Travel | 0.25 |
| `geo_high_risk_country` | High-Risk Country | 0.15 |
| `geo_new_location` | New Login Location | 0.08 |
| `geo_tor_vpn` | TOR/VPN/Proxy Detected | 0.12 |
| `geo_geofence_violation` | Geofence Violation | 0.10 |

### Network (5 signals)

| Signal ID | Name | Default Weight |
|-----------|------|---------------|
| `net_threat_intel` | Threat Intel Match | 0.20 |
| `net_ip_reputation` | IP Reputation Score | 0.10 |
| `net_new_asn` | New ASN | 0.05 |
| `net_ddos_source` | DDoS Source IP | 0.15 |
| `net_port_scan` | Port Scan Detected | 0.08 |

### Behavior (6 signals)

| Signal ID | Name | Default Weight |
|-----------|------|---------------|
| `beh_ueba_anomaly` | UEBA Anomaly Score | 0.18 |
| `beh_off_hours` | Off-Hours Access | 0.06 |
| `beh_bulk_action` | Bulk Action Detected | 0.12 |
| `beh_privilege_escalation` | Privilege Escalation Attempt | 0.20 |
| `beh_mfa_fatigue` | MFA Fatigue Pattern | 0.15 |
| `beh_new_device_user` | First-Time Device for User | 0.08 |

### Session (4 signals)

| Signal ID | Name | Default Weight |
|-----------|------|---------------|
| `sess_concurrent` | Concurrent Sessions | 0.08 |
| `sess_token_anomaly` | Token Usage Anomaly | 0.10 |
| `sess_session_age` | Session Age Exceeded | 0.05 |
| `sess_session_hijack` | Session Hijack Indicator | 0.18 |

## Composite Scoring Algorithm

```
composite_score = Σ(signal_value × signal_weight) × 100

Where:
  signal_value = 0.0-1.0 (normalized risk contribution)
  signal_weight = 0.0-1.0 (per-tenant configurable)
```

The score is clamped to 0-100. Higher = more risky.

## Decision Thresholds

| Score Range | Level | Decision | Action |
|-------------|-------|----------|--------|
| 0-24 | Low | `allow` | Normal access |
| 25-49 | Medium | `step_up` | Require additional MFA |
| 50-74 | High | `step_up_strong` | Require hardware key + admin approval |
| 75-100 | Critical | `block` | Deny access immediately |

Thresholds are configurable per-tenant via risk policies.

## Per-Tenant Policy Configuration

```go
type RiskPolicy struct {
    TenantID        uuid.UUID          `json:"tenant_id"`
    AllowThreshold  int                `json:"allow_threshold"`   // default: 25
    StepUpThreshold int                `json:"step_up_threshold"`  // default: 50
    StrongThreshold int                `json:"strong_threshold"`   // default: 75
    Weights         map[string]float64 `json:"weights"`            // per-signal overrides
}
```

## UEBA Isolation Forest Enhancement

The `beh_ueba_anomaly` signal uses an isolation forest algorithm to detect behavioral anomalies:

1. **Baseline learning**: Establishes per-user behavioral patterns (login times, locations, devices, action types).
2. **Anomaly scoring**: New events are compared against the baseline. Outliers receive higher anomaly scores.
3. **Adaptive**: The baseline updates as user behavior evolves, reducing false positives over time.

## Integration Points

| Integration | How |
|-------------|-----|
| **ITDR** | ITDR detections (mfa_fatigue, token_theft) feed into behavior signals |
| **CAE** | Continuous Authorization Evaluation middleware calls /risk/evaluate on every request |
| **Threat Intel** | IOC matches feed into `net_threat_intel` signal |
| **Unified PDP** | Risk score is the 4th layer in authorization decisions (risk overlay) |

## API Endpoints

### POST `/api/v1/risk/evaluate`

Evaluate risk for a user/session.

```bash
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/risk/evaluate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"user_id":"admin","session_id":"sess-123","context":{"ip":"192.168.1.1","device":"laptop","action":"login"}}'
```

**Response:**
```json
{
  "score": 35,
  "level": "medium",
  "decision": "step_up",
  "signals": [
    {"id":"geo_impossible_travel","value":0.8,"weight":0.25},
    {"id":"device_trust_score","value":0.1,"weight":0.10}
  ],
  "evaluated_at": "2026-07-18T03:15:00Z",
  "evaluation_id": "abc-123"
}
```

### GET `/api/v1/risk/scores/:user_id`

Get risk score history for a user.

```bash
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/risk/scores/admin" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
```

### GET/PUT `/api/v1/risk/policies`

Get or update risk policy for the tenant.

```bash
# Get current policy
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/risk/policies" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"

# Update thresholds
curl -k -H 'Accept-Encoding: identity' \
  -X PUT "https://ggid.iot2.win/api/v1/risk/policies" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"allow_threshold":20,"step_up_threshold":45,"strong_threshold":70,"weights":{"geo_impossible_travel":0.30}}'
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| All users getting step_up | Thresholds too low or signals always high | Check policy thresholds; verify signal values aren't stuck at max |
| Risk score always 0 | Signals not feeding data | Verify ITDR/threat intel integrations are active |
| UEBA never triggers | Insufficient baseline data | UEBA needs 7+ days of history to establish patterns |
| Evaluation latency high | Too many active signals | Disable low-priority signals via weight=0 |

## Best Practices

- **Tune thresholds gradually**: Start with defaults, lower thresholds based on false negative rate.
- **Monitor signal distribution**: Track which signals fire most to identify tuning opportunities.
- **Weight by risk posture**: High-security tenants should weight geo and device signals higher.
- **Correlate with ITDR**: Cross-reference risk evaluations with ITDR detections for comprehensive threat visibility.
