# Identity Recovery Playbook

Account takeover response, mass credential reset, session invalidation, forensics, user communication, legal notification, and post-incident review.

## Incident Severity

| Severity | Scenario | Response Time | Escalation |
|----------|---------|--------------|------------|
| SEV-1 | Mass credential breach | Immediate | CISO + legal |
| SEV-2 | Single admin account takeover | <15 min | Security team |
| SEV-3 | Regular user compromise | <1 hour | IT support |
| SEV-4 | Suspicious activity (no confirmed compromise) | <4 hours | Monitor |

## SEV-1: Mass Credential Breach

### Step 1: Contain (T+0)

```bash
# Force password reset for ALL users
POST /api/v1/admin/incident/force-reset-all
{
  "incident_id": "INC-2025-0001",
  "reason": "Mass credential breach"
}
# → All passwords invalidated, users must reset on next login

# Revoke ALL sessions
POST /api/v1/admin/incident/revoke-all-sessions
# → Every active session terminated

# Suspend OAuth clients
POST /api/v1/admin/incident/suspend-oauth
# → No new token issuance (maintenance mode)
```

### Step 2: Investigate (T+5min)

```bash
# Export forensic evidence
POST /api/v1/audit/export
{
  "query": "result eq 'denied' || action eq 'user.login'",
  "from": "2025-01-14T00:00:00Z",
  "format": "jsonl",
  "include_chain": true
}
```

### Step 3: Remediate (T+30min)

```bash
# Rotate all signing keys
POST /api/v1/admin/keys/rotate-all
# → New JWT signing keys, old keys revoked after grace period

# Check for unauthorized admin accounts
GET /api/v1/admin/users?role=admin&created_after=2025-01-14
# → Verify no attacker-created admin accounts

# Check for unauthorized OAuth clients
GET /api/v1/oauth/clients?created_after=2025-01-14
```

### Step 4: Communicate (T+1h)

```
User email:
"Security Incident — Action Required

We detected unusual activity on our platform. As a precaution:
1. Your password has been reset. Reset it here: [link]
2. If you notice suspicious activity, contact security@corp.com
3. We recommend enabling multi-factor authentication.

We apologize for the inconvenience and are working to resolve this."
```

## SEV-2: Admin Account Takeover

### Step 1: Isolate

```bash
# Suspend compromised admin
PATCH /api/v1/admin/users/{user_id}
{"status": "locked", "reason": "ATO suspected"}

# Revoke all sessions for this user
DELETE /api/v1/admin/users/{user_id}/sessions

# Revoke all tokens
DELETE /api/v1/admin/users/{user_id}/tokens

# Freeze admin actions (prevent further damage)
POST /api/v1/admin/users/{user_id}/freeze-admin
```

### Step 2: Assess Damage

```bash
# What did the attacker do?
GET /api/v1/audit/events?user_id={user_id}&from=2025-01-15T08:00:00Z
# → Review all actions taken during suspected window

# Did they create new users?
GET /api/v1/admin/identity/users?created_by={user_id}

# Did they change policies?
GET /api/v1/audit/events?user_id={user_id}&action_prefix=policy.

# Did they assign roles?
GET /api/v1/audit/events?user_id={user_id}&action=role.assign
```

### Step 3: Reverse Unauthorized Changes

```bash
# Revert unauthorized role assignments
POST /api/v1/admin/incident/revert-actions
{
  "incident_id": "INC-2025-0002",
  "user_id": "compromised-admin",
  "timeframe": {"from": "2025-01-15T08:00:00Z", "to": "now"}
}
# → All actions by compromised admin are audited and reversed if unauthorized
```

## Session Invalidation

### Granular Invalidation

```bash
# Revoke specific user's sessions
DELETE /api/v1/auth/sessions?user_id={user_id}

# Revoke by IP (attacker's IP)
DELETE /api/v1/auth/sessions?ip=192.168.1.50

# Revoke by device
DELETE /api/v1/auth/sessions?device_fingerprint={fingerprint}

# Revoke by tenant (if tenant-wide incident)
DELETE /api/v1/auth/sessions?tenant_id={tenant_id}
```

