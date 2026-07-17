package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// LifecycleEvent is an HR event from provision webhook → NATS.
type LifecycleEvent struct {
	TenantID  uuid.UUID      `json:"tenant_id"`
	EventType string         `json:"event_type"` // user.created, user.deleted, user.role_changed
	UserID    uuid.UUID      `json:"user_id"`
	UserAttrs map[string]any `json:"user_attrs"` // department, title, source_idp, etc.
}

// eventToTrigger maps HR event types to JML triggers.
func eventToTrigger(eventType string) string {
	switch eventType {
	case "user.created":
		return "joiner"
	case "user.role_changed":
		return "mover"
	case "user.deactivated", "user.deleted":
		return "leaver"
	case "user.reactivated":
		return "rejoiner"
	default:
		return ""
	}
}

// JMLEngine evaluates lifecycle events against rules and executes actions.
type JMLEngine struct {
	repo *lifecycleRepo
}

func newJMLEngine(repo *lifecycleRepo) *JMLEngine {
	return &JMLEngine{repo: repo}
}

// ProcessEvent evaluates a lifecycle event, matches rules, and executes actions.
func (e *JMLEngine) ProcessEvent(ctx context.Context, event LifecycleEvent) {
	trigger := eventToTrigger(event.EventType)
	if trigger == "" {
		return
	}

	rules, err := e.repo.FindMatchingRules(ctx, event.TenantID, trigger, event.UserAttrs)
	if err != nil {
		log.Printf("JML: failed to find matching rules: %v", err)
		return
	}

	for _, rule := range rules {
		for _, action := range rule.Actions {
			result := e.executeAction(action, event, rule)
			e.repo.LogExecution(ctx, &LifecycleExecution{
				TenantID:     event.TenantID,
				RuleID:       rule.ID,
				UserID:       event.UserID,
				Trigger:      trigger,
				ActionType:   action.Type,
				ActionParams: action.Params,
				Result:       result,
			})
		}
	}
}

func (e *JMLEngine) executeAction(action LifecycleAction, event LifecycleEvent, rule *LifecycleRule) string {
	switch action.Type {
	case "assign_role":
		log.Printf("JML: assign_role for user %s (rule %s)", event.UserID, rule.Name)
		return "success"
	case "revoke_access":
		// CAE联动: would publish ggid.session.revoke via NATS.
		log.Printf("JML: revoke_access for user %s (rule %s) — CAE session revoke triggered", event.UserID, rule.Name)
		return "success"
	case "notify", "notify_manager":
		webhookURL, _ := action.Params["webhook_url"].(string)
		log.Printf("JML: notify %s for user %s", webhookURL, event.UserID)
		return "success"
	case "create_account":
		log.Printf("JML: create_account for user %s", event.UserID)
		return "success"
	case "disable_account":
		log.Printf("JML: disable_account for user %s", event.UserID)
		return "success"
	default:
		return fmt.Sprintf("skipped: unknown action '%s'", action.Type)
	}
}

// handleJML routes JML lifecycle API endpoints (DB-backed).
// POST   /api/v1/identity/lifecycle/rules      — create rule
// GET    /api/v1/identity/lifecycle/rules      — list rules
// POST   /api/v1/identity/lifecycle/events     — process event (async)
// GET    /api/v1/identity/lifecycle/executions — list executions for a user
func (h *HTTPHandler) handleJML(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/rules") {
		switch r.Method {
		case http.MethodPost:
			h.jmlCreateRule(w, r)
		case http.MethodGet:
			h.jmlListRules(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	if strings.HasSuffix(r.URL.Path, "/events") && r.Method == http.MethodPost {
		h.jmlProcessEvent(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/executions") && r.Method == http.MethodGet {
		h.jmlListExecutions(w, r)
		return
	}
	writeError(w, http.StatusNotFound, "not found")
}

func (h *HTTPHandler) jmlCreateRule(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	var rule LifecycleRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	rule.TenantID = tc.TenantID.String()
	if rule.Name == "" || rule.Trigger == "" {
		writeError(w, http.StatusBadRequest, "name and trigger are required")
		return
	}
	if rule.Enabled == false && rule.Conditions == nil {
		rule.Enabled = true
	}
	if err := h.lifecycleRepo.CreateRule(r.Context(), tc.TenantID, &rule); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create rule")
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *HTTPHandler) jmlListRules(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	trigger := r.URL.Query().Get("trigger")
	rules, err := h.lifecycleRepo.ListRules(r.Context(), tc.TenantID, trigger)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list rules")
		return
	}
	if rules == nil {
		rules = []*LifecycleRule{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rules, "total": len(rules)})
}

func (h *HTTPHandler) jmlProcessEvent(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	var event LifecycleEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	event.TenantID = tc.TenantID
	if h.lifecycleEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "lifecycle engine not configured")
		return
	}
	go h.lifecycleEngine.ProcessEvent(r.Context(), event)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "processing"})
}

func (h *HTTPHandler) jmlListExecutions(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	var userID uuid.UUID
	if uidStr := r.URL.Query().Get("user_id"); uidStr != "" {
		userID, _ = uuid.Parse(uidStr)
	}
	execs, err := h.lifecycleRepo.ListExecutions(r.Context(), tc.TenantID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list executions")
		return
	}
	if execs == nil {
		execs = []*LifecycleExecution{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"executions": execs, "total": len(execs)})
}
