package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// UserRoleAssignment represents a role assigned to a user.
type UserRoleAssignment struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	RoleID    string    `json:"role_id"`
	RoleName  string    `json:"role_name"`
	AssignedAt time.Time `json:"assigned_at"`
	AssignedBy string    `json:"assigned_by"`
}

var (
	userRolesMu sync.RWMutex
	userRoles   = map[uuid.UUID][]UserRoleAssignment{} // keyed by user ID
)

// GET /api/v1/users/{id}/roles — list roles for a user
// POST /api/v1/users/{id}/roles — assign a role to a user
// DELETE /api/v1/users/{id}/roles/{roleId} — revoke a role
func (h *HTTPHandler) handleUserRoles(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	// Parse sub-path after "roles"
	parts := splitUserPath(r.URL.Path)
	// parts: ["api", "v1", "users", "{id}", "roles", ...]

	switch r.Method {
	case http.MethodGet:
		userRolesMu.RLock()
		roles := make([]UserRoleAssignment, len(userRoles[userID]))
		copy(roles, userRoles[userID])
		userRolesMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"roles": roles})

	case http.MethodPost:
		var req struct {
			RoleID   string `json:"role_id"`
			RoleName string `json:"role_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.RoleID == "" {
			writeError(w, http.StatusBadRequest, "role_id is required")
			return
		}
		if req.RoleName == "" {
			req.RoleName = req.RoleID
		}
		assignment := UserRoleAssignment{
			ID:        uuid.New(),
			UserID:    userID,
			RoleID:    req.RoleID,
			RoleName:  req.RoleName,
			AssignedAt: time.Now(),
		}
		userRolesMu.Lock()
		// Avoid duplicates
		for _, existing := range userRoles[userID] {
			if existing.RoleID == req.RoleID {
				userRolesMu.Unlock()
				writeError(w, http.StatusConflict, "role already assigned")
				return
			}
		}
		userRoles[userID] = append(userRoles[userID], assignment)
		userRolesMu.Unlock()
		writeJSON(w, http.StatusCreated, assignment)

	case http.MethodDelete:
		// Extract role ID from path: /api/v1/users/{id}/roles/{roleId}
		if len(parts) < 6 {
			writeError(w, http.StatusBadRequest, "role ID is required")
			return
		}
		roleID := parts[5]
		userRolesMu.Lock()
		roles := userRoles[userID]
		found := false
		for i, r := range roles {
			if r.RoleID == roleID {
				userRoles[userID] = append(roles[:i], roles[i+1:]...)
				found = true
				break
			}
		}
		userRolesMu.Unlock()
		if !found {
			writeError(w, http.StatusNotFound, "role assignment not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
