package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ConditionGroup struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Logic      string            `json:"logic"` // AND, OR, NOT
	Conditions []map[string]any  `json:"conditions"`
	Children   []string          `json:"children,omitempty"` // nested group IDs
	CreatedAt  time.Time         `json:"created_at"`
}

var (
	abacGroupMu sync.RWMutex
	abacGroups  = make(map[string]*ConditionGroup)
)

// POST /api/v1/policies/abac/groups — create condition group
// GET /api/v1/policies/abac/groups — list groups
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
		if req.Logic == "" {
			req.Logic = "AND"
		}
		g := &ConditionGroup{
			ID: "cg-" + uuid.New().String()[:8], Name: req.Name, Logic: req.Logic,
			Conditions: req.Conditions, Children: req.Children,
			CreatedAt: time.Now().UTC(),
		}
		abacGroupMu.Lock()
		abacGroups[g.ID] = g
		abacGroupMu.Unlock()
		writeJSON(w, http.StatusCreated, g)

	case http.MethodGet:
		abacGroupMu.RLock()
		result := make([]*ConditionGroup, 0, len(abacGroups))
		for _, g := range abacGroups {
			result = append(result, g)
		}
		abacGroupMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"groups": result, "count": len(result)})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
