package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// POST /api/v1/policies/delegate — delegate permissions to another user
func (s *HTTPServer) handleDelegate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		DelegatorID  string   `json:"delegator_id"`
		DelegateeID  string   `json:"delegatee_id"`
		Permissions  []string `json:"permissions"`
		MaxDurationH int      `json:"max_duration_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	hours := time.Duration(req.MaxDurationH) * time.Hour
	if hours <= 0 {
		hours = 24 * time.Hour // default 24h
	}
	d, err := s.policySvc.DelegatePermissions(
		context.Background(),
		parseUUIDSafe(req.DelegatorID),
		parseUUIDSafe(req.DelegateeID),
		req.Permissions, hours,
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

// GET /api/v1/policies/delegations?user_id=X — list delegations for a user
func (s *HTTPServer) handleListDelegations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	userID := parseUUIDSafe(r.URL.Query().Get("user_id"))
	delegations, err := s.policySvc.ListDelegations(context.Background(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"delegations": delegations})
}

func parseUUIDSafe(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
