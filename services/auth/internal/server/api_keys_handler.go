package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// APIKey represents an API key for programmatic access.
type APIKey struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Scopes     []string  `json:"scopes"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastUsed   *time.Time `json:"last_used"`
	Status     string    `json:"status"` // active, expired, revoked
	UsageCount int       `json:"usage_count"`
}

var (
	apiKeysMu sync.RWMutex
	apiKeys   = []APIKey{}
)

// GET/POST /api/v1/auth/api-keys
// GET/POST/DELETE /api/v1/auth/api-keys/{id}
func (h *Handler) handleAPIKeys(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/v1/auth/api-keys" && r.Method == http.MethodGet:
		apiKeysMu.RLock()
		keys := make([]APIKey, len(apiKeys))
		copy(keys, apiKeys)
		apiKeysMu.RUnlock()
		writeJSON(w, http.StatusOK, keys)

	case r.URL.Path == "/api/v1/auth/api-keys" && r.Method == http.MethodPost:
		var req struct {
			Name      string   `json:"name"`
			Scopes    []string `json:"scopes"`
			ExpiresAt string   `json:"expires_at"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		key := APIKey{
			ID:        fmt.Sprintf("key-%d", time.Now().UnixNano()),
			Name:      req.Name,
			Scopes:    req.Scopes,
			CreatedAt: time.Now(),
			Status:    "active",
		}
		if req.ExpiresAt != "" {
			t, err := time.Parse(time.RFC3339, req.ExpiresAt)
			if err == nil {
				key.ExpiresAt = t
			}
		}
		apiKeysMu.Lock()
		apiKeys = append(apiKeys, key)
		apiKeysMu.Unlock()
		writeJSON(w, http.StatusCreated, key)

	case strings.HasPrefix(r.URL.Path, "/api/v1/auth/api-keys/") && r.Method == http.MethodPost:
		// Handle /api/v1/auth/api-keys/{id}/rotate
		parts := splitPath(r.URL.Path)
		if len(parts) >= 6 && parts[5] == "rotate" {
			apiKeysMu.Lock()
			defer apiKeysMu.Unlock()
			for i := range apiKeys {
				if apiKeys[i].ID == parts[4] {
					apiKeys[i].ID = fmt.Sprintf("key-%d", time.Now().UnixNano())
					writeJSON(w, http.StatusOK, apiKeys[i])
					return
				}
			}
			writeError(w, http.StatusNotFound, "API key not found")
			return
		}
		writeError(w, http.StatusNotFound, "unknown path")

	case strings.HasPrefix(r.URL.Path, "/api/v1/auth/api-keys/") && r.Method == http.MethodDelete:
		parts := splitPath(r.URL.Path)
		if len(parts) >= 5 {
			apiKeysMu.Lock()
			defer apiKeysMu.Unlock()
			for i := range apiKeys {
				if apiKeys[i].ID == parts[4] {
					apiKeys[i].Status = "revoked"
					writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
					return
				}
			}
		}
		writeError(w, http.StatusNotFound, "API key not found")

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// splitPath splits a URL path into segments.
func splitPath(path string) []string {
	var parts []string
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}
