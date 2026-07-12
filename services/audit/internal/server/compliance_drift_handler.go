package httpserver

import (
	"net/http"
	"time"
)

// GET /api/v1/audit/compliance/drift?framework=X
// Compares last assessment vs current status, returns drift score + changed controls.
func (s *HTTPServer) handleComplianceDrift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	framework := r.URL.Query().Get("framework")
	if framework == "" {
		framework = "soc2"
	}

	// Simulated previous vs current assessment data
	changedControls := []map[string]any{
		{
			"control_id":   "CC3.1",
			"control_name": "Risk Assessment",
			"was":          "non_compliant",
			"now":          "warning",
			"direction":    "improved",
			"delta":        +15,
		},
		{
			"control_id":   "CC6.2",
			"control_name": "Network Security",
			"was":          "compliant",
			"now":          "warning",
			"direction":    "regressed",
			"delta":        -10,
		},
		{
			"control_id":   "CC7.1",
			"control_name": "System Operations",
			"was":          "warning",
			"now":          "compliant",
			"direction":    "improved",
			"delta":        +20,
		},
		{
			"control_id":   "CC8.1",
			"control_name": "Risk Mitigation",
			"was":          "compliant",
			"now":          "non_compliant",
			"direction":    "regressed",
			"delta":        -25,
		},
		{
			"control_id":   "CC5.1",
			"control_name": "Control Activities",
			"was":          "warning",
			"now":          "warning",
			"direction":    "unchanged",
			"delta":        0,
		},
	}

	// Compute drift score: sum of absolute deltas / total controls
	totalControls := 12
	changedCount := 0
	regressedCount := 0
	improvedCount := 0
	totalDelta := 0
	for _, c := range changedControls {
		dir, _ := c["direction"].(string)
		delta, _ := c["delta"].(int)
		if dir != "unchanged" {
			changedCount++
			totalDelta += abs(delta)
		}
		if dir == "regressed" {
			regressedCount++
		} else if dir == "improved" {
			improvedCount++
		}
	}

	driftScore := totalDelta * 100 / (totalControls * 25) // normalize to 0-100

	riskLevel := "low"
	if driftScore >= 40 {
		riskLevel = "high"
	} else if driftScore >= 20 {
		riskLevel = "medium"
	}

	// Previous and current scores
	prevScore := 78
	currentScore := 75

	writeJSON(w, http.StatusOK, map[string]any{
		"framework":         framework,
		"previous_score":    prevScore,
		"current_score":     currentScore,
		"score_delta":       currentScore - prevScore,
		"drift_score":       driftScore,
		"drift_risk_level":  riskLevel,
		"total_controls":    totalControls,
		"changed_controls":  changedControls,
		"changed_count":     changedCount,
		"regressed_count":   regressedCount,
		"improved_count":    improvedCount,
		"unchanged_count":   totalControls - changedCount,
		"previous_assessment": time.Now().UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		"current_assessment":  time.Now().UTC().Format(time.RFC3339),
		"summary":            map[string]string{
			"trend":          "slight_regression",
			"top_concern":    "CC8.1 regressed from compliant to non_compliant",
			"recommendation": "Review CC8.1 control remediation plan",
		},
	})
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
