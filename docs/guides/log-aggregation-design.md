# Log Aggregation Design

Structured JSON logging, log levels, correlation IDs, sensitive data redaction, routing to Loki/ELK, retention policies, and cost optimization.

## Structured Logging

GGID uses structured JSON logging across all services:

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

logger.Info("user.login",
    "user_id", userID,
    "method", "password+mfa",
    "ip", clientIP,
    "trace_id", traceID,
    "duration_ms", 45,
)
```

### Output

```json
{
  "time": "2025-01-15T10:30:00.123Z",
  "level": "INFO",
  "msg": "user.login",
  "user_id": "uuid",
  "method": "password+mfa",
  "ip": "10.0.1.5",
  "trace_id": "0af7651916cd43dd",
  "duration_ms": 45,
  "service": "auth",
  "version": "2.3.1"
}
```

## Log Levels

| Level | When to Use | Example |
|-------|------------|---------|
| `ERROR` | Operation failed, needs attention | DB connection lost |
| `WARN` | Something unexpected but handled | Retried after transient error |
| `INFO` | Normal operations (default) | User login, request completed |
| `DEBUG` | Diagnostic detail (off in prod) | Query plan, cache hit/miss |
| `TRACE` | Very detailed (off in prod) | Function entry/exit |

### Level Configuration

```yaml
log:
  level: INFO
  per_service_override:
    auth: DEBUG      # Troubleshooting auth
    audit: INFO
    gateway: WARN    # High volume, reduce noise
```

## Correlation IDs

Every request gets a unique ID propagated across services:

```go
func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        
        ctx := context.WithValue(r.Context(), "request_id", requestID)
        w.Header().Set("X-Request-ID", requestID)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Propagated Fields

| Field | Source | Purpose |
|-------|--------|---------|
| `request_id` | Gateway generates | Single HTTP request |
| `trace_id` | OpenTelemetry | Distributed trace |
| `user_id` | JWT claims | User correlation |
| `tenant_id` | JWT claims | Tenant isolation |
| `session_id` | Session store | Session tracking |

## Sensitive Data Redaction

```go
var sensitiveFields = []string{
    "password", "token", "secret", "api_key",
    "ssn", "credit_card", "authorization",
}

func redactSensitive(attrs map[string]interface{}) map[string]interface{} {
    for key, val := range attrs {
        if isSensitive(key, sensitiveFields) {
            attrs[key] = "[REDACTED]"
        }
        if isPII(key) {
            attrs[key] = maskValue(val.(string))
        }
    }
    return attrs
}

func isSensitive(key string, patterns []string) bool {
    lower := strings.ToLower(key)
    for _, p := range patterns {
        if strings.Contains(lower, p) { return true }
    }
    return false
}
```

### Redaction Rules

| Field Type | Redaction | Example |
|-----------|-----------|---------|
| password | `[REDACTED]` | `"password": "[REDACTED]"` |
| email | `j***@corp.com` | `"email": "j***@corp.com"` |
| phone | `+1-555-01**` | `"phone": "+1-555-01**"` |
| token/JWT | `eyJ...` (first 8 chars) | `"token": "eyJhbGci*"` |
| IP | `10.0.1.*` | `"ip": "10.0.1.*"` |

## Log Routing

### Loki (Recommended for GGID)

```yaml
# promtail.yml
positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: ggid
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
        filters: [{name: label, values: ["com.docker.compose.project=ggid"]}]
    pipeline_stages:
      - json:
          expressions:
            level: level
            service: service
            trace_id: trace_id
      - labels:
          level:
          service:
```

### ELK Stack

```yaml
# Filebeat → Logstash → Elasticsearch → Kibana
filebeat:
  containers:
    enabled: true
  output.logstash:
    hosts: ["logstash:5044"]
```

### Query Example

```logql
# Loki: Find all ERROR logs for a user
{service="auth", level="ERROR"} |= "user_id=uuid"

# Find slow requests
{level="WARN"} |= "slow" | duration_ms > 500
```

## Retention Policies

| Log Type | Hot Storage | Warm Storage | Cold Storage |
|----------|------------|-------------|-------------|
| Application logs | 7 days | 30 days | — |
| Audit logs | 90 days | 1 year | 7 years (archive) |
| Error logs | 30 days | 90 days | — |
| Debug logs | 24 hours | — | — |

```yaml
retention:
  loki:
    max_age: 30d
    compactor: enabled
    retention_delete_worker_count: 50
```

## Cost Optimization

### Reduce Log Volume

| Strategy | Impact | Implementation |
|----------|--------|----------------|
| Set gateway to WARN level | -60% volume | Config override |
| Sample INFO logs (10%) | -40% volume | Drop 90% of INFO |
| Drop health check logs | -15% volume | Filter `/healthz` |
| Remove redundant fields | -20% per log | Remove duplicate trace context |

### Sample Implementation

```go
func shouldLog(level slog.Level, msg string) bool {
    // Always log errors and warnings
    if level <= slog.LevelWarn { return true }
    
    // Skip health check noise
    if strings.Contains(msg, "health.check") { return false }
    
    // Sample INFO logs (10%)
    if level == slog.LevelInfo && rand.Float32() > 0.1 { return false }
    
    return true
}
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Log volume per service | Spike → possible issue |
| ERROR rate | >0.1% → investigate |
| Log pipeline lag | >30s → scale logstash/loki |
| Disk usage (log storage) | >80% → rotate faster |

## See Also

- [Distributed Tracing Setup](distributed-tracing-setup.md)
- [Monitoring and Alerting](monitoring-and-alerting.md)
- [Audit Log Architecture](audit-log-architecture.md)
- [Health Check Design](health-check-design.md)
