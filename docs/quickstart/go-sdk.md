# Go SDK Quickstart

> Add GGID authentication to any Go app in 3 lines.

---

## Install

```bash
go get github.com/ggid/ggid/sdk/go@latest
```

## Verify a JWT (3 lines)

```go
import ggid "github.com/ggid/ggid/sdk/go"

verifier := ggid.NewVerifier("http://localhost:8080", "your-jwt-secret")
claims, err := verifier.Verify(ctx, tokenString)
// claims.TenantID, claims.UserID, claims.Scope
```
## Protect an HTTP Handler

```go
middleware := ggid.NewMiddleware(verifier)
http.Handle("/api/protected", middleware.Protect(myHandler))
```

## Full Example

```go
package main

import (
    "fmt"
    "net/http"

    ggid "github.com/ggid/ggid/sdk/go"
)

func main() {
    verifier := ggid.NewVerifier("http://localhost:8080", "secret")
    mw := ggid.NewMiddleware(verifier)

    http.Handle("/api/me", mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := ggid.ClaimsFromContext(r.Context())
        fmt.Fprintf(w, "Hello, %s (tenant: %s)", claims.UserID, claims.TenantID)
    })))

    http.ListenAndServe(":8081", nil)
}
```

## Gin Integration

```go
import ggidmw "github.com/ggid/ggid/sdk/go/middleware/gin"

r := gin.Default()
r.Use(ggidmw.Auth(verifier))
r.GET("/api/me", func(c *gin.Context) {
    claims := ggidmw.Claims(c)
    c.JSON(200, gin.H{"user": claims.UserID})
})
```

---

*See: [Gin Integration](../integration-guides/gin.md) | [SDK Reference](../sdk-reference.md)*