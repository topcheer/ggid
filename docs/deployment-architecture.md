# GGID Deployment Architecture

Production topology guide covering container layout, clustering, load
balancing, and horizontal scaling for each GGID service.

---

## Table of Contents

- [Overview](#overview)
- [Container Layout](#container-layout)
- [PostgreSQL High Availability](#postgresql-high-availability)
- [Redis HA (Sentinel/Cluster)](#redis-ha-sentinelcluster)
- [NATS JetStream Clustering](#nats-jetstream-clustering)
- [Load Balancer Configuration](#load-balancer-configuration)
- [Horizontal Scaling Per Service](#horizontal-scaling-per-service)
- [Network Topology](#network-topology)

---

## Overview

GGID is designed for horizontal scaling. All services are stateless and can
be scaled independently based on load.

```
                    ┌─────────────────┐
                    │   Cloud LB /    │
                    │   DNS (weighted)│
                    └───────┬─────────┘
                            │
              ┌─────────────┼─────────────┐
              │             │             │
        ┌─────▼──┐   ┌─────▼──┐   ┌─────▼──┐
        │Gateway │   │Gateway │   │Gateway │
        │  #1    │   │  #2    │   │  #3    │
        │:8080   │   │:8080   │   │:8080   │
        └────┬───┘   └────┬───┘   └────┬───┘
             │            │            │
     ┌───────┼────────────┼────────────┼───────┐
     │       │            │            │       │
  ┌──▼──┐ ┌─▼──┐ ┌──▼──┐ ┌─▼──┐ ┌──▼──┐ ┌─▼──┐
  │Auth │ │Iden│ │OAuth│ │Poli│ │Org │ │Audt│
  │  x2 │ │ x2 │ │  x2 │ │ x2 │ │ x2 │ │ x2 │
  └──┬──┘ └─┬──┘ └──┬──┘ └─┬──┘ └──┬──┘ └─┬──┘
     │       │       │       │       │       │
     └───────┴───────┴───┬───┴───────┴───────┘
                         │
              ┌──────────┼──────────┐
              │          │          │
        ┌─────▼──┐ ┌────▼──┐ ┌────▼────┐
        │PostgreSQL│ │ Redis │ │  NATS   │
        │Primary   │ │Sentinel│ │ Cluster │
        │+Replica  │ │ +2    │ │ 3 nodes │
        └──────────┘ └───────┘ └─────────┘
```

---

## Container Layout

### Recommended Production Replicas

| Service | Min Replicas | Recommended | CPU/Pod | Memory/Pod | Scaling Trigger |
|---------|-------------|-------------|---------|------------|-----------------|
| Gateway | 2 | 3 | 0.5 | 256MB | RPS > 500 |
| Auth | 2 | 3 | 1.0 | 512MB | Login RPS > 100 |
| Identity | 2 | 2 | 0.5 | 256MB | User CRUD RPS |
| OAuth | 2 | 2 | 0.5 | 256MB | Token RPS |
| Policy | 2 | 2 | 0.5 | 256MB | Policy check latency |
| Org | 1 | 2 | 0.25 | 128MB | Org CRUD RPS |
| Audit | 2 | 3 | 0.5 | 256MB | Events/sec |
| Console | 2 | 2 | 0.25 | 128MB | Static (CDN offload) |

### Kubernetes Deployment Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ggid-gateway
  namespace: ggid
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 1
  selector:
    matchLabels:
      app: ggid-gateway
  template:
    metadata:
      labels:
        app: ggid-gateway
    spec:
      containers:
        - name: gateway
          image: ghcr.io/ggid/gateway:latest
          ports:
            - containerPort: 8080
          resources:
            requests:
              cpu: 250m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 256Mi
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 10
          env:
            - name: AUTH_SERVICE_URL
              value: "ggid-auth:9001"
            - name: REDIS_URL
              value: "redis-sentinel:26379"
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: ggid-db-secret
                  key: url
```

### HorizontalPodAutoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: ggid-gateway-hpa
  namespace: ggid
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ggid-gateway
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

---

## PostgreSQL High Availability

### Primary + Streaming Replica

```
┌─────────────────────────────────────────┐
│         PostgreSQL HA Cluster           │
├─────────────────────────────────────────┤
│                                         │
│   Primary (read-write)                  │
│   ├── WAL streaming ──┐                 │
│   │                   │                 │
│   │              ┌────▼────┐            │
│   │              │ Replica │            │
│   │              │(read-   │            │
│   │              │ only)   │            │
│   │              └─────────┘            │
│   │                                     │
│   └── PgBouncer (connection pooler)     │
│                                         │
└─────────────────────────────────────────┘
```

### Streaming Replication Setup

```ini
# postgresql.conf (primary)
wal_level = replica
max_wal_senders = 10
wal_keep_size = 1024  # MB
hot_standby = on

# pg_hba.conf (primary)
host replication replicator 10.0.0.0/8 md5
```

```bash
# Set up replica (initial base backup)
pg_basebackup -h primary-host -U replicator -D /var/lib/postgresql/data -X stream -R
```

### PgBouncer Configuration

```ini
[databases]
ggid = host=postgres-primary port=5432 dbname=ggid

pool_mode = transaction
max_client_conn = 500
default_pool_size = 20
```

> **Critical:** GGID uses `SET LOCAL app.tenant_id` for RLS, which requires
> `transaction` pool mode. Never use `session` mode with PgBouncer.

### Automatic Failover (Patroni)

```yaml
# patroni.yml
scope: ggid-pg
name: pg-node-1

restapi:
  listen: 0.0.0.0:8008

etcd:
  hosts: etcd-1:2379,etcd-2:2379,etcd-3:2379

bootstrap:
  dcs:
    ttl: 30
    loop_wait: 10
    maximum_lag_on_failover: 1048576
    postgresql:
      use_pg_rewind: true
      parameters:
        wal_level: replica
        hot_standby: on
```

---

## Redis HA (Sentinel/Cluster)

### Redis Sentinel (Failover)

```ini
# sentinel.conf (3 sentinel nodes)
sentinel monitor ggid redis-primary 6379 2
sentinel down-after-milliseconds ggid 3000
sentinel failover-timeout ggid 30000
sentinel parallel-syncs ggid 1
```

### Go Client with Sentinel

```go
rdb := redis.NewFailoverClient(&redis.FailoverOptions{
    MasterName:    "ggid",
    SentinelAddrs: []string{":26379", ":26380", ":26381"},
    Password:      os.Getenv("REDIS_PASSWORD"),
    PoolSize:      25,
})
```

### Redis Cluster (Sharding)

For >50GB data or >100k ops/sec:

```go
rdb := redis.NewClusterClient(&redis.ClusterOptions{
    Addrs:     []string{":7000", ":7001", ":7002", ":7003", ":7004", ":7005"},
    Password:  os.Getenv("REDIS_PASSWORD"),
    PoolSize:  25,
    ReadOnly:  true,
})
```

---

## NATS JetStream Clustering

### 3-Node Cluster

```conf
# nats-server.conf (node 1)
port: 4222
http_port: 8222

jetstream {
    store_dir: /data
    max_file_store: 10GB
}

cluster {
    name: ggid-cluster
    routes: [
        nats-route://nats-2:4222
        nats-route://nats-3:4222
    ]
}
```

### Stream with Replication

```bash
# Create stream replicated to all 3 nodes (R=3)
nats stream add GGID_EVENTS \
    --subjects "ggid.events.>" \
    --storage file \
    --replicas 3 \
    --max-age 7d
```

---

## Load Balancer Configuration

### NGINX Load Balancer

```nginx
upstream ggid_gateway {
    least_conn;
    server gateway-1:8080 max_fails=3 fail_timeout=30s;
    server gateway-2:8080 max_fails=3 fail_timeout=30s;
    server gateway-3:8080 max_fails=3 fail_timeout=30s;
    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name iam.example.com;

    ssl_certificate /etc/ssl/ggid.crt;
    ssl_certificate_key /etc/ssl/ggid.key;

    # Security headers
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload";
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;

    location / {
        proxy_pass http://ggid_gateway;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_http_version 1.1;
        proxy_set_header Connection "";

        # WebSocket / SSE support
        proxy_buffering off;
        proxy_read_timeout 300s;
    }

    # Health check endpoint
    location /healthz {
        proxy_pass http://ggid_gateway/healthz;
        access_log off;
    }
}
```

### Kubernetes Ingress (NGINX Ingress Controller)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ggid-ingress
  namespace: ggid
  annotations:
    nginx.ingress.kubernetes.io/backend-protocol: HTTP
    nginx.ingress.kubernetes.io/proxy-buffering: "off"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "300"
    nginx.ingress.kubernetes.io/configuration-snippet: |
      add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload";
spec:
  tls:
    - hosts: [iam.example.com]
      secretName: ggid-tls
  rules:
    - host: iam.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: ggid-gateway
                port:
                  number: 8080
```

---

## Horizontal Scaling Per Service

### Gateway (Scale by RPS)

```bash
# Target: <500 RPS per pod
kubectl scale deployment ggid-gateway --replicas=5

# Or use HPA with custom metric
kubectl autoscale deployment ggid-gateway \
    --cpu-percent=70 --min=3 --max=10
```

### Auth (Scale by Login Rate)

```bash
# Auth is CPU-bound (bcrypt/Argon2id hashing)
# Target: <100 logins/sec per pod
kubectl scale deployment ggid-auth --replicas=4
```

### Audit (Scale by Event Rate)

```bash
# Audit needs high throughput for NATS publishing
# Scale consumers to keep up with event stream
kubectl scale deployment ggid-audit --replicas=3
```

### Scaling Database Connections

Total DB connections = sum of all pods × pool size:

```
3 Gateway × 10  = 30
3 Auth     × 15  = 45
2 Identity × 10  = 20
2 OAuth    × 10  = 20
2 Policy   × 5   = 10
2 Org      × 5   = 10
2 Audit    × 10  = 20
─────────────────────
Total: 155 connections (without PgBouncer)

With PgBouncer (pool_mode=transaction):
  PgBouncer → PostgreSQL: 20-50 server connections
  PgBouncer → Clients: 500+ client connections
```

---

## Network Topology

### Production VPC Layout

```
VPC 10.0.0.0/16
├── Public Subnet (10.0.1.0/24)
│   ├── NGINX Load Balancer
│   └── Bastion Host
│
├── Application Subnet (10.0.10.0/24) — private
│   ├── Gateway pods (3)
│   ├── Auth pods (3)
│   ├── Identity pods (2)
│   ├── OAuth pods (2)
│   ├── Policy pods (2)
│   ├── Org pods (2)
│   └── Audit pods (3)
│
├── Data Subnet (10.0.20.0/24) — private, no internet
│   ├── PostgreSQL Primary + Replica
│   ├── PgBouncer
│   ├── Redis Sentinel (3 nodes)
│   └── NATS Cluster (3 nodes)
│
└── Management Subnet (10.0.30.0/24) — private
    ├── Monitoring (Prometheus + Grafana)
    ├── Jaeger
    └── ELK Stack
```

### Security Groups

| Source | Destination | Port | Protocol | Purpose |
|--------|-------------|------|----------|---------|
| Internet | Public LB | 443 | TCP | HTTPS |
| Public LB | App Subnet | 8080 | TCP | Gateway |
| App Subnet | Data Subnet | 5432 | TCP | PostgreSQL |
| App Subnet | Data Subnet | 6432 | TCP | PgBouncer |
| App Subnet | Data Subnet | 6379 | TCP | Redis |
| App Subnet | Data Subnet | 4222 | TCP | NATS |
| App Subnet | Mgmt Subnet | 9090 | TCP | Prometheus scrape |
| App Subnet | App Subnet | 9001 | TCP | Auth gRPC |
| App Subnet | App Subnet | 50051 | TCP | Identity gRPC |
| App Subnet | App Subnet | 9070-9072 | TCP | Policy/Org/Audit gRPC |

> **Rule:** Data subnet has no internet access (egress denied). App subnet
> can only reach data subnet and mgmt subnet. Internet egress only via NAT
> gateway for Docker image pulls.

---

## Database Strategy: Shared vs Per-Service

GGID uses a **shared database** approach where all 7 microservices connect to
the same PostgreSQL instance. Tenant isolation is enforced via RLS.

### Why Shared Database

| Aspect | Shared DB (GGID) | Per-Service DB |
|--------|------------------|----------------|
| Cross-service queries | Easy (JOINs) | Requires API calls or data sync |
| Migrations | Single migration run | Coordinated per-service |
| Transactions | ACID across services | Distributed transactions (hard) |
| Connection pool | Shared (PgBouncer) | Each service has own pool |
| Complexity | Low | High |
| Scale | Vertical + RLS | Horizontal per service |
| Best for | Microservices with shared data | Truly independent services |

GGID services share `users`, `roles`, `audit_events` tables. A per-service
DB would require expensive cross-service API calls for every auth check.

### Connection Topology

```
                    PgBouncer (transaction pooling)
                           │
                    PostgreSQL (shared)
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                  │
    tenant_id=A       tenant_id=B        tenant_id=C
    (RLS filtered)    (RLS filtered)     (RLS filtered)
```

### When to Split

Split databases only if:
- A single service has dramatically different write patterns (e.g., audit at 10k events/sec)
- Data residency requires separate storage for specific data types
- A service needs a different DB engine (e.g., time-series for metrics)

GGID Audit Service can optionally use a separate database for high-volume
event ingestion while maintaining a summary table in the shared DB.

---

## Redis Patterns

GGID uses Redis for three distinct purposes, each with different access patterns:

### 1. Session Store

```
Key: tid:{tenant_id}:session:{session_id}
Value: {user_id, device, ip, expires_at}
TTL: 24h (sliding expiration)
Pattern: Read-heavy (every API call), Write on login/logout
```

### 2. Rate Limiting (Token Bucket)

```
Key: tid:{tenant_id}:rl:{endpoint}:{ip}
Value: Sorted set of request timestamps (sliding window)
TTL: 60s (window duration)
Pattern: Read + Write on every API call (ZADD + ZCARD via Lua script)
```

### 3. Policy Cache

```
Key: tid:{tenant_id}:policy:{user_id}:{resource}:{action}
Value: allow/deny + computed_at timestamp
TTL: 5min
Pattern: Read-heavy (95%+ hit rate), Write on policy change (cache invalidation)
```

### Redis Configuration

| Setting | Session | Rate Limit | Policy Cache |
|---------|---------|------------|--------------|
| Eviction | TTL | TTL | TTL + maxmemory-policy |
| Persistence | RDB (optional) | AOF (recommended) | Not needed |
| Consistency | Strong (read-after-write) | Strong (Lua atomic) | Eventually consistent |

---

## References

- [Deployment Guide](./deployment-guide.md) — Step-by-step deployment
- [High Availability](./high-availability.md) — HA patterns
- [Multi-Tenancy Guide](./multi-tenancy-guide.md) — RLS deep dive
- [Security Hardening](./security-hardening.md) — Production security
