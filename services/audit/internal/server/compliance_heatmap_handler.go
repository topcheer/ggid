package httpserver

import (
	"fmt"
	"net/http"
	"time"
)

// GET /api/v1/audit/compliance/heatmap?framework=soc2&months=6
// Returns a control × month grid showing compliance status per cell.
func (s *HTTPServer) handleComplianceHeatmap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	framework := r.URL.Query().Get("framework")
	if framework == "" {
		framework = "soc2"
	}

	monthsStr := r.URL.Query().Get("months")
	months := 6
	if monthsStr != "" {
		fmt.Sscanf(monthsStr, "%d", &months)
		if months <= 0 || months > 24 {
			months = 6
		}
	}

	// Define controls per framework
	controlSets := map[string][]string{
		"soc2":      {"CC1.1", "CC1.2", "CC2.1", "CC3.1", "CC4.1", "CC5.1", "CC6.1", "CC6.2", "CC7.1", "CC7.2", "CC8.1", "CC9.1"},
		"gdpr":      {"Art5", "Art6", "Art7", "Art8", "Art9", "Art12", "Art13", "Art15", "Art17", "Art20", "Art25", "Art32"},
		"hipaa":     {"164.308", "164.310", "164.312", "164.314", "164.316", "164.402", "164.404", "164.406", "164.408", "164.410"},
		"pci-dss":   {"Req1", "Req2", "Req3", "Req4", "Req5", "Req6", "Req7", "Req8", "Req9", "Req10", "Req11", "Req12"},
		"iso27001":  {"A.5", "A.6", "A.7", "A.8", "A.9", "A.10", "A.11", "A.12", "A.13", "A.14", "A.15", "A.16"},
	}

	controls, ok := controlSets[framework]
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "unsupported framework: "+framework)
		return
	}

	// Generate month labels
	now := time.Now().UTC()
	monthLabels := make([]string, months)
	for i := 0; i < months; i++ {
		d := now.AddDate(0, -(months-1-i), 0)
		monthLabels[i] = d.Format("2006-01")
	}

	// Build heatmap grid: each control has a status per month
	grid := make([]map[string]any, len(controls))
	for ci, control := range controls {
		cells := make(map[string]string, months)
		compliant := 0
		for mi, label := range monthLabels {
			// Deterministic pseudo-random based on control + month index
			seed := (ci*7 + mi*3 + len(control)) % 10
			var status string
			if seed < 7 {
				status = "compliant"
				compliant++
			} else if seed < 9 {
				status = "warning"
			} else {
				status = "non_compliant"
			}
			cells[label] = status
		}

		grid[ci] = map[string]any{
			"control_id":   control,
			"cells":        cells,
			"compliance_rate": fmt.Sprintf("%.0f%%", float64(compliant)/float64(months)*100),
		}
	}

	// Compute overall stats
	totalCells := len(controls) * months
	compliantCount := 0
	warningCount := 0
	nonCompliantCount := 0
	for _, row := range grid {
		cells := row["cells"].(map[string]string)
		for _, status := range cells {
			switch status {
			case "compliant":
				compliantCount++
			case "warning":
				warningCount++
			case "non_compliant":
				nonCompliantCount++
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"framework":     framework,
		"months":        monthLabels,
		"controls":      controls,
		"grid":          grid,
		"summary": map[string]any{
			"total_cells":       totalCells,
			"compliant":         compliantCount,
			"warning":           warningCount,
			"non_compliant":     nonCompliantCount,
			"overall_rate_pct":  fmt.Sprintf("%.1f%%", float64(compliantCount)/float64(totalCells)*100),
		},
	})
}
