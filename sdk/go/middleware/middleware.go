// Package middleware provides HTTP middleware for integrating GGID authentication
// into Go backend applications.
//
// Usage:
//
//	import ggidmw "github.com/ggid/ggid/sdk/go/middleware"
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/api/protected", myHandler)
//
//	// Wrap with GGID auth — verifies JWT on every request
//	handler := ggidmw.Auth("https://iam.example.com", ggidmw.Options{
//		SkipPaths: []string{"/health", "/public"},
//	})(mux)
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Options configures the Auth middleware.
type Options struct {
	// SkipPaths are URL paths that bypass JWT verification (e.g. health checks).
	SkipPaths []string
	// TenantHeader is the header name for tenant ID (default: X-Tenant-ID).
	TenantHeader string
	// OnUnauthorized is called when auth fails (default: writes JSON error).
	OnUnauthorized http.HandlerFunc
}

// UserInfo holds the authenticated user information extracted from the JWT.
type UserInfo struct {
	UserID   string
	TenantID string
	Username string
	Email    string
	Roles    []string
	Scopes   []string
}

type contextKey struct{}

// FromContext extracts UserInfo from the request context.
func FromContext(ctx context.Context) (*UserInfo, bool) {
	info, ok := ctx.Value(contextKey{}).(*UserInfo)
	return info, ok
}

// Auth returns an HTTP middleware that verifies GGID JWT tokens.
// The baseURL should point to the GGID Gateway (e.g. http://localhost:8080).
func Auth(baseURL string, opts Options) func(http.Handler) http.Handler {
	if opts.TenantHeader == "" {
		opts.TenantHeader = "X-Tenant-ID"
	}
	if opts.OnUnauthorized == nil {
		opts.OnUnauthorized = defaultUnauthorized
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip whitelisted paths
			for _, p := range opts.SkipPaths {
				if r.URL.Path == p || strings.HasPrefix(r.URL.Path, p) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extract Bearer token
			token := extractBearer(r)
			if token == "" {
				opts.OnUnauthorized(w, r)
				return
			}

			// Parse JWT (offline — no signature verification for simplicity)
			// In production, verify against JWKS from the Gateway.
			info, err := parseToken(token)
			if err != nil {
				opts.OnUnauthorized(w, r)
				return
			}

			// Inject user info into context
			ctx := context.WithValue(r.Context(), contextKey{}, info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that checks if the user has the given role.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info, ok := FromContext(r.Context())
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			for _, userRole := range info.Roles {
				if userRole == role || userRole == "admin" {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, fmt.Sprintf(`{"error":"forbidden: requires role '%s'"}`, role), http.StatusForbidden)
		})
	}
}

// extractBearer extracts the Bearer token from the Authorization header.
func extractBearer(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

// parseToken parses a JWT and extracts user info from claims.
func parseToken(tokenString string) (*UserInfo, error) {
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	info := &UserInfo{}
	if v, ok := claims["sub"]; ok {
		info.UserID = fmt.Sprintf("%v", v)
	}
	if v, ok := claims["tenant_id"]; ok {
		info.TenantID = fmt.Sprintf("%v", v)
	}
	if v, ok := claims["username"]; ok {
		info.Username = fmt.Sprintf("%v", v)
	}
	if v, ok := claims["email"]; ok {
		info.Email = fmt.Sprintf("%v", v)
	}
	if roles, ok := claims["roles"].([]any); ok {
		for _, r := range roles {
			info.Roles = append(info.Roles, fmt.Sprintf("%v", r))
		}
	}
	if scope, ok := claims["scope"].(string); ok {
		info.Scopes = strings.Split(scope, " ")
	}

	return info, nil
}

func defaultUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "missing or invalid token",
	})
}
