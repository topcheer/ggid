package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ggid/ggid/services/policy/internal/repository"
	"github.com/google/uuid"
)

// POST /api/v1/policies/jit/elevate — request just-in-time privilege elevation.
// SECURITY: Always requires approval — no-approval mode is dangerous and disabled.
func (s *HTTPServer) handleJITElevate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// SECURITY: require tenant context.
	tid, ok := requireTenantHeader(w, r)
	if !ok {
		return
	}
	tenantID := uuid.MustParse(tid)

	// SECURITY: require authenticated user.
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		writeJSONError(w, http.StatusForbidden, "authentication required")
		return
	}
	userID := uuid.MustParse(userIDStr)

	var req struct {
		UserID           string `json:"user_id"`
		RequestedRole    string `json:"requested_role"`
		Duration         string `json:"duration"`
		Justification    string `json:"justification"`
		ApprovalRequired bool   `json:"approval_required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.RequestedRole == "" {
		writeJSONError(w, http.StatusBadRequest, "requested_role required")
		return
	}

	// SECURITY: Use the authenticated caller's user ID, not a body-supplied one.
	req.UserID = userID.String()

	dur, _ := time.ParseDuration(req.Duration)
	if dur == 0 {
		dur = 4 * time.Hour
	}
	// Cap duration to prevent excessive elevation windows.
	if dur > 8*time.Hour {
		dur = 8 * time.Hour
	}
	expiresAt := time.Now().UTC().Add(dur)

	// SECURITY: Always require approval — no-approval mode bypasses PAM controls.
	// Create a pending JIT request in the database.
	jitReq := &repository.JITRequest{
		ID:          uuid.New(),
		TenantID:    tenantID,
		UserID:      userID,
		Reason:      req.RequestedRole + ": " + req.Justification,
		DurationMin: int(dur.Minutes()),
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if s.jitRepo != nil {
		if err := s.jitRepo.Create(r.Context(), jitReq); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create JIT request")
			return
		}
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":         "pending_approval",
		"request_id":     jitReq.ID.String(),
		"requested_role": req.RequestedRole,
		"duration":       dur.String(),
		"justification":  req.Justification,
		"expires_at":     expiresAt.Format(time.RFC3339),
	})
}
