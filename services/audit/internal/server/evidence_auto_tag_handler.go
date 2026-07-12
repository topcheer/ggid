package httpserver

import (
	"net/http"
	"strings"
	"sync"
)

// evidenceTag stores auto-generated tags for evidence.
type evidenceTag struct {
	Framework string  `json:"framework"`
	ControlID string  `json:"control_id"`
	Confidence float64 `json:"confidence"`
}

var evidenceTagStore = struct {
	sync.RWMutex
	data map[string][]evidenceTag
}{data: make(map[string][]evidenceTag)}

// POST /api/v1/audit/compliance/evidence/{id}/auto-tag
func (s *HTTPServer) handleEvidenceAutoTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract evidence ID
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/compliance/evidence/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[1] != "auto-tag" {
		writeJSONError(w, http.StatusBadRequest, "invalid path")
		return
	}
	evidenceID := parts[0]
	if evidenceID == "" {
		writeJSONError(w, http.StatusBadRequest, "evidence ID is required")
		return
	}

	// Simulate content analysis to auto-tag
	// In production, this would use NLP/ML on the evidence content
	tags := []evidenceTag{
		{Framework: "soc2", ControlID: "CC6.1", Confidence: 0.92},
		{Framework: "soc2", ControlID: "CC7.1", Confidence: 0.78},
		{Framework: "iso27001", ControlID: "A.9", Confidence: 0.85},
		{Framework: "gdpr", ControlID: "Art32", Confidence: 0.71},
	}

	evidenceTagStore.Lock()
	evidenceTagStore.data[evidenceID] = tags
	evidenceTagStore.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"evidence_id":  evidenceID,
		"tags":         tags,
		"total_tags":   len(tags),
		"avg_confidence": func() float64 {
			sum := 0.0
			for _, t := range tags {
				sum += t.Confidence
			}
			return sum / float64(len(tags))
		}(),
		"tagged_at": "auto-analysis",
	})
}
