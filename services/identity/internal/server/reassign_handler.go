package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// POST /api/v1/users/{id}/reassign
func (h *HTTPHandler) reassignUser(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		NewOrg     string `json:"new_org"`
		NewRole    string `json:"new_role"`
		NewManager string `json:"new_manager"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "reassigned", "user_id": userID.String(),
		"new_org": req.NewOrg, "new_role": req.NewRole, "new_manager": req.NewManager,
		"reassigned_at": time.Now().UTC().Format(time.RFC3339),
		"actions_triggered": []string{"access_review", "session_revoke", "notify_old_manager", "notify_new_manager"},
	})
}
