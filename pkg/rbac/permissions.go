// Package rbac provides the system-level permission definitions for GGID.
//
// System permissions are code-defined and immutable. They are synced to the
// database on service startup via EnsureSystemPermissions(). New permissions
// are added with each release; existing ones are never modified or deleted
// from code (backward compatibility). Custom (user-created) permissions have
// system_perm=false and can be freely edited/deleted.
package rbac

import (
	"context"
	"fmt"
	"log/slog"
)

// SystemPermission defines an immutable, code-defined permission.
type SystemPermission struct {
	Key         string `json:"key"`          // e.g. "users:read:tenant"
	Name        string `json:"name"`         // e.g. "View Tenant Users"
	Description string `json:"description"`  // human-readable
	Resource    string `json:"resource"`     // e.g. "users"
	Action      string `json:"action"`       // read, create, update, delete, assign_role, etc.
	Scope       string `json:"scope"`        // all, tenant, own, department
}

// scope levels for hierarchy checking: all > tenant > department > own
var scopeOrder = map[string]int{
	"all":        4,
	"tenant":     3,
	"department": 2,
	"own":        1,
}

// ScopeCovers returns true if the granted scope is broader or equal to required.
// e.g. "all" covers "tenant" and "own"; "tenant" covers "own".
func ScopeCovers(granted, required string) bool {
	if granted == required {
		return true
	}
	return scopeOrder[granted] >= scopeOrder[required]
}

// HasPermission checks if the user's permission list satisfies the required
// resource:action:scope. Uses scope hierarchy: broader scope covers narrower.
func HasPermission(userPerms []string, resource, action, scope string) bool {
	for _, p := range userPerms {
		if p == "admin" {
			return true // admin bypass
		}
		// Parse "resource:action:scope"
		var pr, pa, ps string
		n, _ := fmt.Sscanf(p, "%[^:]:%[^:]:%s", &pr, &pa, &ps)
		if n < 3 {
			// Legacy format "resource:action" — treat as "resource:action:tenant"
			if n == 2 {
				ps = "tenant"
			} else {
				continue
			}
		}
		if pr == resource && pa == action && ScopeCovers(ps, scope) {
			return true
		}
	}
	return false
}

