# Cost Optimization Guide

This guide covers optimizing GGID infrastructure costs — resource right-sizing, connection pool tuning, cache strategies, query optimization, storage tiering, and cost monitoring.

## Resource Right-Sizing

### Service Resource Recommendations

| Service | CPU Request | CPU Limit | Memory Request | Memory Limit |
|---------|------------|-----------|---------------|-------------|
| Gateway | 100m | 500m | 128Mi | 256Mi |
| Auth | 100m | 500m | 128Mi | 256Mi |
| Identity | 100m | 500m | 128Mi | 256Mi |
| OAuth | 50m | 250m | 64Mi | 128Mi |
| Policy | 50m | 250m | 64Mi | 128Mi |
| Org | 50m | 250m | 64Mi | 128Mi |
| Audit | 100m | 500m | 128Mi | 256Mi |

### Right-Sizing Process

1. Monitor actual usage for 2 weeks (Prometheus/Grafana)
2. Set requests to p95 CPU/memory
3. Set limits to 2x requests
4. Review quarterly

```bash
# Get resource usage
kubectl top pods -n ggid --sort-by=cpu
kubectl top pods -n ggid --sort-by=memory
```

## Connection Pool Tuning

Oversized connection pools waste DB connections (max_connections=200):

| Service | Before | After | Savings |
|---------|--------|-------|---------|
| Gateway | 20 | 5 | 15 |
| Identity | 30 | 15 | 15 |
| Auth | 20 | 10 | 10 |
| Policy | 20 | 10 | 10 |
| Org | 20 | 10 | 10 |
| Audit | 30 | 15 | 15 |
| **Total** | **140** | **65** | **75 (53%)** |

```bash
# Right-size pools
DB_MAX_CONNS=10  # Per service instance
DB_MIN_CONNS=2
```

## Cache Strategies

### JWT Claims Cache

Cache JWT verification results in Redis (TTL = token expiry):
```
Without cache: ~5ms per verify (JWKS fetch + signature check)
With cache: ~0.1ms per verify (Redis GET)
→ 98% latency reduction for hot tokens
```

### Policy Decision Cache

Cache RBAC check results:
```go
cacheKey := fmt.Sprintf("policy:%s:%s:%s", userID, resource, action)
if cached := redis.Get(cacheKey); cached != nil {
    return cached // < 1ms
}
result := policyEngine.Check(userID, resource, action)
redis.Set(cacheKey, result, 5*time.Minute) // Cache 5 min
```

## Query Optimization

### Index Audit

```sql
-- Find slow queries
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
ORDER BY mean_exec_time DESC LIMIT 20;

-- Find unused indexes (waste)
SELECT relname, indexrelname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0;
```

### Key Indexes

```sql
-- RLS-filtered queries (tenant_id first)
CREATE INDEX idx_users_tenant_status ON users (tenant_id, status);
CREATE INDEX idx_audit_tenant_created ON audit_events (tenant_id, created_at DESC);
CREATE INDEX idx_roles_tenant_key ON roles (tenant_id, key);
```

## Storage Tiering

| Data | Tier | Retention | Cost |
|------|------|-----------|------|
| Hot: recent audit events | SSD | 90 days | $$$ |
| Warm: older audit events | HDD | 1 year | $$ |
| Cold: compliance archives | S3 Glacier | 7 years | $ |

### Audit Partitioning

```sql
-- Monthly partitions enable cheap archival
CREATE TABLE audit_events_2025_01 PARTITION OF audit_events
  FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- Move old partition to cold storage
ALTER TABLE audit_events_2024_01 DETACH PARTITION;
pg_dump --table=audit_events_2024_01 | aws s3 cp - s3://ggid-archive/
DROP TABLE audit_events_2024_01;
```

## Infrastructure Cost Breakdown

| Component | Monthly Cost | Optimization |
|-----------|-------------|-------------|
| Compute (K8s nodes) | $1,500 | Right-size, spot instances |
| PostgreSQL | $500 | Shared instance for small tenants |
| Redis | $200 | Single instance (non-HA dev) |
| NATS | $100 | Single node (non-prod) |
| Load balancer | $100 | Shared ALB |
| DNS | $50 | — |
| Network egress | $100 | Compress responses |
| **Total** | **$2,550** | |

## Spot Instances (Non-Critical)

Use spot/preemptible nodes for:
- Audit consumer pods (stateless, retriable)
- Console (stateless)
- CI/CD runners

**Not for**: Gateway, Auth, PostgreSQL, Redis, NATS (critical path).

```yaml
nodeSelector:
  node-role: spot
 tolerations:
  - key: spot
    value: "true"
```

## Cost Monitoring Alerts

| Alert | Threshold |
|-------|----------|
| Monthly compute spend | > $2,000 |
| DB storage | > 100 GB |
| NATS storage | > 10 GB |
| Redis memory | > 512 MB |
| Network egress | > 500 GB |

## See Also

- [Performance Tuning](performance-tuning.md)
- [High Availability](high-availability.md)
- [Production Checklist](production-checklist.md)
- [Data Retention Policy](data-retention-policy.md)
