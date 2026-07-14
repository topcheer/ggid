# Real-Time Alerting Design

> Anomaly detection and alerting engine for GGID audit events via NATS.

---

## Competitor Analysis

### Auth0 Anomaly Detection
- Brute-force protection (auto-block after N failed logins)
- Impossible travel detection (login from geographically distant IPs)
- New device/location alerts
- Breached password detection (HaveIBeenPwned integration)

### Okta Threat Insights
- Risk scoring per login (low/medium/high)
- IP reputation checks
- Device fingerprinting
- Adaptive MFA (require MFA when risk > threshold)

---

## GGID Alert Engine Design

### Architecture

```
NATS AUDIT_EVENTS → Alert Engine (rules consumer) → Notification Service
                         ↓                            ↓
                    Rule Evaluation              Email / Slack / Webhook
                    (Go, in-memory)
```

### Alert Rules Table

```sql
CREATE TABLE alert_rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    condition   JSONB NOT NULL,  -- e.g. {"action":"user.login","metadata.success":false,"count_gt":5,"window":"60s"}
    severity    VARCHAR(20) DEFAULT 'medium',  -- low, medium, high, critical
    channels    TEXT[] NOT NULL DEFAULT '{email}',  -- email, slack, webhook
    enabled     BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

### Rule Evaluation (Go)

```go
type AlertRule struct {
    Name      string
    Condition AlertCondition
    Severity  string
}

type AlertCondition struct {
    Action      string `json:"action"`      // e.g. "user.login"
    SuccessEq   *bool  `json:"metadata.success"` // nil = any
    CountGT     int    `json:"count_gt"`    // threshold
    Window      string `json:"window"`      // e.g. "60s"
}

func (e *AlertEngine) Evaluate(ctx context.Context, event *AuditEvent) {
    for _, rule := range e.rules {
        if rule.Condition.Action != "" && rule.Condition.Action != event.Action {
            continue
        }
        if rule.Condition.SuccessEq != nil && *rule.Condition.SuccessEq != event.Metadata.Success {
            continue
        }
        // Check count threshold in sliding window
        count := e.counter.Count(event.ActorID, rule.Condition.Window)
        if count >= rule.Condition.CountGT {
            e.notify(ctx, rule, event, count)
        }
    }
}
```

### Built-in Alert Templates

| Template | Condition | Severity |
|----------|-----------|----------|
| Brute force | 5+ failed logins in 60s | High |
| Impossible travel | Login from 2 IPs >1000km apart in 10min | High |
| New admin role | `role.assign` to admin | Medium |
| Mass deletion | 10+ `user.delete` in 5min | Critical |
| Hash chain tamper | `security.hash_chain_tamper` | Critical |
| Off-hours access | Login outside business hours | Low |

### Notification Channels

```go
type Notification struct {
    Channel string // email, slack, webhook
    Target  string // address, webhook URL
    Subject string
    Body    string
}
```

Slack example:
```json
{
  "text": "[HIGH] Brute force detected: 6 failed logins for usr_abc from 192.168.1.50 in 60s"
}
```

---

## Implementation Estimate

| Component | Effort |
|-----------|--------|
| Alert rules table + CRUD | 1 day |
| Rule evaluation engine | 2 days |
| Sliding window counter (Redis) | 1 day |
| Notification dispatcher | 1 day |
| Console UI (rule management) | 2 days |
| Impossible travel detection | 1 day |
| **Total** | **8 days** |

Priority: P2 (GGID has rate limiting + audit, but no proactive alerting).

---

*See: [Event-Driven Architecture](../architecture/event-driven.md) | Audit Compliance | [SIEM Connector Design](siem-connector-design.md)*

*Last updated: 2025-07-11*
