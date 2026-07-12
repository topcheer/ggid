# Distributed Tracing Setup

OpenTelemetry integration, trace propagation across gRPC/HTTP, span per service, sampling config, Jaeger/Tempo backend, and audit log correlation.

## Overview

Distributed tracing follows requests across all GGID microservices, enabling latency analysis, error diagnosis, and request flow visualization.

## Architecture

```
Gateway → Auth → Identity → PostgreSQL
   │         │        │
   │         │        └── Span: DB query (2ms)
   │         └── Span: verify credentials (5ms)
   └── Span: route + JWT verify (1ms)
       
All spans share trace_id → visible as one waterfall in Jaeger
```

## OpenTelemetry Integration

### Service Instrumentation

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/propagation"
)

func InitTracer(serviceName string) (*trace.TracerProvider, error) {
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint("otel-collector:4317"),
        otlptracegrpc.WithTLSConfig(tlsConfig),
    )
    if err != nil { return nil, err }
    
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.ServiceNameKey.String(serviceName),
            semconv.ServiceVersionKey.String(version),
        )),
        trace.WithSampler(trace.TraceIDRatioBased(0.1)), // 10% sampling
    )
    
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.TraceContext{})
    
    return tp, nil
}
```

## Trace Propagation

### HTTP (Gateway → Service)

```go
// Gateway injects trace context into outgoing HTTP request
func injectTrace(ctx context.Context, req *http.Request) {
    propagation.TraceContext{}.Inject(ctx,
        propagation.HeaderCarrier(req.Header))
    // Headers: traceparent, tracestate
}

// Service extracts trace context from incoming request
func extractTrace(ctx context.Context, r *http.Request) context.Context {
    ctx = propagation.TraceContext{}.Extract(ctx,
        propagation.HeaderCarrier(r.Header))
    return ctx
}
```

### gRPC (Service → Service)

```go
// gRPC automatically propagates trace context via metadata
// Using otelgrpc interceptor:

srv := grpc.NewServer(
    grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
    grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
)

// Client side
conn, _ := grpc.Dial("identity-svc:9080",
    grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
)
```

### traceparent Header

```http
traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
```

| Part | Meaning |
|------|---------|
| `00` | Version |
| `0af7651916cd43dd8448eb211c80319c` | Trace ID (shared across all spans) |
| `b7ad6b7169203331` | Span ID (unique per service) |
| `01` | Trace flags (01 = sampled) |

## Span per Service

### Automatic Spans

```go
// HTTP handler span (automatic via otelhttp)
handler := otelhttp.NewHandler(myHandler, "users.list")

// gRPC method span (automatic via otelgrpc)
// Each RPC call creates a span

// Database span (automatic via otelsql)
db := otelsql.Open("pgx", connString)
```

### Manual Spans

```go
func (s *UserService) Create(ctx context.Context, req *CreateRequest) (*User, error) {
    ctx, span := otel.Tracer("identity").Start(ctx, "UserService.Create")
    defer span.End()
    
    // Add attributes
    span.SetAttributes(
        attribute.String("user.email", req.Email),
        attribute.String("user.department", req.Department),
    )
    
    // Validate (child span)
    ctx, validateSpan := otel.Tracer("identity").Start(ctx, "validate")
    err := validate(req)
    validateSpan.End()
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    
    // DB insert (child span via otelsql)
    user, err := s.repo.Create(ctx, req)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }
    
    return user, nil
}
```

## Sampling Configuration

| Strategy | Rate | Use Case |
|----------|------|----------|
| Always | 100% | Development/debugging |
| Ratio 10% | 0.1 | Production default |
| Ratio 1% | 0.01 | High-volume production |
| Tail-based | Dynamic | Capture all errors + slow requests |

### Tail-Based Sampling

```yaml
tail_sampling:
  policies:
    - name: "errors"
      type: status_code
      status_code: {status_codes: [ERROR]}
      
    - name: "slow_requests"
      type: latency
      latency: {threshold_ms: 1000}
      
    - name: "sample_rest"
      type: probabilistic
      probabilistic: {sampling_percentage: 5}
```

Captures 100% of errors + slow requests, 5% of normal requests.

## Jaeger/Tempo Backend

### Jaeger

```yaml
# docker-compose
jaeger:
  image: jaegertracing/all-in-one:latest
  ports:
    - "16686:16686"  # UI
    - "4317:4317"    # OTLP gRPC
  environment:
    COLLECTOR_OTLP_ENABLED: "true"
    SPAN_STORAGE_TYPE: "elasticsearch"
```

### Tempo

```yaml
tempo:
  image: grafana/tempo:latest
  ports:
    - "4317:4317"
  config:
    storage: {trace: {backend: s3}}
```

### Querying Traces

```bash
# By trace ID
GET http://jaeger:16686/api/traces/{trace_id}

# By service + operation
GET http://jaeger:16686/api/traces?service=identity&operation=UserService.Create
```

## Audit Log Correlation

```go
// Inject trace_id into audit events
func auditWithTrace(ctx context.Context, action string, data map[string]interface{}) {
    span := trace.SpanFromContext(ctx)
    traceID := span.SpanContext().TraceID().String()
    
    data["trace_id"] = traceID
    audit.Log(action, data)
}
```

### Cross-Referencing

```bash
# Find audit events for a specific trace
GET /api/v1/audit/events?trace_id=0af7651916cd43dd8448eb211c80319c

# Or find the trace from an audit event
GET /api/v1/audit/events/evt-uuid
# → {trace_id: "0af7651916cd43dd8448eb211c80319c"}
# → Open Jaeger with that trace_id
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Trace export latency | >1s → batch too large |
| Sampling rate accuracy | ±2% of configured |
| Span export errors | >1% → collector down |
| Trace completeness | Missing spans → instrumentation gap |

## See Also

- [Monitoring and Alerting](monitoring-and-alerting.md)
- [Gateway Architecture](gateway-architecture.md)
- [Audit Log Architecture](audit-log-architecture.md)
- [Observability Guide](observability-guide.md)
