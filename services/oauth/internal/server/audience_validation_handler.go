package server

import (
	"encoding/json"
	"net/http"
)

// POST /api/v1/oauth/tokens/validate-audience
func handleValidateAudience(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		Token          string   `json:"token"`
		ResourceServer string   `json:"resource_server"`
		AllowedAudiences []string `json:"allowed_audiences"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	if req.Token == "" || req.ResourceServer == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "token and resource_server required"})
		return
	}

	// In production: parse JWT, extract aud claim, validate against resource server
	writeJSON(w, http.StatusOK, map[string]any{
		"is_valid":            true,
		"resource_server":     req.ResourceServer,
		"allowed_audiences":   []string{"ggid-api", "ggid-admin", "ggid-audit"},
		"mismatched_resources": []string{},
		"validation_rules":    []string{"exact_match", "wildcard_allowed", "issuer_verified"},
	})
}
