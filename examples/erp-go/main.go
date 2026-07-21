// Cross-Board ERP Demo — Go implementation
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	ggid "github.com/ggid/ggid/sdk/go"
)

var (
	ggidURL    = getEnv("GGID_URL", "http://localhost:8080")
	tenantID   = getEnv("GGID_TENANT_ID", "00000001-0000-0000-0000-000000000001")
	listenAddr = getEnv("ERP_LISTEN", ":9090")
	ggidClient *ggid.Client
)

type ctxKey string

const userKey ctxKey = "user"

func main() {
	ggidClient = ggid.New(ggidURL, ggid.WithDiscovery())

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", handleLogin)
	mux.HandleFunc("/api/auth/refresh", handleRefresh)
	mux.HandleFunc("/api/auth/verify", handleVerify)
	mux.HandleFunc("/api/auth/oauth/login", handleOAuthLogin)
	mux.HandleFunc("/api/auth/callback", handleOAuthCallback)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/users", withAuth(handleUsers))
	mux.HandleFunc("/api/users/", withAuth(handleUserByID))
	mux.HandleFunc("/api/roles", withAuth(handleRoles))
	mux.HandleFunc("/api/orgs", withAuth(handleOrgs))
	mux.HandleFunc("/api/inventory", withAuth(handleInventory))
	mux.HandleFunc("/api/inventory/", withAuth(handleInventoryByID))
	mux.HandleFunc("/api/orders", withAuth(handleOrders))
	mux.HandleFunc("/api/orders/", withAuth(handleOrderByID))
	mux.HandleFunc("/api/audit", withAuth(handleAudit))
	mux.HandleFunc("/api/dashboard", withAuth(handleDashboard))

	fmt.Printf("ERP Go Demo on %s | GGID: %s\n", listenAddr, ggidURL)
	log.Fatal(http.ListenAndServe(listenAddr, mux))
}

func withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := getTokenFromHeader(r)
		if token == "" {
			writeJSON(w, 401, map[string]string{"error": "Bearer token required"})
			return
		}
		info, err := ggidClient.VerifyToken(r.Context(), token)
		if err != nil {
			writeJSON(w, 401, map[string]string{"error": "invalid token"})
			return
		}
		ctx := context.WithValue(r.Context(), userKey, info)
		next(w, r.WithContext(ctx))
	}
}

func getUser(ctx context.Context) *ggid.UserInfo {
	v, _ := ctx.Value(userKey).(*ggid.UserInfo)
	return v
}

func currentUserID(r *http.Request) string {
	if info := getUser(r.Context()); info != nil {
		return info.UserID
	}
	return ""
}

func requirePerm(w http.ResponseWriter, r *http.Request, perm string) bool {
	info := getUser(r.Context())
	if info == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return false
	}
	for _, p := range info.Permissions {
		if p == perm || p == "admin" { return true }
	}
	for _, s := range info.Scopes {
		if s == perm || s == "admin" { return true }
	}
	writeJSON(w, 403, map[string]string{"error": "missing permission: " + perm})
	return false
}

func getEnv(k, fb string) string { if v := os.Getenv(k); v != "" { return v }; return fb }

func writeJSON(w http.ResponseWriter, s int, d any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)
	json.NewEncoder(w).Encode(d)
}

func writeError(w http.ResponseWriter, s int, m string) { writeJSON(w, s, map[string]string{"error": m}) }

func methodAllowed(w http.ResponseWriter, r *http.Request, m string) bool {
	if r.Method != m { writeJSON(w, 405, map[string]string{"error": "method not allowed"}); return false }
	return true
}

func parseID(r *http.Request) string {
	parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
	if len(parts) > 0 { return parts[len(parts)-1] }
	return ""
}

func getTokenFromHeader(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") { return strings.TrimPrefix(auth, "Bearer ") }
	return ""
}
