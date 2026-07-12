# Break Glass (Emergency Access) Procedure

This guide covers configuring and using emergency access (break glass) procedures in GGID.

## Overview

Break glass accounts provide emergency administrative access when normal authentication is unavailable (IdP outage, MFA device lost, account compromise). They must be tightly controlled, monitored, and reviewed.

## Break Glass Roles

### Emergency Admin

A pre-provisioned role with full access, activated only during emergencies:

```bash
# Create break glass role
curl -X POST https://api.ggid.example.com/api/v1/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "key": "break-glass-admin",
    "name": "Break Glass Administrator",
    "description": "Emergency access only — requires post-incident review",
    "permissions": ["*"]
  }'
```
### Break Glass Accounts

Create dedicated accounts (not personal accounts):

- `breakglass-01` — Primary emergency account
- `breakglass-02` — Secondary (in case 01 is compromised)
- Store credentials in: sealed envelope in safe + secrets manager
- No MFA (or MFA with backup codes stored separately)

## Activation Procedure

### 1. Justification Required

Every break glass activation must include:

```bash
curl -X POST https://api.ggid.example.com/api/v1/auth/break-glass/activate \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "username": "breakglass-01",
    "password": "emergency-password",
    "justification": "Production IdP outage — need to restore user access",
    "incident_id": "INC-2025-0142",
    "requested_by": "on-call-engineer",
    "approver": "security-oncall"
  }'
```

### 2. Approval Workflow

| Scenario | Approval Required |
|----------|-----------------|
| During incident (page) | Verbal from security on-call |
| Post-incident activation | Written (ticket) within 24h |
| Unplanned (no approval) | Auto-escalate to CISO |

### 3. Time-Limited Session

Break glass sessions are automatically time-limited:

```yaml
break_glass:
  max_session_minutes: 60      # 1 hour max
  cooldown_hours: 24           # Can't reuse for 24h after
  require_justification: true
  auto_revoke_on_expiry: true
```

## Monitoring

### Real-Time Alerts

Every break glass activation triggers immediate alerts:

```bash
curl -X POST https://api.ggid.example.com/api/v1/audit/alerts/rules \
  -d '{
    "name": "Break glass activation",
    "condition": "event_type=break_glass.activate",
    "severity": "critical",
    "action": "webhook",
    "recipients": ["security@company.com"],
    "webhook_url": "https://hooks.slack.com/services/xxx"
  }'
```

### Session Recording

All break glass sessions are fully logged:
- Every API call recorded
- Source IP and user agent
- Commands executed
- Data accessed
- Duration

## Cooldown Period

After a break glass session ends, a 24-hour cooldown prevents immediate re-use:

```
Break glass activated at 14:00
Session ends at 14:45
Cooldown: 14:45 → next day 14:45
Next activation requires: CISO approval
```

## Post-Incident Review

### Within 48 Hours

1. **Document**: Timeline, justification, actions taken
2. **Review**: Security team reviews all actions
3. **Verify**: No unauthorized changes
4. **Rotate**: Change break glass password
5. **Close incident**: Link incident report

### Template

```markdown
## Break Glass Review — INC-2025-0142

**Activated by**: Jane Doe (on-call engineer)
**Time**: 2025-01-24 14:00 — 14:45 (45 min)
**Justification**: Production IdP outage
**Actions taken**:
  1. Identified IdP database connection failure
  2. Restarted IdP service
  3. Verified user logins restored
**Data accessed**: User list (read-only)
**Changes made**: None
**Verdict**: [ ] Justified [ ] Unjustified
**Reviewed by**: ___________ Date: ______
```

## Audit Requirements

Break glass events are retained for **7 years** (SOX compliance):

```bash
# Query all break glass events
curl "https://api.ggid.example.com/api/v1/audit/events?actor_id=breakglass-01" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Security Checklist

- [ ] Break glass accounts created (2 minimum)
- [ ] Credentials stored in physical safe + secrets manager
- [ ] Activation requires justification + approval
- [ ] Sessions time-limited (max 60 min)
- [ ] Cooldown period enforced (24h)
- [ ] Real-time alerts configured
- [ ] All actions logged with full context
- [ ] Post-incident review within 48h
- [ ] Password rotated after each use
- [ ] Quarterly drill conducted
- [ ] Retention: 7 years (SOX)

## See Also

- [Security Audit Checklist](security-audit-checklist.md)
- [ITDR Implementation](itdr-implementation.md)
- [Compliance Guide](compliance-guide.md)
