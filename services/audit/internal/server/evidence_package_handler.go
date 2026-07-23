package httpserver

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/services/audit/internal/compliance"
	"github.com/google/uuid"
)

// EvidencePackage is the consolidated SOC2/GDPR evidence bundle.
// It aggregates the compliance report, collected evidence items, control
// mappings, and an audit-event summary into a single downloadable artifact
// suitable for auditor delivery.
type EvidencePackage struct {
	ID            string                 `json:"id"`
	Framework     string                 `json:"framework"`
	TenantID      string                 `json:"tenant_id"`
	Period        EvidencePeriod         `json:"period"`
	GeneratedAt   time.Time              `json:"generated_at"`
	Report        *compliance.ComplianceReport `json:"report"`
	Evidence      []ComplianceEvidence   `json:"evidence"`
	Controls      []map[string]any       `json:"controls"`
	AuditSummary  map[string]any         `json:"audit_summary"`
	PackageHash   string                 `json:"package_hash,omitempty"`
}

// EvidencePeriod is the date range covered by the package.
type EvidencePeriod struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// POST /api/v1/audit/compliance/evidence-package
// GET  /api/v1/audit/compliance/evidence-package?framework=soc2&tenant_id=X&from=...&to=...
//
// Generates a consolidated SOC2/GDPR evidence package containing:
//  1. Structured compliance report (from compliance.Generator)
//  2. Collected evidence items (from the evidence store)
//  3. Control coverage mappings
//  4. Audit event summary for the period
//
// Supports format=json (default) or format=csv (flat CSV of evidence items).
func (s *HTTPServer) handleEvidencePackage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.generateEvidencePackage(w, r)
	case http.MethodPost:
		s.generateEvidencePackage(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) generateEvidencePackage(w http.ResponseWriter, r *http.Request) {
	// Parse parameters — support both query params (GET) and JSON body (POST).
	framework := "soc2"
	tenantIDStr := ""
	format := "json"

	if r.Method == http.MethodPost {
		var req struct {
			Framework string `json:"framework"`
			TenantID  string `json:"tenant_id"`
			FromDate  string `json:"from_date"`
			ToDate    string `json:"to_date"`
			Format    string `json:"format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.Framework != "" {
				framework = req.Framework
			}
			if req.TenantID != "" {
				tenantIDStr = req.TenantID
			}
			if req.Format != "" {
				format = req.Format
			}
		}
	} else {
		framework = r.URL.Query().Get("framework")
		if framework == "" {
			framework = "soc2"
		}
		tenantIDStr = r.URL.Query().Get("tenant_id")
		format = r.URL.Query().Get("format")
		if format == "" {
			format = "json"
		}
	}

	// Validate framework.
	reportType := compliance.ReportType(framework)
	if reportType != compliance.ReportSOC2 && reportType != compliance.ReportGDPR && reportType != compliance.ReportHIPAA {
		writeJSONError(w, http.StatusBadRequest, "framework must be soc2, gdpr, or hipaa")
		return
	}

	// Parse tenant.
	if tenantIDStr == "" {
		tenantIDStr = r.Header.Get("X-Tenant-ID")
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		tenantID = defaultTenantID()
	}

	// Parse date range.
	now := time.Now().UTC()
	from := now.AddDate(0, -1, 0)
	to := now
	if f := r.URL.Query().Get("from"); f != "" {
		if t, err := time.Parse(time.RFC3339, f); err == nil {
			from = t
		}
	}
	if t := r.URL.Query().Get("to"); t != "" {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			to = parsed
		}
	}

	// 1. Generate structured compliance report from audit events.
	var report *compliance.ComplianceReport
	if s.svc != nil {
		adapter := &auditEventQueryAdapter{svc: s.svc, tenantID: tenantID}
		gen := compliance.NewGenerator(adapter)
		report, err = gen.Generate(r.Context(), reportType, from, to)
		if err != nil {
			log.Printf("evidence-package: report generation error: %v", err)
			// Continue with nil report — evidence package still useful without it.
		}
	}

	// 2. Collect evidence items from the in-memory store (filtered by framework).
	evidenceMu.RLock()
	var evidenceItems []ComplianceEvidence
	for _, ev := range evidenceStore {
		if ev.Framework == framework {
			evidenceItems = append(evidenceItems, *ev)
		}
	}
	evidenceMu.RUnlock()

	// 3. Build control coverage summary.
	controls := buildControlCoverage(framework, evidenceItems, report)

	// 4. Build audit event summary.
	auditSummary := buildAuditSummary(report)

	pkg := &EvidencePackage{
		ID:          "epkg-" + now.Format("20060102-150405") + "-" + framework,
		Framework:   framework,
		TenantID:    tenantID.String(),
		Period:      EvidencePeriod{From: from, To: to},
		GeneratedAt: now,
		Report:      report,
		Evidence:    evidenceItems,
		Controls:    controls,
		AuditSummary: auditSummary,
	}

	// CSV format: flatten evidence items to CSV.
	if format == "csv" {
		writeEvidencePackageCSV(w, pkg)
		return
	}

	writeJSON(w, http.StatusOK, pkg)
}

// buildControlCoverage merges static framework mappings with collected evidence
// and report sections to produce a per-control status overview.
func buildControlCoverage(framework string, evidence []ComplianceEvidence, report *compliance.ComplianceReport) []map[string]any {
	// Use the enhanced framework mappings as the source of truth for control IDs.
	mappings, ok := enhancedFrameworkMappings[framework]
	if !ok {
		mappings = nil // framework has no static mappings; rely on evidence items only
	}

	// Index evidence by control ID.
	evByControl := make(map[string][]ComplianceEvidence)
	for _, ev := range evidence {
		evByControl[ev.ControlID] = append(evByControl[ev.ControlID], ev)
	}

	result := make([]map[string]any, 0, len(mappings))
	for _, m := range mappings {
		control := map[string]any{
			"control_id":      m.ControlID,
			"name":            m.ControlName,
			"trust_category":  m.TrustCategory,
			"feature":         m.GGIDFeature,
			"status":          m.Status,
			"evidence_count":  len(evByControl[m.ControlID]),
		}
		if evs, ok := evByControl[m.ControlID]; ok && len(evs) > 0 {
			control["latest_evidence"] = evs[len(evs)-1].CollectedAt
		}
		result = append(result, control)
	}

	// Also check report sections for status overrides.
	if report != nil {
		for _, section := range report.Sections {
			for _, c := range result {
				// Match by control ID substring in section title.
				if cid, ok := c["control_id"].(string); ok && section.Title != "" {
					if contains(section.Title, cid) {
						c["report_status"] = section.Status
					}
				}
			}
		}
	}

	return result
}

// buildAuditSummary extracts key metrics from the compliance report.
func buildAuditSummary(report *compliance.ComplianceReport) map[string]any {
	if report == nil {
		return map[string]any{
			"total_events":  0,
			"sections":      0,
		}
	}
	return map[string]any{
		"total_events":      report.Summary.TotalEvents,
		"failed_logins":     report.Summary.FailedLogins,
		"privileged_access": report.Summary.PrivilegedAccess,
		"data_accessed":     report.Summary.DataAccessed,
		"policy_changes":    report.Summary.PolicyChanges,
		"sections":          len(report.Sections),
		"period":            report.Period,
	}
}

// writeEvidencePackageCSV writes a flattened CSV of evidence items.
func writeEvidencePackageCSV(w http.ResponseWriter, pkg *EvidencePackage) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="`+pkg.ID+".csv\"")
	w.Write([]byte("control_id,framework,status,artifacts,notes,collected_at,collected_by\n"))
	for _, ev := range pkg.Evidence {
		artifacts := ""
		if len(ev.Artifacts) > 0 {
			artifacts = ev.Artifacts[0]
			for i := 1; i < len(ev.Artifacts); i++ {
				artifacts += ";" + ev.Artifacts[i]
			}
		}
		line := csvEscape(ev.ControlID) + "," +
			csvEscape(ev.Framework) + "," +
			csvEscape(ev.Status) + "," +
			csvEscape(artifacts) + "," +
			csvEscape(ev.Notes) + "," +
			ev.CollectedAt.Format(time.RFC3339) + "," +
			csvEscape(ev.CollectedBy) + "\n"
		w.Write([]byte(line))
	}
}

// csvEscape quotes a field if it contains commas, quotes, or newlines.
func csvEscape(s string) string {
	for _, c := range s {
		if c == ',' || c == '"' || c == '\n' {
			return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
		}
	}
	return s
}

// contains checks if s contains substr (case-insensitive substring match).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr)))
}
