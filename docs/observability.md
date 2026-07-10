# Observability Guide

Monitoring, tracing, and alerting for GGID deployments.

---

## Three Pillars

| Pillar | Tool | What It Measures |
|--------|------|-----------------|
| Metrics | Prometheus + Grafana | Request rate, error rate, latency, resource usage |
| Traces | OpenTelemetry (OTLP) | Request flow across services |
| Logs | ELK / Loki | Structured event details |

---

## Prometheus Metrics

### Exposed Endpoints

Each service exposes `/metrics` on its HTTP port:

```bash
GET http://gateway:8080/metrics
GET http://auth:9001/metrics
GET http://policy:8070/metrics
```

### Key Metrics

#### Gateway

| Metric | Type | Description |
|--------|------|-------------|
| `ggid_http_requests_total{method,path,status}` | Counter | Total HTTP requests |
| `ggid_http_request_duration_seconds{method,path}` | Histogram | Request latency |
| `ggid_rate_limit_hits_total{path}` | Counter | Rate-limited requests |
| `ggid_circuit_breaker_state{backend}` | Gauge | 0=closed, 1=open, 2=half-open |
| `ggid_backend_healthy{backend}` | Gauge | 1=healthy, 0=unhealthy |
| `ggid_jwt_verifications_total` | Counter | JWT verification count |
| `ggid_jwt_verification_errors_total{reason}` | Counter | JWT verification failures |

#### Auth Service

| Metric | Type | Description |
|--------|------|-------------|
| `ggid_auth_login_total{result}` | Counter | Login attempts (success/failure) |
| `ggid_auth_register_total` | Counter | New registrations |
| `ggid_auth_mfa_total{method,result}` | Counter | MFA verifications |
| `ggid_auth_active_sessions` | Gauge | Current active sessions |

#### Audit Pipeline

| Metric | Type | Description |
|--------|------|-------------|
| `ggid_nats_published_total` | Counter | Events published |
| `ggid_nats_dropped_total` | Counter | Events dropped (NATS unavailable) |
| `ggid_nats_consumer_lag` | Gauge | Unprocessed events in stream |

---

## OpenTelemetry Tracing

### Configuration

```bash
# Set OTLP endpoint for all services
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
OTEL_SERVICE_NAME=ggid-gateway
```

### Trace Flow

```
Client Request
  └─ Gateway span (entry)
       ├─ JWT verification span
       ├─ Rate limit check span
       └─ Backend proxy span
            ├─ Auth Service span
            │   ├─ DB query span
            │   └─ NATS publish span
            └─ Response span
```

### Viewing Traces

Use **Jaeger** or **Zipkin**:

```bash
# Start Jaeger (Docker)
docker run -d -p 16686:16686 jaegertracing/all-in-one

# Access UI
open http://localhost:16686
```

Search by:
- Service name: `ggid-gateway`
- Operation: `POST /api/v1/auth/login`
- Tags: `http.status_code=500`

---

## Health Checks

### Liveness Probe

```bash
GET /healthz
```

Returns `200 OK` if the process is running and can accept connections.

### Readiness Probe

```bash
GET /readyz
```

Returns `200 OK` if the service can serve requests (DB connected, Redis connected).

### Backend Health

```bash
GET /healthz/backends
```

Returns health status of all backend services:

```json
{
  "auth": {"status": "healthy", "latency_ms": 5},
  "identity": {"status": "healthy", "latency_ms": 3},
  "policy": {"status": "unhealthy", "latency_ms": 0, "error": "connection refused"}
}
```

---

## Grafana Dashboard

### Pre-provisioned Dashboard

GGID includes a Grafana dashboard at `deploy/grafana/dashboards/ggid-overview.json`.

Import manually:
1. Grafana → **Dashboards** → **Import**
2. Upload `deploy/grafana/dashboards/ggid-overview.json`

### Key Panels

| Panel | PromQL |
|-------|--------|
| Request Rate | `rate(ggid_http_requests_total[5m])` |
| Error Rate | `rate(ggid_http_requests_total{status=~"5.."}[5m]) / rate(ggid_http_requests_total[5m])` |
| p95 Latency | `histogram_quantile(0.95, rate(ggid_http_request_duration_seconds_bucket[5m]))` |
| Active Sessions | `ggid_auth_active_sessions` |
| Backend Health | `ggid_backend_healthy` |
| Circuit Breaker | `ggid_circuit_breaker_state` |
| NATS Lag | `ggid_nats_consumer_lag` |

---

## Alerting Rules

```yaml
# deploy/prometheus/alerts.yml
groups:
  - name: ggid
    rules:
      # Service down
      - alert: GGIDServiceDown
        expr: up{job=~"ggid-.*"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Service {{ $labels.job }} is down"

      # High error rate
      - alert: GGIDHighErrorRate
        expr: |
          rate(ggid_http_requests_total{status=~"5.."}[5m])
          / rate(ggid_http_requests_total[5m]) > 0.01
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Error rate above 1%"

      # High latency
      - alert: GGIDHighLatency
        expr: |
          histogram_quantile(0.95,
            rate(ggid_http_request_duration_seconds_bucket[5m])) > 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "p95 latency above 500ms"

      # Backend unhealthy
      - alert: GGIDBackendUnhealthy
        expr: ggid_backend_healthy == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Backend {{ $labels.backend }} is unhealthy"

      # Circuit breaker open
      - alert: GGIDCircuitBreakerOpen
        expr: ggid_circuit_breaker_state == 1
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Circuit breaker open for {{ $labels.backend }}"

      # NATS consumer lag
      - alert: GGIDAuditConsumerLag
        expr: ggid_nats_consumer_lag > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Audit consumer lag > 1000 events"

      # Login failure spike (brute force)
      - alert: GGIDBruteForceAttack
        expr: |
          rate(ggid_auth_login_total{result="failure"}[5m]) > 10
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Possible brute-force attack (>10 failed logins/min)"
```
