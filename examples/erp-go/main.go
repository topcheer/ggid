// Cross-Board ERP Demo — Go implementation
// Tests all GGID core features via Go SDK
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	ggid "github.com/ggid/ggid/sdk/go"
	ggidmw "github.com/ggid/ggid/sdk/go/middleware"
)

var (
	ggidURL    = getEnv("GGID_URL", "http://localhost:8080")
	tenantID   = getEnv("GGID_TENANT_ID", "00000000-0000-0000-0000-000000000001")
	clientID   = getEnv("GGID_CLIENT_ID", "erp-go-demo")
	listenAddr = getEnv("ERP_LISTEN", ":9090")
	ggidClient *ggid.Client
)

func main() {
	// Initialize GGID SDK client
	ggidClient = ggid.NewClient(ggidURL, tenantID, clientID, "")

	mux := http.NewServeMux()

	// === Public routes (no auth) ===
	mux.HandleFunc("/api/auth/login", handleLogin)
	mux.HandleFunc("/api/auth/refresh", handleRefresh)
	mux.HandleFunc("/api/auth/verify", handleVerify)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})

	// === Protected routes (JWT + permission check) ===
	// Users
	mux.Handle("/api/users", authMiddleware(http.HandlerFunc(handleUsers)))
	mux.Handle("/api/users/", authMiddleware(http.HandlerFunc(handleUserByID)))

	// Roles
	mux.Handle("/api/roles", authMiddleware(http.HandlerFunc(handleRoles)))

	// Organizations
	mux.Handle("/api/orgs", authMiddleware(http.HandlerFunc(handleOrgs)))

	// Inventory
	mux.Handle("/api/inventory", authMiddleware(http.HandlerFunc(handleInventory)))
	mux.Handle("/api/inventory/", authMiddleware(http.HandlerFunc(handleInventoryByID)))

	// Orders
	mux.Handle("/api/orders", authMiddleware(http.HandlerFunc(handleOrders)))
	mux.Handle("/api/orders/", authMiddleware(http.HandlerFunc(handleOrderByID)))

	// Audit
	mux.Handle("/api/audit", authMiddleware(http.HandlerFunc(handleAudit)))

	// Dashboard
	mux.Handle("/api/dashboard", authMiddleware(http.HandlerFunc(handleDashboard)))

	fmt.Printf("ERP Go Demo running on %s\n", listenAddr)
	fmt.Printf("GGID URL: %s | Tenant: %s | Client: %s\n", ggidURL, tenantID, clientID)
	log.Fatal(http.ListenAndServe(listenAddr, mux))
}

// authMiddleware verifies JWT and extracts user info + permissions
func authMiddleware(next http.Handler) http.Handler {
	return ggidmw.Middleware(ggidmw.Config{
		GGIDURL:  ggidURL,
		TenantID: tenantID,
	})(next)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// requirePerm checks if the user has a fine-grained permission
func requirePerm(w http.ResponseWriter, r *http.Request, perm string) bool {
	info, ok := ggidmw.FromContext(r.Context())
	if !ok {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return false
	}
	// Check fine-grained permissions claim (new JWT structure)
	for _, p := range info.Permissions {
		if p == perm || p == "admin" {
			return true
		}
	}
	// Fallback: check scopes for backward compatibility (old tokens)
	for _, s := range info.Scopes {
		if s == perm || s == "admin" {
			return true
		}
	}
	writeJSON(w, 403, map[string]string{"error": "missing permission: " + perm})
	return false
}

// requireRole checks if the user has a role
func requireRole(w http.ResponseWriter, r *http.Request, role string) bool {
	info, ok := ggidmw.FromContext(r.Context())
	if !ok {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return false
	}
	for _, r2 := range info.Roles {
		if r2 == role || r2 == "admin" {
			return true
		}
	}
	writeJSON(w, 403, map[string]string{"error": "missing role: " + role})
	return false
}
