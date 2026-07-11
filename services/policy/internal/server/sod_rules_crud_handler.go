package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SoDRulePair defines a mutually exclusive role pair.
type SoDRulePair struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	RoleA       string    `json:"role_a"`
	RoleB       string    `json:"role_b"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

type sodRuleStore struct {
	mu    sync.RWMutex
	rules map[string]*SoDRulePair
}

var sodRulesCRUD = &sodRuleStore{rules: make(map[string]*SoDRulePair)}

// POST/GET/PUT/DELETE /api/v1/policies/sod/rules
func (s *HTTPServer) handleSoDRules(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	switch r.Method {
	case http.MethodPost:
		var req struct {
			TenantID    string `json:"tenant_id"`
			RoleA       string `json:"role_a"`
			RoleB       string `json:"role_b"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.RoleA == "" || req.RoleB == "" {
			writeJSONError(w, http.StatusBadRequest, "role_a and role_b are required")
			return
		}
		rule := &SoDRulePair{
			ID:          uuid.New().String(),
			TenantID:    req.TenantID,
			RoleA:       req.RoleA,
			RoleB:       req.RoleB,
			Description: req.Description,
			Enabled:     true,
			CreatedAt:   time.Now().UTC(),
		}
		sodRulesCRUD.mu.Lock()
		sodRulesCRUD.rules[rule.ID] = rule
		sodRulesCRUD.mu.Unlock()
		writeJSON(w, http.StatusCreated, rule)

	case http.MethodGet:
		tenantID := r.URL.Query().Get("tenant_id")
		sodRulesCRUD.mu.RLock()
		result := []*SoDRulePair{}
		for _, rl := range sodRulesCRUD.rules {
			if id != "" && rl.ID != id {
				continue
			}
			if tenantID != "" && rl.TenantID != tenantID {
				continue
			}
			result = append(result, rl)
		}
		sodRulesCRUD.mu.RUnlock()
		if id != "" && len(result) == 1 {
			writeJSON(w, http.StatusOK, result[0])
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"rules": result, "count": len(result)})

	case http.MethodPut:
		var req struct {
			ID          string `json:"id"`
			RoleA       string `json:"role_a"`
			RoleB       string `json:"role_b"`
			Description string `json:"description"`
			Enabled     *bool  `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.ID == "" {
			req.ID = id
		}
		sodRulesCRUD.mu.Lock()
		rl, ok := sodRulesCRUD.rules[req.ID]
		if !ok {
			sodRulesCRUD.mu.Unlock()
			writeJSONError(w, http.StatusNotFound, "rule not found")
			return
		}
		if req.RoleA != "" { rl.RoleA = req.RoleA }
		if req.RoleB != "" { rl.RoleB = req.RoleB }
		if req.Description != "" { rl.Description = req.Description }
		if req.Enabled != nil { rl.Enabled = *req.Enabled }
		sodRulesCRUD.mu.Unlock()
		writeJSON(w, http.StatusOK, rl)

	case http.MethodDelete:
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "id is required")
			return
		}
		sodRulesCRUD.mu.Lock()
		if _, ok := sodRulesCRUD.rules[id]; !ok {
			sodRulesCRUD.mu.Unlock()
			writeJSONError(w, http.StatusNotFound, "rule not found")
			return
		}
		delete(sodRulesCRUD.rules, id)
		sodRulesCRUD.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "id": id})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
