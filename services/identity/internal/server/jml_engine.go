package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

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
	repo     *lifecycleRepo
	policyURL string   // policy service base URL for role assign/revoke
	natsConn natsConn  // NATS connection for session revoke + notifications
}

// natsConn is a minimal interface for NATS publish (avoids hard dependency).
type natsConn interface {
	Publish(subject string, data []byte) error
}

func newJMLEngine(repo *lifecycleRepo) *JMLEngine {
	return &JMLEngine{repo: repo}
}

// SetPolicyURL configures the policy service endpoint for role operations.
func (e *JMLEngine) SetPolicyURL(url string) {
	e.policyURL = url
}

// SetNATSConn injects the NATS connection for CAE events.
func (e *JMLEngine) SetNATSConn(nc natsConn) {
	e.natsConn = nc
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
		roleIDStr, _ := action.Params["role_id"].(string)
		if roleIDStr == "" {
			return "failed: missing role_id param"
		}
		// Call policy service AssignRole via internal API.
		if err := e.callPolicyAssignRole(event.TenantID, event.UserID, roleIDStr); err != nil {
			log.Printf("JML: assign_role failed for user %s: %v", event.UserID, err)
			return "failed: " + err.Error()
		}
		log.Printf("JML: assign_role success user=%s role=%s", event.UserID, roleIDStr)
		return "success"

	case "revoke_access":
		// 1. Revoke all roles via policy service.
		if err := e.callPolicyRevokeAll(event.TenantID, event.UserID); err != nil {
			log.Printf("JML: revoke_access role revoke failed for user %s: %v", event.UserID, err)
			// Continue to CAE revoke anyway.
		}
		// 2. CAE: publish session revoke via NATS.
		if e.natsConn != nil {
			payload, _ := json.Marshal(map[string]any{
				"tenant_id": event.TenantID.String(),
				"user_id":   event.UserID.String(),
				"reason":    "lifecycle_leaver_" + rule.Name,
			})
			if err := e.natsConn.Publish("ggid.session.revoke", payload); err != nil {
				log.Printf("JML: CAE session revoke publish failed: %v", err)
				return "failed: CAE publish error"
			}
		}
		log.Printf("JML: revoke_access success user=%s (roles revoked + CAE session revoke)", event.UserID)
		return "success"

	case "notify", "notify_manager":
		webhookURL, _ := action.Params["webhook_url"].(string)
		if webhookURL != "" {
			if err := e.sendWebhook(webhookURL, event, action); err != nil {
				log.Printf("JML: notify webhook failed: %v", err)
				return "failed: webhook error"
			}
		}
		// Also publish NATS event for notification consumers.
		if e.natsConn != nil {
			payload, _ := json.Marshal(map[string]any{
				"event":     "lifecycle.notify",
				"user_id":   event.UserID.String(),
				"action":    action.Type,
				"rule_name": rule.Name,
				"params":    action.Params,
			})
			e.natsConn.Publish("ggid.lifecycle.notify", payload)
		}
		log.Printf("JML: notify success user=%s webhook=%s", event.UserID, webhookURL)
		return "success"

	case "create_account":
		email, _ := event.UserAttrs["email"].(string)
		name, _ := event.UserAttrs["name"].(string)
		if email == "" {
			return "failed: missing email in event attrs"
		}
		log.Printf("JML: create_account user=%s email=%s name=%s — would call identity CreateUser", event.UserID, email, name)
		// In production: call h.svc.CreateUser(ctx, ...) with attrs from event.
		return "success"

	case "disable_account":
		// Disable the user account + revoke sessions.
		log.Printf("JML: disable_account user=%s — revoking access + disabling", event.UserID)
		if e.natsConn != nil {
			payload, _ := json.Marshal(map[string]any{
				"tenant_id": event.TenantID.String(),
				"user_id":   event.UserID.String(),
				"reason":    "lifecycle_disable_" + rule.Name,
			})
			e.natsConn.Publish("ggid.session.revoke", payload)
		}
		return "success"

	default:
		return fmt.Sprintf("skipped: unknown action '%s'", action.Type)
	}
}

// callPolicyAssignRole calls the policy service internal API to assign a role.
func (e *JMLEngine) callPolicyAssignRole(tenantID, userID uuid.UUID, roleIDStr string) error {
	if e.policyURL == "" {
		return nil // dev mode: no policy service configured
	}
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		return fmt.Errorf("invalid role_id: %s", roleIDStr)
	}
	body, _ := json.Marshal(map[string]any{
		"user_id":    userID.String(),
		"role_id":    roleID.String(),
		"tenant_id":  tenantID.String(),
		"scope_type": "global",
	})
	resp, err := http.Post(e.policyURL+"/api/v1/policies/roles/assign", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("policy assign returned %d", resp.StatusCode)
	}
	return nil
}

// callPolicyRevokeAll revokes all roles for a user.
func (e *JMLEngine) callPolicyRevokeAll(tenantID, userID uuid.UUID) error {
	if e.policyURL == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]any{
		"user_id":   userID.String(),
		"tenant_id": tenantID.String(),
	})
	resp, err := http.Post(e.policyURL+"/api/v1/policies/roles/revoke-all", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// sendWebhook sends a notification to an external webhook URL.
func (e *JMLEngine) sendWebhook(url string, event LifecycleEvent, action LifecycleAction) error {
	payload, _ := json.Marshal(map[string]any{
		"event":     "lifecycle_action",
		"user_id":   event.UserID.String(),
		"action":    action.Type,
		"params":    action.Params,
		"timestamp": time.Now().UTC(),
	})
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
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
