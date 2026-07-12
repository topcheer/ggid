# Incident Response Automation

This guide covers detection triggers, automated response actions, SOAR integration, incident severity matrix, playbook templates, post-incident review, and GGID's automated response.

## Detection Triggers

### Automated Detection Signals

| Trigger | Condition | Source | Severity |
|---|---|---|---|
| Failed login spike | >20 failed logins in 5 min per user | Audit | High |
| IP login burst | >100 logins from 1 IP in 10 min | Audit | High |
| Anomalous access | User accesses unusual resources | Audit | Medium |
| Data exfiltration | >10,000 records exported in 1h | Audit | Critical |
| MFA fatigue | >5 MFA pushes in 10 min per user | Auth | High |
| Impossible travel | Logins from distant locations too fast | Auth | High |
| Admin after hours | Admin action outside business hours | Audit | Medium |
| Token reuse | Refresh token used after rotation | OAuth | Critical |
| Policy bypass attempt | Repeated denied access attempts | Policy | Medium |
| New device flood | >5 new devices in 1h per user | Auth | Medium |

### Detection Implementation

```go
type DetectionEngine struct {
    rules    []DetectionRule
    audit    AuditStore
    alerting AlertService
}

type DetectionRule struct {
    Name      string
    Condition func(events []*AuditEvent) bool
    Severity  string
    Actions   []ResponseAction
}

func (e *DetectionEngine) Evaluate(event *AuditEvent) {
    for _, rule := range e.rules {
        recentEvents := e.audit.GetRecent(event.UserID, 10*time.Minute)
        if rule.Condition(append(recentEvents, event)) {
            e.executeActions(rule.Actions, event)
            e.alerting.Alert(rule.Name, rule.Severity, event)
        }
    }
}
```

## Automated Response Actions

### Action Catalog

| Action | Description | Trigger |
|---|---|---|
| Account lockout | Disable user account | Credential stuffing, token reuse |
| IP block | Block source IP | IP burst, scanning |
| Step-up auth | Require MFA re-verification | New device, anomalous access |
| Session revoke | Terminate all user sessions | Token reuse, account compromise |
| Rate limit increase | Tighten rate limits | Failed login spike |
| SIEM alert | Forward to SIEM | All triggers |
| Admin notification | Email/Slack security team | High/Critical severity |
| User notification | Email user about activity | Suspicious activity |
| Webhook fire | Trigger external webhook | Configured events |
| Quarantine | Isolate affected resources | Data exfiltration |

### Action Implementation

```go
type ResponseAction interface {
    Execute(event *AuditEvent) error
}

type AccountLockoutAction struct{}

func (a *AccountLockoutAction) Execute(event *AuditEvent) error {
    user := getUser(event.UserID)
    user.Locked = true
    user.LockReason = "automated: credential_stuffing_detected"
    user.LockedAt = time.Now()
    saveUser(user)
    
    // Revoke all sessions
    revokeAllSessions(event.UserID)
    
    // Audit
    audit.Log("account_locked", event.UserID, "automated_response")
    
    // Notify
    notifyUser(event.UserID, "Your account has been locked due to suspicious activity")
    notifyAdmin("Account locked: " + event.UserID)
    
    return nil
}

type IPBlockAction struct{}

func (a *IPBlockAction) Execute(event *AuditEvent) error {
    redis.Set(ctx, "ip:blocked:"+event.IP, "1", 1*time.Hour)
    audit.Log("ip_blocked", event.IP, "automated_response")
    return nil
}
```

## SOAR Integration

### Security Orchestration, Automation, and Response

```yaml
soar:
  enabled: true
  platform: "splunk-soar"  # or "cortex-xsoar", "ibm-resilient"
  api_url: "https://soar.example.com/api"
  api_key: "<soar-api-key>"
  
  # Map GGID events to SOAR incidents
  event_mapping:
    critical:
      soar_severity: "critical"
      auto_create_incident: true
      assign_to: "security-oncall"
    high:
      soar_severity: "high"
      auto_create_incident: true
      assign_to: "security-team"
    medium:
      soar_severity: "medium"
      auto_create_incident: false
      log_only: true
```

### Webhook Integration

```yaml
soar:
  webhook:
    url: "https://soar.example.com/webhooks/ggid"
    secret: "<hmac-secret>"
    events:
      - type: "security.critical"
        forward: true
      - type: "security.high"
        forward: true
      - type: "auth.account_locked"
        forward: true
```

## Incident Severity → Response Matrix

| Severity | Detection | Auto Response | Notification | Escalation |
|---|---|---|---|---|
| Critical | Immediate | Lock + block + revoke + alert | PagerDuty + Email + Slack | CISO + Security team |
| High | <5 min | Challenge + rate limit + alert | Slack + Email | Security team |
| Medium | <15 min | Log + monitor | Slack channel | Security analyst |
| Low | <1 hour | Log only | Audit log | None |

## Playbook Templates

### Credential Stuffing Playbook

