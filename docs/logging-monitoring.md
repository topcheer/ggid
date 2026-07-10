# Logging and Monitoring

Structured logging, Prometheus metrics, Grafana dashboards, alerting rules,
log retention, and secret redaction in GGID.

---

## Table of Contents

- [Structured Logging](#structured-logging)
- [Log Levels](#log-levels)
- [Correlation IDs](#correlation-ids)
- [Secret Redaction](#secret-redaction)
- [Prometheus Metrics](#prometheus-metrics)
- [Grafana Dashboards](#grafana-dashboards)
- [Alerting Rules](#alerting-rules)
- [Log Retention](#log-retention)

---

## Structured Logging

All GGID services emit JSON-formatted logs to stdout:

```json
{
  "timestamp": "2024-01-15T10:30:00.123Z",
  "level": "info",
  "service": "auth",
  "method": "POST",
  "path": "/api/v1/auth/login",
  "status": 200,
  "duration": "45ms",
  "request_id": "req-abc123",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "login successful",
  "source_ip": "192.168.1.50"
}
```

### Standard Fields

| Field | Always Present | Description |
|-------|:--------------:|-------------|
| `timestamp` | Yes | RFC 3339 with milliseconds |
| `level` | Yes | `debug`, `info`, `warn`, `error` |
| `service` | Yes | Service name (auth, gateway, etc.) |
| `message` | Yes | Human-readable message |
| `request_id` | On HTTP | Correlation ID (UUID) |
| `tenant_id` | When available | Tenant context |
| `user_id` | When authenticated | Acting user |
| `method` | On HTTP | HTTP method |
| `path` | On HTTP | Request path |
| `status` | On HTTP | Response status code |
| `duration` | On HTTP | Request duration |
| `source_ip` | On HTTP | Client IP |
| `user_agent` | On HTTP | Client user agent |
| `error` | On error | Error message |
| `stack` | On panic/error | Stack trace |

### Configuration

```bash
LOG_LEVEL=info       # debug, info, warn, error
LOG_FORMAT=json      # json (production) or text (dev)
```

---

## Log Levels

| Level | When to Use | Production |
|-------|-------------|:----------:|
| `debug` | Detailed diagnostic info | Disabled |
| `info` | Normal operation (logins, API calls) | Enabled |
| `warn` | Unexpected but non-fatal (rate limit, cache miss) | Enabled |
| `error` | Errors requiring attention (500, DB failure, panic) | Enabled |

### Per-Service Defaults

```yaml
logging:
  gateway: info
  auth: info
  oauth: info
  identity: info
  policy: info
  org: info
  audit: info
```

---

## Correlation IDs

Every HTTP request gets a `request_id` (UUID) that flows through all
services via gRPC metadata and HTTP headers.

### Request Flow

```
Client → Gateway (generates request_id)
  → Auth Service (inherits request_id via gRPC metadata)
  → PostgreSQL (request_id in log context)
  → Audit Service (request_id stored in audit event)
```

### Header Propagation

```
X-Request-ID: req-abc123      # Set by Gateway, propagated downstream
X-Correlation-ID: req-abc123  # Alias for distributed tracing
```

### Tracing in Logs

```bash
# Find all logs for a specific request across all services
docker logs ggid-gateway 2>&1 | jq 'select(.request_id == "req-abc123")'
docker logs ggid-auth 2>&1 | jq 'select(.request_id == "req-abc123")'
```

---

## Secret Redaction

GGID automatically redacts sensitive fields from logs:

### Redacted Fields

| Pattern | Replacement |
|---------|-------------|
| `password` | `"[REDACTED]"` |
| `token` | `"[REDACTED]"` |
| `secret` | `"[REDACTED]"` |
| `api_key` | `"[REDACTED]"` |
| `authorization` | `"[REDACTED]"` |
| `refresh_token` | `"[REDACTED]"` |
| `client_secret` | `"[REDACTED]"` |
| `private_key` | `"[REDACTED]"` |
| Credit card numbers | `"[REDACTED]"` |
| SSN (xxx-xx-xxxx) | `"[REDACTED]"` |

### Implementation

```go
var redactKeys = []string{
    "password", "token", "secret", "api_key", "authorization",
    "refresh_token", "client_secret", "private_key",
}

func redactFields(fields map[string]any) map[string]any {
    for k := range fields {
        for _, pattern := range redactKeys {
            if strings.Contains(strings.ToLower(k), pattern) {
                fields[k] = "[REDACTED]"
            }
        }
    }
    return fields
}
```

---

## Prometheus Metrics

### Metrics Endpoint

Each service exposes Prometheus metrics at `/metrics`:

```
http://gateway:8080/metrics
http://auth:9001/metrics
http://oauth:9005/metrics
```

### Key Metrics

#### HTTP Metrics

```
# Request rate
ggid_http_requests_total{service="auth",method="POST",path="/api/v1/auth/login",status="200"}

# Request duration histogram
ggid_http_request_duration_seconds_bucket{service="auth",le="0.1"}
ggid_http_request_duration_seconds_p99{service="auth"}

# Active connections
ggid_http_active_connections{service="gateway"}
```

#### Auth Metrics

```
# Login attempts
ggid_auth_login_total{result="success"}
ggid_auth_login_total{result="failed"}

# Active sessions
ggid_auth_active_sessions{tenant_id="..."}

# MFA enrollments
ggid_auth_mfa_enabled_total{method="totp"}
ggid_auth_mfa_enabled_total{method="webauthn"}

# Account lockouts
ggid_auth_account_locked_total
```

#### OAuth Metrics

```
# Token issuance
ggid_oauth_tokens_issued_total{grant_type="authorization_code"}
ggid_oauth_tokens_issued_total{grant_type="client_credentials"}
ggid_oauth_tokens_issued_total{grant_type="refresh_token"}

# Refresh token rotation
ggid_oauth_refresh_rotations_total
ggid_oauth_refresh_reuse_detected_total

# Active tokens
ggid_oauth_active_access_tokens
ggid_oauth_active_refresh_tokens
```

#### System Metrics

```
# Go runtime
go_goroutines
go_memstats_alloc_bytes
go_memstats_gc_duration_seconds

# Database
ggid_db_connections{state="idle"}
ggid_db_connections{state="active"}
ggid_db_query_duration_seconds_bucket

# Redis
ggid_redis_operations_total{op="GET"}
ggid_redis_operations_total{op="SET"}

# NATS
ggid_nats_publish_total
ggid_nats_subscribe_total
```

### Prometheus Scrape Config

```yaml
scrape_configs:
  - job_name: 'ggid'
    static_configs:
      - targets:
        - gateway:8080
        - auth:9001
        - oauth:9005
        - identity:8080
        - policy:8070
        - org:8071
        - audit:8072
    scrape_interval: 15s
    metrics_path: /metrics
```

---

## Grafana Dashboards

### Import Dashboards

```bash
# Import pre-built GGID dashboard
curl -X POST http://grafana:3000/api/dashboards/db \
  -H "Authorization: Bearer $GRAFANA_TOKEN" \
  -H "Content-Type: application/json" \
  -d @docs/grafana/ggid-overview.json
```

### Key Dashboard Panels

| Panel | Query | Purpose |
|-------|-------|---------|
| Request Rate | `rate(ggid_http_requests_total[5m])` | QPS per service |
| Error Rate | `rate(ggid_http_requests_total{status=~"5.."}[5m])` | 5xx percentage |
| P99 Latency | `histogram_quantile(0.99, ggid_http_request_duration_seconds_bucket)` | Slowest requests |
| Active Sessions | `ggid_auth_active_sessions` | Concurrent users |
| Login Failures | `rate(ggid_auth_login_total{result="failed"}[5m])` | Attack detection |
| Token Issuance | `rate(ggid_oauth_tokens_issued_total[5m])` | OAuth volume |
| DB Connections | `ggid_db_connections` | Pool health |
| Goroutines | `go_goroutines` | Memory leak detection |
| GC Duration | `rate(go_memstats_gc_duration_seconds_sum[5m])` | GC pressure |

---

## Alerting Rules

### Critical Alerts

```yaml
groups:
  - name: ggid-critical
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate(ggid_http_requests_total{status=~"5.."}[5m])) by (service)
          / sum(rate(ggid_http_requests_total[5m])) by (service) > 0.05
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "{{ $labels.service }} error rate > 5%"

      - alert: LoginFailuresSpike
        expr: rate(ggid_auth_login_total{result="failed"}[5m]) > 10
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Login failure rate > 10/min (possible attack)"

      - alert: RefreshTokenReuse
        expr: rate(ggid_oauth_refresh_reuse_detected_total[5m]) > 0
        for: 0s
        labels:
          severity: critical
        annotations:
          summary: "Refresh token reuse detected (possible token theft)"

      - alert: DBConnectionExhaustion
        expr: ggid_db_connections{state="active"} / (ggid_db_connections{state="active"} + ggid_db_connections{state="idle"}) > 0.8
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "DB connection pool > 80% utilized"
```

### Warning Alerts

```yaml
  - name: ggid-warning
    rules:
      - alert: HighLatency
        expr: histogram_quantile(0.99, ggid_http_request_duration_seconds_bucket) > 2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "P99 latency > 2 seconds"

      - alert: GoroutineLeak
        expr: go_goroutines > 1000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Goroutine count > 1000 (possible leak)"

      - alert: DBPoolGrowth
        expr: ggid_db_connections{state="active"} > 50
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "DB active connections > 50"
```

---

## Log Retention

### Retention Policy

| Log Type | Hot Storage | Cold Storage | Total |
|----------|------------|-------------|-------|
| Application logs (stdout) | 7 days (Loki) | 30 days (S3) | 37 days |
| Audit logs (PostgreSQL) | 90 days | 1 year (archive) | 1 year |
| Audit logs (NATS) | 30 days | 30 days (replay) | 60 days |
| Access logs (Nginx) | 7 days | 30 days (S3) | 37 days |
| Prometheus metrics | 15 days | 90 days (Thanos) | 105 days |

### Loki Configuration

```yaml
loki:
  limits_config:
    retention_period: 168h      # 7 days hot
    max_query_series: 500
  compactor:
    working_directory: /tmp/loki/compactor
    shared_store: s3
    retention_enabled: true
    retention_delete_delay: 2h
```

### Audit Log Retention SQL

```sql
-- Archive old audit events (scheduled daily)
INSERT INTO audit_events_archive
SELECT * FROM audit_events
WHERE created_at < NOW() - INTERVAL '90 days';

DELETE FROM audit_events
WHERE created_at < NOW() - INTERVAL '90 days';
```

### Log Rotation (Docker)

```yaml
# /etc/docker/daemon.json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "5"
  }
}
```

Each container gets max 250MB of logs (5 files × 50MB).
