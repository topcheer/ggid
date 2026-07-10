# Performance Tuning Guide

> PostgreSQL, Redis, NATS JetStream, and application-level tuning for
> production GGID deployments.

---

## PostgreSQL Tuning

### Memory Configuration

| Parameter | Default | Recommended (8GB) | Recommended (32GB) | Description |
|-----------|---------|-------------------|--------------------|-------------| 
| `shared_buffers` | 128MB | 2GB | 8GB | Shared memory buffer pool (25% of RAM) |
| `effective_cache_size` | 4GB | 6GB | 24GB | OS cache estimate (75% of RAM) |
| `work_mem` | 4MB | 16MB | 64MB | Per-sort/hash memory |
| `maintenance_work_mem` | 64MB | 512MB | 2GB | VACUUM/CREATE INDEX memory |
| `wal_buffers` | -1 | 16MB | 16MB | WAL write buffer |
| `temp_buffers` | 8MB | 32MB | 32MB | Temp table memory |

```ini
# postgresql.conf — 32GB RAM server
shared_buffers = 8GB
effective_cache_size = 24GB
work_mem = 64MB
maintenance_work_mem = 2GB
wal_buffers = 16MB
max_connections = 200
```

### Connection Pooling (pgxpool)

GGID uses `pgxpool` for connection pooling. Configure per-service:

```go
config, _ := pgxpool.ParseConfig(databaseURL)
config.MaxConns = 25                    // Max connections per service instance
config.MinConns = 5                     // Keep warm connections
config.MaxConnLifetime = time.Hour      // Recycle connections
config.MaxConnIdleTime = 30 * time.Minute
config.HealthCheckPeriod = time.Minute

pool, _ := pgxpool.NewWithConfig(ctx, config)
```

**Recommended pool sizes:**

| Service | MaxConns | MinConns | Rationale |
|---------|----------|----------|-----------|
| Gateway | 0 (no DB) | 0 | Stateless proxy |
| Auth | 25 | 5 | High query volume |
| Identity | 20 | 5 | Moderate CRUD |
| Policy | 15 | 3 | Read-heavy, cached |
| Org | 10 | 2 | Lower traffic |
| Audit | 15 | 3 | Write-heavy |

**Total DB connections:** (25+20+15+10+15) × replicas ≤ `max_connections` (200)

### Index Strategy

```sql
-- Every tenant-scoped table must have tenant_id index
CREATE INDEX CONCURRENTLY idx_users_tenant
    ON users(tenant_id);
CREATE INDEX CONCURRENTLY idx_users_tenant_created
    ON users(tenant_id, created_at DESC);

-- Composite indexes for common query patterns
CREATE INDEX CONCURRENTLY idx_credentials_identifier_tenant
    ON credentials(tenant_id, identifier);

CREATE INDEX CONCURRENTLY idx_roles_tenant_key
    ON roles(tenant_id, key);

CREATE INDEX CONCURRENTLY idx_audit_tenant_action_time
    ON audit_events(tenant_id, action, created_at DESC);

-- Partial indexes for filtered queries
CREATE INDEX CONCURRENTLY idx_users_active
    ON users(tenant_id)
    WHERE deleted_at IS NULL;

-- Expression indexes for case-insensitive search
CREATE INDEX CONCURRENTLY idx_users_email_lower
    ON users(LOWER(email));
```

### Query Optimization

```sql
-- Use EXPLAIN ANALYZE to verify query plans
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM users
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND created_at > '2024-01-01'
ORDER BY created_at DESC
LIMIT 20;

-- Good plan: Index Scan using idx_users_tenant_created
-- Bad plan: Seq Scan → needs VACUUM ANALYZE or new index
```

### VACUUM and Autovacuum

```ini
# postgresql.conf
autovacuum = on
autovacuum_max_workers = 6
autovacuum_naptime = 30s
autovacuum_vacuum_threshold = 50
autovacuum_vacuum_scale_factor = 0.1    # Vacuum when 10% rows change
autovacuum_analyze_scale_factor = 0.05  # Analyze when 5% rows change

# Audit table needs aggressive vacuuming (high write volume)
ALTER TABLE audit_events SET (
    autovacuum_vacuum_scale_factor = 0.02,
    autovacuum_analyze_scale_factor = 0.01
);
```

---

## Redis Tuning

### Memory Policy

```conf
# redis.conf
maxmemory 2gb
maxmemory-policy allkeys-lru     # Evict least recently used
# Alternatives:
#   volatile-lru   — Only evict keys with TTL
#   allkeys-lfu    — Evict least frequently used
#   volatile-ttl   — Evict keys with shortest TTL first
```

### Connection Pooling

