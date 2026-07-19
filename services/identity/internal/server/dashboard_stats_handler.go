package server

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// GET /api/v1/identity/dashboard/stats
// Returns aggregate statistics for the dashboard widget.
func (h *HTTPHandler) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	type dashboardStats struct {
		TotalUsers            int `json:"total_users"`
		ActiveSessions        int `json:"active_sessions"`
		FailedLogins24h       int `json:"failed_logins_24h"`
		SuccessfulLogins24h   int `json:"successful_logins_24h"`
		MFAEnrollmentRate     int `json:"mfa_enrollment_rate"`
		AuditEvents24h        int `json:"audit_events_24h"`
		PendingAccessRequests int `json:"pending_access_requests"`
		OAuthClients          int `json:"oauth_clients"`
	}

	stats := dashboardStats{}

		// Real DB queries when pool is available
	if pool := h.svc.Pool(); pool != nil {
		tenantID := tenantIDFromContext(ctx)
		// Fallback: parse from X-Tenant-ID header
		if tenantID == nil {
			if tidStr := r.Header.Get("X-Tenant-ID"); tidStr != "" {
				if tid, err := uuid.Parse(tidStr); err == nil {
					tenantID = &tid
				}
			}
		}

		// Set RLS context for this pooled connection's queries
		if tenantID != nil {
			_, _ = pool.Exec(ctx, `SET app.tenant_id = $1`, tenantID.String())
		}

		// Total users (non-deleted)
		_ = pool.QueryRow(ctx, `
			SELECT count(*) FROM users WHERE deleted_at IS NULL AND ($1::uuid IS NULL OR tenant_id = $1)
		`, tenantID).Scan(&stats.TotalUsers)

		// Active sessions (non-revoked, created in last 24h)
		_ = pool.QueryRow(ctx, `
			SELECT count(*) FROM sessions WHERE revoked_at IS NULL AND created_at > NOW() - INTERVAL '24 hours'
		`).Scan(&stats.ActiveSessions)

		// OAuth clients count
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM oauth_clients WHERE enabled = true`).Scan(&stats.OAuthClients)

		// Login stats from audit_events (auth_events table doesn't exist)
		since := time.Now().Add(-24 * time.Hour)
		_ = pool.QueryRow(ctx, `
			SELECT count(*) FILTER (WHERE result = 'failure') AS failed,
			       count(*) FILTER (WHERE result = 'success') AS success
			FROM audit_events WHERE action = 'user.login' AND created_at > $1
		`, since).Scan(&stats.FailedLogins24h, &stats.SuccessfulLogins24h)

		// Audit events from last 24h
		_ = pool.QueryRow(ctx, `
			SELECT count(*) FROM audit_events WHERE created_at > $1
		`, since).Scan(&stats.AuditEvents24h)

		// MFA enrollment rate from user_credentials table
		if stats.TotalUsers > 0 {
			var mfaCount int
			_ = pool.QueryRow(ctx, `
				SELECT count(*) FROM user_credentials WHERE mfa_enabled = true
			`).Scan(&mfaCount)
			if mfaCount > 0 {
				stats.MFAEnrollmentRate = (mfaCount * 100) / stats.TotalUsers
			}
		}
	}

	writeJSON(w, http.StatusOK, stats)
}

// tenantIDFromContext extracts tenant ID from context, returns nil if absent.
func tenantIDFromContext(ctx context.Context) *uuid.UUID {
	v := ctx.Value("tenant_id")
	if v == nil {
		return nil
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		return nil
	}
	return &id
}
