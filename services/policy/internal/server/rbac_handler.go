package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleRolesSubpath dispatches /api/v1/roles/{id} and /api/v1/roles/{id}/permissions
func (s *HTTPServer) handleRolesSubpath(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// /api/v1/roles/{id}/permissions
	if strings.HasSuffix(path, "/permissions") {
		s.handleRoleRoutePermissions(w, r)
		return
	}
	// /api/v1/roles/{id}
	s.handleRoleByID(w, r)
}

// RoleRoutePermission represents a single route permission for a role.
type RoleRoutePermission struct {
	RoleID          string `json:"role_id"`
	RoutePrefix     string `json:"route_prefix"`
	PermissionLevel string `json:"permission_level"` // read/write/admin
}

// System roles whose permissions cannot be modified.
var systemRoles = map[string]bool{
	"platform:admin": true,
	"tenant:admin":   true,
	"tenant:auditor": true,
	"user:self":      true,
}

// GET /api/v1/roles/{id}/permissions — list route permissions for a role
// PUT /api/v1/roles/{id}/permissions — replace all route permissions (overwrite)
func (s *HTTPServer) handleRoleRoutePermissions(w http.ResponseWriter, r *http.Request) {
	roleID := strings.TrimPrefix(r.URL.Path, "/api/v1/roles/")
	roleID = strings.TrimSuffix(roleID, "/permissions")
	roleID = strings.TrimSpace(roleID)
	if roleID == "" || roleID == "/permissions" {
		writeJSONError(w, http.StatusBadRequest, "role ID is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		if s.pool == nil {
			writeJSON(w, http.StatusOK, []RoleRoutePermission{})
			return
		}
		rows, err := s.pool.Query(r.Context(), `
			SELECT role_id::text, route_prefix, permission_level
			FROM role_route_permissions
			WHERE role_id = $1
			ORDER BY route_prefix
		`, roleID)
		if err != nil {
			writeJSON(w, http.StatusOK, []RoleRoutePermission{})
			return
		}
		defer rows.Close()

		perms := []RoleRoutePermission{}
		for rows.Next() {
			var p RoleRoutePermission
			if err := rows.Scan(&p.RoleID, &p.RoutePrefix, &p.PermissionLevel); err != nil {
				continue
			}
			perms = append(perms, p)
		}
		writeJSON(w, http.StatusOK, map[string]any{"permissions": perms, "total": len(perms)})

	case http.MethodPut:
		if s.pool == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "database not available")
			return
		}

		// Check if role is a system role
		var roleKey string
		_ = s.pool.QueryRow(r.Context(), `SELECT key FROM roles WHERE id = $1`, roleID).Scan(&roleKey)
		if systemRoles[roleKey] {
			writeJSONError(w, http.StatusForbidden, "cannot modify system role permissions")
			return
		}

		var req struct {
			Permissions []struct {
				RoutePrefix     string `json:"route_prefix"`
				PermissionLevel string `json:"permission_level"`
			} `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Validate permission levels
		validLevels := map[string]bool{"read": true, "write": true, "admin": true}
		for _, p := range req.Permissions {
			if !validLevels[p.PermissionLevel] {
				writeJSONError(w, http.StatusBadRequest, "invalid permission_level: "+p.PermissionLevel)
				return
			}
			if p.RoutePrefix == "" {
				writeJSONError(w, http.StatusBadRequest, "route_prefix is required")
				return
			}
		}

		// Overwrite: delete existing, insert new
		tx, err := s.pool.Begin(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to begin transaction")
			return
		}
		defer tx.Rollback(r.Context())

		_, _ = tx.Exec(r.Context(), `DELETE FROM role_route_permissions WHERE role_id = $1`, roleID)
		for _, p := range req.Permissions {
			_, err := tx.Exec(r.Context(), `
				INSERT INTO role_route_permissions (role_id, route_prefix, permission_level)
				VALUES ($1, $2, $3)
				ON CONFLICT (role_id, route_prefix) DO UPDATE SET permission_level = $3
			`, roleID, p.RoutePrefix, p.PermissionLevel)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "failed to set permission")
				return
			}
		}

		if err := tx.Commit(r.Context()); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to commit")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "updated",
			"role_id":     roleID,
			"permissions": req.Permissions,
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleUserMePermissions returns the merged route permissions for the
// authenticated user (union of all assigned roles).
// GET /api/v1/users/me/permissions
func (s *HTTPServer) handleUserMePermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if s.pool == nil {
		// Fallback: admin sees everything, others see profile only
		writeJSON(w, http.StatusOK, map[string]any{
			"permissions": []RoleRoutePermission{
				{RoutePrefix: "/dashboard", PermissionLevel: "read"},
				{RoutePrefix: "/profile", PermissionLevel: "read"},
			},
		})
		return
	}

	// Union of all role permissions for this user.
	// If multiple roles grant different levels for the same route, take the highest.
	rows, err := s.pool.Query(r.Context(), `
		SELECT rrp.route_prefix,
		       CASE
				WHEN MAX(CASE WHEN rrp.permission_level = 'admin' THEN 1 ELSE 0 END) = 1 THEN 'admin'
				WHEN MAX(CASE WHEN rrp.permission_level = 'write' THEN 1 ELSE 0 END) = 1 THEN 'write'
				ELSE 'read'
		       END AS max_level
		FROM role_route_permissions rrp
		JOIN user_roles ur ON ur.role_id = rrp.role_id
		WHERE ur.user_id = $1
		GROUP BY rrp.route_prefix
		ORDER BY rrp.route_prefix
	`, userIDStr)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"permissions": []RoleRoutePermission{}})
		return
	}
	defer rows.Close()

	perms := []RoleRoutePermission{}
	for rows.Next() {
		var p RoleRoutePermission
		p.RoleID = userIDStr
		if err := rows.Scan(&p.RoutePrefix, &p.PermissionLevel); err != nil {
			continue
		}
		perms = append(perms, p)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"permissions": perms,
		"total":       len(perms),
	})
}
