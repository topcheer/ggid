# Performance Benchmarking

Load testing methodology, tools setup, baseline metrics, regression detection, and CI integration.

## Tools

| Tool | Protocol | Best For |
|------|----------|---------|
| k6 | HTTP/REST | Gateway + API endpoints |
| ghz | gRPC | Inter-service calls |
| Locust | HTTP/WebSocket | Custom scenarios |

## Baseline Metrics per Service

| Service | Endpoint | Target P50 | Target P99 | Max QPS |
|---------|----------|-----------|-----------|---------|
| Gateway | GET /healthz | 2ms | 10ms | 10000 |
| Auth | POST /login | 50ms | 200ms | 1000 |
| Identity | GET /users/{id} | 5ms | 50ms | 5000 |
| Policy | POST /evaluate | 2ms | 20ms | 10000 |
| Audit | POST /events | 3ms | 30ms | 5000 |

## k6 Load Test

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 100 },   // Ramp up
    { duration: '2m', target: 100 },     // Steady
    { duration: '30s', target: 500 },    // Spike
    { duration: '1m', target: 500 },     // Sustain spike
    { duration: '30s', target: 0 },      // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(99)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  let res = http.post('https://gateway.ggid.dev/api/v1/auth/login', JSON.stringify({
    username: 'bench@corp.com',
    password: 'test-password',
  }), { headers: { 'Content-Type': 'application/json' } });

  check(res, {
    'status 200': (r) => r.status === 200,
    'has token': (r) => r.json('access_token') !== undefined,
  });
  sleep(0.1);
}
```

## ghz gRPC Test

```bash
ghz --proto=policy.proto \
    --call=policy.PolicyService.Evaluate \
    --insecure \
    --concurrency=50 \
    --total=10000 \
    --timeout=5s \
    policy-svc:9070
```

## Regression Detection

```yaml
regression_rules:
  - metric: p99_latency
    threshold: "baseline * 1.5"
    action: "fail CI"
    
  - metric: error_rate
    threshold: "0.01"
    action: "fail CI"
    
  - metric: qps_max
    threshold: "baseline * 0.8"
    action: "warn"
```

## CI Integration

```yaml
benchmark_job:
  schedule: "0 2 * * *"  # Nightly
  steps:
    - name: Deploy to bench env
      run: make deploy-bench
    - name: Run k6
      run: k6 run --out json=results.json tests/load/login.js
    - name: Compare baseline
      run: ./scripts/compare-baseline.sh results.json
    - name: Alert on regression
      if: failure()
      run: ./scripts/alert-slack.sh "Performance regression detected"
```

## Result Analysis

| Metric | Healthy | Warning | Critical |
|--------|---------|---------|----------|
| P99 latency | <target | 1.2× target | 1.5× target |
| Error rate | <0.1% | 0.1-1% | >1% |
| Throughput | >target | 80-100% | <80% |
| CPU usage | <70% | 70-90% | >90% |

## See Also

- [Auto-Scaling Strategy](auto-scaling-strategy.md)
- [Connection Pool Tuning](connection-pool-tuning.md)
- [SRE Practices](sre-practices.md)
- [Monitoring and Alerting](monitoring-and-alerting.md)
