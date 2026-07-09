package ggid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// contextKey is used to store user info in request context.
type contextKey string

const (
	// ContextKeyUser is the context key for the authenticated user info.
	ContextKeyUser contextKey = "ggid_user"
)

// MiddlewareConfig controls which paths require authentication.
type MiddlewareConfig struct {
	// PublicPaths are path prefixes that skip JWT verification (e.g. /healthz, /public).
	PublicPaths []string
	// TenantID is injected as X-Tenant-ID header on all proxied requests.
	TenantID string
}

// Middleware wraps an http.Handler with GGID JWT verification.
func (c *Client) Middleware(next http.Handler, cfg MiddlewareConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path is public
		for _, prefix := range cfg.PublicPaths {
			if strings.HasPrefix(r.URL.Path, prefix) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Extract Bearer token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeUnauthorized(w, "missing authorization header")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			writeUnauthorized(w, "invalid authorization scheme")
			return
		}

		// Verify token
		userInfo, err := c.VerifyToken(r.Context(), token)
		if err != nil {
			writeUnauthorized(w, "invalid or expired token")
			return
		}

		// Inject tenant header if configured
		if cfg.TenantID != "" {
			r.Header.Set("X-Tenant-ID", cfg.TenantID)
		}

		// Inject user info into context
		ctx := context.WithValue(r.Context(), ContextKeyUser, userInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFromContext extracts the authenticated user info from request context.
func UserFromContext(ctx context.Context) *UserInfo {
	user, _ := ctx.Value(ContextKeyUser).(*UserInfo)
	return user
}

// RequirePermission returns middleware that checks user permission.
func (c *Client) RequirePermission(resource, action string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			writeUnauthorized(w, "not authenticated")
			return
		}

		allowed, err := c.CheckPermission(r.Context(), user.UserID, resource, action)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "permission check failed")
			return
		}

		if !allowed {
			writeError(w, http.StatusForbidden, "permission denied")
			return
		}

		next(w, r)
	}
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// fmt import is needed for potential future error formatting
var _ = fmt.Sprintf
