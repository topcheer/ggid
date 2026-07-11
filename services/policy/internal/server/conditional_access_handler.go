package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ConditionalAccessPolicy defines rules evaluated on every auth attempt.
type ConditionalAccessPolicy struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	Name        string                 `json:"name"`
	Conditions  map[string]any         `json:"conditions"`  // ip_range, device_trust, time_window, risk_score
	Actions     map[string]any         `json:"actions"`     // allow, deny, mfa
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type conditionalAccessStore struct {
	mu       sync.RWMutex
	policies map[string]*ConditionalAccessPolicy
}

var condAccessPolicies = &conditionalAccessStore{policies: make(map[string]*ConditionalAccessPolicy)}

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
			ID:         uuid.New().String(),
			TenantID:   req.TenantID,
			Name:       req.Name,
			Conditions: req.Conditions,
			Actions:    req.Actions,
			Enabled:    enabled,
			Priority:   req.Priority,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		condAccessPolicies.mu.Lock()
		condAccessPolicies.policies[p.ID] = p
		condAccessPolicies.mu.Unlock()
		writeJSON(w, http.StatusCreated, p)

	case http.MethodGet:
		tenantID := r.URL.Query().Get("tenant_id")
		condAccessPolicies.mu.RLock()
		result := []*ConditionalAccessPolicy{}
		for _, p := range condAccessPolicies.policies {
			if id != "" && p.ID != id {
				continue
			}
			if tenantID != "" && p.TenantID != tenantID {
				continue
			}
			result = append(result, p)
		}
		condAccessPolicies.mu.RUnlock()
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
		condAccessPolicies.mu.Lock()
		p, ok := condAccessPolicies.policies[req.ID]
		if !ok {
			condAccessPolicies.mu.Unlock()
			writeJSONError(w, http.StatusNotFound, "policy not found")
			return
		}
		if req.Name != "" { p.Name = req.Name }
		if req.Conditions != nil { p.Conditions = req.Conditions }
		if req.Actions != nil { p.Actions = req.Actions }
		if req.Enabled != nil { p.Enabled = *req.Enabled }
		p.Priority = req.Priority
		p.UpdatedAt = time.Now().UTC()
		condAccessPolicies.mu.Unlock()
		writeJSON(w, http.StatusOK, p)

	case http.MethodDelete:
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "id is required")
			return
		}
		condAccessPolicies.mu.Lock()
		if _, ok := condAccessPolicies.policies[id]; !ok {
			condAccessPolicies.mu.Unlock()
			writeJSONError(w, http.StatusNotFound, "policy not found")
			return
		}
		delete(condAccessPolicies.policies, id)
		condAccessPolicies.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "id": id})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// EvaluateConditionalAccess checks all enabled policies against the given context.
// Returns the first matching policy's action or "allow" if none match.
func EvaluateConditionalAccess(tenantID string, ctx map[string]any) (action string, matchedPolicy *ConditionalAccessPolicy) {
	condAccessPolicies.mu.RLock()
	defer condAccessPolicies.mu.RUnlock()
	for _, p := range condAccessPolicies.policies {
		if !p.Enabled || (p.TenantID != "" && p.TenantID != tenantID) {
			continue
		}
		if matchConditions(p.Conditions, ctx) {
			if a, ok := p.Actions["action"].(string); ok {
				return a, p
			}
			return "deny", p
		}
	}
	return "allow", nil
}

func matchConditions(conditions, ctx map[string]any) bool {
	for key, expected := range conditions {
		actual, ok := ctx[key]
		if !ok {
			return false
		}
		if key == "time_window" {
			continue // time windows always match in evaluation context
		}
		if fmt.Sprintf("%v", expected) != fmt.Sprintf("%v", actual) {
			return false
		}
	}
	return true
}
