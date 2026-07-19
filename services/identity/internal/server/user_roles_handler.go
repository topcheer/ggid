package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// UserRoleAssignment represents a role assigned to a user.
type UserRoleAssignment struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	RoleID     string    `json:"role_id"`
	RoleName   string    `json:"role_name"`
	AssignedAt time.Time `json:"assigned_at"`
	AssignedBy string    `json:"assigned_by"`
}

// GET /api/v1/users/{id}/roles — list roles for a user (from DB)
// POST /api/v1/users/{id}/roles — assign a role to a user (writes to DB)
// DELETE /api/v1/users/{id}/roles/{roleId} — revoke a role (deletes from DB)
func (h *HTTPHandler) handleUserRoles(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	parts := splitUserPath(r.URL.Path)
	pool := h.svc.Pool()

	switch r.Method {
	case http.MethodGet:
		if pool == nil {
			writeJSON(w, http.StatusOK, map[string]any{"roles": []UserRoleAssignment{}})
			return
		}
		rows, err := pool.Query(ctx, `
			SELECT ur.role_id::text, COALESCE(r.key, r.name, ur.role_id::text), ur.created_at, COALESCE(ur.granted_by::text, '')
			FROM user_roles ur
			LEFT JOIN roles r ON r.id = ur.role_id
			WHERE ur.user_id = $1
			ORDER BY ur.created_at DESC
		`, userID)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"roles": []UserRoleAssignment{}})
			return
		}
		defer rows.Close()

		roles := []UserRoleAssignment{}
		for rows.Next() {
			var a UserRoleAssignment
			if err := rows.Scan(&a.RoleID, &a.RoleName, &a.AssignedAt, &a.AssignedBy); err != nil {
				continue
			}
			roles = append(roles, a)
		}
		writeJSON(w, http.StatusOK, map[string]any{"roles": roles})

	case http.MethodPost:
		var req struct {
			RoleID   string `json:"role_id"`
			RoleName string `json:"role_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.RoleID == "" {
			writeJSONError(w, http.StatusBadRequest, "role_id is required")
			return
		}

		roleUUID, err := uuid.Parse(req.RoleID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid role_id: must be a valid UUID")
			return
		}

		if pool == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "database not available")
			return
		}

		// Get role name from roles table if not provided
		if req.RoleName == "" {
			_ = pool.QueryRow(ctx, `SELECT name FROM roles WHERE id = $1`, roleUUID).Scan(&req.RoleName)
		}

		assignment := UserRoleAssignment{
			ID:         uuid.New(),
			UserID:     userID,
			RoleID:     req.RoleID,
			RoleName:   req.RoleName,
			AssignedAt: time.Now(),
		}

		// Insert into user_roles table (ON CONFLICT DO NOTHING for idempotency)
		_, err = pool.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, granted_by)
			VALUES ($1, $2, 'tenant', NULL, $3)
			ON CONFLICT DO NOTHING
		`, userID, roleUUID, uuid.Nil)
		if err != nil {
			// Check if it's a duplicate
			writeJSON(w, http.StatusCreated, assignment)
			return
		}
		writeJSON(w, http.StatusCreated, assignment)

	case http.MethodDelete:
		if len(parts) < 6 {
			writeJSONError(w, http.StatusBadRequest, "role ID is required")
			return
		}
		roleID := parts[5]
		roleUUID, err := uuid.Parse(roleID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid role_id")
			return
		}

		if pool == nil {
			writeJSONError(w, http.StatusNotFound, "role assignment not found")
			return
		}

		cmd, err := pool.Exec(ctx, `
			DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2
		`, userID, roleUUID)
		if err != nil || cmd.RowsAffected() == 0 {
			writeJSONError(w, http.StatusNotFound, "role assignment not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// splitUserPath splits a URL path into non-empty segments.
func splitUserPath(path string) []string {
	var parts []string
	cur := ""
	for _, c := range path {
		if c == '/' {
			if cur != "" {
				parts = append(parts, cur)
				cur = ""
			}
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}
