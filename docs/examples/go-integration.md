# Go Backend Integration Example

> Complete, runnable Go HTTP server using the GGID Go SDK to protect handlers with JWT verification and permission checks.

---

## Prerequisites

- Go 1.21+
- GGID Gateway running at `http://localhost:8080`

---

## Project Setup

```bash
mkdir ggid-go-demo && cd ggid-go-demo
go mod init demo

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
	"log"
	"net/http"
	"os"

	ggid "github.com/ggid/ggid/sdk/go"
	ggidmw "github.com/ggid/ggid/sdk/go/middleware"
)

// Configuration from environment
var (
	ggidURL   = getenv("GGID_URL", "http://localhost:8080")
	jwtSecret = getenv("JWT_SECRET", "your-shared-secret")
	port      = getenv("PORT", "8081")
)

func main() {
	// Create GGID SDK client
	client := ggid.New(ggidURL,
		ggid.WithAPIKey(os.Getenv("GGID_API_KEY")),
		ggid.WithJWKS(15*60*1e9), // 15-minute JWKS cache (nanoseconds)
	)

	mux := http.NewServeMux()

	// ─── Public Routes ────────────────────────────────────────
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "go-demo",
		})
	})

	// ─── Protected Routes ─────────────────────────────────────
	// Apply auth middleware to /api/* routes
	protected := http.NewServeMux()

	// GET /api/me — return authenticated user info
	protected.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		info, ok := ggidmw.FromContext(r.Context())
		if !ok {
			http.Error(w, `{"error":"not authenticated"}`, http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"user_id":   info.UserID,
			"tenant_id": info.TenantID,
			"username":  info.Username,
			"email":     info.Email,
			"roles":     info.Roles,
			"scopes":    info.Scopes,
		})
	})

	// GET /api/users — requires read:users permission
	protected.Handle("/api/users",
		client.RequirePermission("users", "read")(
			http.HandlerFunc(listUsers),
		),
	)

	// POST /api/users — requires write:users permission
	protected.Handle("/api/users/create",
		client.RequirePermission("users", "write")(
			http.HandlerFunc(createUser),
		),
	)

	// DELETE /api/users/{id} — requires delete:users permission
	protected.Handle("/api/users/delete/",
		client.RequirePermission("users", "delete")(
			http.HandlerFunc(deleteUser),
		),
	)

	// Wrap protected routes with auth middleware
	authMiddleware := ggidmw.Auth(ggidURL, ggidmw.Options{
		SkipPaths: []string{"/health"},
	})

	// Route dispatch: public vs protected
	topMux := http.NewServeMux()
	topMux.Handle("/health", http.HandlerFunc(healthHandler))
	topMux.Handle("/api/", authMiddleware(protected))

	log.Printf("Go demo server starting on :%s", port)
	log.Printf("GGID Gateway: %s", ggidURL)
	log.Fatal(http.ListenAndServe(":"+port, topMux))
}

// ─── Handlers ────────────────────────────────────────────────

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "go-demo",
	})
}

func listUsers(w http.ResponseWriter, r *http.Request) {
	info, _ := ggidmw.FromContext(r.Context())

	// In production, query your database scoped to info.TenantID
	// SELECT * FROM users WHERE tenant_id = $1
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"users": []map[string]string{
			{"id": "usr_001", "username": "alice", "tenant_id": info.TenantID},
			{"id": "usr_002", "username": "bob", "tenant_id": info.TenantID},
		},
		"count":     2,
		"requested_by": info.Username,
	})
}

func createUser(w http.ResponseWriter, r *http.Request) {
	info, _ := ggidmw.FromContext(r.Context())

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" {
		http.Error(w, `{"error":"username_and_email_required"}`, http.StatusBadRequest)
		return
	}

	// Create user via GGID client
	client := ggid.New(ggidURL, ggid.WithAPIKey(os.Getenv("GGID_API_KEY")))
	_ = client // Use client.Users.Create() in production

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":         fmt.Sprintf("usr_%s", req.Username),
		"username":   req.Username,
		"email":      req.Email,
		"tenant_id":  info.TenantID,
		"created_by": info.UserID,
	})
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	info, _ := ggidmw.FromContext(r.Context())

	// Extract user ID from path: /api/users/delete/{id}
	userID := r.URL.Path[len("/api/users/delete/"):]
	if userID == "" {
		http.Error(w, `{"error":"user_id_required"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":     "deleted",
		"user_id":    userID,
		"deleted_by": info.UserID,
	})
}