```go
// Go Redis client configuration
rdb := redis.NewClient(&redis.Options{
    Addr:         "redis:6379",
    PoolSize:     20,                    // Max connections
    MinIdleConns: 5,                     // Keep warm
    MaxRetries:   3,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
})
```

### Pipelining for Batch Operations

```go
// BAD: N individual SET commands
for key, val := range data {
    rdb.Set(ctx, key, val, time.Hour)
}

// GOOD: Single pipelined batch
pipe := rdb.Pipeline()
for key, val := range data {
    pipe.Set(ctx, key, val, time.Hour)
}
pipe.Exec(ctx)  // One round-trip
```

### Session Storage Optimization

```go
// Store sessions efficiently with hash tags for cluster routing
key := fmt.Sprintf("session:{%s}", sessionID)
// The {sessionID} hash tag ensures all session keys route to the same slot

// Use SCAN instead of KEYS for bulk operations
iter := rdb.Scan(ctx, 0, "session:*", 100).Iterator()
for iter.Next(ctx) {
    key := iter.Val()
    // Process key
}
```

---

## NATS JetStream Tuning

### Stream Configuration

```yaml
stream_config:
  name: AUDIT_EVENTS
  storage: file               # File-based (persistent)
  retention: limits
  max_age: 72h                # Auto-expire after 72 hours
  max_msgs: 1000000           # Max 1M messages
  max_bytes: 5GB              # Max 5GB storage
  max_msg_size: 1MB           # Individual message size limit
  discard: old                # Discard oldest when full
  duplicates: 2m              # Deduplication window
  replicas: 3                 # RAFT replicated cluster
```

### Consumer Configuration

```yaml
consumer_config:
  durable_name: audit-consumer
  deliver_policy: deliver_all
  ack_policy: explicit
  ack_wait: 30s               # Redeliver unacked after 30s
  max_deliver: 3              # Max delivery attempts
  max_ack_pending: 1000       # In-flight messages
  filter_subject: audit.events
  replay_policy: instant
```

### Performance Tuning Tips

| Setting | Impact | Recommendation |
|---------|--------|----------------|
| `storage: file` | Persistence vs speed | Use `file` for durability, `memory` for ephemeral |
| `max_msgs` | Disk usage | Set to expected 72h volume |
| `replicas: 3` | Availability vs write latency | Use 3 for HA, 1 for dev |
| `ack_wait` | Redelivery latency | Lower = faster retry, higher = fewer false redeliveries |
| `max_ack_pending` | Throughput | Higher = more parallelism, watch memory |

### Disk I/O for File Storage

```bash
# Use SSD/NVMe for JetStream file store
# Expected write throughput: 100K msgs/sec on NVMe, 10K on SSD

# Monitor disk usage
nats stream info AUDIT_EVENTS
# State: Messages: 523,841, Bytes: 245 MiB

# Monitor consumer lag
nats consumer info AUDIT_EVENTS audit-consumer
# Delivered: 523,000, Ack Floor: 522,990, Pending: 10
```

---

## Application-Level Tuning

### HTTP Server Timeouts

```go
srv := &http.Server{
    Addr:              ":8080",
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       10 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       120 * time.Second,
    MaxHeaderBytes:    1 << 20,  // 1MB
}
```

### Graceful Connection Handling

```go
// Use connection limits per service
srv.ConnState = func(conn net.Conn, state http.ConnState) {
    if state == http.StateNew {
        atomic.AddInt64(&activeConns, 1)
    } else if state == http.StateClosed {
        atomic.AddInt64(&activeConns, -1)
    }
}
```

### bcrypt Cost Tuning

```go
// bcrypt cost directly impacts login latency
// Cost 10: ~60ms   Cost 12: ~250ms   Cost 14: ~1s
// Default: 12 (good balance for production)
// For high-traffic: consider caching verified tokens in Redis

const bcryptCost = 12
hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
```

### JWT Caching

```go
// Cache verified JWTs to skip signature verification on repeat requests
type JWTCache struct {
    cache map[string]*Claims  // keyed by jti
    mu    sync.RWMutex
}

func (c *JWTCache) Get(token string) (*Claims, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    claims, ok := c.cache[token]
    return claims, ok
}
```

---

## Performance Benchmark Targets

| Operation | Target p50 | Target p95 | Target p99 |
|-----------|-----------|-----------|-----------|
| User login (bcrypt verify) | 50ms | 250ms | 500ms |
| JWT verification (cached) | 0.1ms | 0.5ms | 1ms |
| JWT verification (uncached) | 2ms | 5ms | 10ms |
| User list (20 rows) | 5ms | 15ms | 30ms |
| Policy check | 1ms | 3ms | 5ms |
| Audit event publish (NATS) | 1ms | 3ms | 5ms |
| Create user (bcrypt hash) | 250ms | 350ms | 500ms |

