# Identity Threat Detection and Response (ITDR)

Overview, detection rules, MITRE ATT&CK mapping, response playbooks, and SIEM integration.

## Overview

ITDR is the continuous process of detecting, investigating, and responding to identity-based threats. GGID provides the telemetry, rules engine, and response automation for identity-centric security.

```
Signals → Detection Rules → Alerts → Triage → Response → Recovery
  ↑                                                    │
  └────────────── Lessons Learned ────────────────────┘
```

## Detection Rules

### Credential Attacks

| Rule | Signal | Threshold | Action |
|------|--------|-----------|--------|
| Brute force | Failed logins per IP | >10/min | Rate limit + block IP |
| Password spraying | Failed logins per user spread across IPs | >5 users/IP/min | Lock + alert |
| Credential stuffing | Multiple login attempts with known breached passwords | HIBP match | Force password reset |
| MFA bombing | Repeated MFA push requests | >5/min/user | Freeze MFA + alert |

### Session Threats

| Rule | Signal | Threshold | Action |
|------|--------|-----------|--------|
| Impossible travel | Geo-IP change faster than flight time | <2h across >5000km | Force re-auth |
| Session hijacking | IP/UA change mid-session | >30% fingerprint delta | Invalidate session |
| Token replay | Same JWT from new IP after revocation | Any | Alert + revoke all |
| Concurrent session abuse | Multiple active sessions from distant IPs | >3 regions | Evict + investigate |

### Privilege Escalation

| Rule | Signal | Threshold | Action |
|------|--------|-----------|--------|
| Role escalation anomaly | User gains admin scope without JIT | Any | Alert + revert |
| Delegation chain abuse | Delegation depth >3 | Any | Block + audit |
| Break-glass misuse | Break-glass for routine ops | Any non-incident | Alert security team |
| Dormant admin activation | Admin account dormant >90 days then active | Any | Require re-approval |

### Account Takeover

| Rule | Signal | Threshold | Action |
|------|--------|-----------|--------|
| Profile change after login | Email/password change within 1h of login | Any | Step-up MFA required |
| New device enrollment spike | >3 new WebAuthn devices in 1h | Any user | Freeze enrollment |
| API key mass creation | >10 keys in 1h | Any user | Rate limit + alert |
| Data exfiltration | Bulk export after identity change | Any | Block + alert |

## MITRE ATT&CK Mapping

| MITRE Technique | GGID Detection | GGID Mitigation |
|-----------------|---------------|-----------------|
| T1078 Valid Accounts | Impossible travel, session anomaly | Conditional access, session binding |
| T1110 Brute Force | Login rate analysis | Rate limiting, account lockout |
| T1621 Multi-Factor Request Generation | MFA bombing detection | MFA fatigue protection |
| T1556 Modify Auth Process | Config change monitoring | Admin scope enforcement |
| T1098 Account Manipulation | Profile change rules | Step-up on changes |
| T1539 Steal Web Session Cookie | Token replay detection | Token binding (DPoP/mTLS) |
| T1528 Steal App Access Token | Token anomaly detection | Short TTL, audience restriction |
| T1606 Forge Web Credentials | JWT algorithm confusion blocked | RS256/ES256 only, JWKS pinning |

## Response Playbooks

### Playbook: Credential Stuffing Detected

```
1. AUTO: Block source IP (15 min)
2. AUTO: Notify affected users (email: "unusual login activity")
3. AUTO: Force password reset for accounts with >3 failures
4. AUTO: Log to SIEM + create incident ticket
5. MANUAL: Security team reviews — extend block if persistent
```

### Playbook: Account Takeover Suspected

```
1. AUTO: Revoke all sessions + tokens for affected user
2. AUTO: Suspend account (status: "locked")
3. AUTO: Freeze MFA enrollment (no new devices)
4. AUTO: Alert user via alternate channel (SMS if email compromised)
5. AUTO: Page security on-call
6. MANUAL: Identity verification → account recovery → forensic audit
```

### Playbook: Privilege Escalation Detected

```
1. AUTO: Revert unauthorized role assignment
2. AUTO: Revoke elevated scopes immediately
3. AUTO: Freeze delegation for affected user
4. AUTO: Full audit log export for investigation
5. MANUAL: Determine if compromise or misconfiguration
```

## SIEM Integration

### Event Forwarding

GGID forwards identity events to SIEM via:

```bash
# Configure SIEM forwarder
POST /api/v1/admin/siem/config
{
  "endpoint": "https://siem.corp.com/events",
  "format": "cef",  // or json, leef
  "events": ["auth.login", "auth.failure", "session.revoke",
             "role.change", "policy.violation", "threat.detected"],
  "tls": {"verify": true, "ca_cert": "..."}
}
```

### CEF Event Format

```
CEF:0|GGID|IAM|2.0|1001|Login Failure|7|src=10.0.1.5 suser=user@corp.com act=login_failure msg=Brute force detected rt=Jan 15 10:30:00
```

### SIEM Health

```bash
GET /api/v1/admin/siem/health
# → {"status":"connected","events_per_min":142,"last_delivery":"2s ago","queue_depth":0}
```

## Risk Scoring

Each identity event contributes to a real-time risk score:

```python
risk_score = base_score
    + credential_attack_weight * recent_failures
    + session_anomaly_weight * impossible_travel
    + privilege_escalation_weight * scope_change
    + data_exfiltration_weight * bulk_export

# 0-20: Low (allow)
# 21-50: Medium (require MFA step-up)
# 51-80: High (require WebAuthn)
# 81-100: Critical (deny + alert)
```

## Monitoring Dashboard

| Widget | Data |
|--------|------|
| Active threats | Count of open incidents |
| Attack trends | Login failures over time |
| Top risky users | Users with highest risk scores |
| MITRE coverage | % of identity techniques with detection |
| Response time | Mean time to detect (MTTD), mean time to respond (MTTR) |

## See Also

- [Conditional Access](conditional-access.md)
- [Audit API](../api/audit-api.md)
- Fraud Detection
- Threat Modeling
- [Competitive Analysis](competitive-analysis.md)
