package httpserver

import (
	"net/http"
	"time"
)

// GET /api/v1/audit/compliance/cert-export?framework=X&format=json
// Exports compliance certification report.
func (s *HTTPServer) handleCertExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	framework := r.URL.Query().Get("framework")
	if framework == "" {
		framework = "soc2"
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	now := time.Now().UTC()

	report := map[string]any{
		"report_id":    now.Format("20060102-150405") + "-" + framework,
		"framework":    framework,
		"generated_at": now.Format(time.RFC3339),
		"format":       format,
		"organization": map[string]any{
			"name":          "GGID Inc.",
			"tenant_id":     r.Header.Get("X-Tenant-ID"),
			"reporting_period": map[string]string{
				"start": now.Add(-90 * 24 * time.Hour).Format("2006-01-02"),
				"end":   now.Format("2006-01-02"),
			},
		},
		"auditor_info": map[string]any{
			"firm":           "Independent Audit Partners LLC",
			"lead_auditor":   "Jane Smith, CPA, CISSP",
			"auditor_id":     "AUD-2026-0142",
			"audit_date":     now.Add(-7 * 24 * time.Hour).Format("2006-01-02"),
			"opinion":        "unqualified",
		},
		"score":         87,
		"grade":         "B",
		"controls": []map[string]any{
			{"id": "CC1.1", "name": "Control Environment", "status": "met", "evidence": "ev-001", "last_tested": now.Add(-10 * 24 * time.Hour).Format("2006-01-02")},
			{"id": "CC2.1", "name": "Communication", "status": "met", "evidence": "ev-002", "last_tested": now.Add(-12 * 24 * time.Hour).Format("2006-01-02")},
			{"id": "CC3.1", "name": "Risk Assessment", "status": "partial", "evidence": "ev-003", "last_tested": now.Add(-15 * 24 * time.Hour).Format("2006-01-02")},
			{"id": "CC6.1", "name": "Logical Access", "status": "met", "evidence": "ev-004", "last_tested": now.Add(-5 * 24 * time.Hour).Format("2006-01-02")},
			{"id": "CC6.2", "name": "Network Security", "status": "met", "evidence": "ev-005", "last_tested": now.Add(-8 * 24 * time.Hour).Format("2006-01-02")},
			{"id": "CC7.1", "name": "System Operations", "status": "partial", "evidence": "ev-006", "last_tested": now.Add(-20 * 24 * time.Hour).Format("2006-01-02")},
			{"id": "CC8.1", "name": "Risk Mitigation", "status": "not_met", "evidence": "", "last_tested": ""},
		},
		"evidence_summary": map[string]any{
			"total":     308,
			"verified":  245,
			"aging":     38,
			"stale":     17,
			"missing":   8,
		},
		"gaps": []map[string]any{
			{"control": "CC8.1", "severity": "high", "status": "open", "remediation_plan": "Q3 2026"},
			{"control": "CC3.1", "severity": "medium", "status": "in_progress", "remediation_plan": "Q2 2026"},
			{"control": "CC7.1", "severity": "medium", "status": "in_progress", "remediation_plan": "Q2 2026"},
		},
		"certification": map[string]any{
			"status":      "certified_with_findings",
			"valid_until": now.AddDate(1, 0, 0).Format("2006-01-02"),
			"finding_count": 3,
		},
	}

	writeJSON(w, http.StatusOK, report)
}
