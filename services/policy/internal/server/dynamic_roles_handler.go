package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type DynamicRoleRule struct {
	ID         string    `json:"id"`
	Condition  string    `json:"condition"`   // e.g. "dept=engineering AND level>=L5"
	Action     string    `json:"action"`      // e.g. "assign_role:senior_engineer"
	Priority   int       `json:"priority"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
}

var (
	dynRoleMu sync.RWMutex
	dynRoles  = []DynamicRoleRule{
		{ID: "dr-default-1", Condition: "dept=engineering AND level>=L5", Action: "assign_role:senior_engineer", Priority: 10, Enabled: true, CreatedAt: time.Now().UTC().Add(-48 * time.Hour)},
		{ID: "dr-default-2", Condition: "title=manager AND direct_reports>0", Action: "assign_role:team_lead", Priority: 20, Enabled: true, CreatedAt: time.Now().UTC().Add(-24 * time.Hour)},
	}
)

// POST /api/v1/policies/dynamic-roles — create dynamic role assignment rule
// GET /api/v1/policies/dynamic-roles/list — list rules
func (s *HTTPServer) handleDynamicRoles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Condition string `json:"condition"`
			Action    string `json:"action"`
			Priority  int    `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Condition == "" || req.Action == "" {
			writeJSONError(w, http.StatusBadRequest, "condition and action required")
			return
		}
		rule := DynamicRoleRule{
			ID: "dr-" + uuid.New().String()[:8], Condition: req.Condition, Action: req.Action,
			Priority: req.Priority, Enabled: true, CreatedAt: time.Now().UTC(),
		}
		dynRoleMu.Lock()
		dynRoles = append(dynRoles, rule)
		dynRoleMu.Unlock()
		writeJSON(w, http.StatusCreated, rule)

	case http.MethodGet:
		dynRoleMu.RLock()
		result := make([]DynamicRoleRule, len(dynRoles))
		copy(result, dynRoles)
		dynRoleMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"rules": result, "count": len(result),
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
