# GGID Observability Guide

Complete guide for monitoring, tracing, and logging GGID in production. Covers
Prometheus metrics, OpenTelemetry distributed tracing, structured logging,
health endpoints, and Grafana dashboard templates.

---

## Table of Contents

- [Metrics (Prometheus)](#metrics-prometheus)
- [Distributed Tracing (OpenTelemetry)](#distributed-tracing-opentelemetry)
- [Structured Logging](#structured-logging)
- [Health Endpoints](#health-endpoints)
- [Grafana Dashboard Templates](#grafana-dashboard-templates)
- [Alerting Rules](#alerting-rules)
- [Deployment](#deployment)

---

## Metrics (Prometheus)

### Enabling Metrics

All GGID services expose a `/metrics` endpoint in Prometheus format:

```bash
# Enable metrics collection (default: enabled)
METRICS_ENABLED=true
METRICS_PATH=/metrics
```

### Metric Naming Convention

All GGID metrics use the `ggid_` prefix:

```
ggid_<service>_<measurement>_<unit>
```

Examples:
- `ggid_gateway_requests_total`
- `ggid_auth_login_duration_seconds`
- `ggid_policy_check_total`

### Available Metrics by Service

#### Gateway

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ggid_gateway_requests_total` | Counter | `method`, `path`, `status`, `tenant_id` | Total HTTP requests |
| `ggid_gateway_request_duration_seconds` | Histogram | `method`, `path` | Request latency |
| `ggid_gateway_response_size_bytes` | Histogram | `method`, `path` | Response body size |
| `ggid_gateway_in_flight_requests` | Gauge | — | Concurrent requests |
| `ggid_gateway_jwt_verify_total` | Counter | `result` | JWT verification count |
| `ggid_gateway_jwt_verify_seconds` | Histogram | — | JWT verification latency |
| `ggid_gateway_rate_limit_denied_total` | Counter | `endpoint`, `tenant_id` | Rate-limited requests |
| `ggid_gateway_webhook deliveries_total` | Counter | `result`, `event_type` | Webhook delivery count |

#### Auth Service

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ggid_auth_login_total` | Counter | `result`, `method` | Login attempts |
| `ggid_auth_login_duration_seconds` | Histogram | `method` | Login processing time |
| `ggid_auth_tokens_issued_total` | Counter | `grant_type` | Tokens issued |
| `ggid_auth_active_sessions` | Gauge | `tenant_id` | Current active sessions |
| `ggid_auth_lockouts_total` | Counter | `tenant_id` | Account lockouts |
| `ggid_auth_mfa_challenge_total` | Counter | `method`, `result` | MFA challenges |
| `ggid_auth_webauthn_registration_total` | Counter | `result` | WebAuthn registrations |
| `ggid_auth_password_reset_total` | Counter | `result` | Password resets |

#### Identity Service

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ggid_identity_users_total` | Gauge | `tenant_id`, `status` | Total user count |
| `ggid_identity_user_operations_total` | Counter | `operation`, `result` | User CRUD operations |
| `ggid_identity_scim_requests_total` | Counter | `method`, `path`, `status` | SCIM API calls |

#### Policy Engine

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ggid_policy_check_total` | Counter | `result`, `policy_type` | Policy evaluations |
| `ggid_policy_check_seconds` | Histogram | `policy_type` | Evaluation latency |
| `ggid_policy_cache_hit_total` | Counter | — | Cache hits |
| `ggid_policy_cache_miss_total` | Counter | — | Cache misses |
| `ggid_policy_cache_size` | Gauge | — | Current cache entries |

#### Audit Service

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ggid_audit_events_total` | Counter | `event_type`, `tenant_id` | Events ingested |
| `ggid_audit_events_queued` | Gauge | — | Events in NATS queue |
| `ggid_audit_query_total` | Counter | `result` | Query API calls |
| `ggid_audit_sse_connections` | Gauge | `tenant_id` | Active SSE streams |

### Infrastructure Metrics

GGID also exposes infrastructure metrics from dependencies:

| Source | Metric | Description |
|--------|--------|-------------|
| PostgreSQL (postgres_exporter) | `pg_stat_activity_count` | Active connections |
| PostgreSQL | `pg_query_duration_seconds` | Query latency |
| PostgreSQL | `pg_database_size_bytes` | Database size |
| Redis (redis_exporter) | `redis_connected_clients` | Connected clients |
| Redis | `redis_memory_used_bytes` | Memory usage |
| Redis | `redis_keyspace_hits_total` | Cache hits |
| NATS | `nats_jetstream_messages` | Total messages |
| NATS | `nats_jetstream_consumer_num_pending` | Consumer lag |

### Prometheus Scrape Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'ggid-gateway'
    static_configs:
      - targets: ['gateway:8080']
    metrics_path: /metrics
    scrape_interval: 15s

  - job_name: 'ggid-auth'
    static_configs:
      - targets: ['auth:9001']

  - job_name: 'ggid-identity'
    static_configs:
      - targets: ['identity:8080']

  - job_name: 'ggid-policy'
    static_configs:
      - targets: ['policy:8070']

  - job_name: 'ggid-audit'
    static_configs:
      - targets: ['audit:8072']

  - job_name: 'postgresql'
    static_configs:
      - targets: ['postgres-exporter:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']

  - job_name: 'nats'
    static_configs:
      - targets: ['nats:8222']
```

---

## Distributed Tracing (OpenTelemetry)

### How GGID Uses Tracing

GGID implements the W3C Trace Context standard (`traceparent` header). The
Gateway generates a trace ID for each incoming request and propagates it to
all downstream services.

```
Client → Gateway (generates traceparent)
              → Auth (child span)
              → Identity (child span)
              → Policy (child span)
              → PostgreSQL (DB span)
```

### Enabling Tracing

```bash
# OTLP exporter endpoint
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
OTEL_EXPORTER_OTLP_PROTOCOL=grpc

# Service name (identifies this service in traces)
OTEL_SERVICE_NAME=ggid

# Sampling rate (1.0 = all traces, 0.1 = 10%)
OTEL_TRACES_SAMPLER_ARG=0.1

# Resource attributes (added to all spans)
OTEL_RESOURCE_ATTRIBUTES=deployment.environment=production,service.namespace=ggid
```

### Span Hierarchy

Each request produces the following spans:

```
trace: 4bf92f3577b34da6a3ce929d0e0e4736
│
├─ gateway.HTTP /api/v1/users GET (120ms)
│  ├─ gateway.JWT.Verify (2.1ms)
│  ├─ gateway.RateLimit.Check (0.8ms)
│  ├─ auth.gRPC Login (45ms)
│  │  ├─ auth.Argon2id.Verify (38ms)
│  │  └─ auth.Redis.SessionWrite (1.2ms)
│  └─ identity.gRPC GetUser (15ms)
│     └─ pgx.Query users SELECT (8ms)
```

### Jaeger Integration

```yaml
# docker-compose.yaml
services:
  jaeger:
    image: jaegertracing/all-in-one:1.55
    ports:
      - "16686:16686"  # Jaeger UI
      - "4317:4317"    # OTLP gRPC
    environment:
      COLLECTOR_OTLP_ENABLED: true

  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.96.0
    command: ["--config=/etc/otel/config.yaml"]
    volumes:
      - ./otel-config.yaml:/etc/otel/config.yaml
```

```yaml
# otel-config.yaml
receivers:
  otlp:
    protocols:
      grpc:

exporters:
  jaeger:
    endpoint: jaeger:14250
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [jaeger]
```

Access Jaeger UI at `http://localhost:16686`.

### Tempo + Grafana Integration

```yaml
services:
  tempo:
    image: grafana/tempo:2.4.0
    command: ["-config.file=/etc/tempo.yaml"]
    volumes:
      - ./tempo.yaml:/etc/tempo.yaml

  grafana:
    image: grafana/grafana:10.3.1
    ports:
      - "3001:3000"
    environment:
      GF_AUTH_ANONYMOUS_ENABLED: "true"
      GF_DATASOURCES_NAME: "Tempo"
      GF_DATASOURCES_TYPE: "tempo"
      GF_DATASOURCES_URL: "http://tempo:3200"
```

---

## Structured Logging

### JSON Log Format

All GGID services emit JSON-structured logs:

```json
{
  "timestamp": "2024-07-15T10:30:45.123Z",
  "level": "info",
  "service": "auth",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "request_id": "req-abc-123",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "message": "login successful",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "password",
  "duration_ms": 8.2,
  "metadata": {
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0...",
    "mfa_used": false
  }
}
```

### Standard Fields

| Field | Type | Always Present | Description |
|-------|------|----------------|-------------|
| `timestamp` | ISO 8601 | Yes | UTC with milliseconds |
| `level` | string | Yes | `debug`, `info`, `warn`, `error` |
| `service` | string | Yes | `gateway`, `auth`, `identity`, etc. |
| `message` | string | Yes | Human-readable |
| `request_id` | string | HTTP requests | Unique per request |
| `trace_id` | string | When tracing enabled | W3C trace ID |
| `span_id` | string | When tracing enabled | W3C span ID |
| `tenant_id` | UUID | Multi-tenant | Tenant context |
| `duration_ms` | float | Operations | Duration |
| `error` | string | Errors | Error message |

### PII Redaction

GGID automatically redacts sensitive fields:

```bash
# Redacted fields
PASSWORD_REDACT_FIELDS=password,email,token,access_token,refresh_token,secret,api_key,ssn,credit_card
```

Redacted output:

```json
{
  "message": "login attempt",
  "email": "[REDACTED]",
  "password": "[REDACTED]"
}
```

### Log Configuration

```bash
# Format: json (production) or text (development)
LOG_FORMAT=json

# Level: debug, info, warn, error
LOG_LEVEL=info

# Include caller info (file:line)
LOG_CALLER=false
```

### ELK / Loki Integration

```yaml
# docker-compose.yaml — Filebeat for log shipping
services:
  filebeat:
    image: elastic/filebeat:8.12.0
    user: root
    volumes:
      - ./filebeat.yml:/usr/share/filebeat/filebeat.yml:ro
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
    command: filebeat -e -strict.perms=false
```

```yaml
# filebeat.yml
filebeat.inputs:
  - type: container
    paths:
      - '/var/lib/docker/containers/*/*.log'
    processors:
      - decode_json_fields:
          fields: ['message']
          target: ''
      - add_docker_metadata: ~

output.elasticsearch:
  hosts: ['elasticsearch:9200']
```

---

## Health Endpoints

Every GGID service exposes three health check endpoints:

| Endpoint | Purpose | Checks | Use Case |
|----------|---------|--------|----------|
| `/healthz` | Liveness | Process is running | K8s liveness probe |
| `/readyz` | Readiness | Service can handle requests | K8s readiness probe |
| `/healthz/deep` | Deep health | DB + Redis + NATS connectivity | Startup probe, manual checks |

### Response Format

```bash
# Liveness — always 200 if process is alive
$ curl localhost:8080/healthz
{"status":"ok"}

# Readiness — 200 if ready, 503 if not
$ curl localhost:8080/readyz
{
  "status": "ok",
  "checks": {
    "database": "ok",
    "redis": "ok"
  }
}

# Deep health — checks all dependencies
$ curl localhost:8080/healthz/deep
{
  "status": "ok",
  "checks": {
    "database": {
      "status": "ok",
      "latency_ms": 1.2,
      "pool_active": 8,
      "pool_idle": 17
    },
    "redis": {
      "status": "ok",
      "latency_ms": 0.3,
      "connected_clients": 5
    },
    "nats": {
      "status": "ok",
      "lag": 0
    }
  }
}
```

### Metrics Endpoint

```bash
$ curl localhost:8080/metrics
# HELP ggid_gateway_requests_total Total HTTP requests
# TYPE ggid_gateway_requests_total counter
ggid_gateway_requests_total{method="GET",path="/api/v1/users",status="200"} 1234
ggid_gateway_requests_total{method="POST",path="/api/v1/auth/login",status="401"} 56
...
```

---

## Grafana Dashboard Templates

### Dashboard JSON Template

Reference dashboards are stored as JSON. Import via Grafana UI or provision
via ConfigMap:

```json
{
  "dashboard": {
    "title": "GGID Overview",
    "tags": ["ggid"],
    "timezone": "browser",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "datasource": "Prometheus",
        "targets": [{
          "expr": "sum(rate(ggid_gateway_requests_total[1m])) by (method)",
          "legendFormat": "{{method}}"
        }]
      },
      {
        "title": "p95 Latency",
        "type": "graph",
        "targets": [{
          "expr": "histogram_quantile(0.95, sum(rate(ggid_gateway_request_duration_seconds_bucket[5m])) by (le))",
          "legendFormat": "p95"
        }]
      },
      {
        "title": "Error Rate",
        "type": "stat",
        "targets": [{
          "expr": "sum(rate(ggid_gateway_requests_total{status=~\"5..\"}[5m])) / sum(rate(ggid_gateway_requests_total[5m])) * 100"
        }],
        "thresholds": [{"value": 1, "color": "red"}]
      },
      {
        "title": "Active Sessions",
        "type": "stat",
        "targets": [{
          "expr": "sum(ggid_auth_active_sessions)"
        }]
      },
      {
        "title": "Login Success Rate",
        "type": "graph",
        "targets": [{
          "expr": "rate(ggid_auth_login_total{result=\"success\"}[5m]) / rate(ggid_auth_login_total[5m]) * 100",
          "legendFormat": "success %"
        }]
      },
      {
        "title": "Rate Limit Denials",
        "type": "graph",
        "targets": [{
          "expr": "sum(rate(ggid_gateway_rate_limit_denied_total[5m])) by (endpoint)",
          "legendFormat": "{{endpoint}}"
        }]
      },
      {
        "title": "DB Connection Pool",
        "type": "graph",
        "targets": [
          {"expr": "pg_stat_activity_count", "legendFormat": "active"},
          {"expr": "pg_settings_max_connections - pg_stat_activity_count", "legendFormat": "idle"}
        ]
      },
      {
        "title": "Redis Memory",
        "type": "gauge",
        "targets": [{
          "expr": "redis_memory_used_bytes / redis_memory_max_bytes * 100"
        }],
        "thresholds": [{"value": 80, "color": "orange"}, {"value": 90, "color": "red"}]
      }
    ],
    "time": {"from": "now-1h", "to": "now"},
    "refresh": "10s"
  }
}
```

### Provisioning via ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ggid-grafana-dashboards
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  ggid-overview.json: |
    { "dashboard": { "title": "GGID Overview", ... } }
  ggid-auth.json: |
    { "dashboard": { "title": "GGID Auth Service", ... } }
```

---

## Alerting Rules

### Prometheus Alert Rules

```yaml
# ggid-alerts.yml
groups:
  - name: ggid
    rules:
      # High error rate
      - alert: GGIDHighErrorRate
        expr: |
          sum(rate(ggid_gateway_requests_total{status=~"5.."}[5m]))
          / sum(rate(ggid_gateway_requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "GGID error rate > 5%"
          description: "Gateway error rate is {{ $value | humanizePercentage }}"

      # High p95 latency
      - alert: GGIDHighLatency
        expr: |
          histogram_quantile(0.95, rate(ggid_gateway_request_duration_seconds_bucket[5m])) > 2
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "GGID p95 latency > 2s"

      # Login failures spike
      - alert: GGIDLoginFailuresSpike
        expr: |
          rate(ggid_auth_login_total{result="failure"}[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Login failures > 10/min"

      # Service down
      - alert: GGIDServiceDown
        expr: up{job=~"ggid-.*"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "{{ $labels.job }} is down"

      # DB connection pool exhausted
      - alert: GGIDDBPoolExhausted
        expr: pg_stat_activity_count / pg_settings_max_connections > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "PostgreSQL connection pool > 80%"

      # Redis memory
      - alert: GGIDRedisMemoryHigh
        expr: redis_memory_used_bytes / redis_memory_max_bytes > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Redis memory > 90%"

      # NATS consumer lag
      - alert: GGIDNATSConsumerLag
        expr: nats_jetstream_consumer_num_pending > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "NATS audit consumer lag > 1000 messages"
```

---

## Deployment

### Full Observability Stack (Docker Compose)

```yaml
# docker-compose.observability.yaml
services:
  prometheus:
    image: prom/prometheus:v2.50.0
    ports: ["9090:9090"]
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus

  grafana:
    image: grafana/grafana:10.3.1
    ports: ["3001:3000"]
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin

  jaeger:
    image: jaegertracing/all-in-one:1.55
    ports: ["16686:16686", "4317:4317"]
    environment:
      COLLECTOR_OTLP_ENABLED: "true"

  postgres-exporter:
    image: prometheuscommunity/postgres-exporter:v0.15.0
    environment:
      DATA_SOURCE_NAME: "postgresql://ggid:password@postgres:5432/ggid?sslmode=disable"
    ports: ["9187:9187"]

  redis-exporter:
    image: oliver006/redis_exporter:v1.59.0
    environment:
      REDIS_ADDR: "redis://redis:6379"
    ports: ["9121:9121"]

volumes:
  prometheus-data:
  grafana-data:
```

```bash
# Start observability stack alongside GGID
docker compose -f docker-compose.yaml -f docker-compose.observability.yaml up -d

# Access:
# Prometheus:  http://localhost:9090
# Grafana:     http://localhost:3001 (admin/admin)
# Jaeger:      http://localhost:16686
```

---

---

## Alerting Integrations

### PagerDuty Integration

Route critical alerts to PagerDuty for on-call escalation:

```yaml
# alertmanager.yml
receivers:
  - name: pagerduty-critical
    pagerduty_configs:
      - service_key: YOUR_PAGERDUTY_KEY
        description: "{{ .GroupLabels.alertname }}: {{ .GroupLabels.service }}"
        severity: critical

route:
  group_by: ['alertname', 'service']
  group_wait: 10s
  group_interval: 5m
  repeat_interval: 30m
  receiver: pagerduty-critical
  routes:
    - matchers: ['severity="critical"']
      receiver: pagerduty-critical
    - matchers: ['severity="warning"']
      receiver: slack-warnings
```

### Slack Notifications

```yaml
receivers:
  - name: slack-warnings
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK'
        channel: '#ggid-alerts'
        title: '[{{ .Status }}] {{ .GroupLabels.alertname }}'
        text: >-
          {{ range .Alerts }}
          *Service:* {{ .Labels.service }}
          *Severity:* {{ .Labels.severity }}
          *Description:* {{ .Annotations.description }}
          {{ end }}
```

### Alert Severity Mapping

| Severity | Examples | Notification | Escalation |
|----------|----------|-------------|------------|
| Critical | Service down, DB unreachable | PagerDuty + Slack | Immediate page |
| Warning | High error rate, slow queries | Slack only | During business hours |
| Info | Deployment complete, config change | Slack (optional) | No escalation |

---

## Distributed Tracing Across 7 Microservices

### Request Flow (Gateway → 6 services)

```
Client → Gateway (span: HTTP request)
         → Auth (span: gRPC VerifyToken)
         → Identity (span: gRPC GetUser)
            → PostgreSQL (span: SELECT users)
         → Policy (span: gRPC CheckPermission)
            → Redis (span: GET policy cache)
         → Audit (span: gRPC PublishEvent)
            → NATS (span: JetStream publish)
```

Each span carries the same `trace_id`. The Jaeger UI shows the full waterfall:

```
[Gateway: 45ms] ────────────────────────────────────────────────
  [Auth VerifyToken: 2ms] ██
  [Identity GetUser: 8ms] ████
    [PostgreSQL: 3ms] ██
  [Policy Check: 4ms] ██
    [Redis: 0.5ms] ▏
  [Audit Publish: 1ms] ▏
```

### Adding Custom Spans

```go
import "go.opentelemetry.io/otel"

func (s *AuthService) Login(ctx context.Context, username, password string) (*Token, error) {
    ctx, span := otel.Tracer("auth").Start(ctx, "AuthService.Login")
    defer span.End()

    span.SetAttributes(
        attribute.String("user.username", username),
    )

    // ... authentication logic ...

    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(attribute.Bool("auth.success", true))
    return token, nil
}
```

---

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/instrumentation/go/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Benchmark Guide](./benchmark.md) — Performance metrics
- [High Availability](./high-availability.md) — HA monitoring