### Load Test Reference

```bash
# k6 load test — 1000 concurrent users for 1 minute
k6 run --vus 1000 --duration 60s benchmark/login-stress.js

# Expected on 4-core / 8GB RAM:
# Throughput: ~2000 req/sec (login)
# Error rate: < 0.1%
# p95 latency: < 300ms
```

---

## Monitoring Queries

### PostgreSQL Active Connections

```sql
SELECT
    state,
    query_start,
    state = 'active' AS is_active,
    query
FROM pg_stat_activity
WHERE datname = 'ggid'
ORDER BY query_start;
```

### Index Usage Statistics

```sql
SELECT
    schemaname,
    relname,
    indexrelname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```

### Table Size and Bloat

```sql
SELECT
    relname,
    pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
    n_live_tup,
    n_dead_tup,
    ROUND(n_dead_tup::FLOAT / NULLIF(n_live_tup, 0) * 100, 2) AS bloat_pct
FROM pg_stat_user_tables
ORDER BY pg_total_relation_size(relid) DESC;
```

---

## References

- [Benchmark Guide](./benchmark.md) — k6 load test scripts
- [Database Optimization](./database-optimization.md) — Query optimization
- [High Availability](./high-availability.md) — Multi-instance deployment

---

## Go Runtime Tuning

### GOMAXPROCS

In containers, `GOMAXPROCS` defaults to the host's CPU count, not the
container's CPU limit. Use `automaxprocs` to fix this:

```dockerfile
# Dockerfile
ENV GOMAXPROCS=4
# Or use uber-go/automaxprocs
```

```go
import _ "go.uber.org/automaxprocs"
```

### Garbage Collection

| Env Var | Default | Effect |
|---------|---------|--------|
| `GOGC=100` | 100% | GC runs when heap doubles (default) |
| `GOGC=200` | 200% | Less frequent GC, more memory usage |
| `GOGC=50` | 50% | More frequent GC, lower latency peaks |
| `GOMEMLIMIT=512MiB` | off | Soft memory limit (Go 1.19+) |

**Recommendation**: `GOGC=100` (default) for most workloads. Use
`GOMEMLIMIT` in containers to match memory limits.

### Memory Ballast

```go
// Allocate a large ballast to stabilize GC frequency
var ballast [1 << 30]byte  // 1GB ballast (never accessed)
```

> Only useful in Go < 1.19. With `GOMEMLIMIT`, ballast is unnecessary.

### Performance Profiling

```bash
# CPU profile
go test -cpuprofile cpu.out ./...

# Memory profile
go test -memprofile mem.out ./...

# Analyze
go tool pprof cpu.out
(pprof) top 10
(pprof) web  # Graphviz visualization

# Live profiling endpoint
curl http://service:port/debug/pprof/profile?seconds=30 > cpu.prof
```

### Benchmarking

```bash
# Run benchmarks
go test -bench=. -benchmem ./...

# Compare benchmarks
go test -bench=. -count=5 ./... | benchstat old.txt new.txt
```

---

## Frontend Tuning (Next.js Console)

### Bundle Analysis

```bash
# Analyze bundle size
cd console
ANALYZE=true npm run build

# Output: .next/analyze/client.html
# Look for: large dependencies, duplicate imports, unoptimized images
```

### Code Splitting and Lazy Loading

```typescript
// Lazy-load heavy components (charts, editors)
import dynamic from 'next/dynamic';

const AuditChart = dynamic(() => import('@/components/AuditChart'), {
  loading: () => <Skeleton />,
  ssr: false,  // Client-only component
});

const RichEditor = dynamic(() => import('@/components/RichEditor'));
```

### Image Optimization

```typescript
import Image from 'next/image';

// Automatic WebP/AVIF conversion, responsive sizes
<Image
  src="/logo.png"
  width={200}
  height={50}
  priority  // Above-the-fold images
  alt="Logo"
/>
```

### Performance Metrics

| Metric | Target | Tool |
|--------|--------|------|
| First Contentful Paint | < 1.5s | Lighthouse |
| Time to Interactive | < 3s | Lighthouse |
| Bundle size (initial) | < 200KB gzip | webpack-bundle-analyzer |
| Lighthouse score | > 90 | PageSpeed Insights |

### Caching Strategy

```typescript
// next.config.js
module.exports = {
  experimental: {
    staleTimer: 1000,  // ISR revalidation
  },
  headers: async () => [
    {
      source: '/_next/static/:path*',
      headers: [
        { key: 'Cache-Control', value: 'public, max-age=31536000, immutable' },
      ],
    },
  ],
};
```
- [Helm Chart Guide](./helm-chart.md) — K8s resource limits
