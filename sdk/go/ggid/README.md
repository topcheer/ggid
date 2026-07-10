# GGID Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/ggid/ggid/sdk/go.svg)](https://pkg.go.dev/github.com/ggid/ggid/sdk/go/ggid)

Go SDK for GGID IAM Platform — JWT verification, user management, RBAC, and HTTP middleware.

## Installation

```bash
go get github.com/ggid/ggid/sdk/go/ggid
```

## Quick Start

### Protect an HTTP server

```go
package main

import (
    "net/http"
    "github.com/ggid/ggid/sdk/go/ggid"
)

func main() {
    client := ggid.NewClient("https://iam.example.com",
        ggid.WithTenantID("00000000-0000-0000-0000-000000000001"),
        ggid.WithJWKS("https://iam.example.com/.well-known/jwks.json"),
    )

    mux := http.NewServeMux()
    mux.HandleFunc("/api/profile", func(w http.ResponseWriter, r *http.Request) {
        claims := ggid.ClaimsFromContext(r.Context())
        json.NewEncoder(w).Encode(claims)
    })

    // Wrap with GGID authentication
    protected := client.Middleware(mux)
    http.ListenAndServe(":8080", protected)
}
```

### API Client

```go
client := ggid.NewClient("https://iam.example.com")

// Login
tokens, _ := client.Login(ctx, "admin", "Admin@123456")

// List users
users, _ := client.ListUsers(ctx, tokens.AccessToken)

// Check permission
result, _ := client.CheckPermission(ctx, tokens.AccessToken, "documents", "read")
if result.Allowed {
    // Access granted
}
```

### Permission Middleware

```go
// Require specific permission for a route
mux.Handle("/api/admin", client.RequirePermission("admin", "access")(adminHandler))
```

## License

Apache 2.0
