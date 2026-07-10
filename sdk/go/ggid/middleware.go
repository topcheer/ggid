package ggid

import (
	"context"
	"net/http"
	"strings"
)

// Middleware returns an http.Handler that authenticates requests via GGID JWT.
// Public paths (login, register, healthz) skip authentication.
func (c *Client) Middleware(next http.Handler) http.Handler {
	var publicPaths = map[string]bool{
		"/": true, "/healthz": true, "/docs": true,
		"/api-docs": true, "/login": true, "/register": true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Skip public paths and auth endpoints
		if publicPaths[path] || strings.HasPrefix(path, "/api/v1/auth/") ||
			strings.HasPrefix(path, "/oauth/") {
			next.ServeHTTP(w, r)
			return
		}

		// Extract Bearer token
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error":"missing bearer token"}`, http.StatusUnauthorized)
			return
		}

		token := authHeader[7:]

		// Verify token
		claims, err := c.VerifyToken(r.Context(), token)
		if err != nil {
			status := http.StatusUnauthorized
			if err == ErrTokenExpired {
				http.Error(w, `{"error":"token expired"}`, status)
			} else {
				http.Error(w, `{"error":"invalid token"}`, status)
			}
			return
		}

		// Inject claims into request context
		ctx := context.WithValue(r.Context(), claimsKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// claimsKey is the context key for JWT claims.
type claimsKey struct{}

// ClaimsFromContext extracts JWT claims from the request context.
func ClaimsFromContext(ctx context.Context) map[string]interface{} {
	if v, ok := ctx.Value(claimsKey{}).(map[string]interface{}); ok {
		return v
	}
	return nil
}

// RequirePermission returns a middleware that checks if the user has permission
// for the given resource and action.
func (c *Client) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				http.Error(w, `{"error":"not authenticated"}`, http.StatusUnauthorized)
				return
			}

			// Get token from header
			authHeader := r.Header.Get("Authorization")
			token := ""
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = authHeader[7:]
			}

			result, err := c.CheckPermission(r.Context(), token, resource, action)
			if err != nil || !result.Allowed {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
