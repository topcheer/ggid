package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type ConditionGroup struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Logic      string           `json:"logic"`
	Conditions []map[string]any `json:"conditions"`
	Children   []string         `json:"children,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
}

func (s *HTTPServer) handleABACGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Name       string           `json:"name"`
			Logic      string           `json:"logic"`
			Conditions []map[string]any `json:"conditions"`
			Children   []string         `json:"children"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Logic == "" { req.Logic = "AND" }
		g := &ConditionGroup{
			ID: "cg-" + uuid.New().String()[:8], Name: req.Name, Logic: req.Logic,
			Conditions: req.Conditions, Children: req.Children,
			CreatedAt: time.Now().UTC(),
		}
		if s.policyMap != nil {
			s.policyMap.Store(r.Context(), "policy_abac_groups", g.ID, map[string]any{
				"name": g.Name, "logic": g.Logic, "conditions": g.Conditions, "children": g.Children,
			})
		}
		writeJSON(w, http.StatusCreated, g)
	case http.MethodGet:
		var result []map[string]any
		if s.policyMap != nil {
			rows, _ := s.policyMap.List(r.Context(), "policy_abac_groups")
			result = rows
		}
		if result == nil { result = []map[string]any{} }
		writeJSON(w, http.StatusOK, map[string]any{"groups": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
