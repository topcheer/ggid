# Monitoring and Alerting Guide

Prometheus metrics, Grafana dashboards, health endpoints, log aggregation, alert rules, escalation policies, and on-call procedures.

## Monitoring Stack

```
GGID Services
    │
    ├── /metrics (Prometheus exposition)
    │       │
    │       ▼
    │   Prometheus ──▶ Alertmanager ──▶ PagerDuty/Slack
    │       │
    │       ▼
    │   Grafana (dashboards)
    │
    ├── structured JSON logs
    │       │
    │       ▼
    │   Loki / Elasticsearch (log aggregation)
    │
    └── OpenTelemetry traces
            │
            ▼
        Jaeger / Tempo (distributed tracing)
```

## Prometheus Metrics

### Exposition Endpoint

Each service exposes metrics at `/metrics`:

```bash
GET http://auth:9001/metrics
```

### Key Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ggid_http_requests_total` | Counter | method, path, status | Total HTTP requests |
| `ggid_http_request_duration_seconds` | Histogram | method, path | Request latency |
| `ggid_auth_login_attempts_total` | Counter | result, provider | Login attempts |
| `ggid_auth_active_sessions` | Gauge | — | Current active sessions |
| `ggid_jwt_verifications_total` | Counter | result | JWT verification count |
| `ggid_policy_evaluations_total` | Counter | decision | Policy decisions |
| `ggid_policy_eval_duration_seconds` | Histogram | — | Policy eval latency |
| `ggid_audit_events_published_total` | Counter | action | Audit events |
| `ggid_nats_publish_duration_seconds` | Histogram | stream | NATS publish latency |
| `ggid_db_connections_active` | Gauge | — | Active DB connections |
| `ggid_db_query_duration_seconds` | Histogram | operation | DB query latency |
| `ggid_rate_limit_hits_total` | Counter | endpoint, scope | Rate limit denials |
| `ggid_circuit_breaker_state` | Gauge | service, state | CB open=1/closed=0 |

### Prometheus Scrape Config

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'ggid-gateway'
    static_configs:
      - targets: ['gateway:8080']
    scrape_interval: 15s
    
  - job_name: 'ggid-services'
    static_configs:
      - targets: ['auth:9001', 'identity:8081', 'oauth:9005',
                   'policy:8070', 'org:8071', 'audit:8072']
    scrape_interval: 15s
    metrics_path: /metrics
```

## Grafana Dashboards

### Overview Dashboard

| Panel | Query | Visualization |
|-------|-------|---------------|
| Request rate | `rate(ggid_http_requests_total[5m])` | Graph |
| Error rate | `rate(ggid_http_requests_total{status=~"5.."}[5m])` | Graph |
| P99 latency | `histogram_quantile(0.99, ggid_http_request_duration_seconds_bucket)` | Graph |
| Active sessions | `ggid_auth_active_sessions` | Stat |
| Policy denials | `rate(ggid_policy_evaluations_total{decision="deny"}[5m])` | Graph |
| DB connections | `ggid_db_connections_active` | Gauge |

### Auth Dashboard

| Panel | Query |
|-------|-------|
| Login success rate | `rate(ggid_auth_login_attempts_total{result="success"}[5m])` |
| Login failures | `rate(ggid_auth_login_attempts_total{result="failure"}[5m])` |
| JWT verifications | `rate(ggid_jwt_verifications_total[5m])` |
| MFA step-up rate | `rate(ggid_auth_login_attempts_total{result="step_up"}[5m])` |

### Audit Dashboard

| Panel | Query |
|-------|-------|
| Events/min | `rate(ggid_audit_events_published_total[5m])` |
| Hash chain status | `ggid_audit_chain_valid` (custom) |
| SIEM delivery lag | `ggid_siem_delivery_lag_seconds` |

## Health Endpoints

### Liveness (am I alive?)

```bash
GET /healthz
# → 200 {"status":"ok"}
# If not 200 → Kubernetes restarts pod
```

### Readiness (am I ready to serve?)

```bash
GET /healthz/ready
# → 200 if all dependencies reachable (DB, Redis, NATS)
# → 503 if any dependency down
# Kubernetes removes pod from service if not ready
```

### Dependency Check

```go
func readinessHandler(w http.ResponseWriter, r *http.Request) {
    checks := map[string]error{
        "postgres": pingPostgres(),
        "redis":    pingRedis(),
        "nats":     pingNATS(),
    }
    
    allHealthy := true
    status := make(map[string]string)
    for dep, err := range checks {
        if err != nil {
            status[dep] = "unhealthy: " + err.Error()
            allHealthy = false
        } else {
            status[dep] = "ok"
        }
    }
    
    code := 200
    if !allHealthy { code = 503 }
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(status)
}
```

## Log Aggregation

### Structured Logging

GGID uses structured JSON logging:

```json
{
  "timestamp": "2025-01-15T10:30:00.123Z",
  "level": "info",
  "service": "auth",
  "message": "user login successful",
  "request_id": "req-abc",
  "trace_id": "trace-123",
  "user_id": "uuid",
  "ip": "10.0.1.5",
  "duration_ms": 45
}
```

### Log Levels

| Level | When to Use |
|-------|-------------|
| ERROR | Failures requiring action (DB down, crash) |
| WARN | Degraded but functional (circuit breaker open, retry) |
| INFO | Normal operations (login, CRUD) |
| DEBUG | Detailed diagnostic (SQL queries, policy decisions) |

### Loki (Log Aggregation)

```yaml
# docker-compose addition
loki:
  image: grafana/loki:latest
  ports: ["3100:3100"]
  
