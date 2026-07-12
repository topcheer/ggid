package httpserver

import (
	"net/http"
	"time"
)

// GET /api/v1/audit/framework-coverage
func (s *HTTPServer) handleFrameworkCoverage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	frameworks := []map[string]any{
		{
			"framework": "SOC2", "total_controls": 64, "covered": 58, "gaps": 6,
			"coverage_pct": 90.6, "evidence_count": 245, "last_assessed": time.Now().UTC().Add(-10 * 24 * time.Hour).Format(time.RFC3339),
			"status": "certified_with_findings",
		},
		{
			"framework": "HIPAA", "total_controls": 55, "covered": 47, "gaps": 8,
			"coverage_pct": 85.5, "evidence_count": 180, "last_assessed": time.Now().UTC().Add(-15 * 24 * time.Hour).Format(time.RFC3339),
			"status": "in_remediation",
		},
		{
			"framework": "ISO 27001", "total_controls": 114, "covered": 98, "gaps": 16,
			"coverage_pct": 86.0, "evidence_count": 320, "last_assessed": time.Now().UTC().Add(-20 * 24 * time.Hour).Format(time.RFC3339),
			"status": "certified",
		},
		{
			"framework": "GDPR", "total_controls": 42, "covered": 39, "gaps": 3,
			"coverage_pct": 92.9, "evidence_count": 145, "last_assessed": time.Now().UTC().Add(-8 * 24 * time.Hour).Format(time.RFC3339),
			"status": "compliant",
		},
		{
			"framework": "PCI-DSS", "total_controls": 78, "covered": 71, "gaps": 7,
			"coverage_pct": 91.0, "evidence_count": 210, "last_assessed": time.Now().UTC().Add(-5 * 24 * time.Hour).Format(time.RFC3339),
			"status": "certified",
		},
	}

	totalControls := 0
	totalCovered := 0
	totalGaps := 0
	for _, fw := range frameworks {
		totalControls += fw["total_controls"].(int)
		totalCovered += fw["covered"].(int)
		totalGaps += fw["gaps"].(int)
	}

	overallPct := 0.0
	if totalControls > 0 {
		overallPct = float64(totalCovered) / float64(totalControls) * 100
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"frameworks":         frameworks,
		"total_frameworks":   len(frameworks),
		"total_controls":     totalControls,
		"total_covered":      totalCovered,
		"total_gaps":         totalGaps,
		"overall_coverage_pct": overallPct,
		"best_framework":     "GDPR",
		"worst_framework":    "HIPAA",
		"checked_at":         time.Now().UTC().Format(time.RFC3339),
	})
}
