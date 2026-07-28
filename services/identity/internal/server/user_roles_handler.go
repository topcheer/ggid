package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// UserRoleAssignment represents a role assigned to a user.
	type UserRoleAssignment struct {
		ID         string    `json:"id"`
		UserID     string    `json:"user_id"`
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

	// SECURITY: extract caller's tenant for cross-tenant protection.
	callerTenantID := r.Header.Get("X-Tenant-ID")

	switch r.Method {
	case http.MethodGet:
		if pool == nil {
			writeJSON(w, http.StatusOK, map[string]any{"roles": []UserRoleAssignment{}})
			return
		}
		// SECURITY: filter by caller's tenant to prevent cross-tenant role access.
		var rows pgx.Rows
		var err error
		if callerTenantID != "" {
			rows, err = pool.Query(ctx, `
				SELECT ur.user_id::text, ur.role_id::text, COALESCE(r.name, r.key, ur.role_id::text), ur.created_at, COALESCE(ur.granted_by::text, '')
				FROM user_roles ur
				LEFT JOIN roles r ON r.id = ur.role_id
				WHERE ur.user_id = $1 AND r.tenant_id = $2
				ORDER BY ur.created_at DESC
			`, userID, callerTenantID)
		} else {
			rows, err = pool.Query(ctx, `
				SELECT ur.user_id::text, ur.role_id::text, COALESCE(r.name, r.key, ur.role_id::text), ur.created_at, COALESCE(ur.granted_by::text, '')
				FROM user_roles ur
				LEFT JOIN roles r ON r.id = ur.role_id
				WHERE ur.user_id = $1
				ORDER BY ur.created_at DESC
			`, userID)
		}
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"roles": []UserRoleAssignment{}})
			return
		}
		defer rows.Close()

		roles := []UserRoleAssignment{}
		for rows.Next() {
			var a UserRoleAssignment
			if err := rows.Scan(&a.UserID, &a.RoleID, &a.RoleName, &a.AssignedAt, &a.AssignedBy); err != nil {
				continue
			}
			a.ID = a.UserID + "/" + a.RoleID
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

		// Get role name and tenant from roles table if not provided
		roleTenant := uuid.Nil
		// SECURITY: verify the role belongs to the caller's tenant.
		if callerTenantID != "" {
			callerTID, _ := uuid.Parse(callerTenantID)
			_ = pool.QueryRow(ctx, `SELECT name, tenant_id FROM roles WHERE id = $1 AND tenant_id = $2`, roleUUID, callerTID).Scan(&req.RoleName, &roleTenant)
			if roleTenant == uuid.Nil {
				writeJSONError(w, http.StatusNotFound, "role not found in your tenant")
				return
			}
		} else {
			_ = pool.QueryRow(ctx, `SELECT name, tenant_id FROM roles WHERE id = $1`, roleUUID).Scan(&req.RoleName, &roleTenant)
		}
		if roleTenant == uuid.Nil {
			roleTenant = defaultTenantID()
		}

		// Get authenticated user ID from gateway header
		grantedByStr := r.Header.Get("X-User-ID")
		grantedBy, err := uuid.Parse(grantedByStr)
		if err != nil {
			grantedBy = uuid.Nil
		}

		assignment := UserRoleAssignment{
			ID:         uuid.NewString(),
			UserID:     userID.String(),
			RoleID:     req.RoleID,
			RoleName:   req.RoleName,
			AssignedAt: time.Now(),
			AssignedBy: grantedByStr,
		}

		// Insert into user_roles table (ON CONFLICT DO NOTHING for idempotency)
		_, err = pool.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, granted_by)
			VALUES ($1, $2, 'global', $3, $4)
			ON CONFLICT DO NOTHING
		`, userID, roleUUID, roleTenant, grantedBy)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to assign role: "+err.Error())
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

		// SECURITY: only delete roles within the caller's tenant.
		var cmd pgx.CommandTag
		var err2 error
		if callerTenantID != "" {
			cmd, err2 = pool.Exec(ctx, `
				DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2
				AND role_id IN (SELECT id FROM roles WHERE tenant_id = $3)
			`, userID, roleUUID, callerTenantID)
		} else {
			cmd, err2 = pool.Exec(ctx, `
				DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2
			`, userID, roleUUID)
		}
		if err2 != nil || cmd.RowsAffected() == 0 {
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
