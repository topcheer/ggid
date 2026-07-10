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

---

## Pod Disruption Budgets

Ensure minimum available replicas during voluntary disruptions (node drain, cluster upgrade).

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: ggid-gateway-pdb
  namespace: ggid
spec:
  minAvailable: 2              # Always keep at least 2 pods running
  selector:
    matchLabels:
      app.kubernetes.io/name: gateway
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: ggid-auth-pdb
  namespace: ggid
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app.kubernetes.io/name: auth
```

### Recommended PDB Settings

| Service | minAvailable | Rationale |
|---------|-------------|-----------|
| Gateway | 2 | Single point of entry; always need redundancy |
| Auth | 2 | Login traffic cannot stop |
| Identity | 1 | Lower traffic, 1 replica sufficient during disruption |
| OAuth | 1 | Only used for OAuth flows |
| Policy | 1 | Fast in-process check |
| Org | 1 | Lower traffic volume |
| Audit | 1 | Async, can tolerate brief gaps |

---

## Health Probes (Kubernetes)

### Liveness Probe

Detects deadlocked or hung processes. If liveness fails, the pod is restarted.

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  failureThreshold: 3      # Restart after 3 consecutive failures (30s)
  timeoutSeconds: 3
```

### Readiness Probe

Determines if the pod can serve traffic. If readiness fails, the pod is removed
from the Service's endpoint list but NOT restarted.

```yaml
readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 2
  timeoutSeconds: 2
```

### Startup Probe

For services with slow initialization (e.g., DB connection, key loading):

```yaml
startupProbe:
  httpGet:
    path: /healthz
    port: 8080
  failureThreshold: 30
  periodSeconds: 5          # Allow up to 150 seconds for startup
```

---

## Circuit Breakers

The Gateway implements circuit breakers for each backend service to prevent
cascade failures.

```mermaid
graph LR
    GW[Gateway] --> CB{Circuit Breaker}
    CB -->|Closed| Backend[Backend Service<br/>requests flow normally]
    CB -->|Open| Fail[Fail Fast<br/>return 503 immediately]
    CB -->|Half-Open| Probe[Test with 1 request<br/>if ok → Closed]
    Backend -->|5 failures<br/>in 10s| CB
    CB -->|after 30s| Probe

    style CB fill:#e74c3c,color:#fff
    style Backend fill:#27ae60,color:#fff
    style Fail fill:#95a5a6,color:#fff
```

### Circuit Breaker Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `CB_ENABLED` | true | Enable circuit breakers |
| `CB_FAILURE_THRESHOLD` | 5 | Failures before opening circuit |
| `CB_RESET_TIMEOUT` | 30s | Time before half-open probe |
| `CB_HALF_OPEN_MAX` | 1 | Max requests in half-open state |
| `CB_WINDOW` | 10s | Sliding window for failure counting |

```bash
# Environment variables
CB_ENABLED=true
CB_FAILURE_THRESHOLD=5
CB_RESET_TIMEOUT=30s
```

---

## Graceful Shutdown

When a pod receives SIGTERM (e.g., during rolling update):

```mermaid
sequenceDiagram
    participant K8s as Kubernetes
    participant Pod as Service Pod
    participant LB as Load Balancer

    K8s->>Pod: SIGTERM
    Pod->>Pod: Stop accepting new connections
    K8s->>LB: Remove from endpoints (concurrent)
    Pod->>Pod: Wait for in-flight requests (grace period: 30s)
    Pod->>Pod: Close DB/Redis/NATS connections
    Pod->>K8s: Exit 0
```

### Grace Period Configuration

```yaml
spec:
  terminationGracePeriodSeconds: 45   # Allow 45s for in-flight requests
  containers:
    - name: gateway
      lifecycle:
        preStop:
          exec:
            command: ["sh", "-c", "sleep 10"]  # Give LB time to deregister
```

---

## Multi-Region Deployment

For active-active multi-region deployments:

```mermaid
graph TB
    subgraph "Region: us-east-1"
        GW1[Gateway ×3] --> PG1[(PG Primary)]
        GW1 --> RD1[(Redis Primary)]
        GW1 --> NT1[(NATS ×3)]
    end

    subgraph "Region: eu-west-1"
        GW2[Gateway ×3] --> PG2[(PG Replica → Primary)]
        GW2 --> RD2[(Redis Replica)]
        GW2 --> NT2[(NATS ×3)]
    end

    subgraph "Replication"
        PG1 -->|async streaming| PG2
        RD1 -->|replication| RD2
        NT1 -->|leaf node| NT2
    end

    DNS[GeoDNS<br/>Route 53] --> GW1
    DNS --> GW2
```

