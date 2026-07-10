# High Availability Deployment Guide

How to deploy GGID with no single point of failure.

---

## Architecture

```
                    ┌──────────────┐
                    │  Load Balancer│
                    │  (nginx/ALB) │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
         ┌────▼───┐  ┌────▼───┐  ┌────▼───┐
         │ GW #1  │  │ GW #2  │  │ GW #3  │
         └────┬───┘  └────┬───┘  └────┬───┘
              │           │            │
     ┌────────┼───────────┼────────────┼──────┐
     │        │           │            │      │
  ┌──▼─┐  ┌──▼─┐  ┌──▼─┐  ┌──▼─┐  ┌──▼─┐  ┌──▼─┐
  │Auth│  │Ident│  │OAuth│  │Pol │  │Org │  │Aud │
  │x3  │  │x2  │  │x2  │  │x2  │  │x2  │  │x2  │
  └──┬─┘  └──┬─┘  └──┬─┘  └──┬─┘  └──┬─┘  └──┬─┘
     │       │       │       │       │       │
     └───────┴───────┴───┬───┴───────┴───────┘
                         │
          ┌──────────────┼──────────────┐
          │              │              │
     ┌────▼────┐  ┌─────▼─────┐  ┌────▼─────┐
     │PostgreSQL│  │  Redis    │  │  NATS    │
     │ Primary  │  │ Sentinel  │  │ Cluster  │
     │ + Replica│  │ (3 nodes) │  │ (3 nodes)│
     └─────────┘  └───────────┘  └──────────┘
```

---

## Stateless Services

All 7 GGID microservices are **stateless** — no session data stored in process memory. Sessions live in Redis, JWTs are self-contained. This means any instance can serve any request.

### Horizontal Scaling

```bash
# Docker Compose
docker compose up --scale gateway=3 --scale auth=3 --scale identity=2

# Kubernetes
kubectl scale deployment ggid-gateway --replicas=3
kubectl scale deployment ggid-auth --replicas=3
```

### Recommended Replica Counts

| Service | Min Replicas | Production | CPU-Intensive? |
|---------|:------------:|:----------:|:--------------:|
| Gateway | 2 | 3+ | No (proxy) |
| Auth | 2 | 3+ | Yes (Argon2id) |
| Identity | 1 | 2+ | No |
| OAuth | 1 | 2 | No |
| Policy | 1 | 2 | Moderate |
| Org | 1 | 2 | No |
| Audit | 1 | 2 | No |

---

## Database HA

### Streaming Replication

```
PostgreSQL Primary (read/write)
    │
    ├── Sync Replica (sync_commit=on)    ← failover target
    └── Async Replica (sync_commit=off)  ← read replica for queries
```

### Failover

Use **Patroni** or **Stolon** for automatic failover:

```yaml
# patroni.yml
patroni:
  scope: ggid-pg
  postgresql:
    use_slots: true
    parameters:
      wal_level: replica
      max_wal_senders: 5
      synchronous_commit: "on"
  tags:
    nofailover: false
    clonefrom: true
```

### RPO/RTO

| Metric | Target | How |
|--------|--------|-----|
| RPO | < 1s | Synchronous streaming replication |
| RTO | < 30s | Patroni automatic failover |

### Read Replicas

Route read-heavy queries (audit, user listing) to replicas:

```yaml
# Application config
database:
  primary: "postgres://...primary:5432/..."
  replica: "postgres://...replica:5432/..."
```

---

## Redis HA

### Redis Sentinel

```
Redis Master ←──── Sentinel #1
     │          ──── Sentinel #2
     │          ──── Sentinel #3
     ▼
Redis Replica
```

3 Sentinel nodes monitor the Redis master. If master fails, Sentinels elect the replica as new master.

### Configuration

```bash
# sentinel.conf
sentinel monitor ggid-redis redis-master 6379 2
sentinel down-after-milliseconds ggid-redis 5000
sentinel failover-timeout ggid-redis 30000
sentinel parallel-syncs ggid-redis 1
```

### Client Configuration

```go
// Go client with Sentinel support
client := redis.NewFailoverClient(&redis.FailoverOptions{
    MasterName: "ggid-redis",
    SentinelAddrs: []string{":26379", ":26380", ":26381"},
    Password: "strong-password",
})
```

---

## NATS HA

### NATS Cluster (RAFT)

```bash
# 3-node NATS cluster
nats-server --cluster_name ggid \
  --routes=nats://nats-1:6222,nats://nats-2:6222,nats://nats-3:6222 \
  --jetstream --store_dir /data/nats
```

JetStream data is replicated via RAFT consensus across all 3 nodes. Tolerates 1 node failure.

---

## Load Balancer Configuration

### nginx

```nginx
upstream ggid_gateway {
    least_conn;
    server gateway-1:8080 max_fails=3 fail_timeout=30s;
    server gateway-2:8080 max_fails=3 fail_timeout=30s;
    server gateway-3:8080 max_fails=3 fail_timeout=30s;
}

server {
    listen 443 ssl http2;
    ssl_certificate /etc/ssl/ggid.crt;
    ssl_certificate_key /etc/ssl/ggid.key;

    location / {
        proxy_pass http://ggid_gateway;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Health Check

```nginx
location /healthz {
    proxy_pass http://ggid_gateway/healthz;
    access_log off;
}
```

---

## Zero-Downtime Deployment

### Rolling Update (Kubernetes)

```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      containers:
        - name: ggid-auth
          lifecycle:
            preStop:
              exec:
                command: ["sleep", "5"]  # drain connections
          readinessProbe:
            httpGet:
              path: /healthz
              port: 9001
            initialDelaySeconds: 3
            periodSeconds: 2
```

### Blue-Green Deployment

```
Blue (active) ←── 100% traffic
Green (idle)  ←── 0% traffic

1. Deploy to Green
2. Run health checks on Green
3. Switch traffic: Blue→Green
4. Keep Blue as rollback target
```

### Database Migrations

Use backward-compatible migrations:

1. **Expand:** Add new column (nullable, no default)
2. **Migrate:** Deploy new code that writes to both old + new columns
3. **Contract:** Remove old column after all instances updated

---

## Disaster Recovery

See [Disaster Recovery Guide](./disaster-recovery.md) for RPO/RTO targets, backup strategy, cross-region replication, and DR runbook.
