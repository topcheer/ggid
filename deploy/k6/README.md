# GGID Performance Benchmarks (k6)

## Prerequisites

```bash
# Install k6
brew install k6  # macOS
# or: https://k6.io/docs/getting-started/installation/
```

## Running Benchmarks

### 1. Register + Login Flow
Tests the full user registration → login → authenticated request pipeline.

```bash
k6 run deploy/k6/register-and-login.js
```

**Thresholds:**
- Error rate < 5%
- p95 latency < 500ms
- Login p95 < 300ms
- Register p95 < 400ms

### 2. API Throughput (Sustained Load)
Tests sustained throughput at 100 req/sec across multiple endpoints.

```bash
k6 run deploy/k6/api-throughput.js
```

**Thresholds:**
- Error rate < 5%
- p95 latency < 500ms
- p99 latency < 1000ms

### 3. JWT Verification Burst
Tests JWT verification under burst load (100 concurrent VUs).

```bash
k6 run deploy/k6/jwt-verify.js
```

**Thresholds:**
- Error rate < 1%
- JWT verify p95 < 100ms
- JWT verify p99 < 200ms

## Custom Configuration

```bash
# Target a different environment
k6 run -e BASE_URL=https://iam.example.com deploy/k6/register-and-login.js

# Use a different tenant
k6 run -e TENANT_ID=your-tenant-uuid deploy/k6/api-throughput.js
```

## Results

Results are written to `deploy/k6/results.json`. Use `k6 inspect` to view:

```bash
k6 inspect deploy/k6/register-and-login.js
```