### Failover Procedure

1. **Database:** Promote eu-west-1 replica to primary (`pg_ctl promote`)
2. **Redis:** Promote eu-west-1 replica, update Sentinel
3. **NATS:** Update DNS to point to eu-west-1 cluster
4. **DNS:** Switch GeoDNS weight to eu-west-1 (100%)
5. **Verify:** Run smoke tests against new region
6. **Reroute:** Application clients reconnect via DNS TTL

---

## PgBouncer Connection Pooling

PgBouncer sits between GGID services and PostgreSQL, pooling connections to
reduce connection overhead.

### Architecture

```
GGID Services (each with 25 pool conns)
    │
    ├── Gateway ×3 ──┐
    ├── Auth ×3 ─────┤
    ├── Identity ×2 ─┤── PgBouncer (100 server conns) ── PostgreSQL
    ├── Policy ×2 ───┤
    ├── Org ×2 ──────┤
    └── Audit ×2 ────┘

Without PgBouncer: 14 pods × 25 conns = 350 PostgreSQL connections
With PgBouncer: 350 client conns → pooled to 100 server conns
```

### PgBouncer Configuration

```ini
# /etc/pgbouncer/pgbouncer.ini
[databases]
ggid = host=postgres-primary port=5432 dbname=ggid

dbname = ggid
listen_port = 6432
listen_addr = 0.0.0.0

auth_type = scram-sha-256
auth_file = /etc/pgbouncer/userlist.txt

; Connection pool sizing
pool_mode = transaction
max_client_conn = 500
default_pool_size = 20
min_pool_size = 5
reserve_pool_size = 5
reserve_pool_timeout = 3

; Timeouts
server_idle_timeout = 300
server_lifetime = 3600
query_wait_timeout = 30

; Logging
log_connections = 1
log_disconnections = 1
log_pooler_errors = 1
```

### Transaction Mode and Session State

GGID uses `SET LOCAL app.tenant_id` for RLS, which requires transaction-mode
pooling. `SET LOCAL` is scoped to the current transaction and automatically
cleared on COMMIT/ROLLBACK:

```go
// Correct: SET LOCAL within transaction
tx, _ := pool.Begin(ctx)
defer tx.Rollback(ctx)

tx.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID)
// Queries within this tx are tenant-scoped
tx.Commit(ctx)
// tenant_id is now cleared
```

> **Important:** Never use `SET app.tenant_id` (without LOCAL) — it persists
> across transactions and breaks with PgBouncer connection reuse.

### Deploy PgBouncer

```yaml
# docker-compose.yaml
pgbouncer:
  image: edoburu/pgbouncer:1.22.0
  environment:
    DB_USER: ggid
    DB_PASSWORD: ${DB_PASSWORD}
    DB_HOST: postgres
    DB_NAME: ggid
    POOL_MODE: transaction
    MAX_CLIENT_CONN: 500
    DEFAULT_POOL_SIZE: 20
  ports:
    - "6432:6432"
  depends_on:
    postgres:
      condition: service_healthy
```

Point GGID services at PgBouncer instead of PostgreSQL directly:

```bash
# Update DATABASE_URL to use PgBouncer port
DATABASE_URL=postgres://ggid:password@pgbouncer:6432/ggid?sslmode=disable

# For services using individual DB env vars:
DB_HOST=pgbouncer
DB_PORT=6432
```

---

## Redis Sentinel / Cluster

### Redis Sentinel (HA Failover)

Sentinel monitors the Redis primary and automatically promotes a replica if it
goes down.

```
┌───────────────────────────────────────────┐
│ Redis Sentinel Architecture               │
├───────────────────────────────────────────┤
│                                           │
│  Sentinel-1    Sentinel-2    Sentinel-3  │
│     │             │             │         │
│     └──── quorum ─┴──── vote ───┘         │
│                     │                     │
│               ┌─────┴─────┐               │
│               │           │               │
│          ┌────▼───┐  ┌───▼────┐           │
│          │ Redis  │  │ Redis  │           │
│          │PRIMARY │──│REPLICA │           │
│          └────────┘  └────────┘           │
└───────────────────────────────────────────┘
```

#### Sentinel Configuration

