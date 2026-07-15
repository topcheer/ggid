# gRPC Service Implementation Guide

*Research document — 2026-07-15*

## Summary

GGID currently defines gRPC service interfaces in `api/proto/{identity,auth,oauth}/v1/*.proto`, but the generated Go server implementations are **not wired** in the corresponding service binaries. This document provides an architecture guide for implementing identity, auth, and oauth gRPC services, reusing the existing HTTP handlers as much as possible.

## Current State

- **Proto files exist:**
  - `api/proto/identity/v1/identity.proto` — 16 methods
  - `api/proto/auth/v1/auth.proto` — 10 methods
  - `api/proto/oauth/v1/oauth.proto` — 5 methods
- **Generated `pb` packages are missing:** no `api/gen/identity/v1`, `api/gen/auth/v1`, or `api/gen/oauth/v1` directories exist.
- **Server structs only implement `Run`:** e.g., `services/identity/internal/server/server.go` has no `RegisterIdentityServiceServer` call.
- **HTTP handlers are complete:** each service already exposes equivalent REST endpoints in `internal/server/server.go` and `internal/server/http.go`.

## Proto Interface Summary

### IdentityService (`api.proto.identity.v1`)

| RPC | HTTP Equivalent | Purpose |
|-----|-----------------|---------|
| `CreateUser` | `POST /api/v1/users` | Create a user |
| `GetUser` | `GET /api/v1/users/{id}` | Fetch user by ID |
| `ListUsers` | `GET /api/v1/users` | Paginated user list |
| `UpdateUser` | `PUT /api/v1/users/{id}` | Update user profile |
| `DeleteUser` | `DELETE /api/v1/users/{id}` | Delete user |
| `LockUser` | `POST /api/v1/users/{id}/lock` | Lock account |
| `UnlockUser` | `POST /api/v1/users/{id}/unlock` | Unlock account |
| `RegisterUser` | `POST /api/v1/auth/register` | Public registration |
| `VerifyEmail` | `POST /api/v1/users/verify-email` | Email verification |
| `ListUserEmails` | `GET /api/v1/users/{id}/emails` | List emails |
| `AddUserEmail` | `POST /api/v1/users/{id}/emails` | Add email |
| `RemoveUserEmail` | `DELETE /api/v1/users/{id}/emails/{emailId}` | Remove email |
| `SetPrimaryEmail` | `PUT /api/v1/users/{id}/emails/primary` | Set primary email |
| `ListExternalIdentities` | `GET /api/v1/users/{id}/external-identities` | Social/external links |
| `LinkExternalIdentity` | `POST /api/v1/users/{id}/external-identities` | Link external identity |
| `UnlinkExternalIdentity` | `DELETE /api/v1/users/{id}/external-identities/{identityId}` | Unlink external identity |

### AuthService (`api.proto.auth.v1`)

| RPC | HTTP Equivalent | Purpose |
|-----|-----------------|---------|
| `Login` | `POST /api/v1/auth/login` | Password login |
| `Register` | `POST /api/v1/auth/register` | Credential registration |
| `Logout` | `POST /api/v1/auth/logout` | Revoke session/tokens |
| `RefreshToken` | `POST /api/v1/auth/refresh` | Refresh token exchange |
| `ForgotPassword` | `POST /api/v1/auth/forgot-password` | Initiate reset flow |
| `ResetPassword` | `POST /api/v1/auth/reset-password` | Complete reset flow |
| `ChangePassword` | `POST /api/v1/auth/change-password` | Authenticated password change |
| `ListSessions` | `GET /api/v1/auth/sessions` | Active sessions |
| `RevokeSession` | `DELETE /api/v1/auth/sessions/{id}` | Revoke a session |

### OAuthService (`api.proto.oauth.v1`)

| RPC | HTTP Equivalent | Purpose |
|-----|-----------------|---------|
| `CreateClient` | `POST /api/v1/oauth/clients` | Dynamic client registration |
| `GetClient` | `GET /api/v1/oauth/clients/{id}` | Fetch client |
| `ListClients` | `GET /api/v1/oauth/clients` | List clients |
| `UpdateClient` | `PUT /api/v1/oauth/clients/{id}` | Update client |
| `DeleteClient` | `DELETE /api/v1/oauth/clients/{id}` | Delete client |

## Service-to-Handler Mapping Strategy

The recommended approach is to **wrap the existing HTTP handler layer** instead of duplicating business logic. This keeps both REST and gRPC paths consistent and reduces maintenance overhead.

### Option A: HTTP-Handler Adapter (Recommended)

For each RPC, create a thin gRPC handler that:

