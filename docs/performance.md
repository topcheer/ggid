# GGID Performance Tuning Guide

Guide to optimizing GGID for high-throughput production workloads.

---

## Table of Contents

- [Performance Baselines](#performance-baselines)
- [Database Indexing](#database-indexing)
- [Connection Pool Configuration](#connection-pool-configuration)
- [Redis Caching Strategy](#redis-caching-strategy)
- [NATS JetStream Tuning](#nats-jetstream-tuning)
- [Gateway Connection Reuse](#gateway-connection-reuse)
- [Go Runtime Tuning](#go-runtime-tuning)
- [pprof Analysis](#pprof-analysis)
- [Load Testing](#load-testing)
- [Monitoring](#monitoring)

---

## Performance Baselines

Expected single-instance performance (no tuning, Docker Compose, M2 Mac):

| Operation | p50 | p95 | p99 |
|-----------|-----|-----|-----|
| Login | 8ms | 25ms | 50ms |
| JWT verify (cached) | 0.1ms | 0.5ms | 1ms |
| JWT verify (JWKS fetch) | 2ms | 10ms | 20ms |
| List users (50 results) | 5ms | 15ms | 30ms |
| Policy check | 3ms | 10ms | 20ms |
| Audit query (100 events) | 10ms | 30ms | 60ms |

Target after tuning (production hardware, 4 cores, 8GB RAM):

| Operation | p50 | p95 | p99 |
|-----------|-----|-----|-----|
| Login | 2ms | 5ms | 10ms |
| JWT verify (cached) | 0.05ms | 0.2ms | 0.5ms |
| List users | 1ms | 3ms | 8ms |
| Policy check | 1ms | 3ms | 6ms |

---

## Database Indexing

### Essential Indexes

Every multi-tenant table should have these indexes:

```sql
-- Primary lookup indexes (tenant-scoped)
CREATE INDEX idx_users_tenant_username ON users (tenant_id, username);
CREATE INDEX idx_users_tenant_email ON users (tenant_id, email);
CREATE INDEX idx_users_tenant_status ON users (tenant_id, status);

CREATE INDEX idx_roles_tenant_key ON roles (tenant_id, key);

CREATE INDEX idx_orgs_tenant_parent ON organizations (tenant_id, parent_id);

CREATE INDEX idx_audit_tenant_time ON audit_events (tenant_id, created_at DESC);
CREATE INDEX idx_audit_tenant_action ON audit_events (tenant_id, action);
CREATE INDEX idx_audit_tenant_actor ON audit_events (tenant_id, actor_id);
```

### Analyze Query Performance

```sql
-- Enable pg_stat_statements
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Find slowest queries
SELECT query, calls, mean_exec_time, total_exec_time, rows
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 20;

-- Check index usage
SELECT relname, indexrelname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- Find unused indexes (candidates for removal)
SELECT relname, indexrelname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0
ORDER BY pg_relation_size(indexrelid) DESC;

-- Check table bloat
SELECT relname, n_live_tup, n_dead_tup,
       round(n_dead_tup::numeric / NULLIF(n_live_tup, 0) * 100, 2) AS dead_pct
FROM pg_stat_user_tables
WHERE n_live_tup > 0
ORDER BY dead_pct DESC;
```

### Partitioning for Large Tables

For audit_events exceeding 10M rows, partition by time range:

```sql
CREATE TABLE audit_events (
    id UUID DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    -- ... other columns
) PARTITION BY RANGE (created_at);

-- Monthly partitions
CREATE TABLE audit_events_2024_01 PARTITION OF audit_events
  FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE audit_events_2024_02 PARTITION OF audit_events
  FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
```

Automate partition creation with `pg_partman` extension.

### Vacuum Tuning

```conf
# postgresql.conf
autovacuum = on
autovacuum_max_workers = 6
autovacuum_naptime = 30s
autovacuum_vacuum_threshold = 50
autovacuum_vacuum_scale_factor = 0.05  -- vacuum when 5% dead tuples
maintenance_work_mem = 512MB
```

---

## Connection Pool Configuration

### pgx Pool Settings

Each service uses pgx v5's connection pool. Tune these parameters:

```go
// In your repository initialization
config, _ := pgxpool.ParseConfig(databaseURL)
config.MaxConns = 20              // max connections per service instance
config.MinConns = 5               // keep warm connections
config.MaxConnLifetime = 30 * time.Minute
config.MaxConnIdleTime = 5 * time.Minute
config.HealthCheckPeriod = 30 * time.Second
pool, _ := pgxpool.NewWithConfig(ctx, config)
```

### Recommended Pool Sizes

| Service | MaxConns | Reason |
|---------|----------|--------|
| Gateway | N/A (no DB) | — |
| Auth | 15 | Login + register + token refresh |
| Identity | 20 | User CRUD + SCIM |
| Policy | 10 | Role/policy CRUD + check |
| Org | 10 | Org tree queries |
| Audit | 5 | Write-heavy (INSERT only) |

### PostgreSQL max_connections

```sql
-- Total connections = sum of all service pools × instance count
-- Example: 5 services × 20 MaxConns × 3 replicas = 300 connections
SHOW max_connections;  -- default: 100

-- Increase for production
ALTER SYSTEM SET max_connections = 300;
ALTER SYSTEM SET shared_buffers = '2GB';
```

Or use PgBouncer to multiplex connections:

```ini
# pgbouncer.ini
[databases]
ggid = host=127.0.0.1 port=5432 dbname=ggid pool_size=50

[pgbouncer]
pool_mode = transaction
max_client_conn = 500
default_pool_size = 50
reserve_pool_size = 10
```

---

## Redis Caching Strategy

### What Redis Stores

| Data | TTL | Key Pattern |
|------|-----|-------------|
| Rate limit buckets | 60s | `ggid:ratelimit:{ip}:{path}` |
| Password reset tokens | 1h | `ggid:pwreset:{hash}` |
| Email verification tokens | 24h | `ggid:emailverify:{hash}` |
| Magic link tokens | 15m | `ggid:magiclink:{hash}` |
| Token blocklist | Until JWT exp | `ggid:blocklist:{jti}` |
| Session data | Configurable | `ggid:session:{id}` |
| MFA challenge | 5m | `ggid:mfa:{challenge_id}` |

### Redis Configuration for Performance

```conf
# redis.conf
maxmemory 512mb
maxmemory-policy allkeys-lru
save ""  # Disable RDB persistence for pure cache use case
appendonly no  # Disable AOF (cache is ephemeral)

# For better latency
tcp-keepalive 60
tcp-nodelay yes
timeout 300
```

### Client-Side Tuning (Go)

```go
// Use connection pooling (built into go-redis)
rdb := redis.NewClient(&redis.Options{
    Addr:         "redis:6379",
    Password:     "strong-password",
    DB:           0,
    PoolSize:     20,           // max connections
    MinIdleConns: 5,            // keep warm
    ReadTimeout:  100 * time.Millisecond,
    WriteTimeout: 100 * time.Millisecond,
    DialTimeout:  500 * time.Millisecond,
})
```

---

## NATS JetStream Tuning

### Stream Configuration

```conf
# nats-server.conf
jetstream {
    store_dir: "/data/nats"
    max_memory_store: 512MB    # in-memory stream buffer
    max_file_store: 10GB      # disk-backed stream
}

# Audit stream
streams = [
    {
        name: "AUDIT-EVENTS"
        subjects: ["audit.>"]
        retention: limits
        max_msgs: 1000000      # max 1M messages
        max_bytes: 5GB
        max_age: 604800000000000  # 7 days in nanoseconds
        storage: file
        discard: old           # discard oldest when limit reached
    }
]
```

### Consumer Configuration

```go
// Audit consumer — durable, manual ack
consumerConfig := &nats.ConsumerConfig{
    Durable:       "audit-consumer",
    AckPolicy:     nats.AckExplicit,
    AckWait:       30 * time.Second,
    MaxDeliver:    3,              // retry up to 3 times
    DeliverPolicy: nats.DeliverAll,
    FilterSubject: "audit.>",
    BatchSize:     100,            // pull 100 messages at a time
}
```

### Performance Tips

1. **Use file storage** for durability (not memory) for audit events
2. **Batch publishes** — publish multiple events in a single transaction:
   ```go
   // Use PublishAsync for fire-and-forget (best-effort)
   js.PublishAsync("audit.events", event)
   ```
3. **Set MaxDeliver to 3** to avoid poison-pill infinite retries
4. **Monitor consumer lag** via NATS monitoring API:
   ```bash
   curl http://localhost:8222/jsz?consumers=true | jq '.[] | .stream | .consumer'
   ```

---

## Gateway Connection Reuse

### Reverse Proxy Optimization

The Gateway reverse-proxies to backend services. Key settings:

```go
// Reuse TCP connections to backends
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 20,        // per backend service
    IdleConnTimeout:     90 * time.Second,
    DisableCompression:  false,
    ForceAttemptHTTP2:   true,       // use HTTP/2 to backends
}

proxy := &httputil.ReverseProxy{
    Transport: transport,
    FlushInterval: 100 * time.Millisecond,  // stream responses
}
```

### JWKS Caching

```bash
# Gateway fetches JWKS on startup and caches
GATEWAY_JWKS_CACHE_TTL=900  # 15 minutes (default)
```

The Gateway verifies JWTs locally — no per-request call to the Auth service.
This is the single biggest performance factor.

---

## Go Runtime Tuning

### GOMAXPROCS

Go automatically sets `GOMAXPROCS` to the number of CPU cores. In containers:

```bash
# Verify correct CPU detection
docker exec ggid-auth sh -c 'GOMAXPROCS'
# Should match the container's CPU limit
```

If using CPU limits in Kubernetes, use [`automaxprocs`](https://github.com/uber-go/automaxprocs):

```go
import _ "github.com/uber-go/automaxprocs"
// Automatically sets GOMAXPROCS based on cgroup CPU quota
```

### Garbage Collection Tuning

```bash
# GOGC controls GC trigger (default: 100 = trigger when heap doubles)
# Lower = more frequent GC (less memory, more CPU)
# Higher = less frequent GC (more memory, less CPU)

# For latency-sensitive (Gateway):
GOGC=50    # More aggressive GC, lower latency spikes

# For throughput (Audit consumer):
GOGC=200   # Less GC, higher throughput

# Or use GOMEMLIMIT (Go 1.19+) for soft memory cap:
GOMEMLIMIT=2GiB  # Cap heap at 2GB (leaving room for cgroups)
```

Set in Docker Compose:
```yaml
services:
  gateway:
    environment:
      GOGC: "50"
      GOMEMLIMIT: "1GiB"
```

### Memory Limits per Service

| Service | Recommended Memory | CPU |
|---------|-------------------|-----|
| Gateway | 256Mi | 500m |
| Auth | 256Mi | 500m |
| Identity | 256Mi | 500m |
| Policy | 256Mi | 250m |
| Org | 128Mi | 250m |
| Audit | 512Mi | 500m |
| Console | 512Mi | 250m |

---

## pprof Analysis

### Enable pprof

Add pprof endpoint to each service:

```go
import _ "net/http/pprof"

// In main.go — start pprof on a separate port
go func() {
    http.ListenAndServe(":6060", nil)
}()
```

### CPU Profiling

```bash
# Capture 30-second CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# In pprof interactive:
(pprof) top 20          # top 20 functions by CPU
(pprof) top20 -cum      # cumulative time
(pprof) list FunctionName  # show source lines
(pprof) web             # open graph in browser
(pprof) svg > cpu.svg   # save flame graph
```

### Heap Profiling

```bash
# Capture heap snapshot
go tool pprof http://localhost:6060/debug/pprof/heap

# In pprof:
(pprof) top 20 -cum     # top allocations
(pprof) list FunctionName
```

### Goroutine Analysis

```bash
# Check for goroutine leaks
curl http://localhost:6060/debug/pprof/goroutine?debug=1 | head -50
go tool pprof http://localhost:6060/debug/pprof/goroutine

# In pprof:
(pprof) top             # which functions created the most goroutines
```

### Block Profiling

```go
// Enable block profiling in main.go
runtime.SetBlockProfileRate(1)  // sample every blocking event

// Then:
// go tool pprof http://localhost:6060/debug/pprof/block
```

### Flame Graphs

```bash
# Install flame graph tool
go install github.com/google/pprof@latest

# Generate flame graph
pprof -flame http://localhost:6060/debug/pprof/profile?seconds=30 > flame.html
```

---

## Load Testing

### k6 Benchmark Suite

GGID includes k6 performance test scripts in `deploy/k6/`:

```bash
# Run login benchmark
k6 run deploy/k6/login-bench.js

# Run full API benchmark
k6 run deploy/k6/api-bench.js

# Run mixed workload
k6 run deploy/k6/mixed-workload.js
```

### Key Metrics to Monitor

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| p50 latency | < 5ms | > 20ms |
| p95 latency | < 50ms | > 200ms |
| p99 latency | < 100ms | > 500ms |
| Error rate | < 0.1% | > 1% |
| Requests/sec | 1000+ (single instance) | < 500 |
| CPU usage | < 70% | > 90% |
| Memory usage | < 80% of limit | > 90% |

### Sample k6 Script

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 50 },   // ramp up
    { duration: '1m', target: 50 },     // steady state
    { duration: '30s', target: 200 },   // spike
    { duration: '1m', target: 200 },
    { duration: '30s', target: 0 },     // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<100', 'p(99)<200'],
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  const res = http.post('http://localhost:8080/api/v1/auth/login', JSON.stringify({
    username: 'loadtest',
    password: 'Test@123456',
  }), {
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': '00000000-0000-0000-0000-000000000001',
    },
  });

  check(res, {
    'status 200': (r) => r.status === 200,
    'has token': (r) => r.json('access_token') !== undefined,
  });

  sleep(0.1);
}
```

---

## Monitoring

### Prometheus Metrics

The Gateway exposes metrics at `/metrics`:

```bash
curl http://localhost:8080/metrics
```

Key metrics:
- `ggid_http_requests_total` — total requests (labels: method, path, status)
- `ggid_http_request_duration_seconds` — latency histogram
- `ggid_jwt_verifications_total` — JWT verify count (labels: result)
- `ggid_backend_healthy` — backend health (0/1 per service)
- `ggid_rate_limit_hits_total` — rate-limited request count

### Grafana Dashboard

Import the GGID dashboard:
1. Open Grafana → Dashboards → Import
2. Upload `deploy/grafana/ggid-dashboard.json`

Key panels:
- Request rate and latency (p50/p95/p99)
- Error rate (4xx/5xx separately)
- Backend health status
- Rate limit hits
- Active sessions

### PostgreSQL Monitoring

```sql
-- Active connections
SELECT count(*), state FROM pg_stat_activity GROUP BY state;

-- Long-running queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query
FROM pg_stat_activity
WHERE state = 'active' AND now() - pg_stat_activity.query_start > interval '5 seconds';

-- Database size
SELECT pg_size_pretty(pg_database_size('ggid'));

-- Table sizes
SELECT relname, pg_size_pretty(pg_total_relation_size(relid))
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC LIMIT 10;
```

### NATS Monitoring

```bash
# Stream stats
curl http://localhost:8222/jsz?streams=true | jq '.[] | {name, msgs, bytes}'

# Consumer lag
curl http://localhost:8222/jsz?consumers=true | \
  jq '.[] | .stream | .consumer[] | {name, delivered, ack_floor, num_pending}'
```

### Alerting Rules

```yaml
# Prometheus alert rules
groups:
  - name: ggid
    rules:
      - alert: HighLatency
        expr: histogram_quantile(0.95, ggid_http_request_duration_seconds_bucket) > 0.2
        for: 5m
        annotations:
          summary: "p95 latency above 200ms"

      - alert: HighErrorRate
        expr: rate(ggid_http_requests_total{status=~"5.."}[5m]) / rate(ggid_http_requests_total[5m]) > 0.05
        for: 2m
        annotations:
          summary: "5xx error rate above 5%"

      - alert: BackendDown
        expr: ggid_backend_healthy == 0
        for: 1m
        annotations:
          summary: "Backend service unhealthy"

```