```ini
# /etc/redis/sentinel.conf
sentinel monitor ggid redis-primary 6379 2
sentinel down-after-milliseconds ggid 3000
sentinel parallel-syncs ggid 1
sentinel failover-timeout ggid 30000
```

#### Go Client with Sentinel

```go
import "github.com/redis/go-redis/v9"

rdb := redis.NewFailoverClient(&redis.FailoverOptions{
    MasterName:       "ggid",
    SentinelAddrs:    []string{":26379", ":26380", ":26381"},
    Password:         os.Getenv("REDIS_PASSWORD"),
    DB:               0,
    PoolSize:         25,
    MinIdleConns:     5,
    MaxRetryAttempts: 3,
})
```

### Redis Cluster (Sharding)

For workloads exceeding single-node Redis capacity (>50GB, >100k ops/sec):

```go
rdb := redis.NewClusterClient(&redis.ClusterOptions{
    Addrs:        []string{":7000", ":7001", ":7002", ":7003", ":7004", ":7005"},
    Password:     os.Getenv("REDIS_PASSWORD"),
    PoolSize:     25,
    MinIdleConns: 5,
    MaxRedirects: 3,
    ReadOnly:     true, // Allow reads from replicas
})
```

> **Note:** Redis Cluster uses hash-slot sharding. Multi-key operations
> (MGET, pipeline) require keys to be in the same slot. GGID's tenant-prefixed
> key naming (`tid:{tenant_id}:...`) may need hash tags: `tid:{tenant_id}:session:*`

### Choosing Sentinel vs Cluster

| Aspect | Sentinel | Cluster |
|--------|----------|--------|
| Use case | HA failover (single shard) | Horizontal scaling |
| Max data | Single node limit | Sharded across nodes |
| Complexity | Low | Medium |
| Multi-key ops | Yes (all keys on primary) | Only same hash slot |
| Best for | GGID sessions & rate limits | Large-scale multi-tenant |

---

## NATS JetStream Clustering

### JetStream Raft Consensus

JetStream uses Raft consensus for replicated streams across a NATS cluster:

```
┌───────────────────────────────────────────────────────┐
│ NATS JetStream Cluster (3 nodes)                     │
├───────────────────────────────────────────────────────┤
│                                                       │
│   Node-1  ←──raft──→  Node-2  ←──raft──→  Node-3    │
│     │                    │                    │       │
│     └─────── GGID_EVENTS stream (replicated) ──┘     │
│                                                       │
│   Stream config: R3 (replicate to 3 nodes)            │
│   Consumer: audit-consumer (durable, replicated)      │
└───────────────────────────────────────────────────────┘
```

### Cluster Configuration

```conf
# nats-server.conf (node 1 of 3)
port: 4222
http_port: 8222

jetstream {
    store_dir: /data
    max_memory_store: 1GB
    max_file_store: 10GB
}

cluster {
    name: ggid-cluster
    routes: [
        nats-route://nats-2:4222
        nats-route://nats-3:4222
    ]
}
``n
### Stream Replication

```bash
# Create stream with R=3 (replicated to all 3 nodes)
nats stream add GGID_EVENTS \
  --subjects "ggid.events.>" \
  --storage file \
  --replicas 3 \
  --retention limits \
  --max-age 7d \
  --server nats://nats-1:4222
```

### Consumer Availability

Durable consumers are also replicated. If a node fails, the consumer
continues processing from the last acknowledged message on another node.

---

## Rolling Updates

### Kubernetes Rolling Update

```bash
# Update image and trigger rolling update
kubectl set image deployment/ggid-gateway \
  gateway=ghcr.io/ggid/gateway:v1.3.1 \
  -n ggid

# Watch the rollout
kubectl rollout status deployment/ggid-gateway -n ggid

# Rollback if needed
kubectl rollout undo deployment/ggid-gateway -n ggid
```

### Rolling Update Strategy

```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0     # Never go below desired replicas
      maxSurge: 1           # Allow 1 extra pod during update
```

### Update Order (Dependency-Aware)

The correct order for rolling updates respects service dependencies:

```
1. Audit Service     (consumers, safe to update first)
2. Org Service       (no downstream dependencies)
3. Policy Service    (no downstream dependencies)
4. OAuth Service     (no downstream dependencies)
5. Identity Service  (provides data to Auth)
6. Auth Service      (depends on Identity)
7. Gateway           (depends on all services)
8. Console           (depends on Gateway)
```

### Database Migration Safety

```bash
# Before deploying new code that requires schema changes:
# 1. Run migrations (backward compatible — additive only)
bash deploy/migrate.sh

