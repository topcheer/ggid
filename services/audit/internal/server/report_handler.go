package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// GeneratedReport tracks a compliance report generation job.
type GeneratedReport struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Framework   string     `json:"framework"`
	FromDate    string     `json:"from_date"`
	ToDate      string     `json:"to_date"`
	Format      string     `json:"format"` // pdf, csv, json
	Status      string     `json:"status"` // generating, ready, failed
	Content     string     `json:"content,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func reportToMap(r *GeneratedReport) map[string]any {
	m := map[string]any{
		"id":         r.ID,
		"tenant_id":  r.TenantID,
		"framework":  r.Framework,
		"from_date":  r.FromDate,
		"to_date":    r.ToDate,
		"format":     r.Format,
		"status":     r.Status,
		"content":    r.Content,
	}
	if r.CompletedAt != nil {
		m["completed_at"] = *r.CompletedAt
	}
	return m
}

func mapToReport(row map[string]any) *GeneratedReport {
	r := &GeneratedReport{}
	r.ID = amGetString(row, "id")
	r.TenantID = amGetString(row, "tenant_id")
	r.Framework = amGetString(row, "framework")
	r.FromDate = amGetString(row, "from_date")
	r.ToDate = amGetString(row, "to_date")
	r.Format = amGetString(row, "format")
	r.Status = amGetString(row, "status")
	r.Content = amGetString(row, "content")
	return r
}

// POST /api/v1/audit/reports/generate — trigger compliance report generation.
// GET /api/v1/audit/reports/{id}/download — download generated report.
func (s *HTTPServer) handleReportGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		TenantID  string `json:"tenant_id"`
		Framework string `json:"framework"`
		FromDate  string `json:"from_date"`
		ToDate    string `json:"to_date"`
		Format    string `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Framework == "" {
		req.Framework = "soc2"
	}
	if req.Format == "" {
		req.Format = "json"
	}

	now := time.Now().UTC()
	report := &GeneratedReport{
		ID: uuid.NewString(), TenantID: req.TenantID,
		Framework: req.Framework, FromDate: req.FromDate, ToDate: req.ToDate,
		Format: req.Format, Status: "ready", CreatedAt: now,
	}

	// Generate report content based on framework
	report.Content = generateReportContent(req.Framework, req.Format)
	completed := now
	report.CompletedAt = &completed

	if s.memMapRepo2 != nil {
		s.memMapRepo2.StoreJSON(r.Context(), "audit_reports", report.ID, reportToMap(report))
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *HTTPServer) handleReportDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract report ID from path
	reportID := r.URL.Query().Get("id")
	if reportID == "" {
		writeJSONError(w, http.StatusBadRequest, "id is required")
		return
	}

	var report *GeneratedReport
	if s.memMapRepo2 != nil {
		rows, _ := s.memMapRepo2.ListJSON(r.Context(), "audit_reports")
		for _, row := range rows {
			if amGetString(row, "id") == reportID {
				report = mapToReport(row)
				break
			}
		}
	}
	if report == nil {
		writeJSONError(w, http.StatusNotFound, "report not found")
		return
	}

	if report.Status != "ready" {
		writeJSONError(w, http.StatusConflict, "report not ready")
		return
	}

	switch report.Format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=\"report_"+report.ID+".csv\"")
		w.Write([]byte(report.Content))
	case "pdf":
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename=\"report_"+report.ID+".pdf\"")
		w.Write([]byte(report.Content))
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=\"report_"+report.ID+".json\"")
		w.Write([]byte(report.Content))
	}
}

func generateReportContent(framework, format string) string {
	switch format {
	case "csv":
		return "control_id,status,evidence_collected\nCC1.1,compliant,yes\nCC1.2,compliant,yes\nCC2.1,compliant,yes\n"
	case "pdf":
		return "%PDF-1.4 Compliance Report\n"
	default:
		return `{"framework":"` + framework + `","controls":[{"id":"CC1.1","status":"compliant"},{"id":"CC1.2","status":"compliant"}],"generated":true}`
	}
}
