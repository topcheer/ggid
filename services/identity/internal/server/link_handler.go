package server

import (
	"encoding/json"
	"net/http"
)

// POST /api/v1/users/link — link external provider to local user
func (h *HTTPHandler) handleLinkAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		UserID     string `json:"user_id"`
		Provider   string `json:"provider"`
		ExternalID string `json:"external_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Provider == "" || req.ExternalID == "" {
		writeError(w, http.StatusBadRequest, "provider and external_id required")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"linked":     true,
		"provider":   req.Provider,
		"external_id": req.ExternalID,
	})
}

// POST /api/v1/users/unlink — unlink external provider
func (h *HTTPHandler) handleUnlinkAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		UserID   string `json:"user_id"`
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"unlinked":  true,
		"provider":  req.Provider,
	})
}
