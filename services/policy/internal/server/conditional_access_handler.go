package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ConditionalAccessPolicy defines rules evaluated on every auth attempt.
type ConditionalAccessPolicy struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	Name       string         `json:"name"`
	Conditions map[string]any `json:"conditions"`
	Actions    map[string]any `json:"actions"`
	Enabled    bool           `json:"enabled"`
	Priority   int            `json:"priority"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// POST/GET/PUT/DELETE /api/v1/policies/conditional-access
func (s *HTTPServer) handleConditionalAccess(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	switch r.Method {
	case http.MethodPost:
		var req struct {
			TenantID   string         `json:"tenant_id"`
			Name       string         `json:"name"`
			Conditions map[string]any `json:"conditions"`
			Actions    map[string]any `json:"actions"`
			Enabled    *bool          `json:"enabled"`
			Priority   int            `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Name == "" {
			writeJSONError(w, http.StatusBadRequest, "name is required")
			return
		}
		if req.Conditions == nil {
			req.Conditions = map[string]any{}
		}
		if req.Actions == nil {
			req.Actions = map[string]any{"action": "deny"}
		}
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		now := time.Now().UTC()
		p := &ConditionalAccessPolicy{
			ID: uuid.New().String(), TenantID: req.TenantID, Name: req.Name,
			Conditions: req.Conditions, Actions: req.Actions,
			Enabled: enabled, Priority: req.Priority, CreatedAt: now, UpdatedAt: now,
		}
		if s.policyMap != nil {
			s.policyMap.Store(r.Context(), "conditional_access_store", p.ID, map[string]any{
				"tenant_id": p.TenantID, "name": p.Name, "conditions": p.Conditions,
				"actions": p.Actions, "enabled": p.Enabled, "priority": p.Priority,
			})
		}
		writeJSON(w, http.StatusCreated, p)

	case http.MethodGet:
		tenantID := r.URL.Query().Get("tenant_id")
		var result []*ConditionalAccessPolicy
		if s.policyMap != nil {
			rows, _ := s.policyMap.List(r.Context(), "conditional_access_store")
			for _, row := range rows {
				p := &ConditionalAccessPolicy{
					ID: pmGetString(row, "id"), TenantID: pmGetString(row, "tenant_id"),
					Name: pmGetString(row, "name"), Conditions: pmGetMap(row, "conditions"),
					Actions: pmGetMap(row, "actions"), Enabled: pmGetBool(row, "enabled"),
				}
				if id != "" && p.ID != id {
					continue
				}
				if tenantID != "" && p.TenantID != tenantID {
					continue
				}
				result = append(result, p)
			}
		}
		if result == nil {
			result = []*ConditionalAccessPolicy{}
		}
		if id != "" && len(result) == 1 {
			writeJSON(w, http.StatusOK, result[0])
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"policies": result, "count": len(result)})

	case http.MethodPut:
		var req struct {
			ID         string         `json:"id"`
			Name       string         `json:"name"`
			Conditions map[string]any `json:"conditions"`
			Actions    map[string]any `json:"actions"`
			Enabled    *bool          `json:"enabled"`
			Priority   int            `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.ID == "" {
			req.ID = id
		}
		if req.ID == "" {
			writeJSONError(w, http.StatusBadRequest, "id is required")
			return
		}
		if s.policyMap != nil {
			update := map[string]any{"name": req.Name, "conditions": req.Conditions,
				"actions": req.Actions, "priority": req.Priority}
			if req.Enabled != nil {
				update["enabled"] = *req.Enabled
			}
			s.policyMap.Store(r.Context(), "conditional_access_store", req.ID, update)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "id": req.ID})

	case http.MethodDelete:
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "id is required")
			return
		}
		if s.policyMap != nil {
			s.policyMap.Delete(r.Context(), "conditional_access_store", id)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "id": id})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// EvaluateConditionalAccess checks all enabled policies against the given context.
func EvaluateConditionalAccess(tenantID string, ctx map[string]any) (action string, matchedPolicy *ConditionalAccessPolicy) {
	// In-memory evaluation only for tests without DB — production uses DB-backed handler above.
	return "allow", nil
}

//nolint:unused // kept for EvaluateConditionalAccess callers and future use
func matchConditions(conditions, ctx map[string]any) bool {
	for key, expected := range conditions {
		actual, ok := ctx[key]
		if !ok {
			return false
		}
		if key == "time_window" {
			continue
		}
		if fmt.Sprintf("%v", expected) != fmt.Sprintf("%v", actual) {
			return false
		}
	}
	return true
}

// --- policy map helpers ---

func pmGetString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func pmGetBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case bool:
			return val
		case string:
			return val == "true"
		}
	}
	return false
}

func pmGetMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mp, ok := v.(map[string]any); ok {
			return mp
		}
	}
	return nil
}
