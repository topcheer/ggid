# Distributed Tracing — Technical Guide

> Feature: W3C Trace Context + OpenTelemetry-style Span Export
> Location: `services/gateway/internal/middleware/otel.go`

## What It Does

GGID implements distributed tracing using the W3C Trace Context standard. Every request passing through the gateway generates a trace with spans across services, enabling end-to-end visibility into request latency, errors, and service dependencies.

## W3C Trace Context

GGID uses the standard `traceparent` header for trace propagation:

```
traceparent: 00-<trace-id>-<span-id>-<trace-flags>
```

- **trace-id**: 32-character hex (128-bit) — unique per request chain.
- **span-id**: 16-character hex (64-bit) — unique per service hop.
- **trace-flags**: 1 byte (e.g., `01` = sampled).

If no `traceparent` header is present, the gateway generates a new trace ID.

## Span Model

```go
type Span struct {
    TraceID     string    `json:"trace_id"`
    SpanID      string    `json:"span_id"`
    ParentID    string    `json:"parent_span_id,omitempty"`
    Operation   string    `json:"operation"`
    ServiceName string    `json:"service"`
    StartTime   time.Time `json:"start_time"`
    DurationMs  int64     `json:"duration_ms"`
    StatusCode  int       `json:"status_code"`
    Attributes  map[string]string `json:"attributes"`
}
```

## Span Attributes

Each span records contextual attributes:

| Attribute | Description |
|-----------|-------------|
| `http.method` | GET, POST, etc. |
| `http.url` | Request URL path |
| `http.status_code` | Response status code |
| `user.id` | Authenticated user ID |
| `tenant.id` | Tenant ID |
| `request.id` | GGID request UUID |
| `service.name` | Backend service name |
| `error` | Error message (if any) |

## Sampling

Not every request is traced — sampling reduces overhead:

| Strategy | Rate | Use Case |
|----------|------|----------|
| **Always** | 100% | Development/debugging |
| **Probabilistic** | 1-10% | Production default |
| **Error-biased** | 100% errors + 1% success | Recommended production |

## Request Flow

```
Client → Gateway (span: gateway.request)
         ↓
         Gateway → Auth Service (span: auth.verify)
         ↓
         Gateway → Backend Service (span: service.handle)
         ↓
         Backend → PostgreSQL (span: db.query)
         ↓
         Gateway → Response (span: gateway.response)
```

All spans share the same `trace_id` but have unique `span_id`s with parent-child links.

## Trace Exporter

The `TraceExporter` buffers spans and exports them:

- **Buffer**: In-memory ring buffer (configurable size).
- **Export**: Batch export to configured backend (Jaeger, Zipkin, Datadog, stdout).
- **Export interval**: Flushes every 5 seconds or when buffer is full.

## Trace in Audit Events

Every audit event includes a `trace_id` field, enabling correlation between audit logs and distributed traces:

```json
{
  "timestamp": "2026-07-18T03:15:00Z",
  "level": "trace",
  "service": "gateway",
  "trace_id": "a1b2c3d4e5f6...",
  "span_id": "a1b2c3d4e5f6",
  "name": "GET /api/v1/users",
  "duration_ms": 45,
  "status": 200
}
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Missing traces | Sampling rate too low or exporter down | Increase sampling rate; check exporter backend |
| Broken trace chain | Service not propagating traceparent | Verify all services forward the header |
| High overhead | 100% sampling in production | Switch to error-biased 1% sampling |
| Missing spans | Service not instrumented | Add tracing middleware to the service |

## Best Practices

- **Propagate traceparent**: All services must forward the W3C header.
- **Use error-biased sampling**: Capture 100% of errors without production overhead.
- **Correlate with audit**: Use trace_id to jump between audit logs and traces.
- **Tag with tenant**: Always include tenant.id for multi-tenant debugging.
- **Monitor p99 latency**: Use traces to identify slow service dependencies.
