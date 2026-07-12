# Observability Stack Design

Three pillars (metrics/traces/logs), Prometheus + Grafana, OpenTelemetry trace propagation, structured logging, alert routing, SLO definition, and error budget tracking.

## Three Pillars

| Pillar | Tool | Purpose |
|--------|------|---------|
| Metrics | Prometheus + Grafana | Quantitative monitoring, alerting |
| Traces | OpenTelemetry → Jaeger/Tempo | Request flow, latency analysis |
| Logs | Structured JSON → Loki/ELK | Debugging, audit trail |

## Metrics

### Prometheus scrape config

```yaml
scrape_configs:
  - job_name: ggid-services
    kubernetes_sd_configs: [{role: pod}]
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
```

### Key Metrics per Service

| Metric | Type | Alert |
|--------|------|-------|
| `http_requests_total` | Counter | Error rate >1% |
| `http_request_duration_seconds` | Histogram | P99 >500ms |
| `grpc_requests_total` | Counter | Error rate >1% |
| `db_pool_acquired` | Gauge | >80% of max |
| `redis_ops_total` | Counter | Track throughput |
| `nats_pending_messages` | Gauge | >1000 → backlog |

## Traces

OpenTelemetry propagation across all services. See [Distributed Tracing Setup](distributed-tracing-setup.md).

- 10% sampling in production
- Tail-based sampling: 100% of errors + slow requests
- Correlation with audit logs via `trace_id`

## Structured Logging

```go
log.Info("user.login",
    "user_id", userID,
    "ip", clientIP,
    "method", "webauthn",
    "duration_ms", duration.Milliseconds(),
    "trace_id", traceID,
    "tenant_id", tenantID,
)
```

All logs are JSON with: timestamp, level, message, trace_id, tenant_id, request_id.

### Sensitive Data Redaction

```go
// PII automatically masked in logs
log.Info("user.access",
    "email", pii.MaskEmail(email),     // j***@corp.com
    "ip", pii.MaskIP(ip),              // 10.0.1.*
)
```

## Alert Routing

```
Prometheus Alert → AlertManager → Route by severity:
  ├── critical → PagerDuty (24/7 on-call)
  ├── warning → Slack #alerts
  └── info → Slack #monitoring (no page)
```

### Alert Rules

```yaml
groups:
  - name: ggid-critical
    rules:
    - alert: HighErrorRate
      expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.01
      for: 2m
      labels: {severity: critical}
      annotations: {summary: "Error rate >1%"}
    
    - alert: HighLatency
      expr: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 0.5
      for: 5m
      labels: {severity: warning}
```

## SLO Definition

| Service | SLO | Error Budget |
|---------|-----|-------------|
| Gateway | 99.95% availability | 22 min/month |
| Auth | 99.9% login success | 43 min/month |
| Identity | P99 < 200ms | — |
| Policy | P99 < 50ms | — |

### Error Budget Burn Rate

```promql
# 1-hour burn rate (should be <1)
1 - (
  sum(rate(http_requests_total{status!~"5.."}[1h]))
  / sum(rate(http_requests_total[1h]))
)
/ (1 - 0.9995)
```

Alert if burn rate >14.4 (consumes month's budget in 2 hours).

## Grafana Dashboards

| Dashboard | Panels |
|-----------|--------|
| Overview | Golden signals per service, traffic map |
| Auth | Login rate, MFA usage, error breakdown |
| Audit | Events/sec, chain health, query latency |
| Infrastructure | DB pool, Redis, NATS queue depth |
| SLO | Error budget remaining, burn rate |

## Monitoring

| Metric | Alert |
|--------|-------|
| Metrics scrape failures | >1% → Prometheus can't reach pods |
| Trace export latency | >5s → OTel collector overloaded |
| Log ingestion rate | Track for capacity |
| Alert noise (firing/resolved ratio) | >5:1 → tune alert rules |

## See Also

- [Distributed Tracing Setup](distributed-tracing-setup.md)
- [Log Aggregation Design](log-aggregation-design.md)
- [Monitoring and Alerting](monitoring-and-alerting.md)
- [Health Check Design](health-check-design.md)
- [SRE Practices](sre-practices.md)
