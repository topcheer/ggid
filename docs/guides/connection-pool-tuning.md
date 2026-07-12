# Connection Pool Tuning

PostgreSQL, Redis, gRPC, and HTTP connection pool sizing formulas, monitoring, and leak detection.

## Sizing Formula

```
Pool Size = (Max Concurrent Requests) × (Queries per Request) × (Avg Query Duration in seconds)

Example:
  100 concurrent requests × 2 queries each × 0.005s avg = 1 connection needed
  Add 2x headroom for spikes → Pool Size = 25
```

### PostgreSQL (pgxpool)

```go
config, _ := pgxpool.ParseConfig(databaseURL)
config.MaxConns = 25
config.MinConns = 5
config.MaxConnLifetime = 30 * time.Minute
config.MaxConnIdleTime = 5 * time.Minute
config.HealthCheckPeriod = 1 * time.Minute

pool, _ := pgxpool.NewWithConfig(ctx, config)
```

| Parameter | Formula | Default | Tuning |
|-----------|---------|---------|--------|
| MaxConns | `concurrent_requests × queries_per_request × 2` | 25 | Match DB `max_connections / number_of_app_instances` |
| MinConns | `MaxConns / 5` | 5 | Keep warm connections ready |
| MaxConnLifetime | — | 30 min | Prevent stale connections |
| MaxConnIdleTime | — | 5 min | Close idle to free DB resources |
| HealthCheckPeriod | — | 1 min | Detect dead connections early |

### PostgreSQL Server-Side

```sql
-- Check max connections
SHOW max_connections;  -- Default: 100

-- Each app instance uses 25 connections
-- Max instances = max_connections / pool_size = 100 / 25 = 4 instances
-- For more instances, increase max_connections or use PgBouncer
```

#### PgBouncer (Connection Pooler)

```ini
# pgbouncer.ini
[databases]
ggid = host=localhost port=5432 dbname=ggid

[pgbouncer]
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 25
reserve_pool_size = 5
```

### Redis Pool

```go
rdb := redis.NewClient(&redis.Options{
    Addr:         "redis:6379",
    PoolSize:     20,
    MinIdleConns: 5,
    MaxRetries:   3,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
    PoolTimeout:  4 * time.Second,
})
```

| Parameter | Formula | Default |
|-----------|---------|---------|
| PoolSize | `concurrent_requests × redis_ops_per_request × 2` | 20 per CPU |
| MinIdleConns | `PoolSize / 4` | 5 |
| PoolTimeout | `ReadTimeout + 1s` | 4s |

### gRPC Connection Pool

```go
conn, _ := grpc.Dial("identity-svc:9080",
    grpc.WithDefaultServiceConfig(`{
        "methodConfig": [{
            "name": [{}],
            "retryPolicy": {
                "maxAttempts": 3,
                "initialBackoff": "0.1s",
                "maxBackoff": "1s",
                "backoffMultiplier": 2.0,
                "retryableStatusCodes": ["UNAVAILABLE"]
            }
        }]
    }`),
    grpc.WithKeepaliveParams(keepalive.ClientParameters{
        Time:                10 * time.Second,
        Timeout:             5 * time.Second,
        PermitWithoutStream: true,
    }),
)
```

### HTTP Client Pool

```go
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 20,
    MaxConnsPerHost:     50,
    IdleConnTimeout:     90 * time.Second,
    TLSHandshakeTimeout: 5 * time.Second,
}

client := &http.Client{
    Transport: transport,
    Timeout:   10 * time.Second,
}
```

## Monitoring

### PostgreSQL Pool Stats

```go
func logPoolStats(pool *pgxpool.Pool) {
    stats := pool.Stat()
    log.Info("db.pool",
        "acquired_conns", stats.AcquiredConns(),
        "idle_conns", stats.IdleConns(),
        "total_conns", stats.TotalConns(),
        "max_conns", stats.MaxConns(),
        "acquire_count", stats.AcquireCount(),
        "acquire_duration_ms", stats.AcquireDuration().Milliseconds(),
    )
}
// Run every 30 seconds
```

### Pool Saturation Alert

| Metric | Alert Threshold |
|--------|----------------|
| DB acquired / max | >80% → increase pool or scale |
| DB acquire wait time | >50ms → pool too small |
| Redis pool hits | >95% → increase pool size |
| gRPC connections | Track per-service |
| HTTP idle conns | <MaxIdleConnsPerHost → increase |

## Leak Detection

### DB Connection Leak

```go
func withLeakDetection(pool *pgxpool.Pool, fn func(*pgxpool.Pool) error) error {
    before := pool.Stat().AcquiredConns()
    err := fn(pool)
    after := pool.Stat().AcquiredConns()
    
    if before != after {
        log.Error("connection leak detected",
            "before", before,
            "after", after,
        )
    }
    return err
}
```

### Context-Based Cleanup

```go
// Always use context for connection lifecycle
func getUser(ctx context.Context, pool *pgxpool.Pool, id string) (*User, error) {
    conn, err := pool.Acquire(ctx)
    if err != nil { return nil, err }
    defer conn.Release() // ALWAYS release
    
    return queryUser(ctx, conn, id)
}
```

## Per-Service Defaults

| Service | DB Pool | Redis Pool | gRPC Pool |
|---------|---------|-----------|-----------|
| Gateway | — | 10 | 20 per backend |
| Identity | 25 | 10 | 10 |
| Auth | 15 | 20 | 10 |
| OAuth | 15 | 15 | 10 |
| Policy | 15 | 20 | 10 |
| Org | 15 | 5 | 10 |
| Audit | 20 (writes) | 5 | 10 |

## See Also

- [Gateway Architecture](gateway-architecture.md)
- [gRPC Interceptor Patterns](grpc-interceptor-patterns.md)
- [Database Migration Playbook](database-migration-playbook.md)
- [Monitoring and Alerting](monitoring-and-alerting.md)