# 2. Deploy new code (reads new columns, ignores old ones)
kubectl rollout restart deployment/ggid-auth -n ggid

# 3. In a later release, remove deprecated columns
# (never in the same release as the additive migration)
```

### Session Affinity and Rolling Updates

GGID services are **stateless** — any instance can serve any request. No
session affinity (sticky sessions) is required.

- Session data lives in Redis (shared across all instances)
- JWTs are self-contained (no server-side state needed for verification)
- Rate limiting is shared via Redis token buckets
- The only consideration: in-flight requests during pod termination

```yaml
# Give in-flight requests time to complete
terminationGracePeriodSeconds: 45

preStop:
  exec:
    command: ["sh", "-c", "sleep 10"]  # LB deregister delay
```

---

## HA Monitoring Checklist

| Component | Metric to Watch | Alert Threshold |
|-----------|----------------|-----------------|
| Gateway pods | `kube_deployment_status_replicas_available` | < 2 for 2m |
| Auth pods | `kube_deployment_status_replicas_available` | < 2 for 2m |
| PostgreSQL | `pg_replication_lag_seconds` | > 10s |
| Redis | `redis_connected_clients` | < 1 for 1m |
| NATS | `nats_jetstream_stream_messages` | consumer lag > 1000 |
| Load balancer | `nginx_upstream_health_status` | any upstream down |

```promql
# Alert: insufficient replicas
kube_deployment_status_replicas_available{deployment="ggid-gateway"} < 2

# Alert: replication lag
pg_replication_lag_seconds > 10

# Alert: NATS consumer lag
nats_jetstream_consumer_num_ack_pending > 100
```

---

## Blue-Green Deployment

Blue-green deployment eliminates downtime by running two identical environments
and switching traffic between them.

### Architecture

```
                   ┌──────────────────┐
                   │  Load Balancer   │
                   │  (weighted DNS)  │
                   └────┬───────┬─────┘
                        │       │
          100% traffic  │       │  0% traffic
                   ┌────▼──┐ ┌──▼──────┐
                   │ BLUE  │ │  GREEN  │
                   │ v1.3  │ │  v1.4   │
                   │ (live)│ │ (idle)  │
                   └───┬───┘ └────┬────┘
                       │          │
                   ┌───▼──────────▼───┐
                   │  Shared Database  │
                   │  (backward compat)│
                   └──────────────────┘
```

### Kubernetes Blue-Green

```bash
# 1. Deploy green (new version) alongside blue
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ggid-gateway-green
spec:
  replicas: 3
  selector:
    matchLabels: {app: ggid-gateway, slot: green}
  template:
    metadata:
      labels: {app: ggid-gateway, slot: green}
    spec:
      containers:
        - name: gateway
          image: ghcr.io/ggid/gateway:v1.4.0
          readinessProbe:
            httpGet: {path: /readyz, port: 8080}
EOF

# 2. Wait for green to be healthy
kubectl rollout status deployment/ggid-gateway-green
kubectl wait --for=condition=available deployment/ggid-gateway-green --timeout=120s

# 3. Switch service to green
kubectl patch service ggid-gateway -p '{"spec":{"selector":{"slot":"green"}}}'

# 4. Verify traffic on green
curl -s https://iam.example.com/healthz

# 5. Scale down blue (keep for rollback)
kubectl scale deployment ggid-gateway-blue --replicas=0

# Or rollback if issues:
# kubectl patch service ggid-gateway -p '{"spec":{"selector":{"slot":"blue"}}}'
```

### Blue-Green Database Considerations

Database schema changes must be **backward compatible** during blue-green:

| Migration Type | Safe? | Example |
|----------------|-------|---------|
| Add column (nullable) | Yes | `ALTER TABLE users ADD COLUMN phone TEXT` |
| Add table | Yes | `CREATE TABLE notifications (...)` |
| Add index (CONCURRENT) | Yes | `CREATE INDEX CONCURRENTLY ...` |
| Remove column | No | Blue code still reads it |
| Rename column | No | Blue code uses old name |
| Change type | No | May break old code |

**Strategy for breaking changes:** Two-phase deploy over two releases:

```
Release 1: Add new column, keep old column (dual-write)
Release 2: Switch reads to new column, drop old column
```

---

## Zero-Downtime Database Migration

### Online Schema Change Process

```bash
# 1. Add nullable column (instant, no lock)
psql -c "ALTER TABLE users ADD COLUMN phone VARCHAR(20);"

