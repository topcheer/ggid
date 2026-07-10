# GGID SLA Monitoring

SLO, SLI, and error budget configuration for GGID deployments.

---

## Definitions

| Term | Definition |
|------|------------|
| **SLA** (Service Level Agreement) | Contractual promise to users (e.g., 99.9% uptime) |
| **SLO** (Service Level Objective) | Internal target (e.g., 99.95% uptime) |
| **SLI** (Service Level Indicator) | Measured metric (e.g., successful requests / total requests) |
| **Error Budget** | Allowable failures: `1 - SLO` (e.g., 99.95% → 43.2 min/month downtime allowed) |

---

## SLO Targets

### Platform-Level SLO

| SLO | Target | Measurement Window | Error Budget |
|-----|--------|---------------------|--------------|
| API Uptime | 99.9% | 30 days | 43.2 min/month |
| API Success Rate | 99.5% | 30 days | 0.5% of requests |
| API p95 Latency | < 200ms | 5 min window | N/A |
| API p99 Latency | < 500ms | 5 min window | N/A |

### Per-Service SLO

| Service | Availability SLO | p95 Latency SLO |
|---------|:----------------:|:----------------:|
| Gateway | 99.95% | 50ms (proxy overhead) |
| Auth | 99.9% | 300ms (Argon2id hashing) |
| Identity | 99.9% | 100ms |
| Policy | 99.9% | 50ms |
| OAuth | 99.9% | 100ms |
| Org | 99.5% | 100ms |
| Audit | 99.5% | 100ms (query) |

---

## SLI Definitions

### Availability SLI

```
Availability = successful_requests / total_requests
```

Where `successful_requests` = HTTP status 200-399 (excluding 429 and 401).

### Latency SLI

```
p95_latency = 95th percentile of request duration
p99_latency = 99th percentile of request duration
```

### Authentication SLI

```
Login success rate = successful_logins / total_login_attempts
```

### Audit Pipeline SLI

```
Consumer lag = delivered_seq - acked_seq (on NATS consumer)
```

---

## Error Budgets

### Calculation

For 99.9% SLO over 30 days (43,200 minutes):

```
Error budget = 43,200 × (1 - 0.999) = 43.2 minutes of downtime per month
```

### Budget Tracking

| Budget % Remaining | Action |
|:------------------:|--------|
| 100% - 50% | Normal operations. Deploy freely. |
| 50% - 25% | Caution. Deploy during low-traffic. Monitor closely. |
| 25% - 10% | Warning. Only critical fixes. Freeze feature deploys. |
| 10% - 0% | Critical. Deploy freeze. Focus on reliability. |
| 0% (exhausted) | SLO breach. All deploys frozen until budget recovers. |

### Budget Burn Rate Alerts

| Burn Rate | Window | Action |
|-----------|--------|--------|
| 10x normal | 1 hour | Page on-call (fast burn) |
| 3x normal | 6 hours | Page on-call (slow burn) |
| 1x normal | 3 days | Create ticket (trend analysis) |

---

## Prometheus Configuration

### Recording Rules

```yaml
# prometheus-rules.yml
groups:
  - name: ggid_slo
    interval: 30s
    rules:
      # Availability SLI (per service)
      - record: ggid:availability:ratio_rate5m
        expr: |
          sum(rate(ggid_http_requests_total{status!~"4..|5.."}[5m]))
          /
          sum(rate(ggid_http_requests_total[5m]))

      # Latency SLI (p95)
      - record: ggid:latency:p95_5m
        expr: |
          histogram_quantile(0.95,
            sum(rate(ggid_http_request_duration_seconds_bucket[5m])) by (le))

      # Latency SLI (p99)
      - record: ggid:latency:p99_5m
        expr: |
          histogram_quantile(0.99,
            sum(rate(ggid_http_request_duration_seconds_bucket[5m])) by (le))

      # Error rate
      - record: ggid:error_rate:ratio_rate5m
        expr: |
          sum(rate(ggid_http_requests_total{status=~"5.."}[5m]))
          /
          sum(rate(ggid_http_requests_total[5m]))

      # 30-day availability
      - record: ggid:availability:ratio_rate30d
        expr: |
          sum(rate(ggid_http_requests_total{status!~"4..|5.."}[30d]))
          /
          sum(rate(ggid_http_requests_total[30d]))
```

