# GGID Multi-Region High Availability Design

> Status: Design Phase
> Depends on: R3-02 Helm Chart (complete)

## Architecture Overview

```
Region A (Primary)                    Region B (DR)
┌─────────────────────┐              ┌─────────────────────┐
│  Gateway (3 replicas)│              │  Gateway (3 replicas)│
│  Auth/Identity/OAuth │              │  Auth/Identity/OAuth │
│  Policy/Audit/Org    │              │  Policy/Audit/Org    │
│                      │              │                      │
│  PostgreSQL Primary  │──streaming──►│  PostgreSQL Standby  │
│  Redis Primary       │──replication►│  Redis Replica       │
│                      │              │                      │
│  PgBackRest → S3     │              │  PgBackRest → S3     │
└─────────┬───────────┘              └─────────┬───────────┘
          │                                      │
          └────────── GeoDNS / Traffic Manager ──┘
                     (health-based failover)
```

## Component HA Strategy

### 1. PostgreSQL (Patroni + Streaming Replication)

**Option A: CloudNativePG Operator (recommended)**
- CNPG manages primary/replica failover automatically
- WAL archiving to S3-compatible storage
- Point-in-time recovery (PITR) via Barman
- Connection pooling via PgBouncer sidecar

```yaml
# values-prod.yaml addition
postgresql:
  ha:
    enabled: true
    replicas: 3  # 1 primary + 2 replicas
    mode: streaming_replication
    walArchiving:
      enabled: true
      destination: s3://ggid-pg-wal/
    failover:
      mode: automatic  # Patroni-style leader election
      timeout: 30s
    connectionPooling:
      enabled: true
      maxClientConn: 1000
      poolMode: transaction
```

**Option B: Stolon (alternative)**
- Kubernetes-native PG HA
- Sentinel-based leader election
- Less mature than CNPG for production

### 2. Redis (Sentinel or Cluster)

**For GGID: Redis Sentinel (simpler, sufficient)**

```yaml
redis:
  architecture: replication
  sentinel:
    enabled: true
    quorum: 2
    masterSet: ggid
  master:
    persistence:
      size: 10Gi
      storageClass: fast-ssd
  replica:
    replicaCount: 2
    persistence:
      size: 10Gi
```

GGID uses Redis for:
- RBAC route permission cache (stale-while-revalidate acceptable)
- Session token blocklist (CAE jti check)
- Rate limiting counters

All are tolerant of brief Redis unavailability (gateway has in-memory fallback).

### 3. Service-Level HA

All stateless services (gateway, auth, identity, oauth, policy, org, audit) 
run with `replicaCount >= 3` and spread across nodes using topology spread constraints:

```yaml
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: ScheduleAnyway
```

### 4. Cross-Region Failover

**DNS-based (GeoDNS)**:
- Primary: `ggid.iot2.win` → Region A LB
- Failover: health check fails → auto-route to Region B
- TTL: 30s (fast DNS propagation)
- Provider: Cloudflare Load Balancer or AWS Route 53

**Database Promotion**:
1. Region A PG primary goes down
2. Patroni promotes Region A replica or Region B standby
3. DNS fails traffic to Region B
4. Services reconnect to new primary via ClusterIP (CNPG updates service)

**Split-Brain Prevention**:
- Use synchronous_commit=on for cross-region replication
- Fencing: Patroni marks old primary as unfenced before promoting standby
- Application-level: gateway checks DB write success before responding

### 5. Data Synchronization

| Data Type | Sync Strategy | RPO | RTO |
|-----------|--------------|-----|-----|
| PostgreSQL | Streaming replication (sync) | 0s | <30s |
| Redis | Async replication | <1s | <10s |
| JWT keys | Shared across regions (configmap) | N/A | N/A |
| S3 artifacts | Cross-region replication | <60s | <60s |

### 6. Health-Based Failover Flow

```
1. Health checker detects Region A degradation
2. DNS provider updates ggid.iot2.win → Region B
3. Region B services already running (warm standby)
4. PG standby promoted to primary (if needed)
5. Redis sentinel fails over to Region B replica
6. Users re-authenticate against Region B
```

## Implementation Checklist

- [ ] Deploy CNPG operator to both regions
- [ ] Configure PG streaming replication with WAL archiving
- [ ] Deploy Redis sentinel cluster in both regions
- [ ] Configure topology spread constraints on all deployments
- [ ] Set up GeoDNS health checks
- [ ] Test planned failover (maintenance window)
- [ ] Test unplanned failover (chaos engineering)
- [ ] Document recovery procedures in RUNBOOK.md

## Prerequisites

- Two Kubernetes clusters in different regions/zones
- Low-latency network between regions (< 50ms for sync replication)
- S3-compatible object storage for WAL archives
- DNS provider with health-check-based failover (Cloudflare/AWS Route 53)
- Container registry accessible from both regions
