# High Availability Deployment Guide

This guide covers deploying GGID for high availability — multi-AZ, active-active services, DB replication, Redis sentinel, NATS clustering, load balancer config, health checks, and failover automation.

## Architecture Overview

```
AZ-A                          AZ-B
┌────────────────────┐       ┌────────────────────┐
│ Gateway (2 pods)    │       │ Gateway (2 pods)    │
│ Auth (2 pods)       │       │ Auth (2 pods)       │
│ Identity (2 pods)   │       │ Identity (2 pods)   │
│ OAuth (2 pods)      │       │ OAuth (2 pods)      │
│ Policy (2 pods)     │       │ Policy (2 pods)     │
│ Org (2 pods)        │       │ Org (2 pods)        │
│ Audit (2 pods)      │       │ Audit (2 pods)      │
├────────────────────┤       ├────────────────────┤
│ PG Primary (RW)     │← WAL →│ PG Replica (RO)     │
│ Redis Primary       │←Sync→ │ Redis Replica       │
│ NATS Node 1         │←Clust→│ NATS Node 2         │
└────────────────────┘       └────────────────────┘
         ↕                              ↕
    ┌────────────────────────────────────┐
    │     Global Load Balancer (Route53)   │
    │     Latency-based + health checks    │
    └────────────────────────────────────┘
```

## Service Replicas

| Service | Min Replicas | Max Replicas | CPU Target |
|---------|-------------|-------------|------------|
| Gateway | 3 (multi-AZ) | 10 | 70% |
| Auth | 3 | 8 | 70% |
| Identity | 3 | 8 | 70% |
| OAuth | 2 | 5 | 60% |
| Policy | 2 | 5 | 60% |
| Org | 2 | 5 | 60% |
| Audit | 2 | 5 | 60% |

### Pod Anti-Affinity (Multi-AZ)

```yaml
spec:
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            app: gateway
        topologyKey: topology.kubernetes.io/zone
```

## PostgreSQL HA

### Streaming Replication

```ini
# Primary
wal_level = replica
max_wal_senders = 10
hot_standby = on
synchronous_commit = on
synchronous_standby_names = '*'  # Wait for replica
```

### Automatic Failover (Patroni/pg_auto_failover)

```yaml
patroni:
  scope: ggid
  postgresql:
    use_pg_rewind: true
  restapi:
    connect_address: 10.0.1.10:8008
  etcd:
    hosts: etcd-1:2379,etcd-2:2379,etcd-3:2379
```

**Failover procedure**:
1. Primary health check fails (3 consecutive)
2. Patroni promotes replica to primary
3. Services reconnect (connection pool handles)
4. Old primary re-joins as replica
5. Total failover: 10-30 seconds

## Redis HA (Sentinel)

```yaml
sentinel:
  quorum: 2  # Out of 3 sentinels
  down_after_milliseconds: 5000
  failover_timeout: 30000
  parallel_syncs: 1
```

**Failover**: Sentinel detects primary failure → promotes replica → notifies clients → GGID services auto-reconnect.

## NATS Clustering

```yaml
nats:
  cluster:
    name: ggid-cluster
    routes:
      - nats://nats-1.internal:6222
      - nats://nats-2.internal:6222
      - nats://nats-3.internal:6222
  jetstream:
    store_dir: /data
    max_mem: 1GB
```

**RA=3 for stream replicas**: Data survives single node loss.

## Load Balancer

### Health Check Configuration

```yaml
health_check:
  path: /healthz
  interval: 5s
  timeout: 2s
  healthy_threshold: 2
  unhealthy_threshold: 3
```

### Circuit Breaker

If a service consistently fails health checks:
1. LB marks instance as unhealthy (removes from pool)
2. GGID circuit breaker opens (fails fast)
3. Requests routed to healthy instances
4. After recovery: auto-add back to pool

## Failover Automation

### Kubernetes HPA + PDB

```yaml
# Pod Disruption Budget — always keep min available
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: gateway-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: gateway
```

### Failover Checklist

| Component | Detection | Failover Time | Data Loss |
|-----------|-----------|---------------|-----------|
| Gateway pod | Liveness probe | 5s | None |
| Auth pod | Liveness probe | 5s | None |
| PostgreSQL | Patroni | 10-30s | 0 (sync) |
| Redis | Sentinel | 5-10s | < 1s |
| NATS | Cluster RA3 | Instant | None |
| AZ failure | Route53 health | 60s | 0 |

## Monitoring

| Metric | Alert Threshold |
|--------|----------------|
| Pod restart count | > 3 in 10m |
| DB replication lag | > 5s |
| Redis failover | Any occurrence |
| AZ health | Any AZ unhealthy |
| HPA scale events | > 5 in 1h |

## See Also

- [Performance Tuning](performance-tuning.md)
- [Multi-Region Deployment](../research/multi-region.md)
- [Production Checklist](production-checklist.md)
- [Disaster Recovery](disaster-recovery.md)