// ─── Helpers ─────────────────────────────────────────────────

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

---

## Environment Variables

```bash
export GGID_URL=http://localhost:8080
export JWT_SECRET=your-shared-secret
export GGID_API_KEY=your-api-key
export PORT=8081
```

---

## Run

```bash
go run main.go
# → Go demo server starting on :8081
# → GGID Gateway: http://localhost:8080
```

---

## Test the Endpoints

### Health Check (public)

```bash
curl http://localhost:8081/health | jq .
# → {"status":"ok","service":"go-demo"}
```

### Protected Route Without Token (401)

```bash
curl http://localhost:8081/api/me
# → {"error":"missing or invalid token"}
```

### Get Current User Info

```bash
# Login to get JWT
JWT=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","password":"Admin123!"}' | jq -r .access_token)

# Call protected endpoint
curl -s http://localhost:8081/api/me \
  -H "Authorization: Bearer $JWT" | jq .
```

**Response:**

```json
{
  "user_id": "usr_abc123",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "username": "admin",
  "email": "admin@example.com",
  "roles": ["admin"],
  "scopes": ["read:users", "write:users", "delete:users"]
}
```

### List Users (requires read:users)

```bash
curl -s http://localhost:8081/api/users \
  -H "Authorization: Bearer $JWT" | jq .
```

### Create User (requires write:users)

```bash
curl -s -X POST http://localhost:8081/api/users/create \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"username":"newuser","email":"new@test.com"}' | jq .
# → {"id":"usr_newuser","username":"newuser","email":"new@test.com", ...}
```

### Delete User (requires delete:users)

```bash
curl -s -X DELETE http://localhost:8081/api/users/delete/usr_001 \
  -H "Authorization: Bearer $JWT" | jq .
# → {"status":"deleted","user_id":"usr_001","deleted_by":"usr_abc123"}
```

### Permission Denied (403)

If a user without `write:users` tries to create a user:

```bash
curl -s -X POST http://localhost:8081/api/users/create \
  -H "Authorization: Bearer $READONLY_JWT" \
  -H "Content-Type: application/json" \
  -d '{"username":"hack","email":"hack@test.com"}' | jq .
# → {"error":"forbidden: requires permission users:write"}
```

---

## How It Works

| Layer | Component | Purpose |
|-------|-----------|---------|
| **Auth** | `ggidmw.Auth(baseURL, opts)` | Verifies JWT on every request |
| **Context** | `ggidmw.FromContext(ctx)` | Extracts `UserInfo` (ID, tenant, roles, scopes) |
| **Permission** | `client.RequirePermission(resource, action)` | Checks Policy Engine |
| **Client** | `ggid.New(url, WithAPIKey(...))` | Call GGID management APIs |
| **JWKS** | `ggid.WithJWKS(ttl)` | Cache signing keys for signature verification |

### Middleware Chain

```
Request → ggidmw.Auth (verify JWT)
        → ggidmw.FromContext (extract user)
        → client.RequirePermission (check policy)
        → Handler (business logic)
```

---

## Key Takeaways

1. **`ggidmw.Auth`** wraps any `http.Handler` with JWT verification.
2. **`FromContext`** gives type-safe access to user identity in handlers.
3. **`RequirePermission`** checks the Policy Engine before the handler runs.
4. **Tenant isolation** is automatic — use `info.TenantID` for all database queries.
5. **JWKS caching** avoids per-request signature verification overhead.

---

*See also: [Go SDK Quickstart](../quickstart/go-sdk.md) | [Gin Integration Guide](../integration-guides/gin.md) | [RBAC Guide](../guides/role-based-access.md)*

*Last updated: 2025-07-11*
