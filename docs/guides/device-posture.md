# Device Posture Evaluation — Technical Guide

> Feature: Zero-Trust Device Posture + Compliance
> Location: `services/identity/internal/server/device_posture.go`, `zt_posture_handler.go`
> Endpoints: `/api/v1/zt/posture`, device posture evaluation

## What It Does

GGID's Device Posture system evaluates device security health before granting access. Each device receives a compliance score based on disk encryption, OS version, jailbreak status, and other security signals. Low-score devices trigger step-up authentication or access denial via the CAE (Continuous Authorization Evaluation) middleware.

## Posture Evaluation

### DevicePosture Model

```go
type DevicePosture struct {
    DeviceID        string `json:"device_id"`
    UserID          string `json:"user_id"`
    Platform        string `json:"platform"`     // iOS, macOS, Android, Windows, Linux
    ComplianceScore int    `json:"compliance_score"` // 0-100
    Status          string `json:"status"`       // compliant, non_compliant, unknown
    EvaluatedAt     string `json:"evaluated_at"`
}
```

### PostureCheckInput

The device agent submits posture data for evaluation:

| Field | Description |
|-------|-------------|
| `disk_encrypted` | Full disk encryption enabled (FileVault/BitLocker) |
| `os_version` | OS version string |
| `os_up_to_date` | OS has latest security patches |
| `jailbroken` | Device is jailbroken/rooted |
| `screen_lock_enabled` | Auto-lock with passcode |
| `antivirus_installed` | Antivirus/EDR agent present |

### PostureResult

```go
type PostureResult struct {
    Score     int               `json:"score"`
    Status    string            `json:"status"`
    Failures  []string          `json:"failures"`
    Action    string            `json:"action"` // allow, step_up, block
}
```

## Compliance Rules

Each rule contributes to the compliance score:

| Rule | Points | Penalty |
|------|--------|---------|
| Disk encrypted | +20 | -20 if missing |
| OS up to date | +20 | -20 if outdated |
| Not jailbroken | +20 | -40 if jailbroken |
| Screen lock enabled | +15 | -15 if missing |
| Antivirus installed | +15 | -15 if missing |
| Managed device | +10 | -10 if unmanaged |

Final score = sum of all checks, clamped to 0-100.

### Decision Thresholds

| Score | Status | CAE Action |
|-------|--------|------------|
| 80-100 | Compliant | Allow |
| 50-79 | Partial | Step-up authentication required |
| 0-49 | Non-compliant | Block access |

## SCEP Device Certificates

GGID supports Simple Certificate Enrollment Protocol (SCEP) for managed device certificate distribution:

1. **Enrollment**: MDM pushes SCEP profile → device generates CSR → GGID CA signs → device receives certificate.
2. **Authentication**: Device presents certificate for mTLS authentication instead of passwords.
3. **Revocation**: Admin can revoke certificates via the console or API.
4. **Auto-renewal**: Certificates auto-renew before expiration via MDM.

## CAE Integration

The CAE (Continuous Authorization Evaluation) middleware in the gateway calls the device posture system on every request:

1. **Extract device ID** from the request (JWT claim or certificate).
2. **Query posture** from the device posture repository.
3. **Evaluate**: If score < threshold, middleware blocks or requires step-up.
4. **Cache**: Posture scores are cached per session to reduce latency.

## Zero-Trust Posture Aggregation

The `/api/v1/zt/posture` endpoint provides a tenant-wide view:

```json
{
  "total_devices": 250,
  "compliant": 210,
  "non_compliant": 30,
  "unknown": 10,
  "compliance_pct": 84,
  "platform_breakdown": {
    "macOS": 100, "Windows": 80, "iOS": 40, "Android": 20, "Linux": 10
  }
}
```

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/zt/posture` | GET | Aggregated ZT posture for tenant |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# Get aggregated zero-trust posture
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/zt/posture" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| All devices show "unknown" | No posture data submitted | Deploy device agent that submits posture checks |
| Low compliance percentage | Many unmanaged devices | Enroll devices via MDM; enforce encryption + screen lock |
| SCEP enrollment fails | CA certificate expired or SCEP URL wrong | Verify CA is running; check SCEP challenge password |
| CAE blocks legitimate users | Threshold too high | Lower step-up threshold; review failing posture checks |

## Best Practices

- **Enforce encryption**: Disk encryption should be mandatory for all devices.
- **Block jailbroken devices**: Jailbreak/root is a critical security risk.
- **Regular OS updates**: Enforce OS patch levels within 30 days of release.
- **Use SCEP certificates**: Eliminates password-based device auth.
- **Monitor compliance trend**: Track compliance percentage over time — declining trends indicate policy gaps.
