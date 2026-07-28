package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/services/oauth/internal/service"
	"github.com/google/uuid"
)

// POST /api/v1/oauth/token/downscope — RFC 8693 token exchange (downscope)
// SECURITY: verifies the source JWT and ensures requested scopes are a subset.
func handleTokenDownscope(oauthSvc *service.OAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}

		// Require authentication — the caller must present a valid Bearer token.
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authentication required"})
			return
		}

		var req struct {
			SourceToken     string   `json:"source_token"`
			RequestedScopes []string `json:"requested_scopes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
			return
		}
		if req.SourceToken == "" || len(req.RequestedScopes) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "source_token and requested_scopes required"})
			return
		}

		// Verify the source token is a valid JWT issued by this service.
		claims, err := oauthSvc.ParseAccessToken(req.SourceToken)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid source token"})
			return
		}

		// Extract the source token's scopes.
		sourceScopeStr, _ := claims["scope"].(string)
		sourceScopes := strings.Fields(sourceScopeStr)
		if len(sourceScopes) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "source token has no scopes to downscope"})
			return
		}
		sourceSet := make(map[string]bool, len(sourceScopes))
		for _, s := range sourceScopes {
			sourceSet[s] = true
		}

		// Ensure requested scopes are a subset of source scopes.
		validScopes := []string{}
		for _, s := range req.RequestedScopes {
			if sourceSet[s] {
				validScopes = append(validScopes, s)
			}
		}
		if len(validScopes) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "requested scopes not subset of source token scopes"})
			return
		}

		// Issue a real downscoped JWT (not a random UUID).
		sub, _ := claims["sub"].(string)
		tenantIDStr, _ := claims["tenant_id"].(string)
		aud, _ := claims["aud"].(string)

		subUUID, err := uuid.Parse(sub)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid subject UUID in token"})
			return
		}
		tenantUUID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid tenant_id in token"})
			return
		}
		subToken, expiresIn, err := oauthSvc.DownscopeToken(
			subUUID,
			tenantUUID,
			aud,
			strings.Join(validScopes, " "),
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to issue downscoped token"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"access_token": subToken,
			"token_type":   "Bearer",
			"expires_in":   expiresIn,
			"scope":        validScopes,
			"downscoped":   true,
		})
	}
}
