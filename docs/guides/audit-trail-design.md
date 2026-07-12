# Audit Trail Design

This guide covers audit event schema, event types, immutable storage, hash chain integrity, query API, retention policy, PII handling, real-time alerting, and GGID's audit trail implementation.

## Audit Event Schema

### Core Schema

```json
{
  "event_id": "evt-uuid-12345678",
  "timestamp": "2026-07-12T10:00:00.123456789Z",
  "event_type": "user.login",
  "severity": "info",
  "tenant_id": "tenant-uuid-9012",
  "actor": {
    "user_id": "user-uuid-3456",
    "user_name": "jdoe@example.com",
    "ip": "192.168.1.50",
    "user_agent": "Mozilla/5.0 ...",
    "session_id": "sess-uuid-7890"
  },
  "action": {"name": "login", "category": "auth", "method": "password+mfa"},
  "resource": {"type": "auth-service", "id": "auth-service"},
  "result": "success",
  "details": {"mfa_type": "totp", "mfa_verified": true, "duration_ms": 1200},
  "hash_chain": {"sequence": 12345, "prev_hash": "abc123...", "block_hash": "def456..."}
}
```

### WHO / WHAT / WHEN / WHERE / RESULT

| Element | Fields | Purpose |
|---|---|---|
| WHO | actor.user_id, actor.user_name | Who performed the action |
| WHAT | action.name, action.category, resource | What was done to what |
| WHEN | timestamp, event_id | When it happened |
| WHERE | actor.ip, actor.user_agent, actor.session_id | Where it originated |
| RESULT | result, details | Whether it succeeded |

## Event Types

### Event Type Taxonomy

| Category | Event Types | Severity Range |
|---|---|---|
| auth | login, login_failed, logout, token_refresh, mfa_challenge | info - high |
| access | api_call, resource_access, permission_denied | info - medium |
| admin | user_create, user_delete, role_assign, config_change | info - critical |
| config | sso_config, mfa_config, policy_change, tenant_config | info - high |
| data | data_export, data_import, data_delete, data_modify | info - high |
| security | intrusion_detected, token_revoked, session_terminated, breach_alert | high - critical |

### Severity Levels

| Level | Numeric | Use Case |
|---|---|---|
| debug | 0 | Detailed debugging (not stored by default) |
| info | 1 | Normal operations (login success, API call) |
| warn | 2 | Suspicious but allowed (rate limit hit, MFA denied) |
| error | 3 | Failed operations (login failed, permission denied) |
| critical | 4 | Security events (breach, admin action, data export) |

## Immutable Storage

### Append-Only Design

```sql
CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(64) UNIQUE NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    tenant_id UUID NOT NULL,
    user_id VARCHAR(64),
    action_name VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    result VARCHAR(20) NOT NULL,
    ip VARCHAR(45),
    details JSONB,
    sequence BIGINT NOT NULL,
    prev_hash VARCHAR(64) NOT NULL,
    block_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### WORM (Write Once Read Many)

```sql
CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit events are immutable: cannot % on audit_events', TG_OP;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER no_update_audit BEFORE UPDATE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

CREATE TRIGGER no_delete_audit BEFORE DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();
```

## Hash Chain Integrity

### Block Structure

```
Block N-1:                Block N:                 Block N+1:
+-------------+          +-------------+          +-------------+
| event data  |          | event data  |          | event data  |
| sequence: N |          | sequence:N+1|          | sequence:N+2|
| prev_hash: H|--------->| prev_hash: H|--------->| prev_hash: H|
| block_hash: |          | block_hash: |          | block_hash: |
| SHA256(data |          | SHA256(data |          | SHA256(data |
| + prev_hash)|          | + prev_hash)|          | + prev_hash)|
+-------------+          +-------------+          +-------------+
```

### Implementation

```go
func computeBlockHash(event *AuditEvent, prevHash string) string {
    data := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s",
        event.Sequence, event.EventID, event.Timestamp.Format(time.RFC3339Nano),
        event.EventType, event.TenantID, event.UserID, event.Result)
    h := sha256.Sum256([]byte(data + "|" + prevHash))
    return hex.EncodeToString(h[:])
}

func appendAuditEvent(event *AuditEvent) error {
    lastEvent := getLastAuditEvent()
    event.Sequence = lastEvent.Sequence + 1
    event.PrevHash = lastEvent.BlockHash
    event.BlockHash = computeBlockHash(event, lastEvent.BlockHash)
    return store(event)
}
```

### Tamper Detection

```go
func verifyHashChain(start, end int64) error {
    events := getEvents(start, end)
    for i := 1; i < len(events); i++ {
        expectedHash := computeBlockHash(events[i], events[i-1].BlockHash)
        if events[i].BlockHash != expectedHash {
            return fmt.Errorf("hash chain broken at sequence %d", events[i].Sequence)
        }
    }
    return nil
}
```

## Query API

### Filter Parameters

```bash
GET /api/v1/audit/events?
  tenant_id=tenant-uuid&
  event_type=user.login&
  severity=error&
  user_id=user-uuid&
  start_time=2026-07-12T00:00:00Z&
  end_time=2026-07-12T23:59:59Z&
  page=1&per_page=50&sort=timestamp:desc
