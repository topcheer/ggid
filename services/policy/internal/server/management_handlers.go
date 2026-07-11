package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// GET /api/v1/policies/permissions/tree?node_id=X
func (s *HTTPServer) handlePermissionTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tree":   []any{},
		"message": "permission inheritance tree — use service.PermissionTree for queries",
	})
}

// --- Rate limit per-tenant config ---
type tenantRateLimit struct {
	mu     sync.RWMutex
	config map[string]any
}

var globalRateLimits = &tenantRateLimit{config: map[string]any{
	"default_rpm":     100,
	"default_burst":   20,
	"per_tenant":      map[string]any{},
}}

func (s *HTTPServer) handleRateLimits(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		globalRateLimits.mu.RLock()
		defer globalRateLimits.mu.RUnlock()
		writeJSON(w, http.StatusOK, globalRateLimits.config)
	case http.MethodPut:
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		globalRateLimits.mu.Lock()
		for k, v := range req {
			globalRateLimits.config[k] = v
		}
		globalRateLimits.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "config": globalRateLimits.config})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

var _ = uuid.Nil
var _ = time.Now
