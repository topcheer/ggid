package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LifecycleRule defines an automated user lifecycle action.
type LifecycleRule struct {
	ID         string            `json:"id"`
	TenantID   string            `json:"tenant_id"`
	Name       string            `json:"name"`
	Trigger    string            `json:"trigger"`  // joiner, mover, leaver
	Conditions map[string]any    `json:"conditions,omitempty"`
	Actions    []LifecycleAction `json:"actions"`
	Enabled    bool              `json:"enabled"`
	CreatedAt  time.Time         `json:"created_at"`
}

// LifecycleAction represents a single action in a lifecycle rule.
type LifecycleAction struct {
	Type  string         `json:"type"`  // assign_role, revoke_access, notify_manager
	Params map[string]any `json:"params,omitempty"`
}

type lifecycleRuleStore struct {
	mu    sync.RWMutex
	rules map[string]*LifecycleRule
}

var lifecycleRules = &lifecycleRuleStore{rules: make(map[string]*LifecycleRule)}

// POST /api/v1/users/lifecycle/rules          — create rule
// GET  /api/v1/users/lifecycle/rules          — list rules
// GET  /api/v1/users/{id}/lifecycle-preview   — preview applicable rules for user
func (h *HTTPHandler) handleLifecycleRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			TenantID   string            `json:"tenant_id"`
			Name       string            `json:"name"`
			Trigger    string            `json:"trigger"`
			Conditions map[string]any    `json:"conditions"`
			Actions    []LifecycleAction `json:"actions"`
			Enabled    *bool             `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Name == "" || req.Trigger == "" {
			writeError(w, http.StatusBadRequest, "name and trigger are required")
			return
		}
		if req.Trigger != "joiner" && req.Trigger != "mover" && req.Trigger != "leaver" {
			writeError(w, http.StatusBadRequest, "trigger must be joiner, mover, or leaver")
			return
		}
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		rule := &LifecycleRule{
			ID:         uuid.New().String(),
			TenantID:   req.TenantID,
			Name:       req.Name,
			Trigger:    req.Trigger,
			Conditions: req.Conditions,
			Actions:    req.Actions,
			Enabled:    enabled,
			CreatedAt:  time.Now().UTC(),
		}
		lifecycleRules.mu.Lock()
		lifecycleRules.rules[rule.ID] = rule
		lifecycleRules.mu.Unlock()
		writeJSON(w, http.StatusCreated, rule)

	case http.MethodGet:
		trigger := r.URL.Query().Get("trigger")
		tenantID := r.URL.Query().Get("tenant_id")
		lifecycleRules.mu.RLock()
		result := []*LifecycleRule{}
		for _, rl := range lifecycleRules.rules {
			if trigger != "" && rl.Trigger != trigger {
				continue
			}
			if tenantID != "" && rl.TenantID != tenantID {
				continue
			}
			result = append(result, rl)
		}
		lifecycleRules.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"rules": result, "count": len(result)})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// GET /api/v1/users/{id}/lifecycle-preview — preview applicable rules for a user.
func (h *HTTPHandler) handleLifecyclePreview(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	// Determine which triggers apply based on user status
	var applicableTriggers []string
	switch string(user.Status) {
	case "pending", "invited":
		applicableTriggers = []string{"joiner"}
	case "active":
		applicableTriggers = []string{"mover"}
	case "disabled", "deprovisioned":
		applicableTriggers = []string{"leaver"}
	default:
		applicableTriggers = []string{"joiner", "mover", "leaver"}
	}

	lifecycleRules.mu.RLock()
	applicableRules := []*LifecycleRule{}
	for _, rl := range lifecycleRules.rules {
		if !rl.Enabled {
			continue
		}
		for _, t := range applicableTriggers {
			if rl.Trigger == t {
				applicableRules = append(applicableRules, rl)
				break
			}
		}
	}
	lifecycleRules.mu.RUnlock()

	// Build preview actions
	previewActions := []map[string]any{}
	for _, rl := range applicableRules {
		for _, a := range rl.Actions {
			previewActions = append(previewActions, map[string]any{
				"rule_id":    rl.ID,
				"rule_name":  rl.Name,
				"trigger":    rl.Trigger,
				"action":     a.Type,
				"params":     a.Params,
				"would_execute": true,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":      userID.String(),
		"username":     user.Username,
		"status":       string(user.Status),
		"triggers":     applicableTriggers,
		"rules":        applicableRules,
		"actions":      previewActions,
		"action_count": len(previewActions),
	})
}
