# ITDR Implementation Guide

## Overview

Identity Threat Detection and Response (ITDR) extends GGID's IAM capabilities to detect, investigate, and respond to identity-based attacks. This guide covers architecture, detection rules, and response playbooks.

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                   ITDR Engine                         │
│                                                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │
│  │ Signal   │→ │ Risk     │→ │ Response         │   │
│  │ Collector│  │ Scorer   │  │ Orchestrator     │   │
│  └──────────┘  └──────────┘  └──────────────────┘   │
│       ↑              ↑              ↓                 │
│  ┌────┴────┐   ┌─────┴─────┐  ┌────┴────────────┐   │
│  │ Auth    │   │ Detection │  │ Action Executor  │   │
│  │ Events  │   │ Rules     │  │ (block/revoke)   │   │
│  └─────────┘   └───────────┘  └──────────────────┘   │
│  ┌─────────┐   ┌───────────┐  ┌──────────────────┐   │
│  │ Audit   │   │ MITRE     │  │ Notification     │   │
│  │ Events  │   │ ATT&CK    │  │ (webhook/SIEM)   │   │
│  └─────────┘   └───────────┘  └──────────────────┘   │
│  ┌─────────┐                                          │
│  │ Session │                                          │
│  │ Events  │                                          │
│  └─────────┘                                          │
└──────────────────────────────────────────────────────┘
```

## Detection Rules Catalog

### 1. Brute Force Detection
```yaml
rule_id: ITDR-001
name: Brute Force Attack
severity: high
mitre_technique: T1110
signals:
  - source: auth_events
    condition: failed_login_count >= 10
    window: 5m
    group_by: [ip_address, user_id]
actions:
  - block_ip (duration: 1h)
  - lock_account (duration: 30m)
  - notify (channel: security-team)
```

### 2. Credential Stuffing
```yaml
rule_id: ITDR-002
name: Credential Stuffing
severity: high
mitre_technique: T1110.004
signals:
  - source: auth_events
    condition: distinct_userids_failed >= 20
    window: 10m
    group_by: [ip_address]
  - source: auth_events
    condition: success_ratio < 0.05
actions:
  - block_ip (duration: 24h)
  - add_to_watchlist (entity: ip)
  - alert (severity: critical)
```

### 3. Lateral Movement Detection
```yaml
rule_id: ITDR-003
name: Impossible Travel
severity: medium
mitre_technique: T1021
signals:
  - source: session_events
    condition: geo_distance > 1000km
    window: 2h
    comparison: consecutive_logins
actions:
  - require_mfa (user_id)
  - flag_session (risk_level: elevated)
  - log_anomaly
```

### 4. Privilege Escalation
```yaml
rule_id: ITDR-004
name: Suspicious Role Assignment
severity: critical
mitre_technique: T1098
signals:
  - source: audit_events
    condition: role_change == "escalation"
    filters:
      - outside_business_hours: true
      - or self_grant: true
      - or skip_approval: true
actions:
  - hold_change (review_required: true)
  - notify (channel: security-team, priority: urgent)
  - create_ticket (type: security-review)
```

### 5. Golden Ticket Detection
```yaml
rule_id: ITDR-005
name: Forged Token Detection
severity: critical
mitre_technique: T1098.004
signals:
  - source: auth_events
    condition: jwt_iss_mismatch == true
  - source: auth_events
    condition: jwt_kid_unknown == true
  - source: audit_events
    condition: admin_scope_without_chain == true
actions:
  - revoke_token (immediate: true)
  - block_user (pending_investigation: true)
  - page_oncall (severity: critical)
```

### 6. Session Hijacking
```yaml
rule_id: ITDR-006
name: Session Anomaly
severity: high
mitre_technique: T1185
signals:
  - source: session_events
    condition: device_fingerprint_changed == true
    window: 1h
  - source: session_events
    condition: user_agent_changed == true
    window: 1h
  - source: session_events
    condition: ip_asn_changed == true
    window: 1h
actions:
  - force_reauth (user_id)
  - invalidate_other_sessions (user_id)
  - flag_session (risk_level: high)
```

### 7. Account Takeover
```yaml
rule_id: ITDR-007
name: Account Takeover Indicators
severity: critical
mitre_technique: T1078
signals:
  - combo_rule:
      any_of:
        - password_change_followed_by_email_change (window: 1h)
        - mfa_disabled_followed_by_login (window: 30m)
        - new_device_followed_by_financial_action (window: 24h)
      min_matches: 1
actions:
  - freeze_account (pending_verification: true)
  - send_recovery_email
  - create_incident (type: account-takeover)
```

## MITRE ATT&CK Mapping

| Technique | ID | ITDR Detection |
|-----------|----|----------------|
| Brute Force | T1110 | ITDR-001, ITDR-002 |
| Lateral Movement | T1021 | ITDR-003 |
| Account Manipulation | T1098 | ITDR-004, ITDR-005 |
| Session Hijacking | T1185 | ITDR-006 |
| Valid Accounts | T1078 | ITDR-007 |
| OS Credential Dumping | T1003 | (via SIEM integration) |
| Kerberoasting | T1558 | (via SIEM integration) |

## Response Playbooks

### Automated Response Matrix

| Risk Score | Action | Duration |
|------------|--------|----------|
| Low (0-30) | Log + monitor | Indefinite |
| Medium (31-60) | Require MFA re-challenge | Current session |
| High (61-85) | Block source IP + flag session | 1-24h |
| Critical (86-100) | Freeze account + revoke all tokens | Pending investigation |

### Manual Investigation Workflow

1. **Alert received** → Analyst opens ITDR dashboard
2. **Correlate signals** → Review user activity, device history, geo patterns
3. **Assess impact** → What data/resources were accessed?
4. **Containment** → Revoke sessions, reset credentials, block IPs
5. **Eradication** → Remove malicious OAuth grants, revoke agent tokens
6. **Recovery** → User identity verification, credential reset, monitoring period
7. **Lessons learned** → Update detection rules, adjust thresholds

## Integration Points

### SIEM Integration
- Forward all ITDR events to SIEM (Splunk/Elastic/Datadog)
- Configured via `/api/v1/audit/siem/forwarder-config`
- Batch delivery with retry and circuit breaker

### Webhook Notifications
- Real-time alerts to security team webhooks
- Configured via tenant settings
- Includes full event context + MITRE mapping

### API Access
```http
GET /api/v1/identity/itdr/threats?status=active&severity=critical
GET /api/v1/identity/itdr/detection-rules
PUT /api/v1/identity/itdr/detection-rules/{id}
POST /api/v1/identity/itdr/threats/{id}/respond
```

## Best Practices

1. **Tune thresholds**: Start conservative, adjust based on false positive rate
2. **Layer detection**: Use multiple signal types (behavioral, contextual, technical)
3. **Automate response**: Critical threats should auto-contain without human delay
4. **Preserve evidence**: All ITDR events retained in audit log with hash chain
5. **Test regularly**: Run red team exercises against detection rules
6. **Integrate broadly**: Connect with EDR, NDR, and SIEM for full coverage
