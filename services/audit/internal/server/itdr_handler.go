package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// handleITDR is the main router for /api/v1/audit/itdr/* endpoints.
func (s *HTTPServer) handleITDR(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/itdr/")

	switch {
	case strings.HasPrefix(path, "detections/") && strings.HasSuffix(path, "/acknowledge"):
		s.handleITDRAcknowledge(w, r)
	case strings.HasPrefix(path, "detections/") && strings.HasSuffix(path, "/resolve"):
		s.handleITDRResolve(w, r)
	case strings.HasPrefix(path, "detections/"):
		s.handleITDRDetectionByID(w, r)
	case path == "detections":
		s.handleITDRListDetections(w, r)
	case path == "stats":
		s.handleITDRStats(w, r)
	case path == "rules":
		s.handleITDRRules(w, r)
	case strings.HasPrefix(path, "rules/"):
		s.handleITDRRuleByID(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

func (s *HTTPServer) handleITDRListDetections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	tenantID := getTenantID(r)
	f := domain.DetectionFilter{
		TenantID: tenantID,
		Page:     1,
		PageSize: 20,
	}

	q := r.URL.Query()
	if sev := q.Get("severity"); sev != "" {
		s := domain.Severity(sev)
		f.Severity = &s
	}
	if st := q.Get("status"); st != "" {
		s := domain.DetectionStatus(st)
		f.Status = &s
	}
	if rule := q.Get("rule_id"); rule != "" {
		f.RuleID = &rule
	}
	if pageStr := q.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			f.Page = p
		}
	}

	detections, total, err := s.itdrRepo.ListDetections(r.Context(), f)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"detections": detections,
		"total":      total,
		"page":       f.Page,
		"page_size":  f.PageSize,
	})
}

func (s *HTTPServer) handleITDRDetectionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "detection ID required"})
		return
	}
	id, err := uuid.Parse(parts[len(parts)-1])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid detection ID"})
		return
	}

	det, err := s.itdrRepo.GetDetection(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "detection not found"})
		return
	}

	writeJSON(w, http.StatusOK, det)
}

func (s *HTTPServer) handleITDRAcknowledge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "detection ID required"})
		return
	}
	id, err := uuid.Parse(parts[len(parts)-2])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid detection ID"})
		return
	}

	if err := s.itdrRepo.UpdateStatus(r.Context(), id, domain.DetectionAcknowledged); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

func (s *HTTPServer) handleITDRResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var body struct {
		FalsePositive bool `json:"false_positive"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "detection ID required"})
		return
	}
	id, err := uuid.Parse(parts[len(parts)-2])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid detection ID"})
		return
	}

	status := domain.DetectionResolved
	if body.FalsePositive {
		status = domain.DetectionFalsePositive
	}

	if err := s.itdrRepo.UpdateStatus(r.Context(), id, status); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": string(status)})
}

func (s *HTTPServer) handleITDRStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	tenantID := getTenantID(r)
	window := 24 * time.Hour
	if w := r.URL.Query().Get("window"); w != "" {
		if d, err := time.ParseDuration(w); err == nil && d > 0 && d <= 168*time.Hour {
			window = d
		}
	}

	since := time.Now().UTC().Add(-window)
	stats, err := s.itdrRepo.GetStats(r.Context(), tenantID, since)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func (s *HTTPServer) handleITDRRules(w http.ResponseWriter, r *http.Request) {
	// Placeholder — rules CRUD returns empty list for now.
	writeJSON(w, http.StatusOK, map[string]any{"rules": []any{}})
}

func (s *HTTPServer) handleITDRRuleByID(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "rule not found"})
}

// Helper to get tenant ID from request header.
func getTenantID(r *http.Request) uuid.UUID {
	idStr := r.Header.Get("X-Tenant-ID")
	if idStr == "" {
		return uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}
	return id
}