// SystemPermissions is the complete list of code-defined permissions.
// Each Console release updates this list. EnsureSystemPermissions() syncs to DB.
var SystemPermissions = []SystemPermission{
	// ═══════════════════════════════════════════
	// USERS
	// ═══════════════════════════════════════════
	{"users:read:all", "View All Users", "View users across all tenants", "users", "read", "all"},
	{"users:read:tenant", "View Tenant Users", "View users within own tenant", "users", "read", "tenant"},
	{"users:read:own", "View Own Profile", "View own user profile only", "users", "read", "own"},
	{"users:create:tenant", "Create Users", "Create new users in own tenant", "users", "create", "tenant"},
	{"users:update:all", "Update All Users", "Edit any user's profile", "users", "update", "all"},
	{"users:update:tenant", "Update Tenant Users", "Edit users within own tenant", "users", "update", "tenant"},
	{"users:update:own", "Update Own Profile", "Edit own profile only", "users", "update", "own"},
	{"users:delete:tenant", "Delete Users", "Soft-delete users in own tenant", "users", "delete", "tenant"},
	{"users:reset_password:tenant", "Reset User Passwords", "Reset passwords for tenant users", "users", "reset_password", "tenant"},
	{"users:assign_role:tenant", "Assign Roles", "Assign/revoke roles to users", "users", "assign_role", "tenant"},
	{"users:import:tenant", "Import Users", "Bulk import users via CSV", "users", "import", "tenant"},
	{"users:export:tenant", "Export Users", "Export user data", "users", "export", "tenant"},
	{"users:freeze:tenant", "Freeze Users", "Emergency freeze/unfreeze user accounts", "users", "freeze", "tenant"},

	// ═══════════════════════════════════════════
	// ROLES
	// ═══════════════════════════════════════════
	{"roles:read:tenant", "View Roles", "View roles and permissions", "roles", "read", "tenant"},
	{"roles:create:tenant", "Create Roles", "Create new custom roles", "roles", "create", "tenant"},
	{"roles:update:tenant", "Update Roles", "Edit role permissions", "roles", "update", "tenant"},
	{"roles:delete:tenant", "Delete Roles", "Delete custom roles", "roles", "delete", "tenant"},
	{"roles:assign:tenant", "Assign Roles to Users", "Assign roles to users", "roles", "assign", "tenant"},

	// ═══════════════════════════════════════════
	// OAUTH CLIENTS
	// ═══════════════════════════════════════════
	{"oauth_clients:read:tenant", "View OAuth Clients", "View registered OAuth/OIDC clients", "oauth_clients", "read", "tenant"},
	{"oauth_clients:create:tenant", "Create OAuth Clients", "Register new OAuth clients", "oauth_clients", "create", "tenant"},
	{"oauth_clients:update:tenant", "Update OAuth Clients", "Edit client config (redirect URIs, grants)", "oauth_clients", "update", "tenant"},
	{"oauth_clients:delete:tenant", "Delete OAuth Clients", "Remove OAuth client registrations", "oauth_clients", "delete", "tenant"},
	{"oauth_clients:rotate_secret:tenant", "Rotate Client Secret", "Generate new client secret", "oauth_clients", "rotate_secret", "tenant"},

	// ═══════════════════════════════════════════
	// ORGANIZATIONS
	// ═══════════════════════════════════════════
	{"orgs:read:tenant", "View Organizations", "View org structure (departments, teams)", "orgs", "read", "tenant"},
	{"orgs:create:tenant", "Create Organizations", "Create departments and teams", "orgs", "create", "tenant"},
	{"orgs:update:tenant", "Update Organizations", "Edit org structure", "orgs", "update", "tenant"},
	{"orgs:delete:tenant", "Delete Organizations", "Remove departments/teams", "orgs", "delete", "tenant"},

	// ═══════════════════════════════════════════
	// POLICIES (Conditional Access)
	// ═══════════════════════════════════════════
	{"policies:read:tenant", "View Policies", "View conditional access policies", "policies", "read", "tenant"},
	{"policies:create:tenant", "Create Policies", "Create new CAE policies", "policies", "create", "tenant"},
	{"policies:update:tenant", "Update Policies", "Edit CAE policy conditions", "policies", "update", "tenant"},
	{"policies:delete:tenant", "Delete Policies", "Remove CAE policies", "policies", "delete", "tenant"},

	// ═══════════════════════════════════════════
	// AUDIT
	// ═══════════════════════════════════════════
	{"audit:read:tenant", "View Audit Logs", "View audit events and logs", "audit", "read", "tenant"},
	{"audit:export:tenant", "Export Audit Data", "Export audit logs (CSV/JSON)", "audit", "export", "tenant"},
	{"audit:read_integrity:tenant", "View Audit Integrity", "Check hash chain integrity", "audit", "read_integrity", "tenant"},

	// ═══════════════════════════════════════════
	// WEBHOOKS
	// ═══════════════════════════════════════════
	{"webhooks:read:tenant", "View Webhooks", "View webhook configurations", "webhooks", "read", "tenant"},
	{"webhooks:create:tenant", "Create Webhooks", "Register new webhook endpoints", "webhooks", "create", "tenant"},
	{"webhooks:update:tenant", "Update Webhooks", "Edit webhook config", "webhooks", "update", "tenant"},
	{"webhooks:delete:tenant", "Delete Webhooks", "Remove webhook endpoints", "webhooks", "delete", "tenant"},

	// ═══════════════════════════════════════════
	// API KEYS
	// ═══════════════════════════════════════════
	{"api_keys:read:tenant", "View API Keys", "View API key configurations", "api_keys", "read", "tenant"},
	{"api_keys:create:tenant", "Create API Keys", "Generate new API keys", "api_keys", "create", "tenant"},
	{"api_keys:delete:tenant", "Revoke API Keys", "Revoke API keys", "api_keys", "delete", "tenant"},

	// ═══════════════════════════════════════════
	// SECURITY
	// ═══════════════════════════════════════════
	{"security:read:tenant", "View Security Dashboard", "View security posture, risk scores, CAE monitor", "security", "read", "tenant"},
	{"security:update:tenant", "Configure Security", "Edit security policies, MFA enforcement", "security", "update", "tenant"},
	{"security:posture:read:tenant", "View Security Posture", "View zero-trust posture scores", "security", "posture", "read"},

	// ═══════════════════════════════════════════
	// SETTINGS
	// ═══════════════════════════════════════════
	{"settings:read:tenant", "View Settings", "View system settings", "settings", "read", "tenant"},
	{"settings:update:tenant", "Update Settings", "Edit system settings (branding, password policy)", "settings", "update", "tenant"},
	{"settings:feature_flags:write:tenant", "Manage Feature Flags", "Enable/disable feature flags", "settings", "feature_flags", "tenant"},

	// ═══════════════════════════════════════════
	// IDENTITY (Dashboard, SCIM, LDAP)
	// ═══════════════════════════════════════════
	{"identity:read:tenant", "View Identity Dashboard", "View identity metrics and joiner dashboard", "identity", "read", "tenant"},
	{"identity:write:tenant", "Manage Identity", "Import users, manage attribute mappings", "identity", "write", "tenant"},

	// ═══════════════════════════════════════════
	// TENANTS (Platform Admin only)
	// ═══════════════════════════════════════════
	{"tenants:read:all", "View Tenants", "View all tenants (platform)", "tenants", "read", "all"},
	{"tenants:create:all", "Create Tenants", "Create new tenants (platform)", "tenants", "create", "all"},
	{"tenants:update:all", "Update Tenants", "Edit tenant config (platform)", "tenants", "update", "all"},
	{"tenants:delete:all", "Delete Tenants", "Remove tenants (platform)", "tenants", "delete", "all"},
	{"tenants:suspend:all", "Suspend Tenants", "Suspend/activate tenants (platform)", "tenants", "suspend", "all"},

	// ═══════════════════════════════════════════
	// SYSTEM ADMIN
	// ═══════════════════════════════════════════
	{"system:admin", "System Admin", "Full system access (bypass all checks)", "system", "admin", "all"},
	{"system:read:all", "View System Status", "View system health and configuration", "system", "read", "all"},
	{"system:impersonate:tenant", "Impersonate Users", "Impersonate other users for support", "system", "impersonate", "tenant"},
	{"system:key_rotation:tenant", "Manage Key Rotation", "Manage signing key rotation", "system", "key_rotation", "tenant"},
	{"system:secrets:tenant", "Manage Secrets", "View/manage system secrets", "system", "secrets", "tenant"},
	{"system:backup:tenant", "Manage Backups", "Manage system backups", "system", "backup", "tenant"},

	// ═══════════════════════════════════════════
	// DASHBOARD (self-service)
	// ═══════════════════════════════════════════
	{"dashboard:read:all", "View Dashboard", "View main dashboard", "dashboard", "read", "all"},

	// ═══════════════════════════════════════════
	// SESSIONS
	// ═══════════════════════════════════════════
	{"sessions:read:own", "View Own Sessions", "View own active sessions", "sessions", "read", "own"},
	{"sessions:revoke:own", "Revoke Own Sessions", "Revoke own sessions", "sessions", "revoke", "own"},
	{"sessions:read:tenant", "View All Sessions", "View sessions for all tenant users", "sessions", "read", "tenant"},
	{"sessions:revoke:tenant", "Revoke User Sessions", "Force-revoke sessions of other users", "sessions", "revoke", "tenant"},

	// ═══════════════════════════════════════════
	// WEBAUTHN / PASSKEY (self-service)
	// ═══════════════════════════════════════════
	{"webauthn:manage:own", "Manage Own Passkeys", "Register/delete own passkeys", "webauthn", "manage", "own"},

	// ═══════════════════════════════════════════
	// ACCESS REVIEWS
	// ═══════════════════════════════════════════
	{"access_reviews:read:tenant", "View Access Reviews", "View access review schedules and results", "access_reviews", "read", "tenant"},
	{"access_reviews:approve:tenant", "Approve Access Reviews", "Approve/reject access review findings", "access_reviews", "approve", "tenant"},
}

