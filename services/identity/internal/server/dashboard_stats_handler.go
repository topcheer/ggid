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
	}

	stats := dashboardStats{}

	// Real DB queries when pool is available
	if pool := h.svc.Pool(); pool != nil {
		tenantID := tenantIDFromContext(ctx)

		// Total users (non-deleted)
		row := pool.QueryRow(ctx, `
			SELECT count(*) FROM users WHERE deleted_at IS NULL AND ($1::uuid IS NULL OR tenant_id = $1)
		`, tenantID)
		_ = row.Scan(&stats.TotalUsers)

		// Active sessions (updated in last 24h)
		row = pool.QueryRow(ctx, `
			SELECT count(*) FROM sessions WHERE created_at > NOW() - INTERVAL '24 hours'
		`)
		_ = row.Scan(&stats.ActiveSessions)

		// Auth events from last 24h
		since := time.Now().Add(-24 * time.Hour)
		row = pool.QueryRow(ctx, `
			SELECT count(*) FILTER (WHERE event_type = 'login_failed') AS failed,
			       count(*) FILTER (WHERE event_type = 'login_success') AS success
			FROM auth_events WHERE created_at > $1
		`, since)
		_ = row.Scan(&stats.FailedLogins24h, &stats.SuccessfulLogins24h)

		// Pending access requests
		row = pool.QueryRow(ctx, `
			SELECT count(*) FROM access_requests WHERE status = 'pending'
		`)
		_ = row.Scan(&stats.PendingAccessRequests)

		// MFA enrollment rate
		if stats.TotalUsers > 0 {
			var mfaCount int
			row = pool.QueryRow(ctx, `
				SELECT count(DISTINCT user_id) FROM mfa_devices WHERE enabled = true
			`)
			_ = row.Scan(&mfaCount)
			stats.MFAEnrollmentRate = (mfaCount * 100) / stats.TotalUsers
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
