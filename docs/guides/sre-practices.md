# SRE Practices Guide

SLO/SLI definition, error budgets, blameless post-mortems, runbooks, toil reduction, change management, and capacity planning.

## SLI (Service Level Indicator)

Measurable metric of service quality:

| Service | SLI | Target (SLO) |
|---------|-----|-------------|
| Gateway | Availability (200 responses) | 99.95% |
| Auth | Login success rate | 99.9% |
| Identity | User CRUD latency P99 | <200ms |
| Policy | Evaluation latency P99 | <50ms |
| Audit | Event delivery latency P99 | <5s |

### SLI Implementation

```promql
# Gateway availability
sum(rate(istio_requests_total{destination_service="gateway", response_code!~"5.."}[5m]))
/ sum(rate(istio_requests_total{destination_service="gateway"}[5m]))
```

## SLO (Service Level Objective)

| Tier | SLO | Error Budget | Downtime Allowed |
|------|-----|-------------|-----------------|
| Critical (Auth, Gateway) | 99.95% | 0.05% | 22 min/month |
| Standard (Identity, Policy) | 99.9% | 0.1% | 43 min/month |
| Best-effort (Console) | 99.5% | 0.5% | 3.6 hours/month |

## Error Budget

```
Monthly budget = (1 - SLO) × minutes_per_month

For 99.95% SLO: 0.05% × 43200 min = 21.6 min/month
```

### Budget Consumption

| Consumption | Action |
|------------|--------|
| <30% | Normal — deploy freely |
| 30-70% | Cautious — larger changes need approval |
| 70-100% | Freeze — only bug fixes, no new features |
| >100% | Overdrawn — mandatory reliability work, feature freeze |

### Error Budget Alerting

```yaml
alerts:
  - name: error_budget_50pct
    query: "error_budget_remaining < 50"
    severity: warn
    message: "Error budget 50% consumed — review reliability"

  - name: error_budget_90pct
    query: "error_budget_remaining < 10"
    severity: critical
    message: "Error budget 90% consumed — feature freeze"
```

## Blameless Post-Mortems

### Principles

1. **Focus on systems, not people** — "What failed?" not "Who caused it?"
2. **Assume good intent** — Everyone was doing their best with available info
3. **Find systemic issues** — Process gaps, missing tooling, inadequate alerts
4. **Action items** — Every finding has an owner and deadline
5. **Share widely** — Post-mortems are public within the org

### Template

```markdown
## Post-Mortem: [INC-YYYY-NNNN]

**Severity:** SEV-2
**Date:** 2025-01-15
**Duration:** 45 minutes
**Impact:** 5000 users couldn't login

### What Happened
[Concise factual summary]

### Timeline
[detailed events with timestamps]

### Root Cause
[Technical root cause — system, not person]

### Contributing Factors
- Missing alert for X
- Runbook didn't cover Y
- No load test for Z

### Action Items
| # | Action | Owner | Priority | Status |
|---|--------|-------|----------|--------|
| 1 | Add alert | @sre | P1 | Done |
| 2 | Update runbook | @ops | P2 | Open |
```

## Runbooks

```markdown
## Runbook: Auth Service High Error Rate

### Symptoms
- Alert: "auth_error_rate > 1%"

### Quick Diagnose
1. Check recent deploys: `kubectl rollout history deploy/auth`
2. Check DB health: `GET /healthz/ready` on auth pod
3. Check Redis: `redis-cli ping`
4. Check error logs: `kubectl logs deploy/auth | grep ERROR`

### If Recent Deploy
- Rollback: `kubectl rollout undo deploy/auth`
- Verify: Watch error rate for 5 min

### If DB Issue
- Check pool: `GET /debug/pool-stats`
- If exhausted: Restart pods `kubectl rollout restart deploy/auth`

### Escalation
- On-call: @auth-oncall
- Runbook owner: @auth-team-lead
- Escalation: @engineering-manager
```

## Toil Reduction

| Toil Type | Reduction Strategy |
|-----------|-------------------|
| Manual deployments | Automate CI/CD pipeline |
| Manual failover | Automate with health checks |
| Manual provisioning | IaC (Terraform) |
| Manual cert renewal | Auto-renewal (cert-manager) |
| Alert triage | Better alerting, reduce false positives |
| Password resets | Self-service portal |

### Toil Budget

SRE teams should spend **<50% time on toil**. Track monthly:

```
Total SRE hours: 800
Toil hours: 320 (40%) ← Acceptable
Toil hours: 500 (62%) ← Too high, dedicate sprint to automation
```

## Change Management

### Change Classes

| Class | Risk | Approval | Notice |
|-------|------|----------|--------|
| Standard (runbook) | Low | Auto | None |
| Normal (config) | Medium | 1 reviewer | 1 day |
| Emergency (hotfix) | High | IC approval | Immediate |
| Major (schema) | High | 2 reviewers + test | 3 days |

### Change Freeze

During these windows, only emergency changes:
- Error budget >70% consumed
- Major incident in progress
- Holiday blackout (Nov 25 - Jan 2)

## Capacity Planning

| Resource | Current | 6-Month Forecast | Action |
|----------|---------|-----------------|--------|
| DB storage | 120GB | 200GB | Provision 500GB |
| Redis memory | 800MB | 2GB | Scale to 4GB |
| NATS storage | 5GB | 15GB | Provision 50GB |
| Max concurrent users | 5000 | 15000 | Scale gateway to 30 pods |
| API rate | 5K/min | 15K/min | Scale services |

### Capacity Review

```bash
# Monthly capacity report
GET /api/v1/admin/capacity/report
# → Current utilization, growth trends, 6-month forecast, recommended actions
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| SLO compliance | 100% of targets | Miss → reliability sprint |
| Error budget | >30% remaining | <10% → feature freeze |
| Toil percentage | <50% | >50% → automate |
| Post-mortem action closure | >90% on time | Overdue items → escalate |
| Runbook coverage | 100% of alerts | Gap → write runbook |

## See Also

- [Incident Command System](incident-command-system.md)
- [Disaster Recovery Testing](disaster-recovery-testing.md)
- [Auto-Scaling Strategy](auto-scaling-strategy.md)
- [Monitoring and Alerting](monitoring-and-alerting.md)
