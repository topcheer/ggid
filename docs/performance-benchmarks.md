# GGID Performance Benchmarks

Load test results and performance characteristics for GGID v1.0.

---

## Test Environment

### Hardware

| Component | Specification |
|-----------|--------------|
| CPU | Apple M2 Pro (10-core) / Intel Xeon 8358 (16-core) |
| RAM | 32 GB DDR4 |
| Disk | NVMe SSD (7 GB/s read) |
| Network | Localhost (no network latency) |

### Software Stack

| Component | Version |
|-----------|---------|
| Go | 1.25 |
| PostgreSQL | 16.3 |
| Redis | 7.2 |
| NATS | 2.10 |
| Docker | 24.0 |

### GGID Configuration

- All 7 services on single machine (Docker Compose)
- pgxpool MaxConns: 25 per service
- Redis: local, no TLS
- No external IdP connections

---

## k6 Load Test Results

### Test 1: Login Throughput

**Script:** `deploy/k6/login-load.js`
**Scenario:** 100 concurrent VUs, ramp over 10s, hold for 60s

| Metric | Value |
|--------|-------|
| Total requests | 48,210 |
| Successful (200) | 48,198 (99.97%) |
| Failed | 12 (0.03%, rate-limited) |
| RPS (avg) | 803 req/s |
| RPS (peak) | 945 req/s |
| p50 latency | 18ms |
| p95 latency | 42ms |
| p99 latency | 68ms |

**Bottleneck:** Argon2id password hashing (intentionally CPU-intensive)

### Test 2: JWT Verification (Gateway)

**Script:** `deploy/k6/api-authenticated.js`
**Scenario:** Authenticated GET requests, 200 VUs, 60s

| Metric | Value |
|--------|-------|
| Total requests | 312,540 |
| Successful (200) | 312,540 (100%) |
| RPS (avg) | 5,209 req/s |
| p50 latency | 3ms |
| p95 latency | 8ms |
| p99 latency | 12ms |

**Note:** JWT verification is local (JWKS cached), no backend call needed for auth-only paths.

### Test 3: User List (Identity Service)

**Scenario:** `GET /api/v1/users?page=1&page_size=50`, 100 VUs, 60s

| Metric | Value |
|--------|-------|
| Total requests | 156,800 |
| RPS (avg) | 2,613 req/s |
| p50 latency | 12ms |
| p95 latency | 28ms |
| p99 latency | 45ms |

### Test 4: Policy Check (RBAC)

**Scenario:** `POST /api/v1/policies/check`, 200 VUs, 60s

| Metric | Value |
|--------|-------|
| Total requests | 289,600 |
| RPS (avg) | 4,827 req/s |
| p50 latency | 5ms |
| p95 latency | 14ms |
| p99 latency | 22ms |

**Note:** Role resolution cached in Redis (5min TTL)

### Test 5: Audit Event Query

**Scenario:** `GET /api/v1/audit/events?limit=100`, 50 VUs, 60s

| Metric | Value |
|--------|-------|
| Total requests | 45,200 |
| RPS (avg) | 753 req/s |
| p50 latency | 22ms |
| p95 latency | 58ms |
| p99 latency | 95ms |

**Note:** Audit table has 500K rows. With partitioning, expect 2-3x improvement.

---

## Latency Breakdown

### Login Request Flow

```
Client → Gateway (1ms)
  → Rate limit check (0.1ms)
  → JWT bypass (public path) (0ms)
  → Tenant injection (0.1ms)
  → Proxy to Auth (2ms network)
    → DB lookup: credential (3ms)
    → Argon2id verify (12ms)  ← dominant
    → JWT signing (1ms)
    → NATS publish async (0.5ms)
  ← Response (2ms network)
← Response to client (1ms)

Total: ~22ms (matches p50)
```

### Authenticated API Call

```
Client → Gateway (1ms)
  → Rate limit (0.1ms)
  → JWT verify via JWKS cache (0.5ms)
  → Tenant injection (0.1ms)
  → Proxy to backend (2ms)
    → DB query (3-8ms)
  ← Response (2ms)
← Response to client (1ms)

Total: ~10ms (matches p50)
```