# 2. Backfill existing rows in batches (no long lock)
psql <<'SQL'
DO $$
DECLARE
  batch_size INT := 1000;
  rows_affected INT := 1;
BEGIN
  WHILE rows_affected > 0 LOOP
    UPDATE users
    SET phone = (
      SELECT phone_number FROM legacy_phones
      WHERE legacy_phones.user_id = users.id
    )
    WHERE phone IS NULL
      AND id IN (
        SELECT id FROM users WHERE phone IS NULL LIMIT batch_size
      );
    GET DIAGNOSTICS rows_affected = ROW_COUNT;
    PERFORM pg_sleep(0.1);  -- 100ms between batches
  END LOOP;
END $$;
SQL

# 3. Add NOT NULL constraint (with default)
psql -c "ALTER TABLE users ALTER COLUMN phone SET DEFAULT '';"
psql -c "ALTER TABLE users ALTER COLUMN phone SET NOT NULL;"

# 4. Create index concurrently (no write lock)
psql -c "CREATE INDEX CONCURRENTLY idx_users_phone ON users(phone);"
```

### Migration Safety Rules

1. **Never use `ALTER TABLE ... TYPE`** that requires a rewrite — add a new column instead
2. **Never drop a column in the same release** that stops using it
3. **Never add a NOT NULL column without a default** — use nullable first, backfill, then set NOT NULL
4. **Always use `CREATE INDEX CONCURRENTLY`** — non-concurrent index creation locks the table
5. **Always test on staging** with production data volume

### Migration Verification

```bash
# Verify migration applied
psql -c "SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = 'users';"

# Check for long-running queries during migration
psql -c "SELECT pid, now() - query_start AS duration, query FROM pg_stat_activity WHERE state = 'active' AND now() - query_start > interval '5 seconds';"

# Check table locks
psql -c "SELECT relation::regclass, mode, pid FROM pg_locks WHERE NOT granted;"
```

---

## Canary Deployment

Canary releases route a small percentage of traffic to the new version,
monitoring error rates before full rollout.

### Kubernetes Canary (Weighted Service)

```bash
# 1. Deploy canary (1 replica of new version)
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ggid-gateway-canary
spec:
  replicas: 1
  selector:
    matchLabels: {app: ggid-gateway, track: canary}
  template:
    metadata:
      labels: {app: ggid-gateway, track: canary}
    spec:
      containers:
        - name: gateway
          image: ghcr.io/ggid/gateway:v1.4.0
EOF

# 2. Route 10% traffic to canary via Istio / NGINX weighted routing
kubectl apply -f - <<EOF
apiVersion: networking.istio.io/v1
kind: VirtualService
metadata:
  name: ggid-gateway
spec:
  http:
    - route:
        - destination: {host: ggid-gateway-stable, weight: 90}
        - destination: {host: ggid-gateway-canary, weight: 10}
EOF

# 3. Monitor error rate for 15 minutes
watch 'kubectl exec ggid-gateway-canary -- curl -s localhost:8080/metrics | grep error_rate'

# 4. Promote canary (if healthy)
kubectl patch deployment ggid-gateway-stable \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"gateway","image":"ghcr.io/ggid/gateway:v1.4.0"}]}}}}'

# 5. Remove canary
kubectl delete deployment ggid-gateway-canary
kubectl delete virtualservice ggid-gateway

# Or rollback (if issues):
# kubectl delete deployment ggid-gateway-canary
# kubectl delete virtualservice ggid-gateway
```

### Canary Promotion Criteria

| Metric | Threshold | Action |
|--------|-----------|--------|
| Error rate | < 1% | Promote |
| p95 latency | < 100ms | Promote |
| Crash rate | 0 | Promote |
| Error rate | 1-5% | Hold (investigate) |
| Error rate | > 5% | Rollback immediately |

### Automated Canary with Flagger

```yaml
# flagger.yaml
apiVersion: flagger.app/v1
kind: Canary
metadata:
  name: ggid-gateway
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ggid-gateway
  service:
    port: 8080
  analysis:
    interval: 1m
    threshold: 5
    maxWeight: 50
    stepWeight: 10
    metrics:
      - name: error-rate
        threshold: 1
        query: |
          sum(rate(http_requests_total{status=~"5.."}[1m])) /
          sum(rate(http_requests_total[1m]))
```

Flagger automatically promotes or rolls back based on error rate.
