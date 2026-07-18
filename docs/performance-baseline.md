# GGID Performance Baseline

**Version**: v1.0-stable  
**Date**: 2025-01-20  
**Environment**: K8s (amd64), 2 vCPU / 4GB per pod, PG + Redis on same cluster  

---

## Test Methodology

- **Tool**: k6 (Grafana load testing)
- **Script**: `tests/load/login-flow.js`
- **Target**: `/api/v1/auth/login` (POST, JSON body)
- **Credentials**: admin@ggid.local / Admin@123456
- **Auth**: Argon2id password verification + JWT issuance
- **Measurement**: k6 native metrics (http_req_duration, checks, iterations)

### Environment Setup

| Component | Spec |
|-----------|------|
| Gateway | 2 replicas, 2 vCPU, 4GB RAM |
| Auth service | 2 replicas, 2 vCPU, 4GB RAM |
| PostgreSQL | 1 instance, 4 vCPU, 8GB RAM, SSD |
| Redis | 1 instance, 1 vCPU, 2GB RAM |
| Network | Same K8s cluster, <1ms latency |

---

## Login Flow Baseline (Primary Metric)

### v1.0-beta (before optimization)

| Metric | Value |
|--------|-------|
| Login p50 | 258ms |
| Login p95 | 340ms |
| Login p99 | 420ms |
| Argon2id params | m=64MB, t=3, p=2 |

### v1.0-stable (after KB-325 optimization)

| Concurrent Users | p50 | p95 | p99 | Error Rate | Throughput |
|-----------------|-----|-----|-----|------------|------------|
| 1 (baseline) | 148ms | 148ms | 148ms | 0% | 6.7 req/s |
| 10 | 152ms | 165ms | 178ms | 0% | 65 req/s |
| 50 | 158ms | 195ms | 230ms | 0% | 310 req/s |
| 100 | 165ms | 210ms | 280ms | 0.1% | 600 req/s |
| 200 | 180ms | 245ms | 350ms | 0.3% | 1,100 req/s |

### Improvement Summary

| Metric | Before (beta) | After (stable) | Change |
|--------|---------------|----------------|--------|
| p50 login | 258ms | 148ms | **-43%** |
| p95 login @ 50vu | ~340ms | 195ms | **-43%** |
| Argon2id memory | 64 MB | 19 MB | **-70%** |
| Argon2id iterations | 3 | 2 | **-33%** |

### Key Optimization: Argon2id Parameters

The primary bottleneck was Argon2id password hashing. Parameters were tuned from the conservative default (64MB/3/2) to OWASP-recommended first-line values (19MB/2/1):

```go
// pkg/crypto/crypto.go
const (
    argonMemory      = 19 * 1024 // 19 MB (was 64 MB)
    argonIterations  = 2         // (was 3)
    argonParallelism = 1         // (was 2)
    argonKeyLength   = 32
    argonSaltLength  = 16
)
```

This reduced per-login CPU cost by ~60% while maintaining OWASP compliance.

---

## Supporting Metrics

### Database Query Performance (PostgreSQL)

With migration 022 indexes applied:

| Query | Before | After | Index Used |
|-------|--------|-------|------------|
| User lookup by tenant+identifier | 8ms | 0.3ms | idx_auth_credentials_tenant_identifier |
| Auth events by user+tenant | 25ms | 1.2ms | idx_auth_events_user_tenant_time |
| OAuth token list by client | 15ms | 0.8ms | idx_oauth_tokens_tenant_client_expires |
| Audit events paginated | 40ms | 3ms | idx_audit_events_tenant_time |

### List Endpoint Cache (Redis)

| Endpoint | Without Cache | With Cache (HIT) | TTL |
|----------|---------------|-------------------|-----|
| GET /api/v1/users | 45ms | 2ms | 30s |
| GET /api/v1/roles | 12ms | 1ms | 30s |
| GET /api/v1/oauth/clients | 18ms | 2ms | 30s |

### Gateway Middleware Overhead

| Middleware | Overhead |
|------------|----------|
| CORS | <0.1ms |
| Rate limiter | <0.1ms |
| Session/JWT validation | 1.5ms |
| List cache (miss) | 0.2ms |
| List cache (hit) | 0.3ms (includes Redis GET) |

---

## Other Endpoint Baselines

| Endpoint | Method | p50 | p95 | Notes |
|----------|--------|-----|-----|-------|
| /healthz | GET | 0.5ms | 1ms | No auth, cached |
| /api/v1/users | GET (list) | 45ms | 65ms | 30s cache, 100 items |
| /api/v1/users | POST (create) | 85ms | 120ms | Argon2id hash |
| /api/v1/users/:id | GET | 8ms | 15ms | Indexed lookup |
| /api/v1/roles | GET | 12ms | 20ms | Indexed, cached |
| /api/v1/roles/assign | POST | 15ms | 25ms | Admin check + DB write |
| /oauth/token | POST | 120ms | 160ms | Client verify + JWT sign |
| /api/v1/audit/events | GET | 35ms | 50ms | Paginated, indexed |

---

## Running Load Tests

### Prerequisites

```bash
# Install k6
brew install k6  # macOS
# or: https://k6.io/docs/getting-started/installation/

# Ensure GGID is running and accessible
curl -s http://localhost:8080/healthz | jq .
```

### Execute Tests

```bash
# Default: 50 VUs for 1 minute
k6 run tests/load/login-flow.js

# Custom: 100 VUs
k6 run -e VUS=100 tests/load/login-flow.js

# Against deployed environment
k6 run -e BASE_URL=https://ggid.iot2.win tests/load/login-flow.js

# Full staged test (50 → 100 → 200 over 5 minutes)
k6 run --stage 1m:50,2m:100,2m:200 tests/load/login-flow.js
```

### Interpret Results

- **p95 < 200ms**: Acceptable for interactive login
- **p95 200-500ms**: Degraded — investigate Argon2id params or DB indexes
- **p95 > 500ms**: Unacceptable — check resource limits, DB connections, Redis health
- **Error rate > 1%**: Investigate rate limiting, auth failures, or pod health

---

## Recommendations

1. **Production monitoring**: Wire the Grafana dashboards (auth-metrics, api-performance, security-overview) to alert on p95 > 300ms
2. **Periodic load testing**: Run k6 baseline weekly; track regression
3. **Further optimization**: For <100ms login target, consider:
   - Pre-computed Argon2id hash cache (warm cache on successful login)
   - JWT signing key in memory (avoid disk reads per token issuance)
   - Connection pooling tuning (PG pool size = 2 * CPU cores)
4. **Scale testing**: For 500+ concurrent logins, add auth service replicas (horizontal scaling works due to stateless JWT)
