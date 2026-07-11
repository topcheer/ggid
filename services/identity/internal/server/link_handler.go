package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
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

// POST /api/v1/users/import/validate — pre-check CSV/JSON import data
func (h *HTTPHandler) handleImportValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Users []struct {
			Username string `json:"username"`
			Email    string `json:"email"`
		} `json:"users"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	valid, invalid := 0, 0
	var errors []map[string]string
	for i, u := range req.Users {
		if u.Username == "" {
			invalid++
			errors = append(errors, map[string]string{"row": strconv.Itoa(i), "error": "username required"})
			continue
		}
		if u.Email == "" || !strings.Contains(u.Email, "@") {
			invalid++
			errors = append(errors, map[string]string{"row": strconv.Itoa(i), "error": "invalid email"})
			continue
		}
		valid++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"valid_count":   valid,
		"invalid_count": invalid,
		"errors":        errors,
	})
}

// POST /api/v1/users/bulk/status — batch update user status
func (h *HTTPHandler) handleBulkStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		UserIDs []string `json:"user_ids"`
		Status  string   `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Status != "active" && req.Status != "inactive" && req.Status != "locked" && req.Status != "suspended" {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success_count": len(req.UserIDs),
		"failures":      []any{},
		"status":        req.Status,
	})
}
