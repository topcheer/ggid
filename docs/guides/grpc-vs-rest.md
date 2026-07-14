# gRPC vs REST Design Guide

When to use gRPC, when to use REST, and how GGID bridges both.

## Decision Matrix

| Factor | gRPC | REST |
|--------|------|------|
| Internal service-to-service | ✅ Preferred | ❌ Overhead |
| External/public API | ❌ Not browser-native | ✅ Preferred |
| Streaming (real-time) | ✅ Bi/uni streaming | ❌ Polling/SSE |
| Browser clients | ❌ Requires grpc-web | ✅ Native |
| Schema enforcement | ✅ Proto strict types | ⚠️ OpenAPI optional |
| Code generation | ✅ Multi-language | ⚠️ Optional |
| Human readability | ❌ Binary protobuf | ✅ JSON |
| Payload size | ✅ Compact (3-10x smaller) | ❌ Verbose JSON |
| Latency | ✅ HTTP/2 multiplexing | ⚠️ HTTP/1.1 connection overhead |

## GGID Architecture

```
Client (Browser/Mobile)
    │  REST/JSON over HTTPS
    ▼
API Gateway (grpc-gateway)
    │  gRPC over mTLS
    ▼
┌───────┬───────┬───────┬───────┐
│Auth   │Ident  │Policy │ Org   │ ...
└───────┴───────┴───────┴───────┘
    │  gRPC + NATS events
    ▼
  Audit Service
```

**External**: REST only (via grpc-gateway translation)
**Internal**: gRPC with mTLS between all services

## Proto Design Guidelines

### Naming Conventions

```protobuf
// Package: reverse-DNS
package ggid.auth.v1;

// Service: noun, not verb
service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
}
```

### Field Conventions

```protobuf
message GetUserRequest {
  string id = 1;              // UUID format
  string tenant_id = 2;       // Always present for multi-tenancy
  google.protobuf.FieldMask field_mask = 3;  // Partial responses
}

message ListUsersRequest {
  int32 page_size = 1;        // Max 100
  string page_token = 2;      // Opaque cursor
  string filter = 3;          // CEL expression
  string order_by = 4;        // "name ASC, created_at DESC"
}
```

### Streaming

```protobuf
service AuditService {
  // Server streaming: tail audit events
  rpc StreamEvents(StreamEventsRequest) returns (stream AuditEvent);

  // Bidirectional: interactive session inspector
  rpc SessionInspector(stream SessionCommand) returns (stream SessionState);
}
```

## Gateway Translation (grpc-gateway)

GGID uses grpc-gateway annotations to auto-generate REST endpoints:

```protobuf
import "google/api/annotations.proto";

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = {
      get: "/api/v1/users/{id}"
    };
  }

  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse) {
    option (google.api.http) = {
      post: "/api/v1/users"
      body: "*"
    };
  }
}
```

This generates both gRPC and REST endpoints from a single proto definition.

## Versioning

| Protocol | Strategy |
|----------|----------|
| gRPC | Package suffix (`v1`, `v2`) — `package ggid.auth.v2` |
| REST | URL path prefix (`/api/v1/`, `/api/v2/`) |
| Breaking changes | New version, old deprecated 6 months |

```protobuf
// Backward compatible: add field with new number
message GetUserResponse {
  string id = 1;
  string email = 2;
  string phone = 3;  // New field — old clients ignore
}

// Breaking: change type or remove field → bump version
```

## Performance Comparison

Benchmark (1KB payload, 1000 req/s, same machine):

| Metric | gRPC | REST (JSON) |
|--------|------|-------------|
| Avg latency | 1.2 ms | 3.8 ms |
| P99 latency | 4.1 ms | 12.5 ms |
| Throughput | 85,000 rps | 28,000 rps |
| Payload size | 48 bytes | 312 bytes |
| CPU usage | 12% | 34% |

**gRPC is ~3x faster** for internal calls due to HTTP/2 multiplexing and binary encoding.

## Error Handling

### gRPC Status Codes

| Code | REST Equivalent |
|------|-----------------|
| OK (0) | 200 |
| InvalidArgument (3) | 400 |
| Unauthenticated (16) | 401 |
| PermissionDenied (7) | 403 |
| NotFound (5) | 404 |
| AlreadyExists (6) | 409 |
| ResourceExhausted (8) | 429 |
| Internal (13) | 500 |
| Unavailable (14) | 503 |

### Rich Error Details

```protobuf
import "google/rpc/error_details.proto";

// Internal gRPC error with details
err := status.Errorf(codes.InvalidArgument, "validation failed")
detail := &errdetails.BadRequest{
  FieldViolations: []*errdetails.BadRequest_FieldViolation{
    {Field: "email", Description: "must be valid email format"},
  },
}
st, _ := status.New(codes.InvalidArgument, "validation failed").
    WithDetails(detail).ToProto()
```

## Best Practices

1. **Proto-first**: Design proto → generate REST + gRPC + SDKs
2. **Never expose raw gRPC externally**: Always through gateway
3. **Use mTLS internally**: See gRPC TLS
4. **Deadline propagation**: Set deadlines on every call
5. **Interceptors**: Auth, logging, tenant-injection at gRPC layer
6. **Connection pooling**: Keep gRPC channels warm

## See Also

- Architecture Overview
- [Authentication Flows](authentication-flows.md)
- [REST API Reference](../api/rest-api.md)
