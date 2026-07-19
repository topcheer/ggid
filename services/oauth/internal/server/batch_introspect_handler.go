package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/service"
)

type BatchIntrospectionResult struct {
	TokenID string `json:"token_id"`
	Active  bool   `json:"active"`
	Scope   string `json:"scope,omitempty"`
	Exp     int64  `json:"exp,omitempty"`
	UserID  string `json:"user_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// POST /api/v1/oauth/introspect/batch
// Introspects multiple tokens in a single request using the real OAuthService.
func handleBatchIntrospect(w http.ResponseWriter, r *http.Request, svc *service.OAuthService) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	var req struct {
		TokenIDs []string `json:"token_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	if len(req.TokenIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "token_ids required"})
		return
	}

	results := make([]BatchIntrospectionResult, len(req.TokenIDs))
	activeCount := 0

	if svc != nil {
		for i, tokenStr := range req.TokenIDs {
			resp := svc.IntrospectToken(tokenStr)
			result := BatchIntrospectionResult{
				TokenID: tokenStr,
				Active:  resp.Active,
			}
			if resp.Active {
				activeCount++
				result.Scope = resp.Scope
				result.Exp = resp.Exp
				result.UserID = resp.UserID
			} else {
				result.Error = "token inactive or invalid"
			}
			results[i] = result
		}
	} else {
		for i, id := range req.TokenIDs {
			results[i] = BatchIntrospectionResult{
				TokenID: id,
				Active:  false,
				Error:   "introspection service not configured",
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"results":         results,
		"total":           len(results),
		"active":          activeCount,
		"inactive":        len(results) - activeCount,
		"introspected_at": time.Now().UTC().Format(time.RFC3339),
	})
}
