package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type PermissionBoundary struct {
	ID            string    `json:"id"`
	Role          string    `json:"role"`
	MaxScopes     []string  `json:"max_scopes"`
	DeniedActions []string  `json:"denied_actions"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (s *HTTPServer) handlePermissionBoundaries(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost, http.MethodPut:
		var req struct {
			Role          string   `json:"role"`
			MaxScopes     []string `json:"max_scopes"`
			DeniedActions []string `json:"denied_actions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Role == "" {
			writeJSONError(w, http.StatusBadRequest, "role required")
			return
		}
		pb := &PermissionBoundary{
			ID: "pb-" + uuid.New().String()[:8], Role: req.Role,
			MaxScopes: req.MaxScopes, DeniedActions: req.DeniedActions,
			UpdatedAt: time.Now().UTC(),
		}
		if s.policyMap != nil {
			s.policyMap.Store(r.Context(), "policy_permission_boundaries", req.Role, map[string]any{
				"id": pb.ID, "role": pb.Role, "max_scopes": pb.MaxScopes,
				"denied_actions": pb.DeniedActions, "updated_at": pb.UpdatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "configured", "boundary": pb})
	case http.MethodGet:
		var result []map[string]any
		if s.policyMap != nil {
			rows, _ := s.policyMap.List(r.Context(), "policy_permission_boundaries")
			result = rows
		}
		if result == nil { result = []map[string]any{} }
		writeJSON(w, http.StatusOK, map[string]any{"boundaries": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
