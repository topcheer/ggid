package httpserver

import (
	"encoding/json"
	"net/http"
	"context"

	"github.com/ggid/ggid/services/policy/internal/service"
	"github.com/google/uuid"
)

// POST /api/v1/policies/sod/check — check if user's roles violate SoD rules
func (s *HTTPServer) handleSoDCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		UserID string   `json:"user_id"`
		Roles  []string `json:"roles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	uid, _ := uuid.Parse(req.UserID)
	violations := service.CheckSoD(context.Background(), uid, req.Roles)
	writeJSON(w, http.StatusOK, map[string]any{
		"violations": violations,
		"violated":   len(violations) > 0,
	})
}