### Alert Rules

```yaml
  - name: ggid_slo_alerts
    rules:
      # SLO: Availability below 99.9% over 5 min
      - alert: GGIDAvailabilityLow
        expr: ggid:availability:ratio_rate5m < 0.999
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Availability below SLO (99.9%)"

      # SLO: p95 latency above 200ms
      - alert: GGIDLatencyHigh
        expr: ggid:latency:p95_5m > 0.2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "p95 latency above SLO (200ms)"

      # SLO: p99 latency above 500ms
      - alert: GGIDLatencyCritical
        expr: ggid:latency:p99_5m > 0.5
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "p99 latency above SLO (500ms)"

      # Error budget: fast burn (10x in 1h)
      - alert: GGIDErrorBudgetFastBurn
        expr: ggid:error_rate:ratio_rate5m > 0.014
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Error budget burning 10x faster than SLO allows"

      # Service down
      - alert: GGIDServiceDown
        expr: up{job="ggid-gateway"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Gateway is down"

      # Backend unhealthy
      - alert: GGIDBackendUnhealthy
        expr: ggid_backend_healthy == 0
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Backend {{ $labels.backend }} is unhealthy"
```

---

## Grafana Dashboard

### Key Panels

| Panel | Query | Visualization |
|-------|-------|---------------|
| Request Rate | `rate(ggid_http_requests_total[5m])` | Graph (req/sec) |
| Error Rate | `ggid:error_rate:ratio_rate5m` | Stat (percentage) |
| p95 Latency | `ggid:latency:p95_5m` | Graph (ms) |
| p99 Latency | `ggid:latency:p99_5m` | Graph (ms) |
| Availability | `ggid:availability:ratio_rate5m` | Stat (percentage) |
| 30-day Availability | `ggid:availability:ratio_rate30d` | Gauge (percentage) |
| Backend Health | `ggid_backend_healthy` | Table (per backend) |
| Rate Limit Hits | `rate(ggid_rate_limit_hits_total[5m])` | Graph |
| Circuit Breaker | `ggid_circuit_breaker_state` | Table (state per backend) |
| Error Budget Remaining | `1 - ggid:error_rate:ratio_rate30d / (1 - 0.999)` | Gauge |

### Import Dashboard

```bash
# GGID dashboard JSON is pre-provisioned in deploy/grafana/
# If using Helm, the dashboard is included automatically
```

---

## SLO Review Process

### Monthly Review

1. **Measure** — Did we meet all SLOs?
2. **Analyze** — What consumed the most error budget?
3. **Adjust** — Should SLO targets change based on usage patterns?
4. **Action items** — Create tickets for reliability improvements

### SLO Adjustment Criteria

- If SLO met for 6 consecutive months → consider tightening (e.g., 99.9% → 99.95%)
- If SLO breached for 2 consecutive months → consider loosening (e.g., 99.9% → 99.5%)
- Any adjustment requires stakeholder sign-off

---

## Incident Response Integration

### During an Incident

1. **Detect** — Prometheus alert fires
2. **Acknowledge** — On-call engineer acknowledges within 5 min
3. **Investigate** — Check Grafana dashboards, logs, traces
4. **Mitigate** — Rollback, scale up, or hotfix
5. **Resolve** — Confirm SLO metrics return to normal
6. **Post-mortem** — Document root cause and prevention measures

### Error Budget Impact Tracking

After each incident, calculate budget consumed:

```
Incident duration: 15 minutes
Requests during incident: 10,000
Error rate during incident: 50%
Budget consumed: 15 min / 43.2 min = 34.7% of monthly budget
```
