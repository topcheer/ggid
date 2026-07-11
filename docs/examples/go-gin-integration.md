# Go Gin Integration Example

> Complete, runnable Gin web server using the GGID Go SDK for JWT verification and route protection.

---

## Prerequisites

- Go 1.21+
- GGID Gateway running at `http://localhost:8080`

---

## Project Setup

```bash
mkdir ggid-gin-demo && cd ggid-gin-demo
go mod init demo
go get github.com/gin-gonic/gin@latest
go get github.com/ggid/ggid/sdk/go@latest
```

---

## Complete Application

Create `main.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	ggid "github.com/ggid/ggid/sdk/go"
	ggidmw "github.com/ggid/ggid/sdk/go/middleware"
)

var ggidURL = getenv("GGID_URL", "http://localhost:8080")

func main() {
	r := gin.Default()

	// ─── Public Routes ────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "gin-demo"})
	})

	// ─── Protected API Group ──────────────────────────────────
	api := r.Group("/api")

	// GGID JWT verification middleware (adapted for Gin)
	api.Use(ginAuthMiddleware(ggidURL))
	{
		// GET /api/me — any authenticated user
		api.GET("/me", func(c *gin.Context) {
			info, _ := ggidmw.FromContext(c.Request.Context())
			c.JSON(http.StatusOK, gin.H{
				"user_id":   info.UserID,
				"tenant_id": info.TenantID,
				"username":  info.Username,
				"email":     info.Email,
				"roles":     info.Roles,
				"scopes":    info.Scopes,
			})
		})

		// GET /api/users — requires read:users scope
		api.GET("/users", ginRequireScope("read:users"), listUsers)

		// POST /api/users — requires admin role
		api.POST("/users", ginRequireRole("admin"), createUser)

		// DELETE /api/users/:id — requires admin role
		api.DELETE("/users/:id", ginRequireRole("admin"), deleteUser)

		// POST /api/check-permission — check Policy Engine
		api.POST("/check-permission", checkPermission)
	}

	port := getenv("PORT", "8081")
	fmt.Printf("Gin demo on :%s (GGID: %s)\n", port, ggidURL)
	r.Run(":" + port)
}

// ─── GGID Middleware Adapter ─────────────────────────────────
// ginAuthMiddleware wraps the GGID HTTP middleware for Gin.
func ginAuthMiddleware(baseURL string) gin.HandlerFunc {
	auth := ggidmw.Auth(baseURL, ggidmw.Options{})
	return func(c *gin.Context) {
		// Create a response recorder to catch auth failures
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		})
		auth(next).ServeHTTP(c.Writer, c.Request)
		// Abort if middleware wrote a response (e.g., 401)
		if c.Writer.Written() {
			c.Abort()
		}
	}
}

// ─── Guards ──────────────────────────────────────────────────

func ginRequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		info, ok := ggidmw.FromContext(c.Request.Context())
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not_authenticated"})
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

func ginRequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		info, ok := ggidmw.FromContext(c.Request.Context())
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not_authenticated"})
			return
		}
		for _, s := range info.Scopes {
			if s == scope || s == "*:*" {
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

// ─── Handlers ────────────────────────────────────────────────

func listUsers(c *gin.Context) {
	info, _ := ggidmw.FromContext(c.Request.Context())
	// In production: SELECT * FROM users WHERE tenant_id = $1
	c.JSON(http.StatusOK, gin.H{
		"users": []gin.H{
			{"id": "usr_001", "username": "alice", "tenant_id": info.TenantID},
			{"id": "usr_002", "username": "bob", "tenant_id": info.TenantID},
		},
		"count":       2,
		"requested_by": info.Username,
	})
}

func createUser(c *gin.Context) {
	info, _ := ggidmw.FromContext(c.Request.Context())

	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         "usr_" + req.Username,
		"username":   req.Username,
		"email":      req.Email,
		"tenant_id":  info.TenantID,
		"created_by": info.UserID,
	})
}

func deleteUser(c *gin.Context) {
	info, _ := ggidmw.FromContext(c.Request.Context())
	userID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{
		"status":     "deleted",
		"user_id":    userID,
		"deleted_by": info.UserID,
	})
}

func checkPermission(c *gin.Context) {
	info, _ := ggidmw.FromContext(c.Request.Context())

	var req struct {
		Action   string `json:"action"`
		Resource string `json:"resource"`
	}
	c.ShouldBindJSON(&req)

	// Call Policy Engine via GGID client
	client := ggid.New(ggidURL, ggid.WithAPIKey(os.Getenv("GGID_API_KEY")))
	_ = client // client.CheckPermission(ctx, info.UserID, req.Resource, req.Action)

	c.JSON(http.StatusOK, gin.H{
		"user_id":  info.UserID,
		"action":   req.Action,
		"resource": req.Resource,
		"allowed":  true, // simplified
	})
}

// ─── Helpers ─────────────────────────────────────────────────

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Ensure json import is used for the _ variable
var _ = json.Marshal
var _ = strings.Contains
```

---

## Environment Variables

```bash
export GGID_URL=http://localhost:8080
export GGID_API_KEY=your-api-key
export PORT=8081
```

---

## Run

```bash
go run main.go
# → Gin demo on :8081 (GGID: http://localhost:8080)
```

---

## Test the Endpoints

### Health Check (public)

```bash
curl http://localhost:8081/health
# → {"status":"ok","service":"gin-demo"}
```

### Protected Route Without Token (401)

```bash
curl http://localhost:8081/api/me
# → 401 Unauthorized
```

### Get User Info

```bash
JWT=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","password":"Admin123!"}' | jq -r .access_token)

curl -s http://localhost:8081/api/me \
  -H "Authorization: Bearer $JWT" | jq .
```

### List Users (requires read:users)

```bash
curl -s http://localhost:8081/api/users \
  -H "Authorization: Bearer $JWT" | jq .
```

### Create User (requires admin role)

```bash
curl -s -X POST http://localhost:8081/api/users \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"username":"newuser","email":"new@test.com"}' | jq .
```

### Delete User (requires admin role)

```bash
curl -s -X DELETE http://localhost:8081/api/users/usr_001 \
  -H "Authorization: Bearer $JWT" | jq .
```

### Insufficient Scope (403)

```bash
# User without admin role
curl -s -X DELETE http://localhost:8081/api/users/usr_001 \
  -H "Authorization: Bearer $READONLY_JWT" | jq .
# → {"error":"insufficient_role","required":"admin"}
```

---

## Key Takeaways

1. **`ginAuthMiddleware`** adapts GGID's `http.Handler` middleware for Gin's `gin.Context`.
2. **`ggidmw.FromContext`** extracts user info from request context in any handler.
3. **`ginRequireRole` / `ginRequireScope`** are reusable Gin middleware guards.
4. **Tenant isolation** is automatic — use `info.TenantID` for database queries.
5. **`gin.RequirePermission`** can wrap specific handlers for Policy Engine checks.

---

*See also: [Gin Integration Guide](../integration-guides/gin.md) | [Go Integration Example](go-integration.md) | [RBAC Guide](../guides/role-based-access.md)*

*Last updated: 2025-07-11*
