# GGID Go SDK

Integrate [GGID IAM](https://github.com/ggid/ggid) into your Go backend.
Provides JWT verification, user/role/org management, and HTTP middleware.

## Installation

```bash
go get github.com/ggid/ggid/sdk/go
```

## Quick Start

```go
import ggid "github.com/ggid/ggid/sdk/go"

// Create a client
client := ggid.New("https://iam.example.com",
    ggid.WithAPIKey("your-server-api-key"),
    ggid.WithJWKS(15*time.Minute), // enable JWT signature verification
)
```

## Authentication

### Login

```go
tokens, err := client.Login(ctx, &ggid.LoginRequest{
    Username: "admin",
    Password: "Admin@123456",
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Access token: %s\n", tokens.AccessToken)
```

### Verify a JWT

```go
// With WithJWKS enabled, verifies the RS256 signature against the JWKS endpoint.
// Without WithJWKS, parses claims without signature verification (offline mode).
userInfo, err := client.VerifyToken(ctx, accessToken)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("User: %s (%s)\n", userInfo.Username, userInfo.Email)
fmt.Printf("Roles: %v\n", userInfo.Roles)
fmt.Printf("Tenant: %s\n", userInfo.TenantID)
```

### Refresh Token

```go
newTokens, err := client.RefreshToken(ctx, tokens.RefreshToken)
```

### Logout

```go
err := client.Logout(ctx, tokens.AccessToken)
```

## HTTP Middleware

Protect your HTTP routes with JWT verification:

```go
mux := http.NewServeMux()

client := ggid.New("https://iam.example.com", ggid.WithJWKS(15*time.Minute))

// Wrap the entire mux with JWT verification
protectedHandler := client.Middleware(mux, ggid.MiddlewareConfig{
    PublicPaths: []string{"/healthz", "/public"},
    TenantID:    "00000000-0000-0000-0000-000000000001",
})

mux.HandleFunc("/api/profile", func(w http.ResponseWriter, r *http.Request) {
    user := ggid.UserFromContext(r.Context())
    json.NewEncoder(w).Encode(user)
})
```

### Permission-Based Authorization

```go
// Require a specific permission (calls the policy engine)
mux.HandleFunc("/api/admin",
    client.RequirePermission("admin:panel", "access",
        func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("Welcome, admin!"))
        },
    ),
)

// Require a specific role (local check from JWT claims — no API call)
mux.HandleFunc("/api/reports",
    client.RequireRole("analyst",
        func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("Reports dashboard"))
        },
    ),
)

// Require an OAuth scope (local check from JWT claims)
mux.HandleFunc("/api/data",
    client.RequireScope("read:data",
        func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("Data endpoint"))
        },
    ),
)
```

## User Management

```go
// Create a user
user, err := client.CreateUser(ctx, &ggid.CreateUserRequest{
    Username: "john.doe",
    Email:    "john@example.com",
    Password: "SecurePass@123",
    Phone:    "+1234567890",
})

// Get user by ID
user, err := client.GetUser(ctx, user.ID)

// List users with pagination
result, err := client.ListUsers(ctx, &ggid.ListOptions{
    PageSize: 50,
    Search:   "john",
})

// Update user
updated, err := client.UpdateUser(ctx, user.ID, &ggid.UpdateUserRequest{
    Email: ggid.StringPtr("new.email@example.com"),
})

// Delete user
err = client.DeleteUser(ctx, user.ID)

// Assign role
err = client.AssignRole(ctx, user.ID, roleID)
```

## Role & Permission Management

```go
// Create a role
role, err := client.CreateRole(ctx, &ggid.CreateRoleRequest{
    Key:         "editor",
    Name:        "Content Editor",
    Description: "Can edit and publish content",
})

// List roles
roles, err := client.ListRoles(ctx, &ggid.ListOptions{PageSize: 50})

// Check permission
allowed, err := client.CheckPermission(ctx, user.ID, "documents:sensitive", "read")
if allowed {
    fmt.Println("Access granted")
}
```

## Organization Management

```go
// Create an organization
org, err := client.CreateOrg(ctx, &ggid.CreateOrgRequest{
    Name:        "Engineering",
    Description: "Engineering department",
})

// List organizations
orgs, err := client.ListOrgs(ctx, &ggid.ListOptions{PageSize: 50})
```

## Error Handling

The SDK returns structured `*APIError` for non-2xx API responses:

```go
user, err := client.GetUser(ctx, "nonexistent-id")
if err != nil {
    var apiErr *ggid.APIError
    if errors.As(err, &apiErr) {
        switch {
        case apiErr.IsNotFound():
            fmt.Println("User not found")
        case apiErr.IsUnauthorized():
            fmt.Println("Token expired or invalid")
        case apiErr.IsConflict():
            fmt.Println("Resource already exists")
        case apiErr.IsRateLimited():
            fmt.Println("Too many requests, slow down")
        }
    }
}
```

## Configuration Options

| Option | Description |
|--------|-------------|
| `WithAPIKey(key)` | Sets the server-side API key for management operations |
| `WithHTTPClient(hc)` | Use a custom `*http.Client` (e.g., for retries, timeouts) |
| `WithJWKS(ttl)` | Enable JWT signature verification with JWKS caching |

## Types Reference

### `UserInfo`
Extracted from JWT claims after token verification.

| Field | Type | Description |
|-------|------|-------------|
| `UserID` | `string` | User UUID (`sub` claim) |
| `TenantID` | `string` | Tenant UUID |
| `Username` | `string` | Username |
| `Email` | `string` | Email address |
| `Roles` | `[]string` | Assigned role keys |
| `Scopes` | `[]string` | OAuth scopes |
| `Claims` | `map[string]any` | Raw JWT claims |

### `TokenSet`

| Field | Type |
|-------|------|
| `AccessToken` | `string` |
| `RefreshToken` | `string` |
| `ExpiresIn` | `int` |
| `TokenType` | `string` |

## License

Apache 2.0
