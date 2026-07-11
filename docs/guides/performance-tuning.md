# Performance Tuning Guide

This guide covers optimizing GGID throughput across PostgreSQL, Redis, NATS JetStream, gRPC, and JWT caching, with benchmark methods.

## PostgreSQL Optimization

### Connection Pool Sizing

Each GGID service maintains its own pgxpool. Total connections must stay within PostgreSQL `max_connections`:

| Service | MaxConns | MinConns | Rationale |
|---------|----------|----------|-----------|
| Gateway | 5 | 1 | JWT only, no DB |
| Identity | 25 | 5 | High CRUD volume |
| Auth | 15 | 3 | Login bursts |
| Policy | 20 | 5 | Policy evaluation |
| Org | 15 | 3 | Moderate writes |
| Audit | 30 | 10 | High insert volume |
| **Total** | **110** | **27** | Fits max_connections=200 |

```bash
DB_MAX_CONNS=25
DB_MIN_CONNS=5
DB_MAX_CONN_LIFETIME=1h
DB_MAX_CONN_IDLE_TIME=30m
DB_HEALTH_CHECK_PERIOD=1m
```

### RLS Performance

Row-Level Security adds a filter to every query. Optimize with composite indexes:

```sql
-- GOOD: tenant_id first in composite index (RLS filter uses this)
CREATE INDEX idx_users_tenant_status ON users (tenant_id, status);
CREATE INDEX idx_audit_tenant_created ON audit_events (tenant_id, created_at DESC);

-- BAD: tenant_id not in index → RLS filter causes full scan
CREATE INDEX idx_users_status ON users (status);
```

**Verify RLS uses indexes**:
```sql
EXPLAIN ANALYZE SELECT * FROM users WHERE tenant_id = 'UUID' AND status = 'active';
-- Should show "Index Scan" not "Seq Scan"
```

### Partitioning for Audit Events

For high-volume audit tables, use time-based partitioning:

```sql
CREATE TABLE audit_events (
    id UUID DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_type VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    data JSONB
) PARTITION BY RANGE (created_at);

-- Monthly partitions enable fast DROP for retention
CREATE TABLE audit_events_2025_01 PARTITION OF audit_events
  FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
```

### Query Analysis

```sql
-- Enable pg_stat_statements
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Find slowest queries
SELECT query, mean_exec_time, calls, total_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC LIMIT 20;

-- Find unused indexes (candidates for removal)
SELECT schemaname, relname, indexrelname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0;
```

### Autovacuum Tuning

```ini
autovacuum_max_workers = 6
autovacuum_naptime = 30s
autovacuum_vacuum_scale_factor = 0.05
autovacuum_analyze_scale_factor = 0.02
```

## Redis Optimization

### Connection Pool

```go
cfg := &redis.Options{
    PoolSize:         20,
    MinIdleConns:     5,
    ReadTimeout:      100 * time.Millisecond,
    WriteTimeout:     100 * time.Millisecond,
    ConnMaxIdleTime:  15 * time.Minute,
    ConnMaxLifetime:  1 * time.Hour,
}
```

### Key Namespacing

Use tenant-prefixed keys to prevent collisions and enable per-tenant analysis:

```
session:{tenant_id}:{session_id}     TTL: 8h
ratelimit:{tenant_id}:{ip}           TTL: 60s
jti:{tenant_id}:{jti}                TTL: token_expiry
oauth_state:{tenant_id}:{state}      TTL: 10m
```

### Pipeline for Batch Operations

```go
// Batch session lookups
pipe := rdb.Pipeline()
for _, sid := range sessionIDs {
    pipe.Get(ctx, "session:"+tenantID+":"+sid)
}
cmders, err := pipe.Exec(ctx)
```

## NATS JetStream Optimization

### Consumer Configuration

```go
js.Subscribe("AUDIT_EVENTS", handler,
    nats.Durable("audit-consumer"),
    nats.MaxAckPending(1000),       // Process up to 1000 concurrently
    nats.MaxDeliver(3),             // Max retry on failure
    nats.AckWait(30*time.Second),   // Timeout for ack
    nats.PullMaxWaiting(256),       // Pull request queue depth
)
```

### Stream Configuration

```yaml
# High-throughput audit stream
nats:
  stream:
    name: AUDIT_EVENTS
    subjects: ["audit.>"]
    retention: limits
    max_age: 168h              # 7 days in NATS
    max_msgs: 1000000
    max_bytes: 10737418240     # 10 GB
    storage: file              # File-based (survives restart)
    replicas: 1                # Set to 3 for HA
    discard: old               # Drop oldest when full
```

### Publish Batching

