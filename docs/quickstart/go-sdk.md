# Go SDK Quickstart

> Add GGID authentication to any Go app in 3 lines.

---

## Install

```bash
go get github.com/ggid/ggid/sdk/go@latest
```

## Verify a JWT (3 lines)

```go
import (
    "context"

    ggid "github.com/ggid/ggid/sdk/go"
)

client := ggid.New("http://localhost:8080", ggid.WithJWKS(15*time.Minute))
userInfo, err := client.VerifyToken(ctx, accessToken)
// userInfo.UserID, userInfo.TenantID, userInfo.Roles, userInfo.Scopes
```

## Protect an HTTP Handler

```go
client := ggid.New("http://localhost:8080", ggid.WithJWKS(15*time.Minute))

mux := http.NewServeMux()
mux.HandleFunc("/api/profile", func(w http.ResponseWriter, r *http.Request) {
    user := ggid.UserFromContext(r.Context())
    fmt.Fprintf(w, "Hello, %s (tenant: %s)", user.UserID, user.TenantID)
})

// Wrap with JWT verification
handler := client.Middleware(mux, ggid.MiddlewareConfig{
    PublicPaths: []string{"/healthz"},
})
http.ListenAndServe(":8081", handler)
```

## Full Example

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    ggid "github.com/ggid/ggid/sdk/go"
)

func main() {
    client := ggid.New("http://localhost:8080",
        ggid.WithJWKS(15*time.Minute),
    )

    mux := http.NewServeMux()

    // Protected route — require read:users permission
    mux.HandleFunc("/api/me", client.RequirePermission("users", "read",
        func(w http.ResponseWriter, r *http.Request) {
            user := ggid.UserFromContext(r.Context())
            json.NewEncoder(w).Encode(user)
        },
    ))

    // Protected route — require admin role
    mux.HandleFunc("/api/admin", client.RequireRole("admin",
        func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("Admin panel"))
        },
    ))

    // Wrap all routes with JWT verification
    handler := client.Middleware(mux, ggid.MiddlewareConfig{
        PublicPaths: []string{"/healthz"},
    })

    fmt.Println("Server on :8081")
    http.ListenAndServe(":8081", handler)
}
```

## Gin Integration

```go
import ggidmw "github.com/ggid/ggid/sdk/go/middleware"

r := gin.Default()
r.Use(ggidmw.Auth("http://localhost:8080", ggidmw.Options{}))
r.GET("/api/me", func(c *gin.Context) {
    info, _ := ggidmw.FromContext(c.Request.Context())
    c.JSON(200, gin.H{"user": info.UserID})
})
```

---

*See: [Gin Integration](../integration-guides/gin.md) | [Go Integration Example](../examples/go-integration.md) | [SDK Reference](../sdk-reference.md)*
