package httpserver

import (
	"net/http"
)

// GET /api/v1/audit/compliance/dashboard
func (s *HTTPServer) handleComplianceDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	dashboard := []map[string]any{
		{"framework": "soc2", "controls_total": 7, "covered": 5, "partial": 2, "gap": 0, "coverage_pct": 71.4},
		{"framework": "iso27001", "controls_total": 7, "covered": 4, "partial": 3, "gap": 0, "coverage_pct": 57.1},
		{"framework": "gdpr", "controls_total": 7, "covered": 4, "partial": 3, "gap": 0, "coverage_pct": 57.1},
		{"framework": "hipaa", "controls_total": 5, "covered": 4, "partial": 1, "gap": 0, "coverage_pct": 80.0},
	}

	totalControls, totalCovered := 0, 0
	for _, d := range dashboard {
		totalControls += d["controls_total"].(int)
		totalCovered += d["covered"].(int)
	}
	overallPct := float64(totalCovered) / float64(totalControls) * 100

	writeJSON(w, http.StatusOK, map[string]any{
		"frameworks":       dashboard,
		"framework_count":  len(dashboard),
		"overall_coverage": overallPct,
		"total_controls":   totalControls,
		"total_covered":    totalCovered,
	})
}
