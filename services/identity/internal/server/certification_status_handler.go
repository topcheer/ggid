package server

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// GET /api/v1/users/{id}/certification-status
func (h *HTTPHandler) handleCertificationStatus(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": userID.String(),
		"last_certified": "2026-04-15T00:00:00Z",
		"next_due": "2026-10-15T00:00:00Z",
		"status": "current",
		"pending_campaigns": []map[string]any{
			{"campaign_id": "camp-q3-2026", "name": "Q3 Access Review", "deadline": "2026-09-30T00:00:00Z", "status": "pending"},
		},
		"expired_certs": []map[string]any{},
		"days_until_due": 94,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}
