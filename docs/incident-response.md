# Incident Response

> Security incident response runbook for GGID deployments: detection, containment, investigation, eradication, recovery, and post-incident analysis.

---

## Table of Contents

1. [Incident Severity Levels](#incident-severity-levels)
2. [Detection Sources](#detection-sources)
3. [Response Procedure](#response-procedure)
4. [Common Incidents and Playbooks](#common-incidents-and-playbooks)
5. [Forensic Tools](#forensic-tools)
6. [Communication Plan](#communication-plan)
7. [Post-Incident Review](#post-incident-review)

---

## Incident Severity Levels

| Level | Description | Examples | Response Time |
|-------|-------------|----------|---------------|
| **SEV-1** | Critical — active breach | Account takeover in progress, data exfiltration detected | Immediate (< 15 min) |
| **SEV-2** | High — confirmed vulnerability | Credential leak, privilege escalation discovered | < 1 hour |
| **SEV-3** | Medium — potential risk | Brute force detected, anomalous access pattern | < 4 hours |
| **SEV-4** | Low — security hygiene | Expired certificate, missing patch | < 24 hours |

---

## Detection Sources

### Internal Monitoring

| Source | What It Detects | Alert Method |
|--------|----------------|-------------|
| Rate limiter (429 responses) | Brute force, credential stuffing | Prometheus alert |
| JTI anti-replay | Token reuse / replay attack | Redis event + log |
| RLS policy violation | Cross-tenant access attempt | PostgreSQL error log |
| Admin scope check failure | Unauthorized admin access attempt | Gateway audit log |
| Circuit breaker trips | Backend service failure | Prometheus alert |
| Login failure spikes | Account takeover attempts | Anomaly detection (planned) |

### External Sources

| Source | What It Detects |
|--------|----------------|
| SIEM integration (webhooks) | Correlated attack patterns |
| Vulnerability scanners | Known CVEs in dependencies |
| Bug bounty reports | Application-specific vulnerabilities |
| Customer reports | Suspicious activity on their tenant |

### Setting Up SIEM Alerts

```bash
# Register webhook for security events
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer <admin-JWT>" \
  -d '{
    "url": "https://siem.example.com/ggid-events",
    "events": [
      "auth.login_failed",
      "auth.account_locked",
      "auth.token_refreshed"
    ],
    "description": "SIEM security event integration"
  }'
```

---

## Response Procedure

### SEV-1: Active Breach

```
Step 1: CONTAIN (immediate, < 5 min)
├── Revoke compromised user sessions
│   DELETE /api/v1/sessions?user_id={user_id}
├── Suspend compromised user accounts
│   POST /api/v1/users/{user_id}/suspend
├── If tenant-wide: suspend entire tenant
│   POST /api/v1/tenants/{tenant_id}/suspend
└── Block attacker IP (if known)
    Add to IP blocklist

Step 2: INVESTIGATE (< 30 min)
├── Query audit trail for compromised user
│   GET /api/v1/audit/events?user_id={user_id}
├── Check all sessions for the user
├── Review recent role changes
├── Check for data export requests
└── Identify blast radius (what data was accessed?)

Step 3: ERADICATE
├── Patch the vulnerability
├── Rotate all secrets (JWT, DB, Redis)
├── Force password reset for affected users
└── Remove attacker's persisted access

Step 4: RECOVER
├── Verify systems are clean
├── Restore from backup if data was modified
├── Re-enable services
└── Monitor for re-intrusion

Step 5: POST-INCIDENT REVIEW (< 48 hours)
├── Document timeline
├── Root cause analysis
├── Update security architecture
├── Update this runbook
└── Notify affected users
```

---

## Common Incidents and Playbooks

### Playbook 1: Account Takeover

**Detection**: >5 failed logins followed by successful login from new IP.

**Response**:
```bash
# 1. Check if account is compromised
curl ".../api/v1/audit/events?user_id=usr_abc&event_type=auth.login" \
  -H "Authorization: Bearer <admin-JWT>"

# 2. Revoke all sessions
curl -X DELETE ".../api/v1/sessions?user_id=usr_abc" \
  -H "Authorization: Bearer <admin-JWT>"

# 3. Lock account
curl -X POST ".../api/v1/users/usr_abc/lock" \
  -H "Authorization: Bearer <admin-JWT>" \
  -d '{"reason": "Suspected account takeover"}'

# 4. Force password reset
curl -X POST ".../api/v1/auth/password/reset" \
  -d '{"username": "compromised_user"}'

# 5. Check for unauthorized changes
curl ".../api/v1/audit/events?user_id=usr_abc&status_code=200&from=2025-07-11T00:00:00Z" \
  -H "Authorization: Bearer <admin-JWT>"
```

### Playbook 2: Credential Stuffing Attack

**Detection**: Spike in failed logins across many usernames from same IP.

**Response**:
```bash
# 1. Check rate limiter status
curl ".../api/v1/audit/events?event_type=auth.login_failed&from=2025-07-11T10:00:00Z" \
  -H "Authorization: Bearer <admin-JWT>"

# 2. Rate limiter should auto-block (429 responses)
# Verify it's working:
curl ".../api/v1/audit/events?status_code=429&from=2025-07-11T10:00:00Z"

# 3. If attack persists, add IP to blocklist
curl -X POST ".../api/v1/security/ip-blocklist" \
  -d '{"ip": "192.168.1.100", "reason": "Credential stuffing attack"}'

# 4. Force password reset for any successfully compromised accounts
# (Check audit for successful logins from attacker IP)
```

### Playbook 3: Privilege Escalation

**Detection**: User suddenly has admin scope or new role assignment they shouldn't have.

**Response**:
```bash
# 1. Check recent role assignments
curl ".../api/v1/audit/events?event_type=user.role_assigned&from=2025-07-10T00:00:00Z" \
  -H "Authorization: Bearer <admin-JWT>"

# 2. Revoke unauthorized role
curl -X DELETE ".../api/v1/users/usr_abc/roles/role_admin" \
  -H "Authorization: Bearer <admin-JWT>"

# 3. Revoke all sessions (old JWT may still have admin scope)
curl -X DELETE ".../api/v1/sessions?user_id=usr_abc" \
  -H "Authorization: Bearer <admin-JWT>"

# 4. Check what the user accessed with elevated privileges
curl ".../api/v1/audit/events?user_id=usr_abc&from=2025-07-10T00:00:00Z" \
  -H "Authorization: Bearer <admin-JWT>"
```

### Playbook 4: JWT Secret Compromise

**Detection**: Unauthorized tokens are valid, or secret was leaked.

**Response**:
```bash
# 1. Rotate JWT secret immediately
# Update JWT_SECRET environment variable on ALL services
# Restart all services

# 2. All existing tokens become invalid
# Users must re-authenticate

# 3. Clear Redis session store
redis-cli FLUSHDB  # (nuclear option — clears everything)

# 4. Check audit trail for unauthorized token usage
curl ".../api/v1/audit/events?from=2025-07-10T00:00:00Z" \
  -H "Authorization: Bearer <new-JWT>"
```

### Playbook 5: Database Breach

**Detection**: Unauthorized database access, data exfiltration signs.

**Response**:
```bash
# 1. Isolate database (block all connections except from app servers)
# 2. Rotate database credentials
# 3. Check for data modification
psql -c "SELECT count(*) FROM users WHERE updated_at > '2025-07-10';"
# 4. Restore from clean backup if data was modified
# 5. Check PostgreSQL logs for unauthorized queries
tail -f /var/log/postgresql/*.log
```

---

## Forensic Tools

### Audit Trail Query

The audit trail is the primary forensic tool:

```bash
# All events for a user (timeline)
curl ".../api/v1/audit/events?user_id=usr_abc&sort=timestamp" \
  -H "Authorization: Bearer <admin-JWT>"

# All events from an IP
curl ".../api/v1/audit/events?client_ip=192.168.1.50" \
  -H "Authorization: Bearer <admin-JWT>"

# Failed operations only
curl ".../api/v1/audit/events?status_code=403" \
  -H "Authorization: Bearer <admin-JWT>"

# Export for forensic analysis
curl ".../api/v1/audit/events?from=2025-07-10&to=2025-07-11&format=csv" \
  -H "Authorization: Bearer <admin-JWT>" \
  -o forensic_export.csv
```

### Redis Investigation

```bash
# Check active sessions
redis-cli keys "session:*" | wc -l

# Check rate limiter state
redis-cli keys "rate_limit:*"

# Check JTI anti-replay
redis-cli keys "jti:*" | wc -l

# Check revoked sessions
redis-cli keys "revoked:*"
```

### PostgreSQL Investigation

```sql
-- Check RLS policy violations
SELECT * FROM pg_stat_activity WHERE state = 'active';

-- Check recent user changes
SELECT id, username, updated_at FROM users
WHERE updated_at > NOW() - INTERVAL '24 hours'
ORDER BY updated_at DESC;

-- Check role assignments
SELECT user_id, role_id, assigned_at
FROM user_roles
WHERE assigned_at > NOW() - INTERVAL '7 days'
ORDER BY assigned_at DESC;
```

---

## Communication Plan

### Internal

| Severity | Notify | Channel | Timing |
|----------|--------|---------|--------|
| SEV-1 | Security team, CTO, CEO | Phone + Slack #incidents | Immediate |
| SEV-2 | Security team, engineering lead | Slack #incidents | < 1 hour |
| SEV-3 | Security team | Slack #security | < 4 hours |
| SEV-4 | Security team | Email digest | Weekly |

### External (Customer Notification)

| Severity | Notify | Timing | Method |
|----------|--------|--------|--------|
| SEV-1 | Affected customers | < 72 hours (GDPR requirement) | Email + status page |
| SEV-2 | Affected customers | < 7 days | Email |
| SEV-3 | Internal only | N/A | N/A |
| SEV-4 | Internal only | N/A | N/A |

### Regulatory Notification

- **GDPR Article 33**: Notify supervisory authority within 72 hours for personal data breaches
- **CCPA**: Notify affected California residents "without unreasonable delay"
- **HIPAA**: Notify HHS within 60 days for breaches affecting 500+ individuals

---

## Post-Incident Review

### Template

```markdown
## Incident Report: [INC-YYYY-NNN]

### Summary
- Date: [date/time]
- Severity: [SEV-1/2/3/4]
- Duration: [time to resolve]
- Impact: [users/data/systems affected]

### Timeline
| Time | Event |
|------|-------|
| T+0 | Detection |
| T+5m | Containment |
| T+30m | Investigation start |
| T+2h | Root cause identified |
| T+4h | Eradication complete |
| T+6h | Recovery complete |

### Root Cause
[What happened and why]

### What Went Well
- [Quick detection via...]
- [Effective containment using...]

### What Went Wrong
- [Delayed alert because...]
- [Missing monitoring for...]

### Action Items
- [ ] [Action] — Owner: [name] — Due: [date]
- [ ] [Action] — Owner: [name] — Due: [date]

### Lessons Learned
[Key takeaways for improving security posture]
```

---

*Last updated: 2025-07-11*
