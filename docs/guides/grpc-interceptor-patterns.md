# gRPC Interceptor Patterns

Auth, logging, tracing, recovery, rate limiting, and tenant propagation interceptors for server and client.

## Interceptor Ordering

Server-side interceptors execute in chain order (outermost first):

```
Request → Recovery → RateLimit → Tenant → Auth → Logging → Tracing → Handler
```

```go
srv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        RecoveryInterceptor,    // 1. Catch panics
        RateLimitInterceptor,   // 2. Reject early if rate limited
        TenantInterceptor,      // 3. Extract tenant context
        AuthInterceptor,        // 4. Verify JWT
        LoggingInterceptor,     // 5. Log request
        TracingInterceptor,     // 6. Create span
    ),
)
```

## Server Interceptors

### Recovery

```go
func RecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
    defer func() {
        if r := recover(); r != nil {
            log.Error("panic recovered", "panic", r, "method", info.FullMethod)
            err = status.Errorf(codes.Internal, "internal error")
            metric.Inc("grpc.panic", "method", info.FullMethod)
        }
    }()
    return handler(ctx, req)
}
```

### Auth

```go
func AuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    // Skip auth for health checks
    if isHealthCheck(info.FullMethod) {
        return handler(ctx, req)
    }
    
    // Extract token from metadata
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "missing metadata")
    }
    
    tokens := md.Get("authorization")
    if len(tokens) == 0 {
        return nil, status.Error(codes.Unauthenticated, "missing token")
    }
    
    claims, err := verifyJWT(strings.TrimPrefix(tokens[0], "Bearer "))
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, "invalid token")
    }
    
    ctx = context.WithValue(ctx, "claims", claims)
    return handler(ctx, req)
}
```

### Tenant Propagation

```go
func TenantInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    md, _ := metadata.FromIncomingContext(ctx)
    
    tenantIDs := md.Get("x-tenant-id")
    if len(tenantIDs) == 0 {
        return nil, status.Error(codes.InvalidArgument, "missing tenant")
    }
    
    tenantID := tenantIDs[0]
    
    // Verify tenant exists and is active
    tenant := tenantStore.Get(tenantID)
    if tenant == nil || tenant.Status != "active" {
        return nil, status.Error(codes.PermissionDenied, "tenant inactive")
    }
    
    ctx = context.WithValue(ctx, "tenant_id", tenantID)
    return handler(ctx, req)
}
```

### Rate Limiting

```go
func RateLimitInterceptor(limiter *rate.Limiter) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        if !limiter.Allow() {
            return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
        }
        return handler(ctx, req)
    }
}
```

### Logging

```go
func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    start := time.Now()
    
    resp, err := handler(ctx, req)
    
    duration := time.Since(start)
    log.Info("grpc.request",
        "method", info.FullMethod,
        "duration_ms", duration.Milliseconds(),
        "error", err != nil,
    )
    
    if duration > 500*time.Millisecond {
        log.Warn("slow gRPC call", "method", info.FullMethod, "duration", duration)
    }
    
    return resp, err
}
```

## Client Interceptors

### Trace Propagation

```go
func TracingClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
    ctx, span := otel.Tracer("ggid").Start(ctx, method)
    defer span.End()
    
    // Inject trace context into gRPC metadata
    md, _ := metadata.FromOutgoingContext(ctx)
    if md == nil { md = metadata.New(nil) }
    propagation.TraceContext{}.Inject(ctx, &metadataWriter{md: md})
    ctx = metadata.NewOutgoingContext(ctx, md)
    
    err := invoker(ctx, method, req, reply, cc, opts...)
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }
    
    return err
}
```

### Auth Token Injection

```go
func AuthClientInterceptor(token string) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        // Add auth token to metadata
        ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
        return invoker(ctx, method, req, reply, cc, opts)
    }
}
```

### Retry

```go
func RetryInterceptor(maxRetries int) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        var err error
        for i := 0; i <= maxRetries; i++ {
            err = invoker(ctx, method, req, reply, cc, opts)
            if err == nil { return nil }
            
            st, _ := status.FromError(err)
            if st.Code() != codes.Unavailable && st.Code() != codes.DeadlineExceeded {
                return err // Don't retry client errors
            }
            
            time.Sleep(backoff(i))
        }
        return err
    }
}
```

## Testing

### Unit Test

```go
func TestAuthInterceptor(t *testing.T) {
    tests := []struct {
        name    string
        token   string
        wantErr bool
    }{
        {"valid token", validJWT, false},
        {"missing token", "", true},
        {"invalid token", "bad", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := metadata.AppendToIncomingContext(context.Background(), "authorization", "Bearer "+tt.token)
            _, err := AuthInterceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/auth.Login"}, noopHandler)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
        })
    }
}
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Panic recoveries | Any → investigate |
| Auth failures | >1% → cert or token issue |
| Rate limit hits | >5% → scale needed |
| Slow calls (>500ms) | >5% → optimize |

## See Also

- [Gateway Architecture](gateway-architecture.md)
- [Distributed Tracing Setup](distributed-tracing-setup.md)
- [Service Mesh Integration](service-mesh-integration.md)
- [Health Check Design](health-check-design.md)
