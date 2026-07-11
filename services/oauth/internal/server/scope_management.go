package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CustomScope represents a user-defined OAuth/OIDC scope.
type CustomScope struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Attributes  []string  `json:"attributes,omitempty"`
	Required    bool      `json:"required"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// scopeStore holds custom scopes in memory (in production, use DB).
type scopeStore struct {
	mu     sync.RWMutex
	scopes map[string]*CustomScope // keyed by scope name
}

var customScopes = &scopeStore{scopes: make(map[string]*CustomScope)}

// handleScopes handles GET/POST/DELETE /api/v1/oauth/scopes.
// GET    /api/v1/oauth/scopes          — list all custom scopes
// GET    /api/v1/oauth/scopes?name=X    — get specific scope
// POST   /api/v1/oauth/scopes          — create new custom scope
// PUT    /api/v1/oauth/scopes          — update existing scope
// DELETE /api/v1/oauth/scopes?name=X    — delete scope
func handleScopes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		name := r.URL.Query().Get("name")
		if name != "" {
			customScopes.mu.RLock()
			scope, ok := customScopes.scopes[name]
			customScopes.mu.RUnlock()
			if !ok {
				writeJSON(w, http.StatusNotFound, map[string]any{"error": "scope not found"})
				return
			}
			writeJSON(w, http.StatusOK, scope)
			return
		}
		customScopes.mu.RLock()
		scopes := make([]*CustomScope, 0, len(customScopes.scopes))
		for _, s := range customScopes.scopes {
			scopes = append(scopes, s)
		}
		customScopes.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"scopes": scopes,
			"count":  len(scopes),
		})

	case http.MethodPost:
		var req struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Attributes  []string `json:"attributes"`
			Required    bool     `json:"required"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
			return
		}
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name is required"})
			return
		}

		customScopes.mu.Lock()
		if _, exists := customScopes.scopes[req.Name]; exists {
			customScopes.mu.Unlock()
			writeJSON(w, http.StatusConflict, map[string]any{"error": "scope already exists"})
			return
		}
		now := time.Now().UTC()
		scope := &CustomScope{
			ID:          uuid.New().String(),
			Name:        req.Name,
			Description: req.Description,
			Attributes:  req.Attributes,
			Required:    req.Required,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		customScopes.scopes[req.Name] = scope
		customScopes.mu.Unlock()

		writeJSON(w, http.StatusCreated, scope)

	case http.MethodPut:
		var req struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Attributes  []string `json:"attributes"`
			Required    bool     `json:"required"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
			return
		}
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name is required"})
			return
		}

		customScopes.mu.Lock()
		scope, ok := customScopes.scopes[req.Name]
		if !ok {
			customScopes.mu.Unlock()
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "scope not found"})
			return
		}
		scope.Description = req.Description
		scope.Attributes = req.Attributes
		scope.Required = req.Required
		scope.UpdatedAt = time.Now().UTC()
		customScopes.mu.Unlock()

		writeJSON(w, http.StatusOK, scope)

	case http.MethodDelete:
		name := r.URL.Query().Get("name")
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name is required"})
			return
		}
		customScopes.mu.Lock()
		_, ok := customScopes.scopes[name]
		if !ok {
			customScopes.mu.Unlock()
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "scope not found"})
			return
		}
		delete(customScopes.scopes, name)
		customScopes.mu.Unlock()

		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "scope": name})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}
