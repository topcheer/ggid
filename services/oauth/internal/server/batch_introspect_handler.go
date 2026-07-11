package server

import (
	"encoding/json"
	"net/http"
	"time"
)

type BatchIntrospectionResult struct {
	TokenID string                 `json:"token_id"`
	Active  bool                   `json:"active"`
	Scope   string                 `json:"scope,omitempty"`
	Exp     int64                  `json:"exp,omitempty"`
	UserID  string                 `json:"user_id,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// POST /api/v1/oauth/introspect/batch
func handleBatchIntrospect(w http.ResponseWriter, r *http.Request) {
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
	for i, id := range req.TokenIDs {
		// In production: query token store for each token
		// For now return sample introspection data
		results[i] = BatchIntrospectionResult{
			TokenID: id,
			Active:  i%4 != 0, // 75% active
			Scope:   "openid profile",
			Exp:     time.Now().Unix() + 3600,
		}
		if results[i].Active {
			activeCount++
		} else {
			results[i].Error = "token expired"
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"results":      results,
		"total":        len(results),
		"active":       activeCount,
		"inactive":     len(results) - activeCount,
		"introspected_at": time.Now().UTC().Format(time.RFC3339),
	})
}
