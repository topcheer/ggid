# Incident Response Playbook

Severity classification (P0-P3), incident commander role, containment checklist, forensic evidence collection, communication templates, and post-incident review.

## Severity Classification

| Level | Definition | Response | Example |
|-------|-----------|---------|---------|
| P0 | Critical outage, data breach | Immediate, 24/7 | Auth down, credential leak |
| P1 | Major degradation | <15 min, 24/7 | Login failures >10% |
| P2 | Minor degradation | <1 hour | Console slow, partial API degraded |
| P3 | Nuisance, no user impact | Next business day | Intermittent warnings |

## Incident Commander (IC)

The IC coordinates — does NOT debug:

1. Declares severity
2. Opens war room (Slack #inc-NNNN + video bridge)
3. Assigns investigation tasks
4. Posts updates every 15-30 min
5. Makes rollback/failover decisions
6. Documents timeline

## Containment Checklist

### Immediate Actions

```bash
# Block attacker IP
POST /api/v1/admin/security/block-ip {"ip": "192.168.1.50"}

# Freeze affected accounts
PATCH /api/v1/admin/users/{id} {"status": "locked"}

# Revoke all sessions for affected users
DELETE /api/v1/admin/users/{id}/sessions

# Revoke tokens
DELETE /api/v1/admin/users/{id}/tokens

# Isolate service (if compromised)
kubectl scale deploy/{service} --replicas=0
```

## Forensic Evidence Collection

```bash
# Export audit logs with hash chain
POST /api/v1/audit/export
{"from": "2025-01-14T00:00:00Z", "format": "jsonl", "include_chain": true}

# Verify chain integrity
GET /api/v1/audit/verify-chain

# Snapshot database
pg_dump ggid > /forensics/snapshot-$(date +%Y%m%d).dump

# Capture active sessions
GET /api/v1/admin/sessions/active?format=csv
```

## Communication Templates

### Internal (Slack)

```
P0: Auth Service Down — INC-2025-0001
IC: @oncall | Started: 14:00 UTC
Impact: All logins failing, ~5000 users
Status: Investigating
Next update: 14:15
Bridge: meet.ggid.dev/inc-2025-0001
```

### External (Status Page)

```
Investigating — Authentication Service Degradation
We are investigating login failures. Some users may be unable
to authenticate. Next update in 30 minutes.
```

### Regulatory (GDPR 72h)

```
Breach Notification — [Date]
Authority: [DPA Name]
Nature: Unauthorized access to user data
Affected: ~5000 users (email, display_name)
Measures: Sessions revoked, vulnerability patched
Contact: dpo@ggid.dev
```

## Post-Incident Review

```markdown
## Post-Mortem: INC-2025-0001
**Duration:** 1h 30m | **Impact:** 5000 users, 45min outage

### Timeline
14:00 — Error spike detected
14:02 — IC declared P0
14:15 — Root cause: deploy introduced connection leak
14:30 — Rollback applied
14:45 — Services restored

### Root Cause
Deployed code opened DB connections without releasing them.

### Action Items
1. Connection leak detection in CI (P1)
2. Alert at 80% pool utilization (P1)
3. Code review checklist update (P2)
```

## See Also

- [Incident Command System](incident-command-system.md)
- [Identity Recovery Playbook](identity-recovery-playbook.md)
- [Audit Tamper Detection](audit-tamper-detection.md)
- [Compliance Framework Mapping](compliance-framework-mapping.md)