### Token Blacklist

```go
func invalidateAllTokensForUser(userID string) {
    // Get all active JWT IDs for user
    jtis := store.GetActiveJTIs(userID)

    for _, jti := range jtis {
        // Add to Redis blacklist (expires when token would expire)
        redis.Set("jwt:blacklist:"+jti, "1", 15*time.Minute)
    }

    audit.Log("tokens_invalidated", map[string]interface{}{
        "user_id": userID,
        "count": len(jtis),
        "incident": "ATO",
    })
}
```

## Forensics

### Evidence Collection

```bash
# Freeze audit data (read-only)
ALTER DATABASE ggid SET default_transaction_read_only = on;

# Snapshot
pg_dump ggid > /forensics/$(date +%Y%m%d_%H%M%S).dump

# Verify hash chain integrity
GET /api/v1/audit/verify-chain
# → Critical: confirms attacker didn't tamper with logs

# Export attacker timeline
GET /api/v1/audit/events?ip={attacker_ip}&format=timeline
```

### Timeline Reconstruction

```
08:00 — Attacker logs in (credential from breach)
08:02 — Attacker views admin panel
08:05 — Attacker creates OAuth client "data-export"
08:08 — Attacker exports user data
08:12 — SOC alert triggered (new device + impossible travel)
08:13 — Auto-response: account locked, sessions revoked
08:15 — Incident response started
```

## User Communication

### Email Template (ATO)

```
Subject: Security Alert — Unauthorized Access to Your Account

We detected unauthorized access to your account on [date].

What happened:
- Someone accessed your account from [IP/location]
- We have locked the account and revoked all sessions

What we did:
- Your password has been reset
- All active sessions terminated
- MFA factors reviewed

What you should do:
1. Reset your password: [link]
2. Review your account for unauthorized changes
3. Enable MFA if not already enabled
4. Contact security@corp.com if you see anything suspicious

We take your security seriously and apologize for this incident.

— Security Team
```

## Legal Notification

| Jurisdiction | Requirement | Timeline |
|-------------|------------|----------|
| GDPR (EU) | Notify DPA | 72 hours |
| CCPA (CA) | Notify affected users | "Without unreasonable delay" |
| HIPAA (US) | Notify HHS + affected | 60 days |
| PIPL (CN) | Notify CAC + affected | "Without delay" |
| State breach laws | Varies by state | 30-90 days |

```bash
# Generate breach notification report
POST /api/v1/admin/compliance/breach-report
{
  "incident_id": "INC-2025-0001",
  "affected_data": ["email", "display_name", "phone"],
  "affected_users": 5000,
  "jurisdictions": ["EU", "US", "CN"],
  "detection_date": "2025-01-15",
  "notification_deadline": "2025-01-22"
}
```

## Post-Incident Review

### Template

```markdown
## Incident Review: INC-2025-0001

### Summary
- Date: 2025-01-15
- Severity: SEV-1
- Duration: 4 hours (detection to resolution)
- Impact: 5,000 users affected

### Timeline
[Detailed timeline]

### Root Cause
[Credential stuffing via breached passwords]

### What Went Well
- Automated detection triggered within 2 minutes
- Session revocation prevented further data access

### What Went Wrong
- Password breach check (HIBP) was disabled for maintenance
- Rate limiting was temporarily lowered

### Action Items
1. [ ] Re-enable HIBP check (P0, done)
2. [ ] Add alerting for HIBP check downtime
3. [ ] Enforce mandatory password rotation
4. [ ] Deploy WebAuthn as primary factor
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Incident response time | Track against SLA |
| Sessions invalidated | Log for evidence |
| Reverted actions | Track count + types |
| User complaints | Spike → communication issue |

## See Also

- [Identity Threat Detection](identity-threat-detection.md)
- [Audit Log Architecture](audit-log-architecture.md)
- [OAuth Refresh Token Rotation](oauth-refresh-token-rotation.md)
- [Compliance Framework Mapping](compliance-framework-mapping.md)
