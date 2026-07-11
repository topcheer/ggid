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
    ggidmw "github.com/ggid/ggid/sdk/go/middleware"
)

func main() {
    r := gin.Default()

    // Public routes (no auth)
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok"})
    })

    // Protected API group — wrap GGID middleware for Gin
    api := r.Group("/api")
    api.Use(ginAuthMiddleware("http://localhost:8080"))
    {
        api.GET("/me", func(c *gin.Context) {
            info, _ := ggidmw.FromContext(c.Request.Context())
            c.JSON(http.StatusOK, gin.H{
                "user_id":   info.UserID,
                "tenant_id": info.TenantID,
                "username":  info.Username,
                "scopes":    info.Scopes,
            })
        })
    }

    r.Run(":8081")
}

// ginAuthMiddleware adapts the GGID HTTP middleware for Gin.
func ginAuthMiddleware(baseURL string) gin.HandlerFunc {
    auth := ggidmw.Auth(baseURL, ggidmw.Options{})
    return func(c *gin.Context) {
        next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            c.Request = r
            c.Next()
        })
        auth(next).ServeHTTP(c.Writer, c.Request)
        c.Abort()
    }
}
```

## Role Check Middleware

```go
func ginRequireRole(role string) gin.HandlerFunc {
    return func(c *gin.Context) {
        info, ok := ggidmw.FromContext(c.Request.Context())
        if !ok {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
            return
        }
        for _, r := range info.Roles {
            if r == role || r == "admin" {
                c.Next()
                return
            }
        }
        c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
            "error":    "insufficient_role",
            "required": role,
        })
    }
}

// Usage
api.DELETE("/users/:id", ginRequireRole("admin"), deleteUser)
api.GET("/users", ginRequireRole("editor"), listUsers)
```

## Scope Check Middleware

```go
func ginRequireScope(scope string) gin.HandlerFunc {
    return func(c *gin.Context) {
        info, ok := ggidmw.FromContext(c.Request.Context())
        if !ok {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
            return
        }
        for _, s := range info.Scopes {
            if s == scope {
                c.Next()
                return
            }
        }
        c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
            "error":    "insufficient_scope",
            "required": scope,
        })
    }
}
```

## Tenant-Aware Handler

```go
func listUsers(c *gin.Context) {
    info, _ := ggidmw.FromContext(c.Request.Context())

    // Use tenant_id for all queries
    rows, err := db.Query(
        `SELECT id, username, email FROM users WHERE tenant_id = $1`,
        info.TenantID,
    )
    // ...
}
```

## Optional Auth (Some Routes Protected)

```go
// Apply middleware only to specific groups
api := r.Group("/api")
api.Use(ginAuthMiddleware("http://localhost:8080"))

// Public routes don't get middleware
r.GET("/health", healthHandler)
r.POST("/api/auth/login", loginHandler)
```

## Using the GGID Client

```go
import ggid "github.com/ggid/ggid/sdk/go"

func getUser(c *gin.Context) {
    client := ggid.New("http://localhost:8080",
        ggid.WithAPIKey(os.Getenv("GGID_API_KEY")),
    )
    user, err := client.GetUser(c, c.Param("id"))
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
export GGID_API_KEY=your-api-key
export PORT=8081
```

---

*See: [Go SDK Quickstart](../quickstart/go-sdk.md) | [Go Integration Example](../examples/go-integration.md) | [SDK Reference](../sdk-reference.md)*
