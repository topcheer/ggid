package server

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// GET /api/v1/users/{id}/management-chain
func (h *HTTPHandler) handleManagementChain(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": userID.String(),
		"chain": []map[string]any{
			{"level": 1, "user_id": "u-mgr", "name": "Jane Smith", "title": "Engineering Manager", "email": "jane@example.com"},
			{"level": 2, "user_id": "u-dir", "name": "Bob Lee", "title": "Director of Engineering", "email": "bob@example.com"},
			{"level": 3, "user_id": "u-vp", "name": "Sarah Chen", "title": "VP Engineering", "email": "sarah@example.com"},
			{"level": 4, "user_id": "u-cto", "name": "Mike Wong", "title": "CTO", "email": "mike@example.com"},
		},
		"depth": 4,
		"next_approver": "u-mgr",
		"escalation_path": "manager → director → VP → CTO",
	})
}
