# High Availability Architecture

## Overview

GGID IAM Suite is designed for 99.99% uptime (≤52.6 min/year downtime) through multi-zone deployment, stateless services, and graceful degradation patterns.

## Availability Targets by Service Tier

| Tier | Service | SLO | Recovery Time Objective |
|------|---------|-----|------------------------|
| Critical | Gateway, Auth, OAuth | 99.99% | < 30s |
| Important | Identity, Policy | 99.95% | < 2min |
| Standard | Audit, Org | 99.9% | < 5min |

## Deployment Topology

### Multi-Zone Active-Active

```
                    ┌─────────────┐
                    │  Global LB  │
                    │  (Anycast)  │
                    └──────┬──────┘
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        ┌─────────┐  ┌─────────┐  ┌─────────┐
        │ Zone A  │  │ Zone B  │  │ Zone C  │
        │ (us-1a) │  │ (us-1b) │  │ (eu-1a) │
        └────┬────┘  └────┬────┘  └────┬────┘
             │            │            │
        ┌────┴────┐  ┌────┴────┘  ┌────┴────┐
        │ Gateway │  │ Gateway │  │ Gateway │
        │ Auth    │  │ Auth    │  │ Auth    │
        │ OAuth   │  │ OAuth   │  │ OAuth   │
        │ Policy  │  │ Policy  │  │ Policy  │
        └────┬────┘  └────┬────┘  └────┬────┘
             │            │            │
        ┌────┴────────────┴────────────┴────┐
        │     Synchronous Replication       │
        │    PostgreSQL (Patroni + etcd)    │
        │    Redis Sentinel (3 nodes)       │
        │    NATS JetStream (R3, replicated)│
        └───────────────────────────────────┘
```

## Stateless Service Design

### Session Storage
- JWT tokens are self-contained; no session DB needed for verification
- Refresh tokens stored in Redis with zone-local replicas
- Session revocation propagated via NATS JetStream events

### Request Routing
- No server affinity required — any instance can serve any request
- Tenant ID extracted from JWT claim, not server-local state
- Rate limits enforced via Redis token buckets (shared across instances)

## Database High Availability

### PostgreSQL (Primary Data Store)

**Architecture**: Patroni cluster with etcd coordination

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│ Primary  │ ←─→ │ Replica  │ ←─→ │ Replica  │
│ (Zone A) │     │ (Zone B) │     │ (Zone C) │
└────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                 │
     └──────── etcd ────────────────────┘
                 │
           ┌─────┴─────┐
           │  HAProxy  │
           │ (read/write
           │  routing) │
           └───────────┘
```

**Failover Process**:
1. Primary health check fails (3 consecutive misses)
2. etcd promotes highest-priority replica
3. HAProxy updates upstream routing (< 10s)
4. Old primary demoted upon recovery
5. Connection pool handles reconnects automatically

**Replication**:
- Synchronous for Zone A → Zone B (zero data loss)
- Asynchronous for Zone C (acceptable 100ms lag)
- `synchronous_commit = on` for auth/identity writes
- `synchronous_commit = remote_apply` for policy/rbac

### Redis (Cache + Rate Limiting)

**Architecture**: Redis Sentinel with 3-node cluster

- Writes to primary, reads from replicas
- Rate limit counters replicated (single-source-of-truth)
- Session data partitioned by tenant
- Failover: Sentinel promotes replica in < 5s

### NATS JetStream (Event Bus)

**Configuration**:
- `R3` replication factor across 3 zones
- Durable consumers for audit/policy services
- At-least-once delivery with idempotent consumers
- JetStream stream replication: Zone A → Zone B → Zone C

## Graceful Degradation

### Redis Unavailable
- Rate limiting: allow all requests, log warning
- Session cache: fall back to JWT-only validation
- Feature flags: use last-known-good config

### Database Failover
- Read queries: transparent redirect to replica
- Write queries: queue in NATS, replay after failover
- Identity creation: return 202 Accepted, process async

### NATS Unavailable
- Audit events: buffer to local disk, replay on reconnect
- Policy updates: use cached policy version
- Session revocation: mark for propagation, best-effort

## Health Check Architecture

### Liveness Probes
```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 15
  periodSeconds: 10
  failureThreshold: 3
```

### Readiness Probes
```yaml
readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 2
```

**Readiness checks include**:
- Database connection pool (min 1 connection)
- Redis ping (if rate limiting enabled)
- NATS connection (if event publisher)
- Key rotation service (if crypto service)

## Rolling Updates

### Zero-Downtime Strategy
1. Deploy to Zone C first (canary)
2. Health check for 5 minutes
3. If healthy, deploy to Zone B
4. Health check for 3 minutes
5. If healthy, deploy to Zone A
6. Automatic rollback if error rate > 1%

### Connection Draining
```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 1
  terminationGracePeriodSeconds: 60
```

**Draining steps**:
1. Pod receives SIGTERM
2. Stops accepting new connections
3. Completes in-flight requests (up to 60s)
4. Closes database connections cleanly
5. Pod exits

## Capacity Planning

### Per-Zone Resource Allocation

| Service | Replicas | CPU/Replica | Memory/Replica | Connections |
|---------|----------|-------------|----------------|-------------|
| Gateway | 3 | 500m | 256Mi | 10000 |
| Auth | 2 | 1000m | 512Mi | 5000 |
| OAuth | 2 | 500m | 256Mi | 3000 |
| Identity | 2 | 500m | 256Mi | 3000 |
| Policy | 2 | 500m | 256Mi | 2000 |
| Audit | 1 | 250m | 128Mi | 1000 |

### Auto-Scaling Triggers
- CPU > 70% sustained for 2 minutes
- Memory > 80% sustained for 3 minutes
- Request latency p99 > 500ms
- Active connections > 80% of limit

## Disaster Recovery

### RPO and RTO
- **RPO** (Recovery Point Objective): < 1 second (synchronous replication)
- **RTO** (Recovery Time Objective): < 30 seconds (automatic failover)

### Backup Strategy
- **Full backup**: Daily at 02:00 UTC, retained 30 days
- **WAL archiving**: Continuous, retained 7 days
- **PITR**: Point-in-time recovery up to 7 days back
- **Cross-region**: Backup copied to DR region every 6 hours

### DR Failover Procedure
1. Promote DR region database to primary
2. Update Global LB DNS to DR region
3. Scale up DR region service replicas
4. Verify health checks
5. Switch traffic (estimated 2-5 minutes total)

## Monitoring and Alerting

### Critical Alerts (Page immediately)
- Any service down in 2+ zones
- Database replication lag > 5 seconds
- NATS stream unhealthy
- Auth error rate > 5%
- Gateway 5xx rate > 1%

### Warning Alerts (Notify on-call)
- Single zone service degradation
- Replication lag > 1 second
- Connection pool utilization > 80%
- Certificate expiring < 7 days
- Disk usage > 85%
