// GGID Go SDK Quickstart — 5-minute JWT authentication integration.
//
// This example shows how to:
// 1. Login and get a JWT token
// 2. Protect your API routes with GGID middleware
// 3. Access user info from the JWT in your handlers
//
// Prerequisites:
//   - GGID running (cd deploy && docker compose up -d)
//   - Go 1.21+
//
// Run:
//   go run main.go
//
// Test:
//   curl http://localhost:9090/public          → 200 (no auth needed)
//   curl http://localhost:9090/protected        → 401 (missing token)
//   curl -H "Authorization: Bearer <token>" http://localhost:9090/protected → 200
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	ggid "github.com/ggid/ggid/sdk/go/ggid"
	ggidmw "github.com/ggid/ggid/sdk/go/middleware"
)

func main() {
	ctx := context.Background()

	// Step 1: Create a client and login
	// =================================
	client := ggid.NewClient("http://localhost:8080",
		ggid.WithTenantID("00000000-0000-0000-0000-000000000001"))

	tokens, err := client.Login(ctx, "admin", "Admin@123456")
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	fmt.Printf("Login OK — token length: %d\n", len(tokens.AccessToken))

	// Step 2: Create your API routes
	// =================================
	mux := http.NewServeMux()

	// Public route — no auth required
	mux.HandleFunc("/public", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"message": "public endpoint, no auth needed"})
	})

	// Protected route — requires valid JWT
	mux.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		// Extract user info from JWT (set by middleware)
		user, ok := ggidmw.FromContext(r.Context())
		if !ok {
			http.Error(w, "no user in context", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"message": "authenticated!",
			"user":    user.Username,
			"email":   user.Email,
			"roles":   user.Roles,
		})
	})

	// Step 3: Wrap with GGID auth middleware — this is the magic line
	// =================================
	handler := ggidmw.Auth("http://localhost:8080", ggidmw.Options{
		SkipPaths: []string{"/public"},
	})(mux)

	log.Println("Quickstart server running on :9090")
	log.Println("  Public:    http://localhost:9090/public")
	log.Println("  Protected: http://localhost:9090/protected")
	log.Println("  Token:     ", tokens.AccessToken[:50]+"...")
	log.Fatal(http.ListenAndServe(":9090", handler))
}
