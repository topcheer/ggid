# GGID Go SDK

A production-ready Go client SDK for the [GGID](https://github.com/ggid/ggid) IAM platform.

## Installation

```bash
go get github.com/ggid/ggid/sdk/go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    ggid "github.com/ggid/ggid/sdk/go"
)

func main() {
    // Create a client pointing to your GGID gateway.
    client := ggid.New("https://iam.example.com",
        ggid.WithAPIKey("your-api-key"),
    )

    ctx := context.Background()

    // Login a user.
    tokens, err := client.Login(ctx, &ggid.LoginRequest{
        Username: "alice",
        Password: "SecurePass@123",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Access token: %s\n", tokens.AccessToken)

    // Verify a JWT (offline mode, no signature verification).
    userInfo, err := client.VerifyToken(ctx, tokens.AccessToken)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("User: %s (%s)\n", userInfo.Username, userInfo.Email)

    // Create a user (requires API key).
    user, err := client.CreateUser(ctx, &ggid.CreateUserRequest{
        Username: "bob",
        Email:    "bob@example.com",
        Password: "SecurePass@123",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created user ID: %s\n", user.ID)
}
```

## Authentication

### JWT Verification (Offline)

By default, `VerifyToken` parses the JWT claims without signature verification.
This is suitable for trusted environments where the gateway already verifies signatures.

```go
client := ggid.New("https://iam.example.com")
info, err := client.VerifyToken(ctx, accessToken)
// info.UserID, info.Email, info.Roles, info.Scopes
```

### JWT Verification (JWKS)

For production, enable JWKS-based signature verification:

```go
client := ggid.New("https://iam.example.com",
    ggid.WithJWKS(15*time.Minute), // cache keys for 15 min
)
info, err := client.VerifyToken(ctx, accessToken)
// Token signature is verified against the server's JWKS.
```

### Token Refresh

```go
tokens, err := client.RefreshToken(ctx, oldRefreshToken)
// tokens.AccessToken + tokens.RefreshToken (rotated)
```

## User Management

```go
// Create
user, err := client.CreateUser(ctx, &ggid.CreateUserRequest{
    Username: "alice",
    Email:    "alice@example.com",
    Password: "SecurePass@123",
})

// Get
user, err := client.GetUser(ctx, "user-id")

// Update
newEmail := "new@example.com"
user, err := client.UpdateUser(ctx, "user-id", &ggid.UpdateUserRequest{
    Email: &newEmail,
})

// Delete
err := client.DeleteUser(ctx, "user-id")

// List with pagination
result, err := client.ListUsers(ctx, &ggid.ListOptions{
    Page:     1,
    PageSize: 20,
    Search:   "alice",
})

// Assign / remove roles
err := client.AssignRole(ctx, "user-id", "role-id")
err := client.RemoveRole(ctx, "user-id", "role-id")
```

## Role Management

```go
role, err := client.CreateRole(ctx, &ggid.CreateRoleRequest{
    Key:  "admin",
    Name: "Administrator",
})

roles, err := client.ListRoles(ctx, nil)
```

## Organization Management

```go
org, err := client.CreateOrg(ctx, &ggid.CreateOrgRequest{
    Name: "Engineering",
})

orgs, err := client.ListOrgs(ctx, nil)
```

## Permission Check

```go
allowed, err := client.CheckPermission(ctx, "user-id", "documents", "read")
if allowed {
    // grant access
}
```

## HTTP Middleware

The SDK provides middleware for protecting your own Go HTTP services.

### JWT Authentication Middleware

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/data", yourHandler)

client := ggid.New("https://iam.example.com")

// Wrap with JWT verification.
protected := client.Middleware(mux, ggid.MiddlewareConfig{
    PublicPaths: []string{"/healthz", "/public"},
    TenantID:    "your-tenant-id",
})

http.ListenAndServe(":8081", protected)
```

### Role-Based Access Control

```go
// Require specific role (checked locally from JWT claims).
adminOnly := client.RequireRole("admin", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello admin!"))
})

// Require specific OAuth scope.
writeOnly := client.RequireScope("write:docs", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Writing..."))
})
```

### Permission-Based Access Control

```go
// Checks against the GGID policy engine (API call).
canRead := client.RequirePermission("documents", "read", func(w http.ResponseWriter, r *http.Request) {
    // User has permission to read documents.
})
```

### Accessing User Info in Handlers

```go
func yourHandler(w http.ResponseWriter, r *http.Request) {
    user := ggid.UserFromContext(r.Context())
    if user == nil {
        http.Error(w, "unauthorized", 401)
        return
    }
    fmt.Fprintf(w, "Hello, %s!", user.Username)
}
```

## Error Handling

The SDK returns structured `*APIError` for all non-2xx API responses.

```go
_, err := client.GetUser(ctx, "nonexistent")
if err != nil {
    var apiErr *ggid.APIError
    if errors.As(err, &apiErr) {
        switch {
        case apiErr.IsNotFound():
            // 404 - user doesn't exist
        case apiErr.IsUnauthorized():
            // 401 - bad token
        case apiErr.IsForbidden():
            // 403 - insufficient permissions
        case apiErr.IsConflict():
            // 409 - duplicate
        case apiErr.IsRateLimited():
            // 429 - too many requests
        }
    }
}
```

## Configuration Options

| Option | Description |
|--------|-------------|
| `WithAPIKey(key)` | Set the API key for management operations |
| `WithHTTPClient(hc)` | Use a custom `*http.Client` (timeouts, transports) |
| `WithJWKS(ttl)` | Enable JWT signature verification via JWKS (cached for TTL) |

## API Reference

| Method | Description |
|--------|-------------|
| `Login(ctx, req)` | Authenticate with username/password |
| `Logout(ctx, token)` | Invalidate an access token |
| `RefreshToken(ctx, rt)` | Refresh an access token |
| `VerifyToken(ctx, token)` | Verify JWT and extract user info |
| `CreateUser(ctx, req)` | Create a new user |
| `GetUser(ctx, id)` | Get user by ID |
| `UpdateUser(ctx, id, req)` | Update user fields |
| `DeleteUser(ctx, id)` | Delete a user |
| `ListUsers(ctx, opts)` | List users with pagination |
| `AssignRole(ctx, uid, rid)` | Assign a role to a user |
| `RemoveRole(ctx, uid, rid)` | Remove a role from a user |
| `CreateRole(ctx, req)` | Create a role |
| `ListRoles(ctx, opts)` | List roles with pagination |
| `CreateOrg(ctx, req)` | Create an organization |
| `ListOrgs(ctx, opts)` | List organizations |
| `CheckPermission(ctx, uid, res, act)` | Check if user has permission |
| `Middleware(handler, cfg)` | JWT auth middleware for HTTP services |
| `RequireRole(role, handler)` | Role-check middleware |
| `RequireScope(scope, handler)` | Scope-check middleware |
| `RequirePermission(res, act, handler)` | Permission-check middleware |

## License

Apache 2.0