```go
// Publish audit events in batches for throughput
for _, event := range events {
    js.PublishAsync("audit.events", eventData)
}
select {
case <-js.PublishAsyncComplete():  // Wait for all publishes
case <-time.After(5 * time.Second):
    log.Warn("publish timeout")
}
```

## gRPC Optimization

### Connection Reuse

Each service should maintain a single gRPC client connection (not per-request):

```go
// GOOD: Shared connection
conn, err := grpc.Dial(target,
    grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
    grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
)
defer conn.Close()

// Reuse conn for all calls
client := policyv1.NewPolicyServiceClient(conn)
```

### Keepalive Tuning

```go
grpc.WithKeepaliveParams(keepalive.ClientParameters{
    Time:                30 * time.Second,
    Timeout:             10 * time.Second,
    PermitWithoutStream: true,
})
```

### gRPC vs REST Performance

| Metric | REST (JSON) | gRPC (Protobuf) |
|--------|-------------|-----------------|
| Serialization | ~5ms | ~0.1ms |
| Payload size | ~2KB | ~0.5KB |
| Connection | HTTP/1.1 (new) | HTTP/2 (multiplexed) |
| Throughput | ~5K req/s | ~50K req/s |

Use gRPC for inter-service communication, REST for external APIs.

## JWT Caching

### JWKS Caching

```go
// Cache JWKS keys for 15 minutes to avoid network calls
client := ggid.NewClient(url,
    ggid.WithJWKS("https://api.ggid.example.com/.well-known/jwks.json", 15*time.Minute),
)
```

### Claims Caching

For high-throughput APIs, cache parsed JWT claims in Redis:

```go
// Cache key: jwt:{hash_of_token}
// Value: parsed claims
// TTL: token expiry (or 5 min, whichever is sooner)

func (v *JWTVerifier) Verify(ctx context.Context, token string) (Claims, error) {
    cached, err := v.redis.Get(ctx, "jwt:"+sha256Hex(token)).Bytes()
    if err == nil {
        return unmarshalClaims(cached), nil  // Cache hit
    }

    claims, err := v.verifyFromJWKS(ctx, token)
    if err != nil {
        return nil, err
    }

    // Cache for 5 minutes
    v.redis.Set(ctx, "jwt:"+sha256Hex(token), marshalClaims(claims), 5*time.Minute)
    return claims, nil
}
```

### Benchmark: JWT Verification

```
Without cache:  ~5ms (JWKS fetch + signature verify)
With JWKS cache: ~0.5ms (signature verify only)
With claims cache: ~0.1ms (Redis GET)
```

## Profiling

### pprof (CPU & Memory)

```bash
# CPU profile (30 seconds)
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof -web cpu.prof

# Heap profile
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof -web heap.prof

# Goroutine count
curl http://localhost:8080/debug/pprof/goroutine > goroutines.prof

# Block profile (must enable)
curl http://localhost:8080/debug/pprof/block > block.prof
```

### Go Benchmark Tests

```go
func BenchmarkLogin(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _, _ = client.Login(ctx, "bench@example.com", "password")
    }
}
// Run: go test -bench=. -benchmem -count=5
```

## Benchmark Targets

| Metric | Target | Critical Path |
|--------|--------|---------------|
| Login (password) | < 100ms p99 | Argon2id verify |
| Login (WebAuthn) | < 50ms p99 | Signature verify only |
| JWT verify (cached) | < 5ms p99 | JWKS cache hit |
| JWT verify (uncached) | < 50ms p99 | JWKS fetch |
| Policy check | < 10ms p99 | Rule evaluation |
| User create | < 50ms p99 | DB insert |
| User list (100) | < 30ms p99 | DB select (RLS) |
| Audit publish | < 1ms p99 | NATS async |
| API throughput | 5,000 req/s | Per gateway instance |

## Load Testing

```bash
# Using vegeta
echo "GET http://localhost:8080/api/v1/users" | \
  vegeta attack -duration=60s -rate=5000 -header="Authorization: Bearer $TOKEN" | \
  vegeta report

# Using wrk
wrk -t12 -c400 -d30s http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN"
```

## Optimization Checklist

- [ ] Composite indexes with `tenant_id` first on all tenant tables
- [ ] pgxpool MaxConns sized per service
- [ ] Redis pool sized for peak concurrency
- [ ] NATS JetStream file-backed with retention limits
- [ ] gRPC connections reused (not per-request)
- [ ] JWKS cached for 15 minutes
- [ ] Audit table partitioned by month
- [ ] pg_stat_statements enabled
- [ ] pprof endpoints accessible (internal only)
- [ ] Load tested at 2x expected peak

## See Also

- [Multi-Database Deployment](multi-database.md)
- [Observability Guide](observability-guide.md)
- [Gateway Configuration](gateway-config.md)
