# Observability Guide

> Metrics, tracing, logging, and alerting for GGID in production.

---

## Metrics (Prometheus)

GGID exposes metrics at `/metrics`:

```bash
curl http://localhost:8080/metrics | head -10
```

### Key Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `ggid_http_requests_total` | Counter | Total HTTP requests by route/method/status |
| `ggid_http_request_duration_seconds` | Histogram | Request latency |
| `ggid_rate_limit_total` | Counter | Rate limit decisions (allowed/denied) |
| `ggid_auth_attempts_total` | Counter | Login attempts by success/failure |
| `ggid_audit_events_published` | Counter | Events published to NATS |
| `ggid_db_connections_active` | Gauge | Active DB connections |
| `ggid_nats_consumer_lag` | Gauge | Unacked NATS messages |

### Prometheus Scrape Config

```yaml
scrape_configs:
  - job_name: 'ggid-gateway'
    static_configs:
      - targets: ['gateway:8080']
  - job_name: 'ggid-auth'
    static_configs:
      - targets: ['auth:9001']
```

---

## Grafana Dashboard

### Alert Rules

```yaml
groups:
  - name: ggid
    rules:
      - alert: HighErrorRate
        expr: rate(ggid_http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "GGID 5xx error rate > 5%"

      - alert: DBConnectionExhaustion
        expr: ggid_db_connections_active > 80
        for: 2m
        labels:
          severity: warning

      - alert: NATSConsumerLag
        expr: ggid_nats_consumer_lag > 1000
        for: 5m
        labels:
          severity: warning
```

---

## Distributed Tracing (OpenTelemetry)

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
export OTEL_SERVICE_NAME=ggid
```

GGID services emit spans for:
- HTTP request handling (method, path, status, duration)
- gRPC calls (service, method)
- Database queries (table, operation)
- NATS publish/consume

---

## Structured Logging (slog)

All services use Go's `slog` structured logger:

```json
{"time":"2025-07-11T12:00:00Z","level":"INFO","msg":"login","user_id":"usr_abc","ip":"10.0.0.1","duration_ms":45}
```

Log levels: DEBUG, INFO, WARN, ERROR. Configure via `LOG_LEVEL=INFO`.

---

*See: [Performance Tuning](../performance-tuning.md) | [Operations Runbook](../operations-runbook.md) | [SIEM Integration](siem-integration.md)*

*Last updated: 2025-07-11*
