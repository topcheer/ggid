# Disaster Recovery: RTO/RPO Standards & Multi-Region Failover

## Overview

This document analyzes disaster recovery standards, multi-region failover architectures, and DR drill procedures for identity systems. It maps industry RTO/RPO benchmarks to GGID's deployment topology.

> **Related**: [Disaster Recovery Guide](../guides/disaster-recovery.md) (operational procedures), [Disaster Recovery IAM](disaster-recovery-iam.md) (1529 lines, comprehensive design)

## RTO/RPO Fundamentals

| Metric | Definition | Impact |
|--------|------------|--------|
| **RTO** (Recovery Time Objective) | Maximum acceptable downtime | How long can users be without access? |
| **RPO** (Recovery Point Objective) | Maximum acceptable data loss | How much data can you afford to lose? |
| **MTTR** (Mean Time to Recovery) | Average time to restore | Operational metric |
| **MTBF** (Mean Time Between Failures) | Average uptime between incidents | Reliability metric |

## Industry RTO/RPO Benchmarks for IAM

### By Deployment Tier

| Tier | Description | RTO | RPO | Architecture |
|------|-------------|-----|-----|--------------|
| **Tier 1** | Mission-critical (banking, healthcare) | < 5 min | 0 | Active-active multi-region |
| **Tier 2** | Business-critical (enterprise SaaS) | < 30 min | < 5 min | Active-passive with replication |
| **Tier 3** | Standard (B2B SaaS) | < 4 hours | < 15 min | Active-passive with backups |
| **Tier 4** | Development/Staging | < 24 hours | < 24 hours | Single region with backups |

### Competitor Benchmarks

| Provider | RTO | RPO | Method |
|----------|-----|-----|--------|
| Okta | < 5 min | 0 | Active-active multi-region |
| Auth0 (Auth0 by Okta) | < 15 min | < 1 min | Multi-region with failover |
| Azure AD (Entra ID) | < 5 min | 0 | Global active-active |
| Ping Identity | < 30 min | < 5 min | Active-passive |
| Keycloak (self-hosted) | Varies | Varies | Depends on deployment |

## GGID Component RTO/RPO Targets

| Component | Tier 2 Target | Current Capability | Gap |
|-----------|---------------|-------------------|-----|
| PostgreSQL | RTO: 30 min, RPO: 15 min | `pg_basebackup` + WAL streaming | Partial — no automated failover |
| Redis | RTO: 5 min, RPO: 0 | Redis replication (ephemeral) | Done — sessions are ephemeral |
| NATS JetStream | RTO: 15 min, RPO: 0 | File-backed persistence | Done — stream replication available |
| JWT signing keys | RTO: 5 min, RPO: 0 | Keygen container + volume | Partial — single-region keys |
| Gateway | RTO: 5 min, RPO: N/A | Stateless, horizontally scalable | Done |
| Identity/Auth/OAuth | RTO: 10 min, RPO: N/A | Stateless services | Done |
| Policy/Org/Audit | RTO: 10 min, RPO: 15 min | Stateful (DB-backed) | Depends on DB recovery |

## Multi-Region Failover Architecture

### Option A: Active-Passive (Warm Standby)

```
Region A (Primary)                    Region B (Standby)
┌─────────────────┐                  ┌─────────────────┐
│  GGID Services  │                  │  GGID Services  │
│  (Active)       │                  │  (Standby)      │
└───────┬─────────┘                  └───────┬─────────┘
        │                                     │
┌───────┴─────────┐    Streaming     ┌───────┴─────────┐
│  PostgreSQL     │ ──── WAL ──────→ │  PostgreSQL     │
│  (Primary)      │    Replication   │  (Replica)      │
└─────────────────┘                  └─────────────────┘
        │                                     │
┌───────┴─────────┐                  ┌───────┴─────────┐
│  Redis          │    Redis         │  Redis          │
│  (Primary)      │ ─── Replication→ │  (Replica)      │
└─────────────────┘                  └─────────────────┘
```

**Failover procedure**:
1. DNS failover: Route traffic to Region B
2. Promote Region B PostgreSQL replica to primary
3. Start GGID services in Region B
4. Verify health endpoints

**RTO**: 10-30 minutes (DNS TTL + replica promotion)
**RPO**: Near-zero (async streaming replication)

### Option B: Active-Active (Multi-Region)

```
Region A                              Region B
┌─────────────────┐                  ┌─────────────────┐
│  GGID Services  │                  │  GGID Services  │
│  (Active)       │                  │  (Active)       │
└───────┬─────────┘                  └───────┬─────────┘
        │                                     │
┌───────┴─────────┐  Bidirectional  ┌───────┴─────────┐
│  PostgreSQL     │ ←─ Logical ──→ │  PostgreSQL     │
│  (Primary-A)    │    Replication  │  (Primary-B)    │
└─────────────────┘                  └─────────────────┘
```

**Challenges**:
- Conflict resolution for concurrent writes
- Tenant data partitioning (tenant → region affinity)
- JWT key distribution across regions
- Cache invalidation across regions

**RTO**: < 5 minutes (traffic shift only)
**RPO**: 0 (both regions accept writes)

### Option C: Pilot Light (Minimal Standby)

