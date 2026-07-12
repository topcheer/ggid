package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PermissionBoundary struct {
	ID            string   `json:"id"`
	Role          string   `json:"role"`
	MaxScopes     []string `json:"max_scopes"`
	DeniedActions []string `json:"denied_actions"`
	UpdatedAt     time.Time `json:"updated_at"`
}

var (
	pbMu sync.RWMutex
	pbs  = make(map[string]*PermissionBoundary)
)

// POST/GET/PUT /api/v1/policies/permission-boundaries
func (s *HTTPServer) handlePermissionBoundaries(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost, http.MethodPut:
		var req struct {
			Role          string   `json:"role"`
			MaxScopes     []string `json:"max_scopes"`
			DeniedActions []string `json:"denied_actions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSONError(w, http.StatusBadRequest, "invalid JSON"); return }
		if req.Role == "" { writeJSONError(w, http.StatusBadRequest, "role required"); return }
		pb := &PermissionBoundary{ID: "pb-" + uuid.New().String()[:8], Role: req.Role, MaxScopes: req.MaxScopes, DeniedActions: req.DeniedActions, UpdatedAt: time.Now().UTC()}
		pbMu.Lock(); pbs[req.Role] = pb; pbMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "configured", "boundary": pb})
	case http.MethodGet:
		pbMu.RLock(); result := []*PermissionBoundary{}
		for _, pb := range pbs { result = append(result, pb) }
		pbMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"boundaries": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
