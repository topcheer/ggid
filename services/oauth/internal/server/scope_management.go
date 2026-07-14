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

// scopeStore holds custom scopes in memory (fallback when DB unavailable).
type scopeStore struct {
	mu     sync.RWMutex
	scopes map[string]*CustomScope // keyed by scope name
}

var customScopes = &scopeStore{scopes: make(map[string]*CustomScope)}

// scopeAdapterVar holds the active scope store adapter (PG or in-memory fallback).
// Set during server.New() initialization.
var scopeAdapterVar *scopeStoreAdapter

// handleScopes handles GET/POST/PUT/DELETE /api/v1/oauth/scopes.
// Uses the persistent scope store (PostgreSQL) with in-memory fallback.
func handleScopes(w http.ResponseWriter, r *http.Request) {
	store := scopeAdapterVar
	if store == nil {
		store = newScopeStoreAdapter(nil)
	}

	switch r.Method {
	case http.MethodGet:
		name := r.URL.Query().Get("name")
		if name != "" {
			scope, ok := store.Get(name)
			if !ok {
				writeJSON(w, http.StatusNotFound, map[string]any{"error": "scope not found"})
				return
			}
			writeJSON(w, http.StatusOK, scope)
			return
		}
		scopes := store.List()
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
		if _, exists := store.Get(req.Name); exists {
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
		if err := store.Create(scope); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
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
		scope, ok := store.Get(req.Name)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "scope not found"})
			return
		}
		scope.Description = req.Description
		scope.Attributes = req.Attributes
		scope.Required = req.Required
		scope.UpdatedAt = time.Now().UTC()
		if err := store.Update(scope); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, scope)

	case http.MethodDelete:
		name := r.URL.Query().Get("name")
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name is required"})
			return
		}
		if _, ok := store.Get(name); !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "scope not found"})
			return
		}
		store.Delete(name)
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "scope": name})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}
