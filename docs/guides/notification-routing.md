# Notification Routing — Technical Guide

> Feature: Multi-Channel Alert Routing with Escalation
> Location: `services/audit/internal/alerting/`
> Console: `/monitoring/alerts`

## What It Does

GGID routes security alerts and operational notifications across 7 delivery channels with severity-based routing, escalation tiers, and per-user preferences. Alerts from ITDR detections, anomaly rules, compliance gaps, and system events are delivered to the right people through their preferred channels.

## Delivery Channels

| Channel | Mechanism | Use Case |
|---------|----------|----------|
| **Email** | SMTP / SendGrid API | Standard notifications |
| **Webhook** | HTTP POST with HMAC | Integration with Slack, Teams, PagerDuty |
| **SMS** | Twilio / AWS SNS | Critical off-hours alerts |
| **Slack** | Slack Webhook / Bot API | Team-wide visibility |
| **Microsoft Teams** | Teams Webhook | Enterprise collaboration |
| **PagerDuty** | Events API | On-call escalation |
| **In-app** | WebSocket push | Real-time console notifications |

## Severity Routing

Alerts are routed based on severity level:

| Severity | Default Channels | Response SLA |
|----------|-----------------|-------------|
| **Critical** | PagerDuty + SMS + Slack + Email | 5 minutes |
| **High** | Slack + Email + In-app | 15 minutes |
| **Medium** | Email + In-app | 1 hour |
| **Low** | In-app only | Next business day |

## Escalation Tiers

When an alert is not acknowledged within the response SLA, it escalates:

```
Tier 0: Primary on-call (PagerDuty)
    ↓ 5 min no ack
Tier 1: Secondary on-call (SMS + Email)
    ↓ 10 min no ack
Tier 2: Team channel (Slack/Teams broadcast)
    ↓ 15 min no ack
Tier 3: Manager + Security lead (Email + SMS)
```

Each escalation includes the full alert context and acknowledgment history.

## Alert Sources

| Source | Trigger |
|--------|---------|
| ITDR detections | mfa_fatigue, token_theft, etc. |
| Anomaly rules | Unusual login patterns, spike in failures |
| Compliance gaps | Coverage drop below threshold |
| Backup failures | Backup overdue or integrity check failed |
| Secrets health | Provider unreachable or rotation overdue |
| Rate limit | Tenant exceeding quota |

## Per-User Preferences

Users can configure their notification preferences:

```json
{
  "user_id": "admin",
  "channels": {
    "critical": ["pagerduty", "sms", "slack"],
    "high": ["slack", "email"],
    "medium": ["email"],
    "low": ["in_app"]
  },
  "quiet_hours": {
    "start": "22:00",
    "end": "07:00",
    "timezone": "Asia/Shanghai",
    "override_for_critical": true
  }
}
```

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/audit/alerts/config` | GET/PUT | Get/update alert routing config |
| `/api/v1/audit/alerts/test` | POST | Send test alert to verify routing |
| `/api/v1/audit/alerts/evaluate` | POST | Manually trigger alert evaluation |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# Get alert config
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/alerts/config" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Send test alert
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/audit/alerts/test" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"severity":"high","message":"Test alert from admin"}'
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Alerts not received | Channel misconfigured or credentials invalid | Test alert to verify routing |
| Too many alerts | Filter too broad | Adjust severity thresholds; use quiet hours |
| Escalation not triggering | Ack timeout not configured | Check escalation tier SLA settings |
| Slack/Teams delivery fails | Webhook URL expired | Rotate webhook URL in alert config |

## Best Practices

- **Test before relying**: Send test alerts to each channel after setup.
- **Use quiet hours**: Reduce alert fatigue with scheduled quiet periods.
- **Reserve SMS for critical**: SMS costs money and causes fatigue — limit to critical.
- **Monitor delivery rates**: Track which channels have high failure rates.
- **Review escalation monthly**: Ensure escalation contacts are current.
