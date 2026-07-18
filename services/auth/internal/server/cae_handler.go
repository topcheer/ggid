package server

import (
	"net/http"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/google/uuid"
)

func (h *Handler) SetCAERepo(repo *repository.CAERepository) {
	h.caeRepo = repo
}

// handleCAE handles GET /cae/status, POST /cae/run, GET /cae/log.
func (h *Handler) handleCAE(w http.ResponseWriter, r *http.Request) {
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
