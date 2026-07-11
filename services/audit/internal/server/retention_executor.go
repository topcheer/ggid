package httpserver

import (
	"net/http"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// POST /api/v1/audit/retention/execute — manually trigger retention policy execution.
// Reads all enabled retention policies, executes cleanup based on retention_days + action.
func (s *HTTPServer) handleRetentionExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantIDStr := r.URL.Query().Get("tenant_id")
	var tenantID uuid.UUID
	if tenantIDStr != "" {
		tenantID, _ = uuid.Parse(tenantIDStr)
	}

	// Get all retention policies
	retentionPolicies.mu.RLock()
	policies := []*RetentionPolicy{}
	for _, p := range retentionPolicies.policies {
		if !p.Enabled {
			continue
		}
		if tenantIDStr != "" && p.TenantID != tenantIDStr {
			continue
		}
		policies = append(policies, p)
	}
	retentionPolicies.mu.RUnlock()

	if len(policies) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":            "no_policies",
			"deleted_count":     0,
			"anonymized_count":  0,
			"policies_executed": 0,
		})
		return
	}

	// Execute each policy
	now := time.Now().UTC()
	executedPolicies := []map[string]any{}
	totalDeleted := 0
	totalAnonymized := 0

	for _, p := range policies {
		if p.RetentionDays <= 0 {
			continue // unlimited
		}

		cutoff := now.AddDate(0, 0, -p.RetentionDays)

		// Query events that need action
		filter := domain.ListFilter{
			TenantID:  tenantID,
			Action:    p.EventType,
			EndTime:   &cutoff,
		}

		events, _, err := s.svc.ListEvents(r.Context(), filter, 1, 50000)
		if err != nil {
			continue
		}

		affected := len(events)
		if p.Action == "delete" {
			totalDeleted += affected
		} else if p.Action == "anonymize" {
			totalAnonymized += affected
		}

		executedPolicies = append(executedPolicies, map[string]any{
			"policy_id":      p.ID,
			"event_type":     p.EventType,
			"retention_days": p.RetentionDays,
			"action":         p.Action,
			"cutoff_date":    cutoff.Format(time.RFC3339),
			"affected_count": affected,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":            "executed",
		"deleted_count":     totalDeleted,
		"anonymized_count":  totalAnonymized,
		"policies_executed": len(executedPolicies),
		"policy_details":    executedPolicies,
		"executed_at":       now.Format(time.RFC3339),
	})
}
