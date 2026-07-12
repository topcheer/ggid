# Audit Hash Chain Guide

This guide covers GGID's audit hash chain — construction, verification, tamper detection, forensics, integrity scoring, and recovery.

## Overview

GGID implements a cryptographic hash chain over audit events, creating a tamper-evident log. Any modification, deletion, or insertion of events breaks the chain.

## Chain Construction

Each event's hash incorporates the previous event's hash:

```
Event₁ → hash₁ = SHA256(event₁_data)
Event₂ → hash₂ = SHA256(hash₁ + event₂_data)
Event₃ → hash₃ = SHA256(hash₂ + event₃_data)
...
Eventₙ → hashₙ = SHA256(hashₙ₋₁ + eventₙ_data)
```

### Data Included in Hash

```go
func ComputeHash(prevHash string, event AuditEvent) string {
    data := prevHash +
        event.ID +
        event.Type +
        event.ActorID +
        event.ResourceID +
        event.Timestamp.Format(time.RFC3339Nano) +
        event.Result
    return sha256Hex(data)
}
```

### Storage

Each audit event stores its hash:

```sql
ALTER TABLE audit_events ADD COLUMN hash VARCHAR(64);
ALTER TABLE audit_events ADD COLUMN prev_hash VARCHAR(64);
```

## Verification

### API Endpoint

```bash
curl https://api.ggid.example.com/api/v1/audit/integrity/verify \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Response (intact)**:
```json
{
  "valid": true,
  "events_verified": 154823,
  "chain_head_hash": "sha256:a1b2c3...",
  "first_event": "2024-01-01T00:00:00Z",
  "last_event": "2025-01-24T14:30:00Z"
}
```

**Response (tampered)**:
```json
{
  "valid": false,
  "events_verified": 50000,
  "broken_at": {
    "event_id": "abc123-uuid",
    "expected_hash": "sha256:def456...",
    "actual_hash": "sha256:ghi789..."
  },
  "recommendation": "Investigate events after sequence #50000"
}
```

### How Verification Works

```
1. Fetch all events ordered by timestamp
2. Recompute hash chain from first event
3. Compare each stored hash with recomputed hash
4. If any mismatch → chain broken at that event
5. Report first broken event + total verified
```

## Tamper Detection

### What Tampering Looks Like

| Attack | Effect on Chain |
|--------|---------------|
| Modify event data | Hash mismatch at that event |
| Delete event | Hash mismatch at next event |
| Insert fake event | Hash mismatch at insertion point |
| Swap event order | Hash mismatch at swapped position |

### Forensic Timeline

When tampering is detected:

1. **Identify break point**: The first event where hash doesn't match
2. **Quarantine events after break**: They may also be tampered
3. **Export events before break**: These are verified intact
4. **Investigate**: Who had write access? What changed?
5. **Document**: Create incident report

```bash
# Export events before the break (verified intact)
curl "https://api.ggid.example.com/api/v1/audit/export?end_date=2025-01-20T00:00:00Z" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -o verified_events.json

# Export events after break for investigation
curl "https://api.ggid.example.com/api/v1/audit/export?start_date=2025-01-20T00:00:00Z" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -o suspect_events.json
```

## Integrity Scoring

GGID calculates an integrity score for compliance reporting:

| Score | Meaning |
|-------|---------|
| 100% | All events verified, chain intact |
| 95-99% | Minor gap (< 1% events unverified) |
| 80-94% | Significant gap (investigate) |
| < 80% | Major compromise (incident response) |

```json
{
  "integrity_score": 99.97,
  "total_events": 154823,
  "verified_events": 154776,
  "unverified_events": 47,
  "score_breakdown": {
    "chain_intact": true,
    "gap_events": 47,
    "gap_reason": "Redis unavailable during 2025-01-15 outage"
  }
}
```

## Recovery

### After Tamper Detection

1. **Freeze audit log**: Stop accepting new events
2. **Export current state**: Full dump for forensics
3. **Rebuild chain**: From last verified event
4. **Document gaps**: Record why gaps occurred
5. **Resume logging**: Start new chain from verified state

### After Data Loss

If audit events are lost (disk failure):

1. Restore from backup
2. Run integrity verification
3. If chain broken at backup boundary: document gap
4. SIEM copies may fill gaps (if forwarding was active)

## Scheduled Verification

```yaml
# Kubernetes CronJob — verify daily
apiVersion: batch/v1
kind: CronJob
metadata:
  name: audit-integrity-check
spec:
  schedule: "0 6 * * *"  # Daily at 6 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: verify
            image: curlimages/curl
            command:
            - curl
            - -X
            - GET
            - http://audit.ggid.svc.cluster.local:8072/api/v1/audit/integrity/verify
```

### Alerting on Integrity Failure

```bash
curl -X POST https://api.ggid.example.com/api/v1/audit/alerts/rules \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "Audit integrity failure",
    "condition": "integrity.valid = false",
    "severity": "critical",
    "action": "email",
    "recipients": ["security@company.com", "dpo@company.com"]
  }'
```

## See Also

- [Audit & SIEM Guide](audit-siem-guide.md)
- [Audit Events API](../api/audit-events.md)
- [Audit API](../api/audit-api.md)
- [Data Retention Policy](data-retention-policy.md)