```yaml
playbook:
  name: "credential_stuffing"
  trigger:
    condition: "failed_logins > 20 in 5min per user OR failed_logins > 100 in 10min per IP"
    severity: "high"
  steps:
    - name: "lock_account"
      action: "account_lockout"
      duration: "30min"
      condition: "per_user_trigger"
    - name: "block_ip"
      action: "ip_block"
      duration: "1h"
      condition: "per_ip_trigger"
    - name: "notify_user"
      action: "user_notification"
      template: "credential_stuffing_alert"
    - name: "notify_security"
      action: "admin_notification"
      channel: "slack:#security-alerts"
    - name: "siem_forward"
      action: "siem_alert"
      severity: "high"
    - name: "require_password_reset"
      action: "force_password_reset"
      on_unlock: true
```

### MFA Fatigue Playbook

```yaml
playbook:
  name: "mfa_fatigue"
  trigger:
    condition: "mfa_pushes > 5 in 10min per user"
    severity: "high"
  steps:
    - name: "stop_pushes"
      action: "block_mfa_push"
      duration: "30min"
    - name: "switch_method"
      action: "switch_mfa_method"
      alternative: "totp"
    - name: "notify_user"
      action: "user_notification"
      template: "mfa_fatigue_warning"
    - name: "notify_security"
      action: "admin_notification"
      channel: "slack:#security-alerts"
    - name: "check_account"
      action: "investigate"
      check: "password_recently_used"
```

### Data Exfiltration Playbook

```yaml
playbook:
  name: "data_exfiltration"
  trigger:
    condition: "records_exported > 10000 in 1h"
    severity: "critical"
  steps:
    - name: "revoke_sessions"
      action: "session_revoke"
      target: "user"
    - name: "lock_account"
      action: "account_lockout"
      duration: "indefinite"
    - name: "quarantine"
      action: "quarantine_user"
    - name: "escalate"
      action: "escalate"
      to: "ciso"
    - name: "preserve_evidence"
      action: "preserve_audit_trail"
      duration: "legal_hold"
    - name: "siem_forward"
      action: "siem_alert"
      severity: "critical"
    - name: "create_incident"
      action: "soar_create_incident"
      severity: "critical"
```

### Token Reuse Playbook

```yaml
playbook:
  name: "token_reuse"
  trigger:
    condition: "refresh_token_reuse_detected"
    severity: "critical"
  steps:
    - name: "revoke_family"
      action: "revoke_token_family"
    - name: "revoke_sessions"
      action: "session_revoke"
      target: "user"
    - name: "require_reauth"
      action: "force_reauthentication"
    - name: "notify_security"
      action: "admin_notification"
      channel: "pagerduty:security-critical"
    - name: "investigate"
      action: "investigate"
      check: "ip_geolocation"
      check: "device_fingerprint"
```

## Post-Incident Review Template

```markdown
## Post-Incident Review: [Incident ID]

### Summary
- **Date**: [date]
- **Duration**: [time from detection to resolution]
- **Severity**: [Critical/High/Medium/Low]
- **Detected by**: [automated/manual]

### Timeline
| Time | Event | Actor |
|---|---|---|
| 10:00 | Detection trigger fired | Automated |
| 10:01 | Account locked | Automated |
| 10:05 | Security team notified | Automated |
| 10:15 | Investigation started | Security analyst |
| 10:45 | Root cause identified | Security analyst |
| 11:00 | Remediation applied | Engineering |
| 11:30 | Incident resolved | Security team |

### Root Cause
[What caused the incident]

### Impact
- **Users affected**: [number]
- **Data exposed**: [description]
- **Service disruption**: [duration]

### What Went Well
- [Automated detection worked]
- [Response was fast]

### What Went Wrong
- [Detection was delayed]
- [Notification didn't reach on-call]

### Action Items
| # | Action | Owner | Due Date | Status |
|---|---|---|---|---|
| 1 | Update detection rule | Security | 2026-07-20 | Open |
| 2 | Add new playbook | Security | 2026-07-25 | Open |
| 3 | Fix root cause | Engineering | 2026-08-01 | Open |

### Lessons Learned
- [Key takeaway 1]
- [Key takeaway 2]
```

## GGID Automated Response

### Configuration

```yaml
incident_response:
  enabled: true
  detection:
    real_time: true
    evaluation_interval: 10s
  response:
    auto_lock: true
    auto_block_ip: true
    auto_revoke_sessions: true
    auto_step_up: true
  notification:
    slack: "#security-alerts"
    email: "security-team@example.com"
    pagerduty: "security-critical"
  soar:
    enabled: true
    auto_create_incident: true
  playbooks:
    - "credential_stuffing"
    - "mfa_fatigue"
    - "data_exfiltration"
    - "token_reuse"
  post_incident:
    auto_review: true
    review_template: "standard"
    review_within: 48h
```

## Best Practices

1. **Automate detection** — Don't rely on manual monitoring
2. **Auto-respond for critical** — Lock first, investigate second
3. **Notify in real-time** — Use PagerDuty for critical, Slack for high
4. **Create playbooks** — Predefined responses for common scenarios
5. **Test playbooks** — Run tabletop exercises quarterly
6. **Preserve evidence** — Legal hold on audit trail for incidents
7. **Conduct post-incident reviews** — Learn from every incident
8. **Track action items** — Ensure fixes are implemented
9. **Integrate with SOAR** — Orchestrate across tools
10. **Document everything** — Full timeline of detection and response