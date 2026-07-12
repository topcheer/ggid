package httpserver

import (
	"net/http"
	"time"
)

// GET /api/v1/audit/compliance/remediation-progress?framework=X
func (s *HTTPServer) handleRemediationProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	framework := r.URL.Query().Get("framework")
	if framework == "" {
		framework = "soc2"
	}

	// Simulated remediation data
	gaps := []map[string]any{
		{"id": "gap-001", "control": "CC3.1", "title": "Risk Assessment Documentation", "status": "resolved", "assigned_to": "sec-team", "resolved_at": time.Now().UTC().Add(-5 * 24 * time.Hour).Format(time.RFC3339), "resolution_days": 12},
		{"id": "gap-002", "control": "CC6.2", "title": "Network Segmentation Gap", "status": "in_progress", "assigned_to": "infra-team", "started_at": time.Now().UTC().Add(-8 * 24 * time.Hour).Format(time.RFC3339), "est_completion": time.Now().UTC().Add(14 * 24 * time.Hour).Format("2006-01-02")},
		{"id": "gap-003", "control": "CC8.1", "title": "Risk Mitigation Policy", "status": "in_progress", "assigned_to": "compliance", "started_at": time.Now().UTC().Add(-3 * 24 * time.Hour).Format(time.RFC3339), "est_completion": time.Now().UTC().Add(7 * 24 * time.Hour).Format("2006-01-02")},
		{"id": "gap-004", "control": "CC7.1", "title": "Monitoring Coverage", "status": "resolved", "assigned_to": "ops-team", "resolved_at": time.Now().UTC().Add(-15 * 24 * time.Hour).Format(time.RFC3339), "resolution_days": 8},
		{"id": "gap-005", "control": "CC5.1", "title": "Control Activities Review", "status": "new", "assigned_to": "", "created_at": time.Now().UTC().Add(-1 * 24 * time.Hour).Format(time.RFC3339)},
		{"id": "gap-006", "control": "CC9.1", "title": "Risk Management Framework", "status": "resolved", "assigned_to": "sec-team", "resolved_at": time.Now().UTC().Add(-20 * 24 * time.Hour).Format(time.RFC3339), "resolution_days": 15},
		{"id": "gap-007", "control": "CC4.1", "title": "Continuous Monitoring", "status": "new", "assigned_to": "", "created_at": time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)},
	}

	totalGaps := len(gaps)
	resolved := 0
	inProgress := 0
	newCount := 0
	totalResolutionDays := 0
	resolvedCount := 0

	for _, g := range gaps {
		switch g["status"] {
		case "resolved":
			resolved++
			if days, ok := g["resolution_days"].(int); ok {
				totalResolutionDays += days
				resolvedCount++
			}
		case "in_progress":
			inProgress++
		case "new":
			newCount++
		}
	}

	avgResolutionDays := 0
	if resolvedCount > 0 {
		avgResolutionDays = totalResolutionDays / resolvedCount
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"framework":          framework,
		"total_gaps":         totalGaps,
		"resolved":           resolved,
		"in_progress":        inProgress,
		"new":                newCount,
		"resolution_rate_pct": float64(resolved) / float64(totalGaps) * 100,
		"avg_resolution_days": avgResolutionDays,
		"gaps":               gaps,
		"by_status": map[string]int{
			"resolved":    resolved,
			"in_progress": inProgress,
			"new":         newCount,
		},
		"by_assignee": map[string]int{
			"sec-team":   2,
			"infra-team": 1,
			"compliance": 1,
			"ops-team":   1,
			"unassigned": 2,
		},
		"trend": []map[string]any{
			{"week": "W-4", "resolved": 1, "new": 3},
			{"week": "W-3", "resolved": 2, "new": 1},
			{"week": "W-2", "resolved": 1, "new": 2},
			{"week": "W-1", "resolved": 1, "new": 1},
		},
		"checked_at": time.Now().UTC().Format(time.RFC3339),
	})
}
