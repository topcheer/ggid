# Gin Framework Integration Guide

> Add GGID authentication to a Go Gin app using the Go SDK.

---

## Install

```bash
go get github.com/ggid/ggid/sdk/go@latest
go get github.com/gin-gonic/gin@latest
```

## Minimal Setup

```go
package main

import (
    "net/http"

    "github.com/gin-gonic/gin"
    ggid "github.com/ggid/ggid/sdk/go"
    ggidmw "github.com/ggid/ggid/sdk/go/middleware/gin"
)

func main() {
    verifier := ggid.NewVerifier(
        "http://localhost:8080",
        "your-jwt-secret",
    )

    r := gin.Default()

    // Public routes (no auth)
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok"})
    })

    // Protected API group
    api := r.Group("/api")
    api.Use(ggidmw.Auth(verifier))
    {
        api.GET("/me", func(c *gin.Context) {
            claims := ggidmw.Claims(c)
            c.JSON(http.StatusOK, gin.H{
                "user_id":   claims.UserID,
                "tenant_id": claims.TenantID,
                "scope":     claims.Scope,
            })
        })
    }

    r.Run(":8081")
}
```

## Scope Check Middleware

```go
func RequireScope(scope string) gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := ggidmw.Claims(c)
        if !claims.HasScope(scope) {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error":    "insufficient_scope",
                "required": scope,
            })
            return
        }
        c.Next()
    }
}

// Usage
api.DELETE("/users/:id", RequireScope("delete:users"), deleteUser)
api.GET("/users", RequireScope("read:users"), listUsers)
```

## Tenant-Aware Handler

```go
func listUsers(c *gin.Context) {
    claims := ggidmw.Claims(c)

    // Use tenant_id for all queries
    rows, err := db.Query(
        `SELECT id, username, email FROM users WHERE tenant_id = $1`,
        claims.TenantID,
    )
    // ...
}
```

## Optional Auth (Some Routes Protected)

```go
// Apply middleware only to specific groups
api := r.Group("/api")
api.Use(ggidmw.Auth(verifier))

// Public routes don't get middleware
r.GET("/health", healthHandler)
r.POST("/api/auth/login", loginHandler)
```

## Using the GGID Client

```go
func getUser(c *gin.Context) {
    claims := ggidmw.Claims(c)

    client := ggid.NewClient("http://localhost:8080", claims.Token)
    user, err := client.Users.Get(c, c.Param("id"))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, user)
}
```

## Environment Variables

```bash
export GGID_URL=http://localhost:8080
export JWT_SECRET=your-shared-secret
export PORT=8081
```

---

*See: [Go SDK Quickstart](../quickstart/go-sdk.md) | [SDK Reference](../sdk-reference.md)*