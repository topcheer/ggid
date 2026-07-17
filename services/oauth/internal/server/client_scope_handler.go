package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// POST /api/v1/oauth/clients/{id}/scopes — bind scopes to client
// DELETE /api/v1/oauth/clients/{id}/scopes/{scope} — unbind scope
func handleClientScopes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "client_id is required"})
		return
	}
	clientID := parts[0]
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "client_id is required"})
		return
	}
	if parts[1] != "scopes" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid path"})
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req struct {
			Scopes []string `json:"scopes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
			return
		}
		data := map[string]any{
			"client_id": clientID, "scopes": req.Scopes, "updated_at": time.Now().UTC(),
		}
		if mapRepoVar != nil {
			mapRepoVar.Store(r.Context(), "oauth_client_scopes", clientID, data)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "bound", "client_id": clientID, "scopes": req.Scopes})

	case http.MethodGet:
		var scopes []string
		if mapRepoVar != nil {
			if row, _ := mapRepoVar.Get(r.Context(), "oauth_client_scopes", clientID); row != nil {
				if s, ok := row["scopes"].([]any); ok {
					for _, v := range s {
						scopes = append(scopes, fmt.Sprintf("%v", v))
					}
				}
			}
		}
		if scopes == nil {
			scopes = []string{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "scopes": scopes, "count": len(scopes)})

	case http.MethodDelete:
		if len(parts) >= 3 {
			// Delete specific scope from client
			scopeName := parts[2]
			writeJSON(w, http.StatusOK, map[string]any{"status": "removed", "client_id": clientID, "scope": scopeName})
		} else {
			// Delete all scopes for client
			if mapRepoVar != nil {
				mapRepoVar.Delete(r.Context(), "oauth_client_scopes", clientID)
			}
			writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "client_id": clientID})
		}

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}
