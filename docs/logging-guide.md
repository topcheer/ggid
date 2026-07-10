# Structured Logging Guide

GGID structured logging with JSON output, request tracing, PII redaction, and ELK/Loki integration.

---

## Log Format

All GGID services emit JSON structured logs:

```json
{
  "timestamp": "2024-07-10T12:00:00.123Z",
  "level": "info",
  "service": "auth",
  "message": "login successful",
  "method": "POST",
  "path": "/api/v1/auth/login",
  "status": 200,
  "duration": "18ms",
  "remote_addr": "192.168.1.100",
  "request_id": "req-abc123",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

---

## Log Levels

| Level | When to Use | Example |
|-------|------------|---------|
| `debug` | Detailed diagnostic info (disabled in production) | Query plans, cache hits/misses |
| `info` | Normal operations | Login success, user created, server started |
| `warn` | Unexpected but non-critical | Rate limit hit, cache miss fallback, deprecated API call |
| `error` | Failures requiring attention | DB error, NATS publish failure, panic recovered |
| `fatal` | Unrecoverable error (service exits) | Can't connect to database on startup |

---

## Request ID Correlation

Every HTTP request gets a unique `request_id`. This ID propagates through:

```
Client → Gateway (generates request_id)
  → Auth Service (same request_id in log)
  → Database query (logged with request_id)
  → NATS audit event (includes request_id)
  → Response header: X-Request-ID: req-abc123
```

### Gateway Middleware

```go
// RequestID middleware generates a UUID for each request
func RequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        w.Header().Set("X-Request-ID", requestID)
        ctx := context.WithValue(r.Context(), requestIDKey, requestID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## PII Redaction

GGID automatically redacts sensitive fields in logs:

| Field | Redacted As |
|-------|-------------|
| `password` | `"***REDACTED***"` |
| `token` / `access_token` / `refresh_token` | `"***REDACTED***"` |
| `secret` / `client_secret` | `"***REDACTED***"` |
| `Authorization` header | `"Bearer ***"` |
| `api_key` | `"***REDACTED***"` |

```go
// pkg/pii package handles redaction
import "github.com/ggid/ggid/pkg/pii"

redacted := pii.Redact(map[string]any{
    "email":    "john@example.com",
    "password": "Secret123!",
    "token":    "eyJhbGc...",
})
// Result: {"email": "john@example.com", "password": "***REDACTED***", "token": "***REDACTED***"}
```

### Configurable Redaction

```bash
PII_REDACT_FIELDS=password,token,secret,ssn,credit_card
```

---

## ELK Integration

### Filebeat Configuration

```yaml
# filebeat.yml
filebeat.inputs:
  - type: container
    paths:
      - /var/lib/docker/containers/*/*.log
    processors:
      - decode_json_fields:
          fields: ["message"]
          target: ""
      - add_kubernetes_metadata:
          host: ${NODE_NAME}

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  indices:
    - index: "ggid-%{[service]}-%{+yyyy.MM.dd}"
```

### Kibana Dashboard

Create index patterns:
- `ggid-gateway-*` — Gateway logs
- `ggid-auth-*` — Auth service logs
- `ggid-audit-*` — Audit service logs

Useful Kibana queries:
- Failed logins: `level:error AND message:"login_failed"`
- Slow requests: `duration: [500ms TO *]`
- Per tenant: `tenant_id: "00000000-0000-0000-0000-000000000001"`
- Request trace: `request_id: "req-abc123"`

---

## Loki Integration

### Promtail Configuration

```yaml
# promtail.yml
server:
  http_listen_port: 9080

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: ggid
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
        filters:
          - name: label
            values: ["com.docker.compose.project=ggid"]
    pipeline_stages:
      - json:
          expressions:
            level: level
            service: service
            request_id: request_id
            tenant_id: tenant_id
      - labels:
          level:
          service:
```

### Grafana LogQL Queries

```logql
# All errors in last 5 minutes
{service="auth", level="error"}

# Login failures by IP
{service="auth"} |= "login_failed" | json | line_format "{{.remote_addr}}"

# Trace a request across all services
{request_id="req-abc123"}

# Slow requests (>500ms)
{service="gateway"} | json | duration > 500000000
```

---

## Log Rotation

### Docker

```yaml
# docker-compose.yaml
services:
  auth:
    logging:
      driver: json-file
      options:
        max-size: "100m"
        max-file: "5"
```

### Kubernetes

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
    - name: auth
      resources:
        limits:
          memory: 256Mi
```

Kubernetes automatically rotates container logs via `kubelet`.

---

## Best Practices

1. **Always log with request_id** — Enables tracing across services
2. **Use structured JSON** — Never string concatenation for logs
3. **Never log passwords or secrets** — PII redaction handles this
4. **Log at the right level** — `info` for normal ops, `error` for failures
5. **Include duration** — Every HTTP request log should include response time
6. **Don't over-log** — Excessive logging impacts performance and storage
7. **Monitor log volume** — Alert if log rate spikes (possible error loop)
