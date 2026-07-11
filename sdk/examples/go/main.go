// GGID Go SDK — Complete Integration Example
//
// Demonstrates:
// 1. 3-line JWT authentication (client + middleware + protected handler)
// 2. Role-based access control (manual check from JWT claims)
// 3. Scope-based access control (manual check from JWT claims)
// 4. Permission check via policy engine (RequirePermission)
// 5. Auto token refresh (TokenManager)
//
// Prerequisites:
//   - GGID running: cd deploy && docker compose up -d
//   - Default tenant: 00000000-0000-0000-0000-000000000001
//
// Run:
//   export GGID_URL=http://localhost:8080
//   export GGID_USER=admin
//   export GGID_PASS=Admin@123456
//   go run main.go
//
// Test with curl:
//   # Get a token
//   TOKEN=$(curl -s -H 'Content-Type: application/json' \
//     -d '{"username":"admin","password":"Admin@123456"}' \
//     http://localhost:8080/api/v1/auth/login | jq -r .access_token)
//
//   # Public endpoint
//   curl http://localhost:9090/public
//
//   # Protected (any authenticated user)
//   curl -H "Authorization: Bearer $TOKEN" http://localhost:9090/api/me
//
//   # Admin-only (requires "admin" role in JWT)
//   curl -H "Authorization: Bearer $TOKEN" http://localhost:9090/api/admin
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	ggid "github.com/ggid/ggid/sdk/go/ggid"
)

func main() {
	addr := flag.String("addr", ":9090", "listen address")
	flag.Parse()

	ggidURL := envOr("GGID_URL", "http://localhost:8080")
	username := envOr("GGID_USER", "admin")
	password := envOr("GGID_PASS", "Admin@123456")
	tenantID := envOr("GGID_TENANT", "00000000-0000-0000-0000-000000000001")

	ctx := context.Background()

	// === 3-LINE INTEGRATION ===
	// Line 1: Create client with JWKS verification
	client := ggid.NewClient(ggidURL,
		ggid.WithTenantID(tenantID),
		ggid.WithJWKS(ggidURL+"/.well-known/jwks.json"),
	)

	// Line 2: Login with auto-refresh via TokenManager
	tm := ggid.NewTokenManager(client)
	if _, err := tm.Login(ctx, username, password); err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	log.Println("Login OK — auto-refresh enabled")

	// Line 3: Protect routes with middleware
	mux := setupRoutes(client, tm)
	handler := client.Middleware(mux)

	log.Printf("Server running on %s", *addr)
	log.Printf("  GGID URL: %s", ggidURL)
	log.Fatal(http.ListenAndServe(*addr, handler))
}

func setupRoutes(client *ggid.Client, tm *ggid.TokenManager) *http.ServeMux {
	mux := http.NewServeMux()

	// --- Public routes ---
	// Note: GGID middleware auto-skips /healthz, /login, /register, /docs
	// Add custom public paths by checking in your handler.
	mux.HandleFunc("/public", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"message": "public — no auth needed"})
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})

	// --- Protected: any authenticated user ---
	mux.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		claims := ggid.ClaimsFromContext(r.Context())
		if claims == nil {
			writeJSON(w, 401, map[string]string{"error": "not authenticated"})
			return
		}
		writeJSON(w, 200, claims)
	})

	// --- Role-protected: requires "admin" role ---
	mux.HandleFunc("/api/admin", func(w http.ResponseWriter, r *http.Request) {
		claims := ggid.ClaimsFromContext(r.Context())
		if claims == nil {
			writeJSON(w, 401, map[string]string{"error": "not authenticated"})
			return
		}

		// Check roles from JWT claims
		roles := extractRoles(claims)
		if !contains(roles, "admin") {
			writeJSON(w, 403, map[string]any{
				"error":  "insufficient_role",
				"needed": "admin",
				"have":   roles,
			})
			return
		}

		writeJSON(w, 200, map[string]any{
			"message": "admin access granted",
			"user":    claims["sub"],
		})
	})

	// --- Scope-protected: requires "read:users" scope ---
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		claims := ggid.ClaimsFromContext(r.Context())
		if claims == nil {
			writeJSON(w, 401, map[string]string{"error": "not authenticated"})
			return
		}

		// Check scope from JWT claims
		scopes := extractScopes(claims)
		roles := extractRoles(claims)
		if !contains(scopes, "read:users") && !contains(roles, "admin") {
			writeJSON(w, 403, map[string]string{
				"error":  "insufficient_scope",
				"needed": "read:users",
			})
			return
		}

		// Use TokenManager for auto-refreshing token
		token, err := tm.AccessToken(r.Context())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("token refresh: %v", err)})
			return
		}

		users, err := client.ListUsers(r.Context(), token)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"users": users})
	})

	// --- Permission-protected: requires policy check via API ---
	mux.HandleFunc("/api/reports", func(w http.ResponseWriter, r *http.Request) {
		token, err := tm.AccessToken(r.Context())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}

		result, err := client.CheckPermission(r.Context(), token, "reports", "read")
		if err != nil || !result.Allowed {
			writeJSON(w, 403, map[string]string{"error": "permission denied for reports:read"})
			return
		}

		writeJSON(w, 200, map[string]string{"message": "reports access granted by policy"})
	})

	// --- Auto-refresh demo ---
	mux.HandleFunc("/api/token-info", func(w http.ResponseWriter, r *http.Request) {
		token, err := tm.AccessToken(r.Context())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		ts := tm.GetTokens()
		writeJSON(w, 200, map[string]any{
			"token_preview": token[:min(20, len(token))] + "...",
			"expires_in":    ts.ExpiresIn,
			"auto_refresh":  true,
		})
	})

	return mux
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func extractRoles(claims map[string]interface{}) []string {
	if roles, ok := claims["roles"].([]interface{}); ok {
		result := make([]string, 0, len(roles))
		for _, r := range roles {
			if s, ok := r.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

func extractScopes(claims map[string]interface{}) []string {
	if scope, ok := claims["scope"].(string); ok {
		return strings.Split(scope, " ")
	}
	return nil
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
