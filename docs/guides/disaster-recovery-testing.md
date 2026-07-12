# Disaster Recovery Testing

DR drill procedures, RPO/RTO validation, chaos engineering, game days, failure injection, post-drill analysis, and improvement tracking.

## DR Objectives

| Metric | Target | Definition |
|--------|--------|-----------|
| RPO | <5 min | Max acceptable data loss |
| RTO | <30 min | Max acceptable downtime |
| RTO (critical) | <15 min | Auth/gateway services |

## DR Drill Schedule

| Drill Type | Frequency | Scope | Participants |
|-----------|-----------|-------|-------------|
| Tabletop | Quarterly | Scenario discussion | Eng + ops |
| Partial failover | Bi-annually | Single service | On-call team |
| Full failover | Annually | Entire region | All teams |
| Chaos engineering | Monthly | Random injection | SRE team |

## Drill Procedure

### Full Region Failover

```
Step 1 (T+0): Declare drill
  - Notify stakeholders: "DR drill starting"
  - Switch DNS to maintenance page (read-only mode)

Step 2 (T+2min): Promote secondary region
  - PostgreSQL: Promote read replica to primary
  - Redis: Failover to replica
  - NATS: Switch to backup stream
  - Kubernetes: Scale up in DR region

Step 3 (T+10min): Verify services
  - Health checks all green
  - Smoke tests (login, create user, audit query)
  - Data integrity check

Step 4 (T+15min): Switch DNS
  - Route traffic to DR region
  - Monitor for errors

Step 5 (T+30min): Declare success
  - RTO measured: time from declare to traffic served
  - RPO measured: replication lag at time of failover

Step 6 (T+1h): Failback
  - Re-establish replication back to primary region
  - Sync data
  - Switch DNS back
  - Verify
```

## RPO/RTO Measurement

### RPO Check

```bash
# Before failover, measure replication lag
POSTGRES_REPL_LAG=$(psql -c "SELECT EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp()))")
echo "Replication lag: ${POSTGRES_REPL_LAG}s"

REDIS_REPL_LAG=$(redis-cli INFO replication | grep lag)
echo "Redis lag: ${REDIS_REPL_LAG}"
```

### RTO Check

```bash
# Record timestamps
DRILL_START=$(date +%s)

# ... perform failover ...

# First successful health check
HEALTHY=$(wait_for_healthy "https://dr.ggid.dev/healthz")
RTO=$((HEALTHY - DRILL_START))
echo "RTO: ${RTO}s (target: 1800s)"
```

## Chaos Engineering

### Failure Injection

| Experiment | Injection | Expected Outcome |
|-----------|-----------|-----------------|
| Kill pod | `kubectl delete pod` | Auto-restart, no user impact |
| Network latency | `tc netem delay 500ms` | Graceful degradation |
| Network partition | Network policy block | Circuit breaker opens |
| Disk full | Fill disk to 100% | Service degrades, alerts fire |
| CPU stress | `stress-ng --cpu 4` | HPA scales up |
| DB connection exhaustion | Open 100 connections | Pool limit prevents cascade |
| Redis failure | `redis-cli DEBUG SLEEP 60` | Fallback to stateless |

### Game Day

```markdown
## Game Day: 2025-01-20

### Scenario: Primary DB failure

### Timeline
- 14:00 — DB primary killed (injected)
- 14:00:05 — Auto-detection: health check fails
- 14:00:15 — Replica promoted (automatic failover)
- 14:00:30 — App reconnects to new primary
- 14:00:45 — First successful request

### Results
- RTO: 45s (target: 1800s) ✅
- RPO: 2s (target: 300s) ✅
- User impact: 15s of errors (connections in flight)

### Issues Found
1. Connection pool didn't reconnect for 10s (DNS cache)
2. No alert for "replica promotion" event

### Action Items
1. [ ] Reduce DNS TTL for DB endpoint (P1)
2. [ ] Add alert for PostgreSQL role change (P2)
3. [ ] Add chaos test to CI (P2)
```

## Post-Drill Analysis

### Report Template

```markdown
## DR Drill Report

### Summary
- Date: 2025-01-20
- Type: Full region failover
- RTO: 45s (target 1800s) ✅
- RPO: 2s (target 300s) ✅
- Participants: 8

### What Worked
- Automatic PostgreSQL failover (Patroni)
- Health checks detected failure in 5s
- Zero data loss

### What Failed
- DNS caching delayed reconnection
- No alert for failover event
- Runbook was outdated (referenced old IPs)

### Improvement Actions
| # | Action | Owner | Priority | Due |
|---|--------|-------|----------|-----|
| 1 | Reduce DNS TTL | SRE | P1 | 1 week |
| 2 | Add failover alerts | SRE | P2 | 2 weeks |
| 3 | Update runbook | Ops | P2 | 2 weeks |
| 4 | Add chaos test to CI | SRE | P3 | 1 month |
```

## Monitoring

| Metric | Alert |
|--------|-------|
| RPO drill result | >target → investigate replication |
| RTO drill result | >target → optimize failover |
| Drill pass rate | <100% → schedule more drills |
| Unfixed action items | Overdue → escalate |

## See Also

- [Backup and Restore](backup-and-restore.md)
- [Multi-Region Deployment](multi-region-deployment.md)
- [Incident Command System](incident-command-system.md)
- [SRE Practices](sre-practices.md)
