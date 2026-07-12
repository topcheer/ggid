# Incident Command System

SEV definitions, on-call rotation, incident commander role, war room procedures, communication templates, stakeholder updates, and post-mortem.

## Severity Levels

| Level | Definition | Response Time | Example |
|-------|-----------|--------------|---------|
| SEV-1 | Critical outage, data loss risk | Immediate (24/7) | Auth service down, DB failure |
| SEV-2 | Major degradation | <15 min (24/7) | Login latency >5s, partial outage |
| SEV-3 | Minor degradation | <1 hour (business hours) | Console slow, non-critical API degraded |
| SEV-4 | Nuisance, no user impact | Next business day | Intermittent log warnings |

## Incident Commander (IC) Role

The IC owns the incident response — not the fix:

| Responsibility | Detail |
|---------------|--------|
| Coordinate response | Direct responders, track status |
| Communication | Stakeholder updates every 15-30 min |
| Decision authority | Roll back, fail over, escalate |
| Document timeline | Record key events and decisions |
| NOT debugging | Delegate technical work |

### IC Appointment

```
Alert fires → On-call engineer assesses
  ├── SEV-3/4 → On-call handles directly
  └── SEV-1/2 → On-call becomes IC, pages additional responders
```

## On-Call Rotation

| Schedule | Details |
|----------|---------|
| Primary | 1 week shifts, 24/7 |
| Secondary | Backup for primary, 24/7 |
| Escalation | Engineering manager, then CTO |
| Handoff | Monday 10:00, documented in #oncall |

### Escalation Path

```
On-call (primary) → 15 min no response →
On-call (secondary) → 15 min →
Engineering Manager → 10 min →
CTO → immediate
```

## War Room Procedures

### SEV-1/SEV-2 War Room

```
1. IC creates Slack channel: #inc-YYYY-NNNN
2. IC posts initial assessment (see template)
3. Responders join and report status
4. IC assigns investigation tasks
5. Updates every 15 min (SEV-1) or 30 min (SEV-2)
6. Video bridge opened for SEV-1
```

### Initial Assessment Template

```
🔴 SEV-1: Auth Service Down
Incident: INC-2025-0001
IC: @oncall-engineer
Started: 14:00 UTC
Impact: All logins failing, ~5000 users affected
Current status: Investigating auth service health
Next update: 14:15 UTC
Bridge: https://meet.ggid.dev/inc-2025-0001
```

### Status Update Template

```
🟡 Update #3 (14:45 UTC)
Status: Auth service restored, monitoring
Root cause: DB connection pool exhausted after deploy
Impact: Logins restored, ~5000 users affected for 45 min
Actions taken:
  - Rolled back deploy (14:30)
  - Restarted auth pods (14:35)
  - Verified health checks green (14:40)
Next update: 15:00 UTC or resolution
```

## Communication

### Stakeholder Updates

| Audience | Channel | Frequency |
|----------|---------|-----------|
| Engineering team | #inc-NNNN Slack | Every 15 min |
| All employees | #incidents Slack | Every 30 min |
| Customers (SEV-1) | Status page | Every 30 min |
| Executives | Email/Slack DM | Every 60 min |
| Legal/PR (SEV-1) | Phone call | At start + resolution |

### Status Page

```
Investigating — Auth Service Degradation
January 15, 2025 14:05 UTC

We are investigating degraded performance in the authentication
service. Some users may experience login delays or failures.
We'll provide another update in 30 minutes.
```

## Resolution

### Resolution Checklist

- [ ] All health checks green
- [ ] Error rate back to baseline
- [ ] Monitoring shows normal operation for 15 min
- [ ] Stakeholders notified of resolution
- [ ] Status page updated to "Resolved"
- [ ] Timeline documented
- [ ] Post-mortem scheduled (within 48h)

### Resolution Message

```
🟢 Resolved (15:30 UTC)
Incident: INC-2025-0001
Duration: 1h 30m
Impact: Login failures for ~5000 users (14:00-14:45 UTC)
Root cause: DB connection pool exhaustion after deployment
Resolution: Deploy rolled back, connection pool config fixed
Post-mortem: Scheduled for 2025-01-17 14:00 UTC
```

## Post-Mortem Process

### Within 48 Hours

```
Post-Mortem: INC-2025-0001

## Summary
Auth service down for 45 minutes due to DB connection pool exhaustion.

## Timeline
14:00 — Auth error rate spikes to 100%
14:02 — PagerDuty alerts on-call
14:05 — IC declares SEV-1, opens war room
14:10 — Identified recent deploy as trigger
14:15 — Rolled back deployment
14:20 — Auth pods restarted with old config
14:35 — Health checks green
14:45 — Error rate back to baseline

## Root Cause
Deployed code opened DB connections without closing them.
Connection pool exhausted (25/25). New connections blocked.

## What Went Well
- Detection in 2 minutes (good alerting)
- Rollback plan existed and worked
- Quick coordination (8 responders in 5 min)

## What Went Wrong
- Code review missed connection leak
- No load test for connection pooling
- Connection pool monitoring not alerting before exhaustion

## Action Items
1. Add connection leak detection to CI (P1, owner: @dev)
2. Alert at 80% pool utilization (P1, owner: @sre)
3. Add load test simulating 1000 concurrent logins (P2, owner: @qa)
4. Code review checklist: check all db.Acquire has defer Release (P2)
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| SEV-1 incidents/year | <2 | — |
| SEV-2 incidents/month | <2 | — |
| MTTD (detect) | <5 min | >10 min |
| MTTR (resolve) | <30 min | >1h |
| Post-mortem on time | 100% | <48h → overdue |

## See Also

- [SRE Practices](sre-practices.md)
- [Disaster Recovery Testing](disaster-recovery-testing.md)
- [Identity Recovery Playbook](identity-recovery-playbook.md)
- [Monitoring and Alerting](monitoring-and-alerting.md)