// EnsureSystemPermissions syncs system permissions to the database.
// - Inserts new permissions (added in this release)
// - Marks them as system_perm=true (immutable)
// - Does NOT delete permissions removed from code (backward compat)
// - Does NOT modify existing permission keys (only adds new ones)
func EnsureSystemPermissions(ctx context.Context, pool PoolExecutor) error {
	if pool == nil {
		slog.Warn("EnsureSystemPermissions: pool is nil, skipping")
		return nil
	}

	inserted := 0
	for _, sp := range SystemPermissions {
		// Use UPSERT: insert if not exists, do nothing if already present
		// system_perm is always set to true for code-defined permissions
		tag, err := pool.Exec(ctx, `
			INSERT INTO permissions (id, tenant_id, key, name, resource_type, action, description, system_perm)
			VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000000', $1, $2, $3, $4, $5, true)
			ON CONFLICT (key) DO UPDATE SET
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				system_perm = true
		`, sp.Key, sp.Name, sp.Resource, sp.Action+":"+sp.Scope, sp.Description)
		if err != nil {
			slog.Error("EnsureSystemPermissions: failed to upsert", "key", sp.Key, "error", err)
			continue
		}
		if tag.RowsAffected() > 0 {
			inserted++
		}
	}

	if inserted > 0 {
		slog.Info("System permissions synced", "total", len(SystemPermissions), "new_or_updated", inserted)
	}
	return nil
}

// PoolExecutor is the minimal DB interface needed by EnsureSystemPermissions.
type PoolExecutor interface {
	Exec(ctx context.Context, sql string, args ...any) (CommandTag, error)
}

// CommandTag abstracts pgconn.CommandTag for testability.
type CommandTag interface {
	RowsAffected() int64
}
