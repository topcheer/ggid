package httpserver

import (
	"net/http"
	"strings"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
)

// GET /api/v1/policies/standing-access
// Queries real role assignments to identify standing (non-JIT) access
// to privileged resources. Returns roles with admin-level permissions
// that are assigned permanently (not time-bound).
func (s *HTTPServer) handleStandingAccess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}

	// Query real roles from DB
	roles, err := s.roleSvc.ListRoles(r.Context(), tc.TenantID, 1, 100)
	if err != nil || len(roles) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"standing_access": []any{},
			"total":           0,
			"jit_recommended": 0,
			"generated_at":    time.Now().UTC().Format(time.RFC3339),
			"message":         "no roles found",
		})
		return
	}

	var access []map[string]any
	privilegedKeywords := []string{"admin", "write", "delete", "manage", "*"}

	for _, role := range roles {
		// Get permissions for this role
		perms, err := s.roleSvc.GetRolePermissions(r.Context(), role.ID)
		if err != nil {
			continue
		}

		// Identify privileged permissions
		var privilegedPerms []string
		for _, p := range perms {
			resource := strings.ToLower(p.ResourceType)
			action := strings.ToLower(p.Action)
			for _, kw := range privilegedKeywords {
				if strings.Contains(resource, kw) || strings.Contains(action, kw) || action == "*" {
					privilegedPerms = append(privilegedPerms, p.ResourceType+":"+p.Action)
					break
				}
			}
		}

		if len(privilegedPerms) > 0 {
			access = append(access, map[string]any{
				"role_id":         role.ID.String(),
				"role_name":       role.Name,
				"role_key":        role.Key,
				"access_type":     "standing",
				"duration":        "permanent",
				"privileged_perms": privilegedPerms,
				"perm_count":      len(privilegedPerms),
				"recommendation":  "convert_to_jit",
				"max_duration":    "4h",
				"approval_required": true,
			})
		}
	}

	if access == nil {
		access = []map[string]any{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"standing_access": access,
		"total":           len(access),
		"jit_recommended": len(access),
		"generated_at":    time.Now().UTC().Format(time.RFC3339),
	})
}
