# ITDR Implementation Guide

Identity Threat Detection and Response (ITDR) implementation guide for GGID — detection rules, response playbooks, MITRE ATT&CK mapping, and SIEM integration.

## Overview

ITDR focuses on detecting and responding to identity-based attacks: credential stuffing, pass-the-hash, golden ticket, anomalous access patterns, and privilege escalation.

## MITRE ATT&CK Mapping

| Technique | ID | Detection Rule | GGID Control |
|-----------|-----|---------------|--------------|
| Credential Stuffing | T1110.004 | Failed logins > 10 from same IP in 5m | Rate limiting + lockout |
| Brute Force | T1110.001 | Failed logins > 20 for same user | Account lockout |
| Pass the Hash | T1550.002 | JWT from new IP after old IP session | jti anti-replay |
| Golden Ticket | T1558.001 | JWT with forged claims | RS256 signature verification |
| Kerberoasting | T1558.003 | Service account token anomaly | Agent identity monitoring |
| Token Impersonation | T1134.001 | Unexpected token delegation | Delegation depth enforcement |
| Account Manipulation | T1098 | Role assigned outside business hours | Alert + approval workflow |
| Root Account Use | T1078.004 | Admin login from new geo | Step-up MFA |

## Detection Rules

### Rule Configuration

```bash
curl -X POST https://api.ggid.example.com/api/v1/audit/alerts/rules \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "Credential Stuffing",
    "description": "Multiple failed logins from same IP",
    "condition": {
      "event_type": "user.login_failed",
      "aggregation": { "field": "ip", "count_gt": 10, "window": "5m" }
    },
    "severity": "high",
    "mitre_attack": "T1110.004",
    "response": ["block_ip", "alert_soc"]
  }'
```

### Recommended Detection Rules

| Rule Name | Condition | Severity | MITRE | Response |
|-----------|-----------|----------|-------|----------|
| Credential stuffing | 10+ failed logins / IP / 5m | High | T1110.004 | Block IP, alert |
| Brute force | 20+ failed logins / user / 10m | High | T1110.001 | Lock account |
| Impossible travel | Login from 2 countries < 2h apart | Critical | T1027 | Step-up MFA, alert |
| New device admin | Admin login from new device | Medium | T1078 | Step-up WebAuthn |
| Off-hours privilege change | Role assign 22:00-06:00 | Medium | T1098 | Alert, require approval |
| Mass export | 5+ audit exports in 1h | Medium | T1048 | Alert SOC |
| Token reuse | jti reuse detected | Critical | T1550 | Revoke all sessions |
| Agent depth exceeded | Delegation > max_depth | High | T1134 | Deny + suspend agent |
| Dormant account login | Inactive > 90d then login | Medium | T1078 | Step-up MFA |
| VPN/Tor login | Login from known VPN/Tor exit | Medium | T1090 | Step-up MFA |

## Response Playbooks

### Playbook: Credential Stuffing Detected

```
Trigger: Credential stuffing rule fires
  ↓
1. Auto-block source IP (15 min)
2. Lock targeted accounts
3. Send alert to SOC (email + Slack)
4. Create incident ticket
5. If pattern persists → escalate to block IP permanently
6. Record in audit log with MITRE tag
```

### Playbook: Impossible Travel

```
Trigger: Login from EU, then US within 2 hours
  ↓
1. Require step-up WebAuthn
2. If WebAuthn fails → deny login
3. Revoke previous sessions
4. Alert user via email
5. Alert SOC for investigation
```

### Playbook: Privilege Escalation

```
Trigger: Role assigned outside normal pattern
  ↓
1. Require manager approval (access request workflow)
2. Log with full context (assigner, target, role, time, IP)
3. If auto-approved → alert for review
4. SoD check → deny if conflict
```

## SIEM Integration

Forward all identity events to SIEM for correlation:

```yaml
siem:
  provider: splunk
  endpoint: https://splunk.example.com:8088/services/collector
  api_key: hec-token
  index: ggid-itdr
  events:
    - user.login
    - user.login_failed
    - role.assigned
    - role.revoked
    - agent.token_exchanged
    - security.suspicious_activity
```

Events are forwarded in CEF format with MITRE ATT&CK technique IDs:

```
CEF:0|GGID|ITDR|1.0|100|Credential Stuffing|9|
act=login_failed suser=unknown src=1.2.3.4
rt=Jan 24 2025 14:30:00 UTC
cs1Label=MITRE cs1=T1110.004
```

## ITDR Dashboard

Key metrics for the ITDR dashboard:

| Metric | Description |
|--------|-------------|
| Threats detected (24h) | Total alerts triggered |
| Top attack types | MITRE technique distribution |
| Blocked IPs | Auto-blocked source IPs |
| Locked accounts | Accounts locked by rules |
| MTTR | Mean time to respond |
| False positive rate | Alerts dismissed / total |

## Implementation Checklist

- [ ] All 10 detection rules configured
- [ ] SIEM forwarder operational
- [ ] Response playbooks documented
- [ ] SOC alert routing tested
- [ ] MITRE ATT&CK tags on all rules
- [ ] ITDR dashboard live
- [ ] Monthly rule tuning scheduled
- [ ] Tabletop exercise conducted

## See Also

- [Audit & SIEM Guide](audit-siem-guide.md)
- [Security Audit Checklist](security-audit-checklist.md)
- [Rate Limiting](rate-limiting-guide.md)
- [Fraud Detection](fraud-detection.md)
