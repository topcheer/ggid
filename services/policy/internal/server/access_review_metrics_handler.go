package httpserver

import (
	"net/http"
	"time"
)

// GET /api/v1/policies/access-reviews/metrics?from=X&to=Y
// Returns campaign statistics: total, completion_rate, avg_review_time, decisions, overdue.
func (s *HTTPServer) handleAccessReviewMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" {
		from = time.Now().UTC().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().UTC().Format("2006-01-02")
	}

	// Simulated campaign metrics
	writeJSON(w, http.StatusOK, map[string]any{
		"period": map[string]string{"from": from, "to": to},
		"total_campaigns":      12,
		"active_campaigns":     3,
		"completed_campaigns":  9,
		"completion_rate_pct":  75.0,
		"avg_review_time_hours": 4.5,
		"median_review_time_hours": 2.0,
		"total_reviewers":      48,
		"active_reviewers":     31,
		"total_items":          420,
		"items_reviewed":       315,
		"items_overdue":        28,
		"overdue_count":        28,
		"decisions_breakdown": map[string]int{
			"certify":  245,
			"revoke":   38,
			"modify":   32,
			"pending":  105,
		},
		"by_framework": []map[string]any{
			{"framework": "SOC2", "campaigns": 5, "items": 180, "completion_pct": 82.2},
			{"framework": "GDPR", "campaigns": 3, "items": 120, "completion_pct": 75.0},
			{"framework": "HIPAA", "campaigns": 2, "items": 70, "completion_pct": 71.4},
			{"framework": "ISO27001", "campaigns": 2, "items": 50, "completion_pct": 60.0},
		},
		"trend": []map[string]any{
			{"month": "2026-01", "completion_pct": 65.0, "avg_hours": 6.2},
			{"month": "2026-02", "completion_pct": 70.0, "avg_hours": 5.5},
			{"month": "2026-03", "completion_pct": 68.0, "avg_hours": 5.0},
			{"month": "2026-04", "completion_pct": 75.0, "avg_hours": 4.8},
			{"month": "2026-05", "completion_pct": 73.0, "avg_hours": 4.5},
			{"month": "2026-06", "completion_pct": 78.0, "avg_hours": 4.2},
		},
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}