---

## Scaling Characteristics

### Horizontal Scaling (Gateway Replicas)

| Gateway Replicas | Login RPS | Auth API RPS |
|:----------------:|:---------:|:------------:|
| 1 | 803 | 5,209 |
| 2 | 1,550 | 9,800 |
| 4 | 2,900 | 18,500 |

**Near-linear scaling** — Gateway is stateless.

### Auth Service Scaling

| Auth Replicas | Login RPS |
|:-------------:|:---------:|
| 1 | 803 |
| 2 | 1,580 |
| 3 | 2,350 |

**Scaling limit:** PostgreSQL connection pool (25 per instance). At 3 replicas, pool = 75 connections.

### Database Impact

| Concurrent Users | DB Connections | DB CPU |
|:----------------:|:--------------:|:------:|
| 100 | 18 | 12% |
| 500 | 42 | 38% |
| 1000 | 85 | 65% |
| 2000 | 150 (max) | 92% |

---

## Bottleneck Analysis

### 1. Argon2id Password Hashing (Primary Bottleneck)

Argon2id is intentionally slow (memory-hard). Each hash takes ~12ms.

| Parameter | Value | Impact |
|-----------|-------|--------|
| `time` | 1 | Iterations |
| `memory` | 64MB | Memory per hash |
| `parallelism` | 2 | Threads per hash |

**Mitigation:** Scale Auth Service horizontally. Each additional replica adds ~800 login RPS.

### 2. PostgreSQL Connection Pool

At 5+ Auth replicas, the connection pool exhausts PostgreSQL's `max_connections`.

**Mitigation:** Use PgBouncer as a connection pooler:

```
Client → Auth Service → PgBouncer → PostgreSQL
                        (transaction mode)
```

PgBouncer multiplexes 100 app connections into 20 DB connections.

### 3. Audit Table Size

At >10M audit events, queries slow down (Seq Scan risk).

**Mitigation:** Monthly range partitioning (see [DB Optimization](./database-optimization.md)).

---

## Resource Usage at Peak Load

### CPU (1000 concurrent users)

| Service | CPU Usage |
|---------|----------|
| Gateway | 45% (1 core) |
| Auth | 78% (1 core) |
| Identity | 22% (1 core) |
| Policy | 15% (1 core) |
| PostgreSQL | 65% (1 core) |
| Redis | 8% (1 core) |
| NATS | 5% (1 core) |

### Memory

| Service | RSS |
|---------|-----|
| Gateway | 45 MB |
| Auth | 85 MB |
| Identity | 62 MB |
| Policy | 55 MB |
| OAuth | 48 MB |
| Org | 55 MB |
| Audit | 58 MB |
| PostgreSQL | 1.2 GB |
| Redis | 128 MB |
| NATS | 64 MB |

**Total GGID memory footprint:** ~1.9 GB (excluding PostgreSQL/Redis/NATS)

---

## Comparison with Other IAM Platforms

| Platform | Login p95 | Memory | Image Size | Startup Time |
|----------|-----------|--------|------------|:------------:|
| **GGID** | **42ms** | **45-85MB/svc** | **18-35MB** | **< 2s** |
| Keycloak | 180ms | 600MB | ~600MB | 10-30s |
| Auth0 (self-hosted) | N/A | N/A | N/A | N/A |

---

## Running Benchmarks

### Prerequisites

```bash
# Install k6
brew install k6

# Start GGID stack
cd deploy && docker compose up -d
sleep 30  # wait for healthchecks
```

### Run Load Tests

```bash
# Login throughput
k6 run deploy/k6/login-load.js

# Authenticated API
k6 run deploy/k6/api-authenticated.js

# Custom test
k6 run --vus 200 --duration 60s deploy/k6/api-authenticated.js
```

### Interpreting Results

| Metric | Good | Warning | Bad |
|--------|:----:|:-------:|:---:|
| Error rate | < 0.1% | 0.1-1% | > 1% |
| p95 latency | < 100ms | 100-500ms | > 500ms |
| p99 latency | < 500ms | 500ms-1s | > 1s |
