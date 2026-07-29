package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// nullIfEmpty returns nil for an empty string so SQL `($N IS NULL OR ...)`
// optional filters work.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// POST /api/v1/audit/retention/execute — manually trigger retention policy execution.
// Reads all enabled retention policies, executes cleanup based on retention_days + action.
func (s *HTTPServer) handleRetentionExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Header is authoritative; query param must match when present.
	// Otherwise a caller could trigger another tenant's policy deletes.
	headerTenant := r.Header.Get("X-Tenant-ID")
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if headerTenant == "" {
		writeJSONError(w, http.StatusForbidden, "tenant context required")
		return
	}
	if tenantIDStr == "" || tenantIDStr != headerTenant {
		tenantIDStr = headerTenant
	}

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
			if !p.Enabled {
				continue
			}
			// P1: Skip policies with empty tenant_id (they would affect all tenants)
			if p.TenantID == "" {
				continue
			}
			if tenantIDStr != "" && p.TenantID != tenantIDStr {
				continue
			}
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
		if p.TenantID == "" {
			// Fail-closed (R150): a policy without a tenant would delete or
			// anonymize EVERY tenant's events. Skip and surface it.
			slog.Warn("retention execute: skipping policy without tenant_id", "policy_id", p.ID)
			executedPolicies = append(executedPolicies, map[string]any{
				"policy_id": p.ID, "event_type": p.EventType,
				"action": p.Action, "error": "policy has no tenant_id",
			})
			continue
		}
		if p.RetentionDays <= 0 {
			continue // unlimited
		}

		cutoff := now.AddDate(0, 0, -p.RetentionDays)

		if p.Action == "delete" {
			// Scoped delete in a tx: SET LOCAL only works inside a
			// transaction, and the DELETE must honor the policy's tenant
			// and event_type — the old single-Exec multi-statement form
			// failed under pgx AND would have deleted every tenant's
			// events (P0). Error details go to logs, not the client.
			tx, terr := s.pool.Begin(r.Context())
			if terr != nil {
				slog.Error("retention execute: begin tx failed", "policy_id", p.ID, "error", terr)
				executedPolicies = append(executedPolicies, map[string]any{
					"policy_id": p.ID, "event_type": p.EventType,
					"action": p.Action, "error": "execution failed",
				})
				continue
			}
			if _, terr = tx.Exec(r.Context(), `SET LOCAL app.allow_audit_mutation = 'on'`); terr == nil {
				var tag pgconn.CommandTag
				if p.TenantID == "" {
					tag, terr = tx.Exec(r.Context(),
						`DELETE FROM audit_events WHERE created_at < $1 AND ($2::text IS NULL OR event_type = $2)`, cutoff, nullIfEmpty(p.EventType))
				} else {
					tag, terr = tx.Exec(r.Context(),
						`DELETE FROM audit_events WHERE created_at < $1 AND tenant_id = $2 AND ($3::text IS NULL OR event_type = $3)`, cutoff, p.TenantID, nullIfEmpty(p.EventType))
				}
				if terr == nil {
					totalDeleted += int(tag.RowsAffected())
					terr = tx.Commit(r.Context())
				}
			}
			if terr != nil {
				_ = tx.Rollback(r.Context())
				slog.Error("retention execute: delete failed", "policy_id", p.ID, "error", terr)
				executedPolicies = append(executedPolicies, map[string]any{
					"policy_id": p.ID, "event_type": p.EventType,
					"action": p.Action, "error": "execution failed",
				})
				continue
			}
		} else if p.Action == "anonymize" {
			// Anonymize: nullify PII fields but keep the event for compliance.
			// Same scoping rules as delete — tenant + event_type filters,
			// tx for SET LOCAL, generic client error (R149 P1-7).
			tx, terr := s.pool.Begin(r.Context())
			if terr != nil {
				slog.Error("retention execute: begin tx failed", "policy_id", p.ID, "error", terr)
				executedPolicies = append(executedPolicies, map[string]any{
					"policy_id": p.ID, "event_type": p.EventType,
					"action": p.Action, "error": "execution failed",
				})
				continue
			}
			if _, terr = tx.Exec(r.Context(), `SET LOCAL app.allow_audit_mutation = 'on'`); terr == nil {
				var tag pgconn.CommandTag
				if p.TenantID == "" {
					tag, terr = tx.Exec(r.Context(),
						`UPDATE audit_events SET actor_ip = NULL, user_agent = NULL, metadata = '{}'::jsonb WHERE created_at < $1 AND ($2::text IS NULL OR event_type = $2)`, cutoff, nullIfEmpty(p.EventType))
				} else {
					tag, terr = tx.Exec(r.Context(),
						`UPDATE audit_events SET actor_ip = NULL, user_agent = NULL, metadata = '{}'::jsonb WHERE created_at < $1 AND tenant_id = $2 AND ($3::text IS NULL OR event_type = $3)`, cutoff, p.TenantID, nullIfEmpty(p.EventType))
				}
				if terr == nil {
					totalAnonymized += int(tag.RowsAffected())
					terr = tx.Commit(r.Context())
				}
			}
			if terr != nil {
				_ = tx.Rollback(r.Context())
				slog.Error("retention execute: anonymize failed", "policy_id", p.ID, "error", terr)
				executedPolicies = append(executedPolicies, map[string]any{
					"policy_id": p.ID, "event_type": p.EventType,
					"action": p.Action, "error": "execution failed",
				})
				continue
			}
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
