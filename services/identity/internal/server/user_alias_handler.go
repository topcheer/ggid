package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// userAlias represents an alias for a user account.
type userAlias struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	TenantID  string `json:"tenant_id"`
	AliasType string `json:"alias_type"` // email, username, upn
	Value     string `json:"value"`
	CreatedAt string `json:"created_at"`
}

var userAliasStore = struct {
	sync.RWMutex
	data map[string][]userAlias // userID → aliases
}{data: make(map[string][]userAlias)}

// POST   /api/v1/users/{id}/aliases -- add alias
// GET    /api/v1/users/{id}/aliases -- list aliases
// DELETE /api/v1/users/{id}/aliases?id=X -- delete alias
func (h *HTTPHandler) handleUserAliases(w http.ResponseWriter, r *http.Request) {
	// SECURITY: Require tenant context for isolation.
	callerTenant := r.Header.Get("X-Tenant-ID")
	if callerTenant == "" {
		writeJSONError(w, http.StatusForbidden, "tenant context required")
		return
	}

	// SECURITY: Require admin scope for all alias operations.
	// Self-service alias management is not supported -- aliases are identity
	// attributes that feed into login/password-reset flows. Allowing non-admin
	// users to modify other users' aliases could enable account takeover.
	scopes := r.Header.Get("X-Scopes")
	isAdmin := strings.Contains(scopes, "platform:admin") || strings.Contains(scopes, "tenant:admin")
	if !isAdmin {
		// Allow users to read their own aliases (self-service GET).
		callerUserID := r.Header.Get("X-User-ID")
		if r.Method != http.MethodGet || callerUserID == "" {
			writeJSONError(w, http.StatusForbidden, "admin scope required to manage user aliases")
			return
		}
		// Will check ownership after extracting userID from path.
	}

	// Extract user ID from path
	path := r.URL.Path
	userID := ""
	if idx := strings.Index(path, "/users/"); idx >= 0 {
		rest := path[idx+len("/users/"):]
		if aIdx := strings.Index(rest, "/aliases"); aIdx >= 0 {
			userID = rest[:aIdx]
		}
	}
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "user ID is required in path")
		return
	}
	if _, err := uuid.Parse(userID); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	// SECURITY: Non-admin users can only access their own aliases.
	if !isAdmin {
		callerUserID := r.Header.Get("X-User-ID")
		if callerUserID != userID {
			writeJSONError(w, http.StatusForbidden, "cannot access another user's aliases")
			return
		}
	}

	switch r.Method {
	case http.MethodPost:
		var req struct {
			AliasType string `json:"alias_type"`
			Value     string `json:"value"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		validTypes := map[string]bool{"email": true, "username": true, "upn": true}
		if !validTypes[req.AliasType] {
			writeJSONError(w, http.StatusBadRequest, "alias_type must be one of: email, username, upn")
			return
		}
		if req.Value == "" {
			writeJSONError(w, http.StatusBadRequest, "value is required")
			return
		}

		alias := userAlias{
			ID:        uuid.New().String(),
			UserID:    userID,
			TenantID:  callerTenant,
			AliasType: req.AliasType,
			Value:     req.Value,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		userAliasStore.Lock()
		userAliasStore.data[userID] = append(userAliasStore.data[userID], alias)
		userAliasStore.Unlock()

		writeJSON(w, http.StatusCreated, alias)

	case http.MethodGet:
		userAliasStore.RLock()
		allAliases := userAliasStore.data[userID]
		result := make([]userAlias, 0)
		for _, a := range allAliases {
			if a.TenantID == callerTenant {
				result = append(result, a)
			}
		}
		userAliasStore.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"user_id": userID,
			"aliases": result,
			"total":   len(result),
		})

	case http.MethodDelete:
		aliasID := r.URL.Query().Get("id")
		if aliasID == "" {
			writeJSONError(w, http.StatusBadRequest, "id query parameter is required")
			return
		}

		userAliasStore.Lock()
		aliases := userAliasStore.data[userID]
		found := false
		filtered := aliases[:0]
		for _, a := range aliases {
			if a.ID == aliasID && a.TenantID == callerTenant {
				found = true
				continue // skip = delete
			}
			filtered = append(filtered, a)
		}
		userAliasStore.data[userID] = filtered
		userAliasStore.Unlock()

		if !found {
			writeJSONError(w, http.StatusNotFound, "alias not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": true,
			"id":      aliasID,
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
