package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// POST /api/v1/oauth/token/downscope
func handleTokenDownscope(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		SourceToken      string   `json:"source_token"`
		RequestedScopes  []string `json:"requested_scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	if req.SourceToken == "" || len(req.RequestedScopes) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "source_token and requested_scopes required"})
		return
	}

	// In production: verify source token, validate requested scopes are subset, issue new token
	originalScopes := []string{"openid", "profile", "profile.email", "audit.read", "admin.users"}
	validScopes := []string{}
	for _, s := range req.RequestedScopes {
		for _, orig := range originalScopes {
			if s == orig {
				validScopes = append(validScopes, s)
				break
			}
		}
	}

	if len(validScopes) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "requested scopes not subset of source token scopes"})
		return
	}

	newToken := uuid.New().String() + uuid.New().String()
	expiresAt := time.Now().UTC().Add(3600 * time.Second)

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":      newToken,
		"token_type":        "Bearer",
		"expires_in":        3600,
		"expires_at":        expiresAt.Format(time.RFC3339),
		"scope":             validScopes,
		"parent_token_id":   req.SourceToken[:8] + "****",
		"downscoped":        true,
	})
}
