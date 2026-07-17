# Load Testing Strategy & Capacity Planning: Performance Baselines for GGID

> **Focus**: Comprehensive load testing strategy using k6 — baseline RPS per service, capacity model (users → infrastructure), bottleneck identification, performance budgets, and soak/spike/stress test scenarios.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: DoD per backlog item (§8).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Performance Budgets](#2-performance-budgets)
3. [Capacity Model](#3-capacity-model)
4. [k6 Load Test Scripts](#4-k6-load-test-scripts)
5. [Bottleneck Identification](#5-bottleneck-identification)
6. [Test Scenarios](#6-test-scenarios)
7. [Database Performance](#7-database-performance)
8. [Implementation Backlog with DoD](#8-implementation-backlog-with-dod)
9. [Competitive Differentiation](#9-competitive-differentiation)

---

## 1. Executive Summary

GGID has **no load testing baseline** — we don't know how many concurrent users, RPS, or what the p95 latency is under load. This is critical for production capacity planning and SLA commitments.

GGID does have:
- Prometheus metrics (request count + latency histograms) ✅
- Token bucket rate limiting (Redis) ✅
- Connection pooling (pgxpool) ✅
- Redis caching (posture, ReBAC, rate limits) ✅

**Recommendation**: Build k6 load test suite covering 5 critical flows (login, token exchange, user CRUD, policy evaluate, risk evaluate), establish baselines, identify bottlenecks, and create a capacity planning model.

---

## 2. Performance Budgets

### Per-Endpoint Latency Targets

| Endpoint | p50 Target | p95 Target | p99 Target | Notes |
|----------|-----------|-----------|-----------|-------|
| `POST /auth/login` | 50ms | 200ms | 500ms | Includes credential check + risk eval |
| `POST /auth/refresh` | 10ms | 50ms | 100ms | Redis token lookup |
| `POST /oauth/token` | 20ms | 100ms | 200ms | Token exchange + DPoP |
| `GET /identity/users/{id}` | 5ms | 20ms | 50ms | Redis cached |
| `GET /identity/users` (list) | 20ms | 100ms | 200ms | Paginated DB query |
| `POST /policy/check` | 5ms | 15ms | 30ms | Redis cached decisions |
| `GET /audit/events` | 30ms | 150ms | 300ms | Large table scan |
| `POST /risk/assess` | 20ms | 100ms | 200ms | Signal aggregation |
| `GET /healthz` | 1ms | 5ms | 10ms | Health check |

### Throughput Targets (per service instance)

| Service | Target RPS | Max RPS | Bottleneck |
|---------|-----------|---------|-----------|
| Gateway | 5,000 | 10,000 | CPU (TLS + routing) |
| Auth | 2,000 | 5,000 | PG (credential lookup) |
| OAuth | 1,500 | 3,000 | PG + Redis |
| Identity | 3,000 | 8,000 | Redis cache hit rate |
| Policy | 5,000 | 15,000 | Redis cache (decisions) |
| Audit | 1,000 | 3,000 | PG write throughput |

---

## 3. Capacity Model

### Users → Infrastructure

| Metric | Formula | 1K users | 10K users | 100K users |
|--------|---------|---------|-----------|------------|
| Daily logins | users × 2 | 2K | 20K | 200K |
| Peak RPS (login) | logins / 3600 × 5 (peak factor) | 3 | 28 | 278 |
| Daily API calls | users × 50 | 50K | 500K | 5M |
| Peak RPS (API) | calls / 86400 × 10 | 6 | 58 | 579 |
| Active sessions | users × 0.3 | 300 | 3K | 30K |
| PG storage/year | users × 5MB | 5GB | 50GB | 500GB |
| Redis memory | sessions × 1KB | 300KB | 3MB | 30MB |
| Audit events/day | calls × 1.5 | 75K | 750K | 7.5M |

### Infrastructure Requirements

| Scale | CPU | RAM | PG | Redis | NATS |
|-------|-----|-----|-----|-------|------|
| 1K users | 2 cores | 4GB | 2 vCPU / 4GB | 512MB | 256MB |
| 10K users | 4 cores | 8GB | 4 vCPU / 16GB | 2GB | 512MB |
| 100K users | 16 cores | 32GB | 8 vCPU / 64GB + replica | 8GB cluster | 2GB |

---

## 4. k6 Load Test Scripts

### Login Flow Test

```javascript
// load-tests/login.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const loginDuration = new Trend('login_duration');
const errorRate = new Rate('login_errors');

export const options = {
  stages: [
    { duration: '30s', target: 50 },    // Ramp up
    { duration: '2m', target: 50 },      // Sustained
    { duration: '30s', target: 200 },    // Spike
    { duration: '1m', target: 200 },     // Sustained spike
    { duration: '30s', target: 0 },      // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<200', 'p(99)<500'],
    http_req_failed: ['rate<0.01'],
    login_errors: ['rate<0.02'],
  },
};

export default function () {
  const res = http.post('https://ggid.corp.com/api/v1/auth/login', JSON.stringify({
    username: `user${__VU}@loadtest.com`,
    password: 'LoadTest123!',
  }), { headers: { 'Content-Type': 'application/json' } });

  loginDuration.add(res.timings.duration);

  check(res, {
    'login successful': (r) => r.status === 200,
    'has access token': (r) => r.json('access_token') !== undefined,
  });

  errorRate.add(res.status !== 200);
  sleep(1);
}
```

### Policy Evaluate Test

```javascript
// load-tests/policy-check.js
export default function () {
  const token = getAuthToken(); // Cached token

  const res = http.post('https://ggid.corp.com/api/v1/policy/authorize', JSON.stringify({
    subject: { user_id: `user${__VU % 100}` },
    action: 'api:call',
    resource: '/api/v1/identity/users',
  }), {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  });

  check(res, {
    'policy decision': (r) => r.json('decision') !== undefined,
    'cached': (r) => r.json('cached') !== undefined,
  });
}
```

### Token Exchange Test

```javascript
// load-tests/token-exchange.js
export default function () {
  const res = http.post('https://ggid.corp.com/oauth/token', {
    grant_type: 'client_credentials',
    client_id: `client${__VU % 10}`,
    client_secret: 'secret',
    scope: 'users:read',
  });

  check(res, {
    'token issued': (r) => r.json('access_token') !== undefined,
    'has expires_in': (r) => r.json('expires_in') > 0,
  });
}
```

---

## 5. Bottleneck Identification

### Expected Bottlenecks (by order of likelihood)

| # | Bottleneck | Where | Mitigation |
|---|-----------|-------|-----------|
| 1 | PG connection pool exhaustion | auth/identity services | Increase pool size + connection proxy (PgBouncer) |
| 2 | Redis hot keys | session lookup + rate limit | Read replicas + local LRU cache |
| 3 | TLS handshake CPU | Gateway | TLS session resumption + HTTP/2 |
| 4 | Audit write contention | audit service | Batch writes + NATS async |
| 5 | Go GC pauses | All services | GOGC tuning + GOMEMLIMIT |
| 6 | NATS subject contention | Event publishing | Partition by tenant_id |

### Diagnostic Commands

```bash
# PG connection pool status
psql -c "SELECT count(*), state FROM pg_stat_activity GROUP BY state"

# Slow queries
psql -c "SELECT query, mean_exec_time, calls FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10"

# Redis hot keys
redis-cli --hotkeys

# Go goroutine count
curl localhost:6060/debug/pprof/goroutine?debug=1

# CPU profile
curl localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
```

---

## 6. Test Scenarios

### Stress Test (ramp to failure)

```yaml
stages:
  - { duration: '5m', target: 100 }
  - { duration: '5m', target: 500 }
  - { duration: '5m', target: 1000 }
  - { duration: '5m', target: 2000 }
  - { duration: '5m', target: 5000 }
  - { duration: '2m', target: 0 }
# Goal: find the RPS where p95 > target or error rate > 1%
```

### Soak Test (24h sustained)

```yaml
stages:
  - { duration: '5m', target: 200 }    # Ramp up
  - { duration: '23h55m', target: 200 } # Sustained
# Goal: detect memory leaks, connection leaks, GC degradation
# Monitor: RSS memory, goroutine count, PG connections over time
```

### Spike Test (sudden burst)

```yaml
stages:
  - { duration: '10s', target: 50 }    # Baseline
  - { duration: '10s', target: 1000 }  # 20x spike
  - { duration: '30s', target: 1000 }  # Hold spike
  - { duration: '10s', target: 50 }    # Return to baseline
# Goal: verify rate limiting + circuit breaker + graceful degradation
```

---

## 7. Database Performance

### Index Audit Checklist

```sql
-- Missing indexes (high seq_scan on large tables)
SELECT relname, seq_scan, seq_tup_read, n_live_tup
FROM pg_stat_user_tables
WHERE seq_scan > 100 AND n_live_tup > 10000
ORDER BY seq_tup_read DESC;

-- Unused indexes (candidates for removal)
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0 AND schemaname = 'public';

-- Slow queries
SELECT query, mean_exec_time, calls, total_exec_time
FROM pg_stat_statements
WHERE mean_exec_time > 50
ORDER BY mean_exec_time DESC;
```

### Connection Pool Tuning

```go
// Recommended pgxpool config per service
poolConfig := pgxpool.Config{
    MaxConns:          20,              // Default: 4 × CPU cores
    MinConns:          5,
    MaxConnLifetime:   time.Hour,
    MaxConnIdleTime:   30 * time.Minute,
    HealthCheckPeriod: 30 * time.Second,
}
```

---

## 8. Implementation Backlog with DoD

### P0 — k6 Test Suite + Baseline (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | k6 load test scripts (5 flows) | ✅ login + token + user CRUD + policy + risk ✅ Thresholds defined ✅ ≥3 runs | 4d |
| 2 | Baseline metrics establishment | ✅ RPS per service ✅ p95 latency per endpoint ✅ Published results | 2d |
| 3 | Capacity model spreadsheet | ✅ Users → infra mapping ✅ 3 scales (1K/10K/100K) ✅ Published | 1d |
| 4 | Bottleneck identification report | ✅ Top 5 bottlenecks ✅ Mitigation recommendations ✅ Published | 2d |

### P1 — Soak + Spike + DB Tuning (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 5 | 24h soak test | ✅ No memory leak ✅ No connection leak ✅ Report published | 3d |
| 6 | Spike test (20x burst) | ✅ Rate limiting works ✅ Circuit breaker activates ✅ Graceful degradation | 2d |
| 7 | DB index audit + tuning | ✅ Missing indexes identified ✅ Added ✅ Slow queries < 50ms | 3d |
| 8 | Connection pool tuning | ✅ PgBouncer or tuned pools ✅ No pool exhaustion under load | 2d |

### P2 — CI Integration (Future)

| # | Task | DoD |
|---|------|-----|
| 9 | k6 in CI (nightly benchmark) | Regression detection > 10% |
| 10 | Performance dashboard (Grafana) | Real-time RPS + latency + errors |
| 11 | Autoscaling calibration | HPA targets from load test data |

---

## 9. Competitive Differentiation

| Feature | GGID (target) | Okta | Auth0 | Keycloak |
|---------|---------------|------|-------|----------|
| **Load testing** | k6 5-flow suite | Internal | Internal | Community |
| **Published baselines** | Target | Yes | Yes | No |
| **Capacity model** | 1K-100K | Proprietary | Proprietary | No |
| **Soak test** | 24h | Internal | Internal | No |
| **Performance budgets** | Per-endpoint | Yes | Yes | No |
| **Open source** | Yes | No | No | Yes |

---

## References

- [k6 Load Testing](https://k6.io/) — Go-native load testing
- [Locust](https://locust.io/) — Python load testing alternative
- [Vegeta](https://github.com/tsenart/vegeta) — Go CLI load tester
- [PostgreSQL pg_stat_statements](https://www.postgresql.org/docs/current/pgstatstatements.html) — Query analysis
- [PgBouncer](https://www.pgbouncer.org/) — Connection pooler
- [Prometheus Histograms](https://prometheus.io/docs/practices/histograms/) — Latency measurement
- [GGID Metrics](../services/gateway/internal/middleware/metrics.go) — Prometheus at line 13
- [GGID Rate Limit](../services/gateway/internal/middleware/token_bucket.go) — Redis at line 128
- [GGID Production Hardening](./production-hardening-checklist.md) — Load testing flagged
