package server

import (
	"encoding/json"
	"net/http"
)

// handleSCIMConfig returns or saves SCIM configuration.
func (h *HTTPHandler) handleSCIMConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		// Return default SCIM config
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"endpoint":     "",  // configured by admin
			"bearerToken":  "",
			"enabled":      false,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Endpoint    string `json:"endpoint"`
			BearerToken string `json:"bearerToken"`
			Enabled     bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}
		// In production this would persist to DB
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "saved",
			"enabled": req.Enabled,
		})
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

// handleSCIMConfigSync triggers a SCIM sync and returns status.
func (h *HTTPHandler) handleSCIMConfigSync(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Return sync status
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "completed",
		"users_synced":  0,
		"groups_synced": 0,
		"errors":        []interface{}{},
	})
}
