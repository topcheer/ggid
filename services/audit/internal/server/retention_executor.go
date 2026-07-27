package httpserver

import (
	"net/http"
	"time"

)

// POST /api/v1/audit/retention/execute — manually trigger retention policy execution.
// Reads all enabled retention policies, executes cleanup based on retention_days + action.
func (s *HTTPServer) handleRetentionExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantIDStr := r.URL.Query().Get("tenant_id")

	// Get all retention policies from PG
	var policies []*RetentionPolicy
	if s.memMapRepo2 != nil {
		rows, _ := s.memMapRepo2.ListJSON(r.Context(), "audit_retention_policies")
		for _, row := range rows {
			p := &RetentionPolicy{
				ID: amGetString(row, "id"), TenantID: amGetString(row, "tenant_id"),
				EventType: amGetString(row, "event_type"), Action: amGetString(row, "action"),
				Enabled: amGetBool(row, "enabled"),
			}
			if !p.Enabled { continue }
			if tenantIDStr != "" && p.TenantID != tenantIDStr { continue }
			policies = append(policies, p)
		}
	}

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

		if p.Action == "delete" {
			// Actually delete old events (not just count them)
			tag, err := s.pool.Exec(r.Context(),
				`SET LOCAL app.allow_audit_mutation = 'on'; DELETE FROM audit_events WHERE created_at < $1`, cutoff)
			if err != nil {
				executedPolicies = append(executedPolicies, map[string]any{
					"policy_id": p.ID, "event_type": p.EventType,
					"action": p.Action, "error": err.Error(),
				})
				continue
			}
			totalDeleted += int(tag.RowsAffected())
		} else if p.Action == "anonymize" {
			// Anonymize: nullify PII fields but keep the event for compliance
			tag, err := s.pool.Exec(r.Context(),
				`UPDATE audit_events SET actor_ip = NULL, user_agent = NULL, metadata = '{}'::jsonb WHERE created_at < $1`, cutoff)
			if err != nil {
				executedPolicies = append(executedPolicies, map[string]any{
					"policy_id": p.ID, "event_type": p.EventType,
					"action": p.Action, "error": err.Error(),
				})
				continue
			}
			totalAnonymized += int(tag.RowsAffected())
		}

		executedPolicies = append(executedPolicies, map[string]any{
			"policy_id":      p.ID,
			"event_type":     p.EventType,
			"retention_days": p.RetentionDays,
			"action":         p.Action,
			"cutoff_date":    cutoff.Format(time.RFC3339),
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
