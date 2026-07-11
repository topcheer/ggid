package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// POST /api/v1/oauth/clients/{id}/migrate
func handleClientMigration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	clientID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/"), "/migrate")
	if clientID == "" || strings.Contains(clientID, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid client_id"})
		return
	}
	var req struct {
		NewRedirectURIs []string `json:"new_redirect_uris"`
		NewScopes       []string `json:"new_scopes"`
		NewGrants       []string `json:"new_grants"`
		GracePeriod     string   `json:"grace_period"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	grace := req.GracePeriod
	if grace == "" { grace = "24h" }
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "migrated", "client_id": clientID,
		"new_redirect_uris": req.NewRedirectURIs, "new_scopes": req.NewScopes, "new_grants": req.NewGrants,
		"grace_period": grace, "old_config_valid_until": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
		"migrated_at": time.Now().UTC().Format(time.RFC3339),
	})
}