promtail:
  image: grafana/promtail:latest
  volumes:
    - /var/log/ggid:/var/log/ggid
  command: -config.file=/etc/promtail/config.yml
```

Query in Grafana:
```logql
{service="auth"} |= "login" | json | level="error"
```

## Alert Rules

### Critical Alerts (Page immediately)

```yaml
groups:
  - name: critical
    rules:
      - alert: ServiceDown
        expr: up == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "{{ $labels.job }} is down"
      
      - alert: HighErrorRate
        expr: |
          rate(ggid_http_requests_total{status=~"5.."}[5m]) 
          / rate(ggid_http_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Error rate >5% on {{ $labels.job }}"
      
      - alert: DatabaseConnectionsExhausted
        expr: ggid_db_connections_active > 24
        for: 1m
        labels:
          severity: critical
      
      - alert: AuditChainBroken
        expr: ggid_audit_chain_valid == 0
        for: 30s
        labels:
          severity: critical
```

### Warning Alerts (Notify, don't page)

```yaml
  - name: warning
    rules:
      - alert: HighLatency
        expr: |
          histogram_quantile(0.99, rate(ggid_http_request_duration_seconds_bucket[5m])) > 0.5
        for: 5m
        labels:
          severity: warning
      
      - alert: CircuitBreakerOpen
        expr: ggid_circuit_breaker_state{state="open"} == 1
        for: 1m
        labels:
          severity: warning
      
      - alert: RateLimitExceeded
        expr: rate(ggid_rate_limit_hits_total[5m]) > 50
        for: 5m
        labels:
          severity: warning
```

## Escalation Policy

```
Alert fires
    │
    ▼
PagerDuty notifies on-call engineer (Level 1)
    │
    ├── Acknowledged in 5 min → Work incident
    │
    ├── Not acknowledged in 5 min
    │   └── Escalate to Level 2 (backup on-call)
    │
    └── Not resolved in 30 min
        └── Escalate to team lead + manager
```

### On-Call Rotation

| Role | Rotation | Hours |
|------|----------|-------|
| Primary on-call | Weekly | 24/7 |
| Secondary on-call | Weekly | 24/7 (backup) |
| Security on-call | Bi-weekly | Critical security alerts |

## Distributed Tracing

```go
import "go.opentelemetry.io/otel"

func traceHandler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx, span := otel.Tracer("ggid").Start(r.Context(), r.URL.Path)
        defer span.End()
        
        span.SetAttributes(
            attribute.String("user.id", getUserID(ctx)),
            attribute.String("tenant.id", getTenantID(ctx)),
        )
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

Trace shows: Gateway → Auth → PostgreSQL (with per-span latency).

## SLOs

| SLO | Target | Window |
|-----|--------|--------|
| Availability | 99.9% | 30 days |
| P99 latency (read) | <500ms | 30 days |
| P99 latency (write) | <1s | 30 days |
| Auth success rate | >98% | 7 days |
| Audit delivery | >99.9% | 30 days |

Error budget: 0.1% = ~43 min downtime/month.

## See Also

- [Gateway Architecture](gateway-architecture.md)
- [Observability Guide](observability-guide.md)
- [Audit Log Architecture](audit-log-architecture.md)
- [Backup and Restore](backup-and-restore.md)
