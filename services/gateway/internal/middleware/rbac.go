package middleware

import (
	"encoding/json"
	"fmt"
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
	"/api/v1/users",            // User CRUD (except /me which is public-ish)
	"/api/v1/audit/",           // Audit events
	"/api/v1/policies",         // Policy management
	"/api/v1/policy/",          // Policy service mounts singular paths (P1 gap)
	"/api/v1/webhooks",         // Webhook CRUD
	"/api/v1/oauth/clients",    // OAuth client management
	"/api/v1/roles",            // Role management (listing is OK for all, but POST/DELETE need admin)
	"/api/v1/admin/",           // Admin dashboard
	"/api/v1/settings/",        // System settings
	"/api/v1/system/",          // System management
	"/api/v1/tenants",          // Tenant management (except resolve which is public)
	"/api/v1/impersonate",      // Impersonation (platform admin only)
	"/api/v1/auth/impersonate", // Auth service impersonation endpoint
	// Admin-level management endpoints (self-service variants are in publicPaths)
	"/api/v1/auth/mfa/factors",          // MFA factor configuration
	"/api/v1/auth/mfa/admin/",           // Admin MFA management for other users
	"/api/v1/auth/credentials/",         // Credential vault
	"/api/v1/auth/credential-stuffing/", // Credential stuffing config
	"/api/v1/mdm/devices",               // MDM device management
	"/api/v1/identity/devices/",         // Device posture management
	// Paths rewritten to /api/v1/audit/* before proxying — must be
	// listed here because RBAC check runs BEFORE the URL rewrite.
	"/api/v1/access-reviews",   // → /api/v1/audit/access-reviews
	"/api/v1/activity",         // → /api/v1/audit/activity
	"/api/v1/exports",          // → /api/v1/audit/exports
	"/api/v1/providers/config", // Provider config CRUD (admin only; /status stays public)
	// Additional admin paths (merged from former isAdminOnlyPath list — R139 fix)
	"/api/v1/api-keys",           // API key management
	"/api/v1/access-keys",        // Access key management
	"/api/v1/identity/dashboard", // Identity dashboard (admin metrics)
}

// SelfServicePaths are /users/me sub-paths exempt from admin checks.
// Only the exact profile path and explicitly listed sub-paths are exempt
// — deep sub-resources like /users/me/settings are NOT exempt.
var SelfServicePaths = map[string]bool{
	"/api/v1/users/me":             true,
	"/api/v1/users/me/permissions": true, // read-only permission listing
}

// isAdminEndpoint returns true for endpoints that require admin scope.
func isAdminEndpoint(path string) bool {
	for _, prefix := range defaultAdminPrefixes {
		if strings.HasPrefix(path, prefix) {
			// Allow whitelisted self-service paths (exact match only)
			if SelfServicePaths[path] {
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
			if claims.Subject == "" {
				// SECURITY: API key requests (no JWT subject) must still be
				// checked. If JWTAuth has validated an API key with route
				// permissions, let HasPermissionForRoute in JWTAuth handle it.
				// But do NOT skip admin-gate entirely for admin endpoints.
				if !isAdminEndpoint(r.URL.Path) {
					next.ServeHTTP(w, r)
					return
				}
				// Admin endpoint + no JWT: fall through to static check below
				// which will enforce scope requirements.
			} else if allow, handled := res.CheckAccess(r.Context(), r.URL.Path, r.Method, claims); handled {
				if allow {
					next.ServeHTTP(w, r)
					return
				}
				writeAdminForbidden(w, r)
				return
			}
			// No dynamic rule matched → static fallback below.
		}

		if !isAdminEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Admin endpoint: check scope.
		// No JWT subject → check if this is an API key request.
		if claims.Subject == "" {
			// If API key scopes are present, enforce admin scope check.
			if scopes, ok := r.Context().Value(APIKeyScopesKey).([]string); ok && len(scopes) > 0 {
				isAdmin := false
				for _, s := range scopes {
					if s == "platform:admin" || s == "tenant:admin" {
						isAdmin = true
						break
					}
				}
				if !isAdmin {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]any{
						"error": map[string]string{
							"code":    "permission_denied",
							"message": "admin scope required",
						},
					})
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			// No API key either → let JWTAuth handle the 401.
			next.ServeHTTP(w, r)
			return
		}

		// SECURITY: Only OAuth scopes (from token_endpoint, verified against client
		// registration) grant admin access. Roles claim comes from DB role.name which
		// is tenant-controlled — allowing it here would let a tenant admin create a
		// role named "platform:admin" and escalate privileges.
		if hasAdminScope(claims.Scopes) {
			next.ServeHTTP(w, r)
			return
		}

		// M2M tokens (client_credentials) carry permissions but no admin scope.
		// If the token has the required permission for this route, allow it.
		// SECURITY: Block permission-key fallback on admin-only prefixes to prevent
		// privilege escalation (e.g. users:read accessing GET /api/v1/users).
		if HasPermissionForRoute(r.URL.Path, r.Method, claims.Permissions) && !isAdminOnlyPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		writeAdminForbidden(w, r)
	})
}

// writeAdminForbidden emits the standard 403 body for admin-scope failures.
func writeAdminForbidden(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	requestID := GetRequestID(r.Context())
	if requestID != "" {
		w.Header().Set("X-Request-ID", requestID)
	}
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprintf(w, `{"detail":"insufficient permissions for this endpoint","title":"Forbidden","type":"https://ggid.dev/errors/forbidden","request_id":%q}`, requestID)
}

// hasAdminScope checks if any of the user's scopes indicate admin-level access.
// Only the namespaced, non-forgeable scope strings are accepted. Loose values
// like "admin"/"administrator" are NOT accepted here: role keys are
// tenant-controlled, so a tenant admin could otherwise mint a role whose key
// matches and escalate to platform admin.
func hasAdminScope(scopes []string) bool {
	for _, sc := range scopes {
		switch strings.ToLower(sc) {
		case "platform:admin", "tenant:admin":
			return true
		}
	}
	return false
}

// hasPlatformAdminScope checks if the user has platform:admin scope.
// This is stricter than hasAdminScope - it only accepts platform:admin, not tenant:admin.
// Use this for platform-only operations that should not be accessible to tenant admins.
func hasPlatformAdminScope(scopes []string) bool {
	for _, sc := range scopes {
		if strings.EqualFold(sc, "platform:admin") {
			return true
		}
	}
	return false
}

// isAdminOnlyPath returns true for admin-protected path prefixes where
// permission-key fallback must NOT apply. These paths require platform:admin
// or tenant:admin OAuth scope, not just resource-level permissions.
// Uses defaultAdminPrefixes as single source of truth (R139 fix).
func isAdminOnlyPath(path string) bool {
	return isAdminEndpoint(path)
}
