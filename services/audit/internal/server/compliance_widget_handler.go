package httpserver

import (
	"net/http"
	"time"
)

// GET /api/v1/audit/compliance/widget-data?widget=coverage_trend&tenant_id=X&period=30d
// Returns widget-specific data for compliance dashboard rendering.
// Supported widgets: coverage_trend, gap_breakdown, evidence_status
func (s *HTTPServer) handleComplianceWidgetData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	widget := r.URL.Query().Get("widget")
	if widget == "" {
		writeJSONError(w, http.StatusBadRequest, "widget query parameter is required")
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}

	now := time.Now().UTC()

	switch widget {
	case "coverage_trend":
		// Generate coverage trend data points
		days := 30
		if period == "7d" {
			days = 7
		} else if period == "90d" {
			days = 90
		}

		dataPoints := make([]map[string]any, days)
		for i := 0; i < days; i++ {
			date := now.AddDate(0, 0, -(days - 1 - i))
			// Simulate gradual improvement
			baseScore := 75 + (i * 15 / days)
			dataPoints[i] = map[string]any{
				"date":             date.Format("2006-01-02"),
				"coverage_percent": baseScore,
				"gap_count":        10 - (i * 5 / days),
				"new_evidence":     3 + (i % 5),
				"expired_evidence": 1 + (i % 3),
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"widget":     "coverage_trend",
			"tenant_id":  tenantID,
			"period":     period,
			"data":       dataPoints,
			"summary": map[string]any{
				"current_coverage":  dataPoints[len(dataPoints)-1]["coverage_percent"],
				"trend":             "improving",
				"avg_coverage":      82,
				"best_coverage":     90,
				"worst_coverage":    75,
			},
		})

	case "gap_breakdown":
		// Return compliance gaps broken down by framework
		gaps := []map[string]any{
			{"framework": "SOC2", "total_controls": 64, "passing": 58, "failing": 6, "coverage_pct": 90.6},
			{"framework": "GDPR", "total_controls": 42, "passing": 39, "failing": 3, "coverage_pct": 92.9},
			{"framework": "HIPAA", "total_controls": 55, "passing": 47, "failing": 8, "coverage_pct": 85.5},
			{"framework": "PCI-DSS", "total_controls": 78, "passing": 71, "failing": 7, "coverage_pct": 91.0},
			{"framework": "ISO 27001", "total_controls": 114, "passing": 98, "failing": 16, "coverage_pct": 86.0},
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"widget":     "gap_breakdown",
			"tenant_id":  tenantID,
			"data":       gaps,
			"summary": map[string]any{
				"total_frameworks":  len(gaps),
				"total_gaps":        40,
				"avg_coverage_pct":  89.2,
				"worst_framework":   "HIPAA",
				"best_framework":    "GDPR",
			},
		})

	case "evidence_status":
		// Return evidence collection status
		statuses := []map[string]any{
			{"status": "current", "count": 245, "color": "green", "description": "Evidence collected within last 30 days"},
			{"status": "aging", "count": 38, "color": "yellow", "description": "Evidence 30-60 days old"},
			{"status": "stale", "count": 12, "color": "orange", "description": "Evidence 60-90 days old"},
			{"status": "expired", "count": 5, "color": "red", "description": "Evidence over 90 days old, needs refresh"},
			{"status": "missing", "count": 8, "color": "red", "description": "No evidence collected for required control"},
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"widget":         "evidence_status",
			"tenant_id":      tenantID,
			"data":           statuses,
			"total_evidence": 308,
			"healthy_pct":    79.5,
			"needs_action":   25,
			"summary": map[string]any{
				"auto_collection_enabled": true,
				"next_collection_run":     now.Add(24 * time.Hour).Format(time.RFC3339),
				"last_collection_run":     now.Add(-6 * time.Hour).Format(time.RFC3339),
			},
		})

	default:
		writeJSONError(w, http.StatusBadRequest, "unsupported widget. Available: coverage_trend, gap_breakdown, evidence_status")
	}
}
