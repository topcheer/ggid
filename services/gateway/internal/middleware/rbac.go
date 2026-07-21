package middleware

import (
	"net/http"
	"strings"
)

// AdminOnly is a middleware that requires admin-level scopes for sensitive endpoints.
// Endpoints protected: user management, audit events, policies, webhooks, OAuth clients, roles.
// This enforces defense-in-depth at the gateway level, complementing backend service checks.
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the user has admin-level scope.
		claims := ExtractJWTClaims(r)
		if len(claims.Scopes) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		if hasAdminScope(claims.Scopes) {
			next.ServeHTTP(w, r)
			return
		}

		// Non-admin user accessing admin-only endpoint
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"detail":"insufficient permissions","title":"Forbidden","type":"https://ggid.dev/errors/forbidden"}`))
	})
}

// defaultAdminPrefixes is the hardcoded admin-endpoint list, kept as a
// fallback when the dynamic RBAC resolver has no data (ADR-dynamic-rbac).
var defaultAdminPrefixes = []string{
	"/api/v1/users",           // User CRUD (except /me which is public-ish)
	"/api/v1/audit/",          // Audit events
	"/api/v1/policies",        // Policy management
	"/api/v1/webhooks",        // Webhook CRUD
	"/api/v1/oauth/clients",   // OAuth client management
	"/api/v1/roles",           // Role management (listing is OK for all, but POST/DELETE need admin)
	"/api/v1/admin/",          // Admin dashboard
	"/api/v1/settings/",       // System settings
	"/api/v1/system/",         // System management
	"/api/v1/tenants",         // Tenant management (except resolve which is public)
	"/api/v1/impersonate",     // Impersonation (platform admin only)
}

// isAdminEndpoint returns true for endpoints that require admin scope.
func isAdminEndpoint(path string) bool {
	for _, prefix := range defaultAdminPrefixes {
		if strings.HasPrefix(path, prefix) {
			// Allow /api/v1/users/me for self-service
			if path == "/api/v1/users/me" || strings.HasPrefix(path, "/api/v1/users/me/") {
				return false
			}
			// Allow tenant resolve (public lookup)
			if strings.HasPrefix(path, "/api/v1/tenants/resolve") {
				return false
			}
			return true
		}
	}
	return false
}

// RequireAdminScope wraps the proxy handler with admin-only path protection.
// This middleware sits AFTER JWTAuth (which validates tokens and sets claims)
// and BEFORE the reverse proxy, blocking non-admin users from management endpoints.
func RequireAdminScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Public paths are never gated by RBAC (dynamic or static).
		if isRBACExempt(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Dynamic RBAC (ADR-dynamic-rbac): DB-driven route permissions take
		// precedence when the resolver has data. Claims are extracted once and
		// reused for both the dynamic and the fallback path.
		claims := ExtractJWTClaims(r)
		if res := getRBACResolver(); res != nil && res.Available() {
			// No JWT at all → let JWTAuth produce the 401 (same contract as
			// the static path); never 403 anonymous requests here.
			if len(claims.Scopes) == 0 && len(claims.Roles) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			if allow, handled := res.CheckAccess(r.Context(), r.URL.Path, r.Method, claims); handled {
				if allow {
					next.ServeHTTP(w, r)
					return
				}
				writeAdminForbidden(w)
				return
			}
			// No dynamic rule matched → static fallback below.
		}

		if !isAdminEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Admin endpoint: check scope
		if len(claims.Scopes) == 0 && len(claims.Roles) == 0 {
			// No JWT on a protected path — let JWTAuth handle the 401
			next.ServeHTTP(w, r)
			return
		}

		if hasAdminScope(claims.Scopes) || hasAdminScope(claims.Roles) {
			next.ServeHTTP(w, r)
			return
		}

		writeAdminForbidden(w)
	})
}

// writeAdminForbidden emits the standard 403 body for admin-scope failures.
func writeAdminForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"detail":"insufficient permissions for this endpoint","title":"Forbidden","type":"https://ggid.dev/errors/forbidden"}`))
}

// hasAdminScope checks if any of the user's scopes indicate admin-level access.
// Supports both lowercase scope strings (platform:admin, admin) and
// role display names (Administrator, Platform Administrator, Tenant Administrator).
func hasAdminScope(scopes []string) bool {
	for _, sc := range scopes {
		lower := strings.ToLower(sc)
		switch lower {
		case "admin", "superadmin", "administrator", "roles:write", "*",
			"platform:admin", "platform administrator",
			"tenant:admin", "tenant administrator":
			return true
		}
	}
	return false
}