Authorization: Bearer <admin_token>
```

### Export

```bash
POST /api/v1/audit/export
{"format": "csv", "filters": {"tenant_id": "...", "start_time": "...", "end_time": "..."}}
```

### Aggregation

```bash
GET /api/v1/audit/aggregate?group_by=event_type,severity&start_time=...&end_time=...
```

## Retention Policy

### Retention by Event Type

| Event Type | Retention | Rationale |
|---|---|---|
| auth events | 1 year | Security investigation |
| admin events | 2 years | Compliance |
| config events | 2 years | Compliance |
| data events | 1 year | Data access tracking |
| security events | 3 years | Forensic investigation |
| access events | 90 days | Volume management |

### Automated Purging

```go
func purgeOldEvents() {
    now := time.Now()
    purgeBefore("auth", now.AddDate(-1, 0, 0))
    purgeBefore("admin", now.AddDate(-2, 0, 0))
    purgeBefore("security", now.AddDate(-3, 0, 0))
    purgeBefore("access", now.AddDate(0, 0, -90))
}
```

### Legal Hold

```yaml
audit:
  retention:
    legal_hold:
      enabled: true
      override_retention: true
      notify_admin: true
```

## PII Handling in Audit

### Masking

```go
func maskPIIInAudit(event *AuditEvent) *AuditEvent {
    if event.Actor != nil {
        event.Actor.UserName = pii.MaskEmail(event.Actor.UserName)
        event.Actor.IP = pii.MaskIP(event.Actor.IP)
    }
    if event.Details != nil {
        event.Details = pii.Obfuscate(event.Details)
    }
    return event
}
```

### Masking Examples

| Original | Masked |
|---|---|
| jdoe@example.com | j***@example.com |
| 192.168.1.50 | 192.168.x.x |
| +1-555-123-4567 | +1-*-***-**** |
| John Doe | J** D** |

### Hashing for Correlation

```go
event.Details["email_hash"] = sha256Hash(email + salt)
// Can correlate events by same email without storing the email
```

## Real-Time Alerting

### Alert Rules

```yaml
audit:
  alerting:
    rules:
      - name: "failed_login_burst"
        condition: "count(user.login_failed) > 10 in 5min per user"
        severity: "high"
        notify: ["security-team"]
      - name: "admin_after_hours"
        condition: "admin.* AND time > 18:00 OR time < 06:00"
        severity: "medium"
        notify: ["security-team"]
      - name: "mass_data_export"
        condition: "data.export AND count > 1000 in 1h"
        severity: "high"
        notify: ["security-team", "ciso"]
      - name: "privilege_escalation"
        condition: "role.assign AND role IN [admin, security-admin]"
        severity: "critical"
        notify: ["security-team", "ciso"]
```

## GGID Audit Trail Implementation

### Configuration

```yaml
audit:
  enabled: true
  storage:
    type: "postgresql"
    immutable: true
    worm: true
  hash_chain:
    enabled: true
    algorithm: "sha256"
    verification_interval: 1h
  retention:
    auth: 1y
    admin: 2y
    config: 2y
    data: 1y
    security: 3y
    access: 90d
    legal_hold: true
  pii:
    masking: true
    hashing_for_correlation: true
  query:
    max_per_page: 100
    max_export_rows: 100000
    require_admin: true
  alerting:
    enabled: true
    real_time: true
    channels: [slack, email, pagerduty]
  siem:
    forward: true
    format: "json"
```

### API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/audit/events` | GET | Query audit events |
| `/api/v1/audit/events/{id}` | GET | Get single event |
| `/api/v1/audit/export` | POST | Export audit data |
| `/api/v1/audit/aggregate` | GET | Aggregate statistics |
| `/api/v1/audit/verify-chain` | POST | Verify hash chain |
| `/api/v1/audit/alerts` | GET | Get triggered alerts |

## Best Practices

1. **Make audit immutable** — No UPDATE or DELETE, ever
2. **Use hash chains** — Detect any tampering attempt
3. **Mask PII** — Don't store raw PII in audit logs
4. **Set retention policies** — Don't keep data longer than needed
5. **Alert in real-time** — Don't wait for manual review
6. **Use append-only storage** — Database triggers prevent modification
7. **Verify periodically** — Run hash chain verification hourly
8. **Export for SIEM** — Forward to Splunk/Elastic for analysis
9. **Legal hold support** — Override retention for litigation
10. **Query with filters** — Support tenant, type, severity, time range