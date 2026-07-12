package server

import (
	"net/http"
	"time"
)

// GET /api/v1/auth/mfa/enrollment-stats
func (h *Handler) handleMFAEnrollmentStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	totalUsers := 15420
	enrolledUsers := 9620

	writeJSON(w, http.StatusOK, map[string]any{
		"total_users":     totalUsers,
		"enrolled_users":  enrolledUsers,
		"unenrolled_users": totalUsers - enrolledUsers,
		"enrollment_rate_pct": float64(enrolledUsers) / float64(totalUsers) * 100,
		"method_distribution": []map[string]any{
			{"method": "totp", "users": 5800, "percentage": 60.3},
			{"method": "sms", "users": 2100, "percentage": 21.8},
			{"method": "email", "users": 850, "percentage": 8.8},
			{"method": "webauthn", "users": 420, "percentage": 4.4},
			{"method": "backup_codes", "users": 450, "percentage": 4.7},
		},
		"avg_methods_per_user": 1.35,
		"multi_factor_users":   3200,
		"pending_enrollments": []map[string]any{
			{"user_id": "user-015", "username": "new.hire1", "method": "totp", "initiated_at": time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339), "expires_in_hours": 22},
			{"user_id": "user-016", "username": "new.hire2", "method": "sms", "initiated_at": time.Now().UTC().Add(-5 * time.Hour).Format(time.RFC3339), "expires_in_hours": 19},
			{"user_id": "user-017", "username": "contractor1", "method": "email", "initiated_at": time.Now().UTC().Add(-12 * time.Hour).Format(time.RFC3339), "expires_in_hours": 12},
		},
		"pending_count":    3,
		"enforcement": map[string]any{
			"required_for_admin":      true,
			"required_for_all":        false,
			"grace_period_days":       7,
			"enforced_users":          420,
			"enforcement_deadline":    time.Now().UTC().Add(7 * 24 * time.Hour).Format("2006-01-02"),
		},
		"by_org": []map[string]any{
			{"org": "Security", "enrolled": 350, "total": 350, "rate_pct": 100.0},
			{"org": "Engineering", "enrolled": 4800, "total": 5200, "rate_pct": 92.3},
			{"org": "Sales", "enrolled": 2100, "total": 3100, "rate_pct": 67.7},
			{"org": "Marketing", "enrolled": 980, "total": 1800, "rate_pct": 54.4},
		},
		"checked_at": time.Now().UTC().Format(time.RFC3339),
	})
}
