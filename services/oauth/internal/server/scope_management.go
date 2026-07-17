package server

import (
	"encoding/json"
	"fmt"
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

// scopeCache is a read-through cache for scope lookups (hot path during token issuance).
var scopeCache sync.Map // name → *CustomScope

// scopeStoreAdapter is used by the scope management handlers for PG-backed operations.
var scopeAdapterVar *scopeStoreAdapter

func handleScopes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var scope CustomScope
		if err := json.NewDecoder(r.Body).Decode(&scope); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
			return
		}
		if scope.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name is required"})
			return
		}
		scope.ID = uuid.New().String()
		now := time.Now().UTC()
		scope.CreatedAt = now
		scope.UpdatedAt = now
		if mapRepoVar != nil {
			mapRepoVar.Store(r.Context(), "oauth_custom_scopes", scope.Name, map[string]any{
				"id": scope.ID, "name": scope.Name, "description": scope.Description,
				"attributes": scope.Attributes, "required": scope.Required,
			})
		}
		scopeCache.Store(scope.Name, &scope)
		writeJSON(w, http.StatusCreated, scope)
	case http.MethodGet:
		var scopes []*CustomScope
		if mapRepoVar != nil {
			rows, _ := mapRepoVar.List(r.Context(), "oauth_custom_scopes")
			for _, row := range rows {
				s := &CustomScope{
					ID: omGetString(row, "id"), Name: omGetString(row, "name"),
					Description: omGetString(row, "description"),
				}
				scopes = append(scopes, s)
			}
		}
		if scopes == nil {
			scopes = []*CustomScope{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"scopes": scopes, "count": len(scopes)})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

// --- helpers ---

func omGetString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
