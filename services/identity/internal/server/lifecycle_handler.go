package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
		// Persist to PG via identityPolicyMap.
		if h.identityPolicyMap != nil {
			data := map[string]any{
				"tenant_id": rule.TenantID, "name": rule.Name, "trigger": rule.Trigger,
				"conditions": rule.Conditions, "actions": rule.Actions,
				"enabled": rule.Enabled, "created_at": rule.CreatedAt,
			}
			h.identityPolicyMap.Store(r.Context(), "lifecycle_rules_store", rule.ID, data)
		}
		writeJSON(w, http.StatusCreated, rule)

	case http.MethodGet:
		trigger := r.URL.Query().Get("trigger")
		tenantID := r.URL.Query().Get("tenant_id")
		var rules []*LifecycleRule
		if h.identityPolicyMap != nil {
			rows, _ := h.identityPolicyMap.List(r.Context(), "lifecycle_rules_store")
			for _, row := range rows {
				rl := &LifecycleRule{
					ID:        getString(row, "id"),
					TenantID:  getString(row, "tenant_id"),
					Name:      getString(row, "name"),
					Trigger:   getString(row, "trigger"),
					Conditions: getMap(row, "conditions"),
					Enabled:   getBool(row, "enabled"),
				}
				if trigger != "" && rl.Trigger != trigger {
					continue
				}
				if tenantID != "" && rl.TenantID != tenantID {
					continue
				}
				rules = append(rules, rl)
			}
		}
		if rules == nil {
			rules = []*LifecycleRule{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"rules": rules, "count": len(rules)})

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

	// Load rules from PG.
	var applicableRules []*LifecycleRule
	if h.identityPolicyMap != nil {
		rows, _ := h.identityPolicyMap.List(r.Context(), "lifecycle_rules_store")
		for _, row := range rows {
			rl := &LifecycleRule{
				ID:      getString(row, "id"),
				Name:    getString(row, "name"),
				Trigger: getString(row, "trigger"),
				Enabled: getBool(row, "enabled"),
			}
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
	}
	if applicableRules == nil {
		applicableRules = []*LifecycleRule{}
	}

	// Build preview actions
	previewActions := []map[string]any{}
	for _, rl := range applicableRules {
		for _, a := range rl.Actions {
			previewActions = append(previewActions, map[string]any{
				"rule_id":        rl.ID,
				"rule_name":      rl.Name,
				"trigger":        rl.Trigger,
				"action":         a.Type,
				"params":         a.Params,
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

// --- helpers for JSONB row access ---

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getBool(m map[string]any, key string) bool {
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

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mp, ok := v.(map[string]any); ok {
			return mp
		}
	}
	return nil
}
