package rbac

import (
	"context"
	"net/http"
	"strings"
)

// RoutePermission maps an API route pattern + HTTP method to a required permission.
type RoutePermission struct {
	Method   string // HTTP method: GET, POST, PUT, DELETE
	Pattern  string // path pattern with {param} wildcards, e.g. /api/v1/users/{id}
	Resource string // required resource
	Action   string // required action
	Scope    string // required scope (minimum)
}

// MatchRoute checks if a request path matches a pattern with {param} wildcards.
func matchRoute(pattern, path string) bool {
	pp := strings.Split(strings.Trim(pattern, "/"), "/")
	ap := strings.Split(strings.Trim(path, "/"), "/")
	if len(pp) != len(ap) {
		return false
	}
	for i := range pp {
		if strings.HasPrefix(pp[i], "{") && strings.HasSuffix(pp[i], "}") {
			continue // wildcard match
		}
		if pp[i] != ap[i] {
			return false
		}
	}
	return true
}

// RoutePermissions maps all protected API routes to their required permissions.
// This is code-defined and immutable (same as SystemPermissions).
var RoutePermissions = []RoutePermission{
	// ── Users ──
	{"GET", "/api/v1/users", "users", "read", "tenant"},
	{"GET", "/api/v1/users/{id}", "users", "read", "own"},
	{"POST", "/api/v1/users", "users", "create", "tenant"},
	{"PUT", "/api/v1/users/{id}", "users", "update", "own"},
	{"DELETE", "/api/v1/users/{id}", "users", "delete", "tenant"},
	{"POST", "/api/v1/users/{id}/freeze", "users", "freeze", "tenant"},
	{"POST", "/api/v1/users/{id}/unfreeze", "users", "freeze", "tenant"},
	{"POST", "/api/v1/users/import", "users", "import", "tenant"},
	{"GET", "/api/v1/users/export", "users", "export", "tenant"},

	// ── Roles ──
	{"GET", "/api/v1/roles", "roles", "read", "tenant"},
	{"POST", "/api/v1/roles", "roles", "create", "tenant"},
	{"PUT", "/api/v1/roles/{id}", "roles", "update", "tenant"},
	{"DELETE", "/api/v1/roles/{id}", "roles", "delete", "tenant"},
	{"POST", "/api/v1/roles/assign", "roles", "assign", "tenant"},
	{"POST", "/api/v1/roles/revoke", "roles", "assign", "tenant"},
	{"GET", "/api/v1/users/{id}/roles", "roles", "read", "tenant"},

	// ── OAuth Clients ──
	{"GET", "/api/v1/oauth/clients", "oauth_clients", "read", "tenant"},
	{"GET", "/api/v1/oauth/clients/{id}", "oauth_clients", "read", "tenant"},
	{"POST", "/api/v1/oauth/clients", "oauth_clients", "create", "tenant"},
	{"PUT", "/api/v1/oauth/clients/{id}", "oauth_clients", "update", "tenant"},
	{"DELETE", "/api/v1/oauth/clients/{id}", "oauth_clients", "delete", "tenant"},
	{"POST", "/api/v1/oauth/clients/{id}/rotate-secret", "oauth_clients", "rotate_secret", "tenant"},

	// ── Organizations ──
	{"GET", "/api/v1/orgs", "orgs", "read", "tenant"},
	{"POST", "/api/v1/orgs", "orgs", "create", "tenant"},
	{"PUT", "/api/v1/orgs/{id}", "orgs", "update", "tenant"},
	{"DELETE", "/api/v1/orgs/{id}", "orgs", "delete", "tenant"},

	// ── Policies ──
	{"GET", "/api/v1/policies", "policies", "read", "tenant"},
	{"GET", "/api/v1/auth/conditional-access/", "policies", "read", "tenant"},
	{"POST", "/api/v1/auth/conditional-access/", "policies", "create", "tenant"},
	{"PUT", "/api/v1/auth/conditional-access/{id}", "policies", "update", "tenant"},
	{"DELETE", "/api/v1/auth/conditional-access/{id}", "policies", "delete", "tenant"},

	// ── Audit ──
	{"GET", "/api/v1/audit/events", "audit", "read", "tenant"},
	{"GET", "/api/v1/audit/stats", "audit", "read", "tenant"},
	{"GET", "/api/v1/audit/export", "audit", "export", "tenant"},
	{"GET", "/api/v1/audit/integrity", "audit", "read_integrity", "tenant"},

	// ── Webhooks ──
	{"GET", "/api/v1/webhooks", "webhooks", "read", "tenant"},
	{"POST", "/api/v1/webhooks", "webhooks", "create", "tenant"},
	{"PUT", "/api/v1/webhooks/{id}", "webhooks", "update", "tenant"},
	{"DELETE", "/api/v1/webhooks/{id}", "webhooks", "delete", "tenant"},

	// ── API Keys ──
	{"GET", "/api/v1/api-keys", "api_keys", "read", "tenant"},
	{"POST", "/api/v1/api-keys", "api_keys", "create", "tenant"},
	{"DELETE", "/api/v1/api-keys/{id}", "api_keys", "delete", "tenant"},

	// ── Security ──
	{"GET", "/api/v1/security/posture", "security", "read", "tenant"},
	{"GET", "/api/v1/security/dashboard", "security", "read", "tenant"},
	{"GET", "/api/v1/zt/posture", "security", "posture", "read"},

	// ── Settings ──
	{"GET", "/api/v1/settings/", "settings", "read", "tenant"},
	{"PUT", "/api/v1/settings/", "settings", "update", "tenant"},
	{"GET", "/api/v1/admin/feature-flags", "settings", "feature_flags", "tenant"},

	// ── Identity Dashboard ──
	{"GET", "/api/v1/identity/dashboard", "identity", "read", "tenant"},
	{"GET", "/api/v1/identity/dashboard/stats", "identity", "read", "tenant"},

	// ── Tenants (platform admin) ──
	{"GET", "/api/v1/tenants", "tenants", "read", "all"},
	{"POST", "/api/v1/tenants/create", "tenants", "create", "all"},

	// ── Sessions ──
	{"GET", "/api/v1/auth/sessions", "sessions", "read", "own"},

	// ── WebAuthn Credentials (self-service) ──
	{"GET", "/api/v1/auth/webauthn/credentials", "webauthn", "manage", "own"},
	{"DELETE", "/api/v1/auth/webauthn/credentials/{id}", "webauthn", "manage", "own"},

	// ── System Admin ──
	{"POST", "/api/v1/auth/impersonate", "system", "impersonate", "tenant"},
	{"GET", "/api/v1/admin/key-rotation", "system", "key_rotation", "tenant"},
	{"GET", "/api/v1/admin/secrets", "system", "secrets", "tenant"},
	{"GET", "/api/v1/admin/backup", "system", "backup", "tenant"},
}

// CheckRoutePermission finds the required permission for a request
// and checks if the user's permissions satisfy it.
// Returns (matched, allowed) — matched=false if no route rule exists (allow by default).
func CheckRoutePermission(method, path string, userPerms []string) (matched bool, allowed bool) {
	for _, rp := range RoutePermissions {
		if rp.Method == method && matchRoute(rp.Pattern, path) {
			return true, HasPermission(userPerms, rp.Resource, rp.Action, rp.Scope)
		}
	}
	// No route rule found — allow (existing adminOnlyPaths handles coarse-grained checks)
	return false, true
}

// ExtractPermissionsFromRequest extracts the permissions claim from JWT context.
// The gateway middleware sets this after JWT verification.
func ExtractPermissionsFromRequest(r *http.Request) []string {
	if perms, ok := r.Context().Value(permissionsCtxKey{}).([]string); ok {
		return perms
	}
	return nil
}

type permissionsCtxKey struct{}

// WithPermissions injects permissions into request context.
func WithPermissions(r *http.Request, perms []string) *http.Request {
	ctx := context.WithValue(r.Context(), permissionsCtxKey{}, perms)
	return r.WithContext(ctx)
}
