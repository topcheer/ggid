# gRPC Streaming Patterns

Server streaming for audit events, client streaming for batch ops, bidi for real-time, backpressure, error handling, and connection lifecycle.

## Pattern Overview

| Pattern | Direction | Use Case |
|---------|-----------|---------|
| Server streaming | Server → Client | Audit event stream, large result sets |
| Client streaming | Client → Server | Batch operations, bulk import |
| Bidirectional | Both | Real-time chat, live updates |

## Server Streaming: Audit Events

### Proto

```protobuf
service AuditService {
  rpc StreamEvents(StreamRequest) returns (stream AuditEvent);
}

message StreamRequest {
  string tenant_id = 1;
  string filter = 2;  // CEL expression
}

message AuditEvent {
  string event_id = 1;
  string action = 2;
  google.protobuf.Timestamp timestamp = 3;
}
```

### Implementation

```go
func (s *AuditServer) StreamEvents(req *StreamRequest, stream auditpb.AuditService_StreamEventsServer) error {
    sub, err := s.nats.Subscribe("audit."+req.TenantId+".>", func(msg *nats.Msg) {
        event := &auditpb.AuditEvent{}
        proto.Unmarshal(msg.Data, event)
        
        // Send to client stream
        if err := stream.Send(event); err != nil {
            log.Error("stream send failed", err)
            sub.Unsubscribe()
        }
    })
    if err != nil { return err }
    
    // Block until client disconnects or context cancelled
    <-stream.Context().Done()
    sub.Unsubscribe()
    return nil
}
```

### Client

```go
stream, _ := client.StreamEvents(ctx, &auditpb.StreamRequest{
    TenantId: tenantID,
    Filter:   `action == "user.login"`,
})

for {
    event, err := stream.Recv()
    if err == io.EOF { break }
    if err != nil { log.Error("stream error", err); break }
    handleEvent(event)
}
```

## Client Streaming: Batch Operations

### Proto

```protobuf
service IdentityService {
  rpc BatchCreateUsers(stream CreateUserRequest) returns (BatchResponse);
}
```

### Implementation

```go
func (s *IdentityServer) BatchCreateUsers(stream idpb.IdentityService_BatchCreateUsersServer) error {
    results := &idpb.BatchResponse{}
    
    for {
        req, err := stream.Recv()
        if err == io.EOF {
            return stream.SendAndClose(results)
        }
        if err != nil { return err }
        
        user, err := s.createUser(req)
        if err != nil {
            results.Failures = append(results.Failures, &idpb.BatchFailure{
                Email:  req.Email,
                Error:  err.Error(),
            })
        } else {
            results.Created = append(results.Created, user.Id)
        }
    }
}
```

## Bidirectional: Real-Time

### Proto

```protobuf
service PolicyService {
  rpc WatchDecisions(stream DecisionRequest) returns (stream DecisionResponse);
}
```

### Implementation

```go
func (s *PolicyServer) WatchDecisions(stream policypb.PolicyService_WatchDecisionsServer) error {
    // Read requests and send responses concurrently
    errCh := make(chan error, 2)
    
    // Receiver goroutine
    go func() {
        for {
            req, err := stream.Recv()
            if err != nil { errCh <- err; return }
            s.enqueueDecision(req)
        }
    }()
    
    // Sender goroutine
    go func() {
        for {
            resp := s.dequeueDecision()
            if err := stream.Send(resp); err != nil { errCh <- err; return }
        }
    }()
    
    return <-errCh
}
```

## Backpressure

```go
// Server-side flow control
func streamWithBackpressure(stream pb.AuditService_StreamEventsServer, events <-chan *AuditEvent) error {
    for {
        select {
        case <-stream.Context().Done():
            return nil
        case event := <-events:
            // Check if client is keeping up
            if len(events) > 1000 {
                // Buffer too full → slow down source
                time.Sleep(100 * time.Millisecond)
            }
            if err := stream.Send(event); err != nil {
                return err
            }
        }
    }
}
```

## Error Handling

| Error | Action |
|-------|--------|
| `io.EOF` | Stream ended normally |
| `context.Canceled` | Client disconnected |
| `context.DeadlineExceeded` | Timeout |
| Transport error | Reconnect |

## Connection Lifecycle

```go
// Client reconnect on stream error
func watchWithReconnect(client pb.AuditServiceClient) {
    for {
        stream, err := client.StreamEvents(ctx, req)
        if err != nil {
            time.Sleep(5 * time.Second) // Backoff
            continue
        }
        
        consumeStream(stream)
        // Stream ended → loop reconnects
    }
}
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Active streams | Track per service |
| Stream errors | >1% → investigate |
| Stream duration | Very long → possible leak |
| Messages per stream | Track throughput |

## See Also

- [gRPC Interceptor Patterns](grpc-interceptor-patterns.md)
- [Event-Driven Audit](event-driven-audit.md)
- [Audit Query API](audit-query-api.md)
- [Connection Pool Tuning](connection-pool-tuning.md)
