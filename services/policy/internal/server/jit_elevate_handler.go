package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// POST /api/v1/policies/jit-elevate
func (s *HTTPServer) handleJITElevate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	var req struct {
		UserID          string `json:"user_id"`
		RequestedRole   string `json:"requested_role"`
		Duration        string `json:"duration"`
		Justification   string `json:"justification"`
		ApprovalRequired bool  `json:"approval_required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSONError(w, http.StatusBadRequest, "invalid JSON"); return }
	if req.UserID == "" || req.RequestedRole == "" { writeJSONError(w, http.StatusBadRequest, "user_id and requested_role required"); return }
	dur, _ := time.ParseDuration(req.Duration)
	if dur == 0 { dur = 4 * time.Hour }
	expiresAt := time.Now().UTC().Add(dur)
	if req.ApprovalRequired {
		writeJSON(w, http.StatusAccepted, map[string]any{"status": "pending_approval", "request_id": "jit-" + uuid.New().String()[:8], "requested_role": req.RequestedRole, "duration": dur.String(), "justification": req.Justification, "expires_at": expiresAt.Format(time.RFC3339)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "elevated", "user_id": req.UserID, "elevated_role": req.RequestedRole, "elevated_token": uuid.New().String() + uuid.New().String(), "expires_at": expiresAt.Format(time.RFC3339), "duration": dur.String(), "justification": req.Justification})
}