1. Builds an `http.Request` from the gRPC request message.
2. Injects tenant context (e.g., `X-Tenant-ID` from gRPC metadata).
3. Calls the existing `http.HandlerFunc` from the service's `internal/server` package.
4. Parses the JSON response into the gRPC response message.

```go
func (s *identityGRPCServer) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.User, error) {
    // 1. Build HTTP request from proto
    body := map[string]any{
        "username": req.Username,
        "email":    req.Email,
        // ...
    }
    httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
    httpReq.Header.Set("X-Tenant-ID", tenantFromContext(ctx))

    // 2. Delegate to existing HTTP handler
    rr := httptest.NewRecorder()
    s.handlers.CreateUser(rr, httpReq)

    // 3. Convert response to proto
    return userHTTPToProto(rr), nil
}
```

Pros:
- Minimal code duplication.
- Single source of truth for business logic.
- Fast to implement.

Cons:
- Extra JSON serialization round-trip inside process.
- Error mapping from HTTP status codes to gRPC status codes needs explicit handling.

### Option B: Service-Level Refactoring (Long-Term)

Extract the core logic from HTTP handlers into a pure service interface (e.g., `IdentityService`) and have both HTTP and gRPC handlers call that interface.

```go
type IdentityService interface {
    CreateUser(ctx context.Context, req CreateUserInput) (*User, error)
    GetUser(ctx context.Context, id string) (*User, error)
    // ...
}

// HTTP handler
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    user, err := h.svc.CreateUser(r.Context(), decodeCreateUserInput(r))
    // write JSON
}

// gRPC handler
func (s *identityGRPCServer) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.User, error) {
    user, err := s.svc.CreateUser(ctx, decodeCreateUserInputFromProto(req))
    // return proto
}
```

Pros:
- Clean architecture; no HTTP-in-gRPC adapter overhead.
- Better testability.

Cons:
- Larger refactoring effort.
- Requires moving a lot of handler code into service layer.

## Recommended Hybrid Approach

1. **Phase 1 — Adapter:** Implement gRPC endpoints using Option A to unblock microservice-to-microservice gRPC calls quickly.
2. **Phase 2 — Extract common services:** As handlers are refactored, move shared logic into service interfaces and switch to Option B.

## Implementation Steps

1. **Add protobuf generation to build:**
   ```bash
   buf generate --path api/proto
   # or
   protoc --go_out=. --go-grpc_out=. api/proto/identity/v1/identity.proto
   ```
   Output to `api/gen/{identity,auth,oauth}/v1`.

2. **Create gRPC server structs:**
   - `services/identity/internal/server/grpc.go`
   - `services/auth/internal/server/grpc.go`
   - `services/oauth/internal/server/grpc.go`

3. **Register gRPC servers in `main.go`:**
   ```go
   grpcServer := grpc.NewServer()
   identityv1.RegisterIdentityServiceServer(grpcServer, grpcserver.NewIdentity(svc, handlers))
   ```

4. **Wire TLS for gRPC:** reuse existing `GRPC_TLS_*` environment variables already used by policy/org/audit.

5. **Add gRPC health checks:** implement `grpc.health.v1.Health` service for each.

6. **Update gateway:** allow internal services to call identity/auth/oauth via gRPC when available, with HTTP fallback.

## Error Mapping

Map common HTTP errors to gRPC status codes:

| HTTP | gRPC Status |
|------|-------------|
| 400 Bad Request | `InvalidArgument` |
| 401 Unauthorized | `Unauthenticated` |
| 403 Forbidden | `PermissionDenied` |
| 404 Not Found | `NotFound` |
| 409 Conflict | `AlreadyExists` |
| 500 Internal Server Error | `Internal` |

## Reuse Checklist

- [ ] `CreateUser` reuses existing HTTP/JSON handler or service logic.
- [ ] `Login` reuses auth service logic (password verify, MFA, session create).
- [ ] `CreateClient` reuses OAuth DCR validation and PG repository.
- [ ] Tenant context is injected from gRPC metadata for multi-tenancy.
- [ ] gRPC server TLS shares the same cert rotation path as other services.
- [ ] Generated pb packages are excluded from `go test` coverage ceiling (they are generated code).

## References

- `api/proto/identity/v1/identity.proto`
- `api/proto/auth/v1/auth.proto`
- `api/proto/oauth/v1/oauth.proto`
- `services/identity/internal/server/server.go`
- `services/auth/internal/server/http.go`
- `services/oauth/internal/server/server.go`
- `docs/platform-completeness-report.md` (gaps #30, #31, #32)
