# Audit & Compliance Guide

> Query audit events, verify hash chain integrity, export compliance reports, configure data retention.

---

## Prerequisites

- Admin JWT with `read:audit` scope
- `X-Tenant-ID` header

---

## 1. Query Audit Events

### Basic Query

```bash
curl -s "http://localhost:8080/api/v1/audit/events?limit=50" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

**Response (200):**
```json
{
  "events": [
    {
      "id": "evt_001",
      "actor_type": "user",
      "actor_id": "usr_abc123",
      "action": "user.login",
      "resource_type": "auth",
      "metadata": {"ip": "192.168.1.1", "success": true},
      "timestamp": "2025-07-11T12:00:00Z",
      "hash": "sha256:abc..."
    }
  ],
  "total": 15423
}
```

### Filter Parameters

| Param | Example | Description |
|-------|---------|-------------|
| `action` | `user.login` | Filter by action type |
| `actor_type` | `user` | Filter: user, system, api_key |
| `actor_id` | `usr_abc123` | Filter by specific actor |
| `resource_type` | `users` | Filter by resource category |
| `start_time` | `2025-07-01T00:00:00Z` | Events after this time |
| `end_time` | `2025-07-31T23:59:59Z` | Events before this time |
| `limit` | `100` | Results per page (max 200) |
| `offset` | `0` | Pagination offset |

### Example: Find all failed logins

```bash
curl -s "http://localhost:8080/api/v1/audit/events?action=user.login&start_time=2025-07-01T00:00:00Z&limit=100" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq '.events[] | select(.metadata.success == false)'
```

---

## 2. Hash Chain Verification

Every audit event is linked via a cryptographic hash chain, ensuring tamper detection.

### Verify Integrity

```bash
curl -s http://localhost:8080/api/v1/audit/verify \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

**Response (healthy):**
```json
{
  "verified": true,
  "total_events": 15423,
  "tampered_events": []
}
```

**Response (tampered):**
```json
{
  "verified": false,
  "total_events": 15423,
  "tampered_events": ["evt_0042", "evt_0043"]
}
```

### How It Works

```
Event₁ → hash₁ = SHA256("" + Event₁.data)
Event₂ → hash₂ = SHA256(hash₁ + Event₂.data)
Event₃ → hash₃ = SHA256(hash₂ + Event₃.data)
```

Any modification, deletion, or insertion breaks the chain.

---

## 3. Compliance Reports

### PCI-DSS Access Report

Generate card data access logs for PCI-DSS audit:

```bash
curl -s "http://localhost:8080/api/v1/audit/events?resource_type=card_data&start_time=2025-07-01T00:00:00Z&end_time=2025-07-31T23:59:59Z&limit=1000" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" > pci_report.json
```

### HIPAA PHI Access Report

```bash
curl -s "http://localhost:8080/api/v1/audit/events?resource_type=patient_records&limit=1000" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" > hipaa_report.json
```

### SOC 2 Change History

```bash
curl -s "http://localhost:8080/api/v1/audit/events?action=role.update&action=user.update&limit=500" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" > soc2_report.json
```

---

## 4. Data Retention

### Default Retention

| Data | Retention | Storage |
|------|-----------|---------|
| Audit events (DB) | 365 days | PostgreSQL |
| NATS stream | 7 days | NATS file storage |
| Session data | 24 hours | Redis |

### Configuring Retention

```bash
# Database retention (days)
AUDIT_RETENTION_DAYS=365

# NATS stream retention
NATS_RETENTION=168h
```

### Automated Cleanup

The Audit Service runs a daily cleanup job that deletes events older than `AUDIT_RETENTION_DAYS`. The hash chain is recalculated after deletion to maintain integrity.

---

## 5. Export for External SIEM

Export events to Splunk, Datadog, or other SIEM via API polling:

```python
import requests
import schedule
import time

last_timestamp = None

def export_events():
    global last_timestamp
    params = {"limit": 200}
    if last_timestamp:
        params["start_time"] = last_timestamp

    resp = requests.get(
        "http://ggid:8080/api/v1/audit/events",
        headers={"Authorization": f"Bearer {TOKEN}", "X-Tenant-ID": TENANT},
        params=params
    )

    for event in resp.json()["events"]:
        send_to_siem(event)  # Your SIEM integration
        last_timestamp = event["timestamp"]

schedule.every(60).seconds.do(export_events)
while True:
    schedule.run_pending()
    time.sleep(1)
```

---

## 6. Best Practices

1. **Regular verification:** Run hash chain verification daily via cron
2. **Off-site backup:** Export events to cold storage weekly
3. **Access control:** Restrict `read:audit` to compliance/security teams
4. **Monitor failures:** Alert on `verified: false` from hash chain checks
5. **Retention policy:** Set retention to match regulatory requirements (PCI: 1yr, HIPAA: 6yr)

---

*See: [Event-Driven Architecture](../architecture/event-driven.md) | [Security Overview](../architecture/security-overview.md) | [API Reference](../api/rest-api.md)*

*Last updated: 2025-07-11*
