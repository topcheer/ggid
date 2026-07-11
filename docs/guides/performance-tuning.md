# Performance Tuning Guide

> Optimize GGID throughput: DB pooling, Redis, NATS, gRPC, JWT caching, pprof profiling.

---

## PostgreSQL Optimization

### Connection Pooling

```bash
# Recommended pool size = (CPU cores * 2) + effective_io_concurrency
DB_MAX_CONNS=50
DB_MIN_CONNS=10
```

### RLS Performance

RLS adds a filter to every query. Ensure indexes include `tenant_id`:

```sql
-- Good: composite index
CREATE INDEX idx_users_tenant_status ON users (tenant_id, status);

-- Bad: tenant_id not in index (RLS filter does full scan)
CREATE INDEX idx_users_status ON users (status);
```

### Query Optimization

```sql
-- Analyze slow queries
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
ORDER BY mean_exec_time DESC LIMIT 10;
```

---

## Redis Optimization

### Connection Pool

```go
cfg := &redis.Options{
    PoolSize:     20,
    MinIdleConns: 5,
    ReadTimeout:  100 * time.Millisecond,
}
```

### Key Namespacing

Use tenant prefix to avoid key collisions:
```
session:{tenant_id}:{session_id}
ratelimit:{tenant_id}:{ip}
jti:{tenant_id}:{jti}
```

---

## NATS JetStream

### Consumer Configuration

```go
js.Subscribe("AUDIT_EVENTS", handler,
    nats.Durable("audit-consumer"),
    nats.MaxAckPending(1000),  // Process up to 1000 concurrently
    nats.MaxDeliver(3),
    nats.AckWait(30*time.Second),
)
```

---

## JWT Caching

JWKS keys are cached for 15 minutes:

```go
client := ggid.New(url, ggid.WithJWKS(15*time.Minute))
```

For high-throughput: cache parsed JWT claims in Redis (5min TTL) to avoid re-parsing.

---

## Profiling with pprof

```bash
# CPU profile
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Memory profile
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

---

## Benchmark Targets

| Metric | Target |
|--------|--------|
| Login latency | < 100ms p99 |
| JWT verify | < 5ms p99 |
| API throughput | 5,000 req/s per instance |
| Audit publish | < 1ms |

---

*See: [Observability Guide](observability-guide.md) | [Operations Runbook](../operations-runbook.md)*

*Last updated: 2025-07-11*
