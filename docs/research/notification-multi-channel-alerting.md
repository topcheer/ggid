# Multi-Channel Notification & Alerting System for GGID

> **Focus**: Production notification platform — multi-channel delivery (email/SMS/Slack/Teams/PagerDuty/webhook/in-app), severity-based routing, user preferences, deduplication, escalation, and integration with GGID's SOAR, webhook engine, and ITDR.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Related**: `realtime-alerting-design.md`, `siem-connector-design.md`, `audit/internal/alerting/`.
>
> **Checklist Compliance**: DoD per backlog item (§7).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: Alerting](#2-ggid-current-state-alerting)
3. [Gap Analysis](#3-gap-analysis)
4. [Architecture](#4-architecture)
5. [Notification Templates](#5-notification-templates)
6. [Routing Rules](#6-routing-rules)
7. [Deduplication & Escalation](#7-deduplication--escalation)
8. [User Preferences](#8-user-preferences)
9. [Database Schema](#9-database-schema)
10. [Implementation Backlog with DoD](#10-implementation-backlog-with-dod)
11. [Competitive Differentiation](#11-competitive-differentiation)

---

## 1. Executive Summary

GGID has **partial alerting infrastructure** — an alerting package (`audit/internal/alerting/`), webhook engine (`audit/internal/webhook/`), and email provider (`auth/server/email_provider.go`). However, there's no unified notification routing, no multi-channel delivery, no deduplication, and no escalation.

**Existing:**
- Alerting package (`audit/internal/alerting/`) ✅
- Webhook engine with HMAC signing + retry (`audit/internal/webhook/engine.go`) ✅
- Email provider handler (`auth/server/email_provider.go`) ✅
- SOAR `notify_soc` action ✅
- Realtime alerting research (`realtime-alerting-design.md`) ✅

**Missing:**
1. **No multi-channel delivery** — Only webhook/email, no SMS/Slack/Teams/PagerDuty
2. **No routing rules** — All alerts go same channel regardless of severity
3. **No deduplication** — 100 failed logins = 100 notifications
4. **No escalation** — Critical alerts don't escalate if unacknowledged
5. **No user preferences** — Users can't choose channels or quiet hours
6. **No template engine** — No per-event-type message templates

**Recommendation**: Build a **Notification Router** that accepts alerts from ITDR/SOAR/system, applies routing rules (severity→channel), deduplicates within time windows, escalates unacknowledged criticals, and delivers via 7 channels.

---

## 2. GGID Current State

| Component | File | Status |
|-----------|------|--------|
| Alerting | `audit/internal/alerting/` | ✅ Basic |
| Webhook engine | `audit/internal/webhook/engine.go` | ✅ HMAC + retry + dead-letter |
| Email provider | `auth/server/email_provider.go` | ✅ |
| SOAR notify_soc | `audit/internal/soar/` | ✅ |
| Realtime alerting research | `realtime-alerting-design.md` | ✅ Theory |

---

## 3. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | No SMS/Slack/Teams/PagerDuty | Can't reach on-call via preferred channel |
| 2 | No severity routing | Low-priority spam to PagerDuty |
| 3 | No deduplication | Alert fatigue |
| 4 | No escalation | Critical ignored = breach |
| 5 | No user prefs | Users can't opt out of noise |
| 6 | No templates | Messages inconsistent |

---

## 4. Architecture

```
Alert Source (ITDR/SOAR/System)
    │
    ▼
Notification Router
    │
    ├── Deduplication (group within 5 min window)
    │
    ├── Routing Rules (severity → channels)
    │
    ├── Template Engine (per event type)
    │
    ├── Channel Delivery (parallel)
    │   ├── Email (SMTP)
    │   ├── SMS (Twilio)
    │   ├── Slack (webhook)
    │   ├── Teams (webhook)
    │   ├── PagerDuty (Events API)
    │   ├── Webhook (custom)
    │   └── In-App (WebSocket push)
    │
    └── Escalation Tracker
        ├── Critical unacknowledged 5 min → escalate
        ├── Escalate to: manager → director → CISO
        └── Log every step
```

---

## 5. Notification Templates

| Event Type | Template | Channels |
|-----------|----------|----------|
| ITDR: brute_force | "Brute force detected on {user} from {ip} ({count} attempts)" | Slack, Email |
| ITDR: token_theft | "CRITICAL: Token theft detected for {user} — session revoked" | PagerDuty, SMS, Slack |
| ITDR: impossible_travel | "Impossible travel: {user} logged in from {country1} then {country2}" | Slack, Email |
| Risk: score_critical | "Risk score {score} for {user} — access blocked" | PagerDuty, Slack |
| Compliance: gap_detected | "Compliance gap: {control} missing evidence" | Email |
| System: backup_failed | "Backup failed: {reason}" | PagerDuty, SMS |
| System: cert_expiring | "TLS cert expires in {days} days for {domain}" | Email |
| Account: locked | "Account {user} locked due to {reason}" | Email |
| Audit: tamper_detected | "CRITICAL: Audit tamper detected at event {id}" | PagerDuty, SMS, Slack, Teams |

---

## 6. Routing Rules

| Severity | Channels | Latency Target |
|----------|----------|---------------|
| **Critical** | PagerDuty + SMS + Slack + Teams | < 30s |
| **High** | Slack + Email | < 2 min |
| **Medium** | Email | < 5 min |
| **Low** | In-App only | < 15 min |

### Configurable Routing

```json
{
  "routing_rules": [
    { "severity": "critical", "channels": ["pagerduty", "sms", "slack", "teams"] },
    { "severity": "high", "channels": ["slack", "email"] },
    { "severity": "medium", "channels": ["email"] },
    { "severity": "low", "channels": ["in_app"] },
    { "event_type": "audit_tamper", "override_channels": ["pagerduty", "sms", "slack", "teams", "email"] }
  ]
}
```

---

## 7. Deduplication & Escalation

### Deduplication

```go
type DedupConfig struct {
    WindowSeconds    int  // Group identical alerts within window
    MaxPerWindow     int  // Max notifications per window
    GroupByKey       []string // ["event_type", "user_id", "ip_address"]
}

// Example: 10 brute_force alerts for same user+IP in 5 min = 1 notification
// "10 brute force attempts from 203.0.113.42 on alice@corp.com (last 5 min)"
```

### Escalation

```
Critical alert issued → PagerDuty + SMS
  │
  ├── Acknowledged within 5 min → Close
  │
  └── Unacknowledged after 5 min → Escalate Tier 2
      ├── Notify: Engineering Manager (Slack + Call)
      │
      └── Unacknowledged after 15 min → Escalate Tier 3
          ├── Notify: Director + CISO (SMS + Call)
          │
          └── Unacknowledged after 30 min → Auto-respond
              ├── SOAR: Isolate affected user
              ├── Revoke all sessions
              └── Create incident ticket
```

---

## 8. User Preferences

```bash
# User sets notification preferences
PUT /api/v1/self-service/notification-preferences
{
  "channels": {
    "critical": ["sms", "email"],
    "high": ["email"],
    "medium": ["in_app"],
    "low": ["in_app"]
  },
  "quiet_hours": {
    "start": "22:00",
    "end": "07:00",
    "timezone": "America/Los_Angeles",
    "override_for_critical": true
  }
}
```

---

## 9. Database Schema

```sql
CREATE TABLE notification_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    event_type      VARCHAR(64) NOT NULL,
    channel         VARCHAR(32) NOT NULL,
    subject_template TEXT,
    body_template   TEXT NOT NULL,
    enabled         BOOLEAN DEFAULT true,
    UNIQUE(tenant_id, event_type, channel)
);

CREATE TABLE notification_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    event_type      VARCHAR(64) NOT NULL,
    severity        VARCHAR(16) NOT NULL,
    channels_tried  JSONB NOT NULL,        -- ["sms", "email", "slack"]
    channels_delivered JSONB DEFAULT '[]',
    channels_failed JSONB DEFAULT '[]',
    dedup_key       VARCHAR(256),          -- For grouping
    acknowledged    BOOLEAN DEFAULT false,
    acknowledged_by UUID,
    acknowledged_at TIMESTAMPTZ,
    escalated       BOOLEAN DEFAULT false,
    escalation_tier INT DEFAULT 0,
    payload         JSONB,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE notification_preferences (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    channel_prefs   JSONB NOT NULL,        -- {"critical": ["sms","email"], ...}
    quiet_start     VARCHAR(5),            -- "22:00"
    quiet_end       VARCHAR(5),            -- "07:00"
    quiet_timezone  VARCHAR(64),
    override_critical BOOLEAN DEFAULT true,
    UNIQUE(tenant_id, user_id)
);

CREATE TABLE notification_channels (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    channel_type    VARCHAR(32) NOT NULL,  -- 'email','sms','slack','teams','pagerduty','webhook'
    config          JSONB NOT NULL,        -- {"webhook_url": "...", "api_key": "..."}
    enabled         BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, channel_type)
);

CREATE INDEX idx_notif_log_tenant_time ON notification_log (tenant_id, created_at DESC);
CREATE INDEX idx_notif_log_unack ON notification_log (tenant_id, severity) WHERE acknowledged = false;
CREATE INDEX idx_notif_log_dedup ON notification_log (tenant_id, dedup_key, created_at DESC);
```

---

## 10. Implementation Backlog with DoD

### P0 — Router + Channels (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Notification router + dedup | ✅ Severity routing ✅ Dedup window ✅ DB-backed ✅ ≥3 tests | 4d |
| 2 | Channel adapters (email, Slack, Teams, webhook) | ✅ 4 channels ✅ Template rendering ✅ ≥3 tests each | 4d |
| 3 | SMS (Twilio) + PagerDuty adapters | ✅ Twilio API ✅ PagerDuty Events API ✅ ≥3 tests | 3d |
| 4 | Template engine (per event type) | ✅ 9 templates ✅ Variable substitution ✅ ≥3 tests | 2d |

### P1 — Escalation + Preferences (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 5 | Escalation engine (3 tiers) | ✅ Auto-escalate ✅ Tier config ✅ ≥3 tests | 3d |
| 6 | User notification preferences | ✅ Channel selection ✅ Quiet hours ✅ ≥3 tests | 3d |
| 7 | In-app notifications (WebSocket push) | ✅ Real-time push ✅ Read/unread ✅ ≥3 tests | 2d |

### P2 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 8 | Notification analytics dashboard | Delivery rate, ack time, escalation stats |
| 9 | Smart dedup (ML-based grouping) | Cluster similar alerts intelligently |
| 10 | Multi-language templates | i18n for notifications |
| 11 | On-call schedule integration | Respect on-call rotation |

---

## 11. Competitive Differentiation

| Feature | GGID (target) | PagerDuty | Opsgenie | Grafana OnCall | Splunk |
|---------|---------------|-----------|----------|---------------|--------|
| **Channels** | 7 | 8+ | 8+ | 5 | 6 |
| **Routing** | Severity + event type | Advanced | Advanced | Basic | Advanced |
| **Dedup** | Time window | Advanced | Advanced | Basic | Advanced |
| **Escalation** | 3-tier | Multi-tier | Multi-tier | Basic | Multi-tier |
| **User prefs** | Quiet hours + channels | Yes | Yes | Yes | Partial |
| **ITDR-native** | ✅ Integrated | External | External | External | External |
| **Open source** | Yes | No | No | Yes | No |

**Key differentiator**: GGID notifications are **ITDR-native** — detections flow directly to the notification router without external integration. PagerDuty/Opsgenie require webhook setup; GGID does it internally.

---

## References

- [PagerDuty Events API](https://developer.pagerduty.com/docs/events-api-v2/)
- [Twilio SMS API](https://www.twilio.com/docs/sms)
- [Slack Webhooks](https://api.slack.com/messaging/webhooks)
- [Teams Webhooks](https://learn.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/)
- [Opsgenie](https://docs.opsgenie.com/)
- [Grafana OnCall](https://grafana.com/docs/oncall/)
- [GGID Webhook Engine](../services/audit/internal/webhook/engine.go)
- [GGID Alerting Package](../services/audit/internal/alerting/)
- [GGID Realtime Alerting](./realtime-alerting-design.md)
- [GGID SOAR Integration](./itdr-maturity-mitre-attack.md) — SOAR webhook
