package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

// clientScopeBinding holds allowed scopes per OAuth client.
type clientScopeBinding struct {
	mu      sync.RWMutex
	scopes  map[string]map[string]bool // client_id → set of scope names
}

var clientScopes = &clientScopeBinding{scopes: make(map[string]map[string]bool)}

// POST /api/v1/oauth/clients/{id}/scopes — bind scopes to client
// DELETE /api/v1/oauth/clients/{id}/scopes/{scope} — unbind scope
func handleClientScopes(w http.ResponseWriter, r *http.Request) {
	// Path: /api/v1/oauth/clients/{id}/scopes or /api/v1/oauth/clients/{id}/scopes/{scope}
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

	// parts[1] should be "scopes"
	if parts[1] != "scopes" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid path"})
		return
	}

	// POST: bind scopes
	if r.Method == http.MethodPost {
		var req struct {
			Scopes []string `json:"scopes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
			return
		}

		clientScopes.mu.Lock()
		if clientScopes.scopes[clientID] == nil {
			clientScopes.scopes[clientID] = make(map[string]bool)
		}
		for _, sc := range req.Scopes {
			clientScopes.scopes[clientID][sc] = true
		}
		bound := make([]string, 0, len(clientScopes.scopes[clientID]))
		for s := range clientScopes.scopes[clientID] {
			bound = append(bound, s)
		}
		clientScopes.mu.Unlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "bound",
			"client_id":  clientID,
			"scopes":     bound,
		})
		return
	}

	// DELETE: unbind a specific scope
	if r.Method == http.MethodDelete {
		var scopeToRemove string
		if len(parts) >= 3 {
			scopeToRemove = parts[2]
		}
		if scopeToRemove == "" {
			scopeToRemove = r.URL.Query().Get("scope")
		}
		if scopeToRemove == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "scope is required"})
			return
		}

		clientScopes.mu.Lock()
		if clientScopes.scopes[clientID] == nil {
			clientScopes.mu.Unlock()
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "no scopes bound for this client"})
			return
		}
		if !clientScopes.scopes[clientID][scopeToRemove] {
			clientScopes.mu.Unlock()
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "scope not bound to this client"})
			return
		}
		delete(clientScopes.scopes[clientID], scopeToRemove)
		remaining := make([]string, 0, len(clientScopes.scopes[clientID]))
		for s := range clientScopes.scopes[clientID] {
			remaining = append(remaining, s)
		}
		clientScopes.mu.Unlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "unbound",
			"client_id":  clientID,
			"scope":      scopeToRemove,
			"remaining":  remaining,
		})
		return
	}

	// GET: list bound scopes
	if r.Method == http.MethodGet {
		clientScopes.mu.RLock()
		scopes := clientScopes.scopes[clientID]
		result := make([]string, 0, len(scopes))
		for s := range scopes {
			result = append(result, s)
		}
		clientScopes.mu.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"client_id": clientID,
			"scopes":    result,
		})
		return
	}

	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
}