```
Region A (Full)                       Region B (Pilot)
┌─────────────────┐                  ┌─────────────────┐
│  Full Stack     │                  │  DB Replica     │
│  (Active)       │                  │  (Only)         │
└───────┬─────────┘                  └───────┬─────────┘
        │                                     │
┌───────┴─────────┐    WAL Archive   ┌───────┴─────────┐
│  PostgreSQL     │ ─── S3/GCS ────→ │  Restore from   │
│  (Primary)      │                  │  archive        │
└─────────────────┘                  └─────────────────┘
```

**RTO**: 1-4 hours (provision + restore)
**RPO**: 15-60 minutes (WAL archive frequency)
**Cost**: Lowest (only DB storage in standby region)

## Database Recovery Procedures

### PostgreSQL Point-in-Time Recovery (PITR)

```bash
# 1. Restore base backup
pg_basebackup -D /var/lib/postgresql/recovery -X stream -c fast

# 2. Create recovery.signal
touch /var/lib/postgresql/recovery/recovery.signal

# 3. Configure recovery (postgresql.auto.conf)
cat >> /var/lib/postgresql/recovery/postgresql.auto.conf << 'EOF'
restore_command = 'aws s3 cp s3://ggid-wal-archive/%f %p'
recovery_target_time = '2025-01-24 14:30:00+00'
recovery_target_action = 'promote'
EOF

# 4. Start PostgreSQL — it replays WAL up to target time
pg_ctl -D /var/lib/postgresql/recovery start
```

### Redis Failover

Redis is used for:
- Session storage (ephemeral — acceptable to lose)
- Rate limiting counters (ephemeral)
- JWT jti blacklist (ephemeral — tokens expire anyway)

**Recovery**: Simply restart Redis. Users will need to re-authenticate.

### NATS JetStream Recovery

```bash
# JetStream data is file-backed and replicated
# On restart, NATS replays unprocessed messages from disk

# Verify stream health after recovery
nats stream info AUDIT
nats stream repair AUDIT
```

## JWT Key Distribution

### Problem

If the auth service signs JWTs with a key, the gateway needs the public key to verify them. In multi-region:

1. Both regions must use the same signing key (shared key)
2. OR each region has its own key and JWKS endpoint is globally accessible
3. OR use a centralized key management service (KMS)

### Recommended: Per-Region Keys with JWKS Aggregation

```
Region A Auth → JWKS-A ─┐
                         ├→ Global JWKS Aggregator → All Gateways
Region B Auth → JWKS-B ─┘
```

Each gateway fetches JWKS from the aggregator, which merges keys from all regions.

## DR Drill Checklist

A DR drill should be conducted quarterly. Use this checklist:

### Pre-Drill

- [ ] Notify all stakeholders (maintenance window)
- [ ] Verify backup integrity (test restore on staging)
- [ ] Document rollback procedure
- [ ] Prepare monitoring dashboards
- [ ] Set up alert routing to DR team

### Drill Execution

- [ ] **T+0**: Simulate primary region failure (stop services / block traffic)
- [ ] **T+0**: DNS failover to standby region
- [ ] **T+1min**: Verify standby health endpoints
- [ ] **T+5min**: Verify user can log in
- [ ] **T+5min**: Verify audit events are being recorded
- [ ] **T+10min**: Verify policy enforcement works
- [ ] **T+10min**: Verify OAuth flows work
- [ ] **T+15min**: Verify SCIM provisioning works
- [ ] **T+30min**: Full smoke test all 7 services
- [ ] **T+30min**: Measure actual RTO

### Post-Drill

- [ ] Document actual RTO and RPO
- [ ] Identify gaps and bottlenecks
- [ ] Update runbook with lessons learned
- [ ] Schedule next drill (quarterly)
- [ ] Verify failback to primary region

## Backup Strategy

| Data | Method | Frequency | Retention | Location |
|------|--------|-----------|-----------|----------|
| PostgreSQL | `pg_basebackup` + WAL | Daily + continuous | 30 days | S3 cross-region |
| Redis | RDB snapshot | Hourly | 24 hours | S3 same-region |
| NATS JetStream | File copy | Daily | 7 days | S3 same-region |
| JWT keys | Keygen container | On rotation | Indefinite | Secrets manager |
| Configuration | Git repository | Continuous | Indefinite | Git remote |
| TLS certificates | ACME/cert-manager | On renewal | 90 days | Secrets manager |

## GGID DR Readiness Assessment

| Area | Status | Score |
|------|--------|-------|
| Database backup | Partial — manual pg_dump | 3/5 |
| Database replication | Partial — streaming supported, not configured | 2/5 |
| Redis ephemeral recovery | Done — sessions rebuildable | 5/5 |
| NATS persistence | Done — file-backed streams | 4/5 |
| Multi-region failover | Gap — no automated DNS failover | 1/5 |
| JWT key portability | Partial — keygen container exists | 3/5 |
| DR drill process | Gap — no documented drills | 1/5 |
| Monitoring & alerting | Partial — health checks exist | 3/5 |
| **Overall** | | **22/40 (55%)** |

## References

- [NIST SP 800-34: Contingency Planning Guide](https://csrc.nist.gov/pubs/sp/800/34/r1/final)
- [AWS Well-Architected Reliability Pillar](https://docs.aws.amazon.com/wellarchitected/latest/reliability-pillar/)
- [Google SRE Book: Disaster Recovery](https://sre.google/sre-book/)

## See Also

- [Disaster Recovery Guide](../guides/disaster-recovery.md)
- [Disaster Recovery IAM Design](disaster-recovery-iam.md)
- [Zero-Downtime Deployment](zero-downtime-deployment.md)
- [Backup and Restore](backup-restore.md)
