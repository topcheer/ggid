package httpserver

import (
	"encoding/json"
	"net/http"
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
			ID: uuid.New().String(), TenantID: req.TenantID, RoleA: req.RoleA,
			RoleB: req.RoleB, Description: req.Description, Enabled: true,
			CreatedAt: time.Now().UTC(),
		}
		if s.policyMap != nil {
			s.policyMap.Store(r.Context(), "policy_sod_rule_pairs", rule.ID, map[string]any{
				"tenant_id": rule.TenantID, "role_a": rule.RoleA, "role_b": rule.RoleB,
				"description": rule.Description, "enabled": rule.Enabled,
			})
		}
		writeJSON(w, http.StatusCreated, rule)

	case http.MethodGet:
		tenantID := r.URL.Query().Get("tenant_id")
		var result []*SoDRulePair
		if s.policyMap != nil {
			rows, _ := s.policyMap.List(r.Context(), "policy_sod_rule_pairs")
			for _, row := range rows {
				rl := &SoDRulePair{
					ID: pmGetString(row, "id"), TenantID: pmGetString(row, "tenant_id"),
					RoleA: pmGetString(row, "role_a"), RoleB: pmGetString(row, "role_b"),
					Description: pmGetString(row, "description"), Enabled: pmGetBool(row, "enabled"),
				}
				if id != "" && rl.ID != id {
					continue
				}
				if tenantID != "" && rl.TenantID != tenantID {
					continue
				}
				result = append(result, rl)
			}
		}
		if result == nil {
			result = []*SoDRulePair{}
		}
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
		if s.policyMap != nil {
			update := map[string]any{
				"role_a": req.RoleA, "role_b": req.RoleB,
				"description": req.Description,
			}
			if req.Enabled != nil {
				update["enabled"] = *req.Enabled
			}
			s.policyMap.Store(r.Context(), "policy_sod_rule_pairs", req.ID, update)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "id": req.ID})

	case http.MethodDelete:
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "id is required")
			return
		}
		if s.policyMap != nil {
			s.policyMap.Delete(r.Context(), "policy_sod_rule_pairs", id)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "id": id})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
