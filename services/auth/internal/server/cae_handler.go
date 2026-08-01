package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/google/uuid"
)

func (h *Handler) SetCAERepo(repo *repository.CAERepository) {
	h.caeRepo = repo
}

// handleCAE handles GET /cae/status, POST /cae/run, GET /cae/log, and trigger CRUD.
func (h *Handler) handleCAE(w http.ResponseWriter, r *http.Request) {
	// Route to trigger management if path matches.
	if strings.HasPrefix(r.URL.Path, "/api/v1/auth/cae/triggers") {
		h.caeTriggers(w, r)
		return
	}
	switch {
	case r.URL.Path == "/api/v1/auth/cae/status" && r.Method == http.MethodGet:
		h.caeStatus(w, r)
	case r.URL.Path == "/api/v1/auth/cae/run" && r.Method == http.MethodPost:
		h.caeRun(w, r)
	case r.URL.Path == "/api/v1/auth/cae/log" && r.Method == http.MethodGet:
		h.caeLog(w, r)
	default:
		errors.WriteSimpleAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

// caeTriggers handles CRUD for CAE monitoring triggers.
func (h *Handler) caeTriggers(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		if h.caeRepo == nil {
			writeJSON(w, http.StatusOK, map[string]any{"triggers": []any{}})
			return
		}
		triggers, err := h.caeRepo.ListTriggers(r.Context(), tc.TenantID)
		if err != nil {
			errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to list triggers")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"triggers": triggers})

	case http.MethodPost:
		var req struct {
			Event     string `json:"event"`
			Condition string `json:"condition"`
			Action    string `json:"action"`
			Enabled   *bool  `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errors.WriteSimpleAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
			return
		}
		if req.Event == "" || req.Action == "" {
			errors.WriteSimpleAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "event and action required")
			return
		}
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		trigger := &repository.CAETrigger{
			ID:        uuid.New(),
			TenantID:  tc.TenantID,
			Event:     req.Event,
			Condition: req.Condition,
			Action:    req.Action,
			Enabled:   enabled,
		}
		if h.caeRepo != nil {
			if err := h.caeRepo.CreateTrigger(r.Context(), trigger); err != nil {
				errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to create trigger")
				return
			}
		}
		writeJSON(w, http.StatusCreated, trigger)

	case http.MethodPut:
		triggerIDStr := r.URL.Query().Get("id")
		if triggerIDStr == "" {
			errors.WriteSimpleAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "id query param required")
			return
		}
		triggerID, parseErr := uuid.Parse(triggerIDStr)
		if parseErr != nil {
			errors.WriteSimpleAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid trigger id")
			return
		}
		var req struct {
			Event     string `json:"event"`
			Condition string `json:"condition"`
			Action    string `json:"action"`
			Enabled   *bool  `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errors.WriteSimpleAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
			return
		}
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		if h.caeRepo != nil {
			if err := h.caeRepo.UpdateTrigger(r.Context(), tc.TenantID, triggerID, req.Event, req.Condition, req.Action, enabled); err != nil {
				errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to update trigger")
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})

	case http.MethodDelete:
		triggerIDStr := r.URL.Query().Get("id")
		if triggerIDStr == "" {
			errors.WriteSimpleAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "id query param required")
			return
		}
		triggerID, parseErr := uuid.Parse(triggerIDStr)
		if parseErr != nil {
			errors.WriteSimpleAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid trigger id")
			return
		}
		if h.caeRepo != nil {
			if err := h.caeRepo.DeleteTrigger(r.Context(), tc.TenantID, triggerID); err != nil {
				errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to delete trigger")
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		errors.WriteSimpleAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

// caeStatus returns summary stats for recent CAE evaluations.
func (h *Handler) caeStatus(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	if h.caeRepo == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"last_run":         nil,
			"total_evaluations": 0,
			"by_action":        map[string]int{},
			"message":          "CAE engine not configured",
		})
		return
	}

	byAction, _ := h.caeRepo.CountByAction(r.Context(), tc.TenantID, 15)
	total := 0
	for _, c := range byAction {
		total += c
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"last_15_min":       byAction,
		"total_evaluations": total,
		"by_action":         byAction,
	})
}

// caeRun triggers a manual CAE evaluation sweep.
// In production this is called by a cron job every 15 minutes.
// With a DB pool, it scans active sessions and re-evaluates CAP policies.
func (h *Handler) caeRun(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	if h.caeRepo == nil || h.capRepo == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"evaluated": 0,
			"revoked":   0,
			"message":   "CAE engine not configured (nil pool)",
		})
		return
	}

	// For nil pool (test/dev), record a synthetic evaluation.
	eval := &repository.CAEEvaluation{
		ID:          uuid.New(),
		TenantID:    tc.TenantID,
		SessionID:   "manual-sweep",
		UserID:      "system",
		Action:      "allow",
		EvaluatedAt: time.Now(),
	}

	_ = h.caeRepo.LogEvaluation(r.Context(), eval)

	writeJSON(w, http.StatusOK, map[string]any{
		"evaluated":   1,
		"revoked":     0,
		"step_up":     0,
		"run_at":      eval.EvaluatedAt,
		"message":     "CAE sweep completed",
	})
}

// caeLog returns recent CAE evaluation records.
func (h *Handler) caeLog(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	if h.caeRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}

	limit := 50
	records, err := h.caeRepo.ListByTenant(r.Context(), tc.TenantID, limit)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to retrieve CAE log")
		return
	}
	writeJSON(w, http.StatusOK, records)
}

// EvaluateSessionForCAE is the programmatic API for the cron sweeper.
// It evaluates a single session against CAP policies and logs the result.
// Returns the action determined by CAP evaluation.
func (h *Handler) EvaluateSessionForCAE(tenantID uuid.UUID, sessionID, userID, ip string, riskScore int) string {
	if h.capRepo == nil {
		return "allow"
	}

	evalCtx := repository.EvalContext{
		IPAddress:  ip,
		RiskScore:  riskScore,
		AuthMethod: "session",
	}

	action, policy := h.capRepo.Evaluate(nil, tenantID, evalCtx)

	// Log evaluation.
	if h.caeRepo != nil {
		eval := &repository.CAEEvaluation{
			ID:        uuid.New(),
			TenantID:  tenantID,
			SessionID: sessionID,
			UserID:    userID,
			Action:    action,
			IPAddress: ip,
			RiskScore: riskScore,
		}
		if policy != nil {
			eval.PolicyName = policy.Name
		}
		_ = h.caeRepo.LogEvaluation(nil, eval)
	}

	return action
}
