package httpserver

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// evidenceAuditEvent represents a single event in an evidence's lifecycle.
type evidenceAuditEvent struct {
	ID         string `json:"id"`
	EvidenceID string `json:"evidence_id"`
	EventType  string `json:"event_type"` // created, modified, verified, expired, refreshed, downloaded, shared
	Actor      string `json:"actor"`
	Timestamp  string `json:"timestamp"`
	Detail     string `json:"detail"`
}

var evidenceAuditStore = struct {
	sync.RWMutex
	data map[string][]evidenceAuditEvent
}{data: map[string][]evidenceAuditEvent{
	"ev-001": {
		{ID: "1", EvidenceID: "ev-001", EventType: "created", Actor: "compliance-bot", Timestamp: time.Now().UTC().Add(-30*24*time.Hour).Format(time.RFC3339), Detail: "Auto-collected from SOC2 scan"},
		{ID: "2", EvidenceID: "ev-001", EventType: "verified", Actor: "auditor@ex.com", Timestamp: time.Now().UTC().Add(-25*24*time.Hour).Format(time.RFC3339), Detail: "SHA256 verified, hash matches"},
		{ID: "3", EvidenceID: "ev-001", EventType: "modified", Actor: "admin@ex.com", Timestamp: time.Now().UTC().Add(-10*24*time.Hour).Format(time.RFC3339), Detail: "Updated description and added tags"},
		{ID: "4", EvidenceID: "ev-001", EventType: "refreshed", Actor: "compliance-bot", Timestamp: time.Now().UTC().Add(-5*24*time.Hour).Format(time.RFC3339), Detail: "New version collected, v2"},
		{ID: "5", EvidenceID: "ev-001", EventType: "verified", Actor: "auditor@ex.com", Timestamp: time.Now().UTC().Add(-3*24*time.Hour).Format(time.RFC3339), Detail: "v2 SHA256 verified"},
	},
	"ev-002": {
		{ID: "6", EvidenceID: "ev-002", EventType: "created", Actor: "admin@ex.com", Timestamp: time.Now().UTC().Add(-15*24*time.Hour).Format(time.RFC3339), Detail: "Manual upload of access review report"},
		{ID: "7", EvidenceID: "ev-002", EventType: "expired", Actor: "system", Timestamp: time.Now().UTC().Add(-1*24*time.Hour).Format(time.RFC3339), Detail: "Evidence over 90 days old, needs refresh"},
	},
}}

// GET /api/v1/audit/compliance/evidence/{id}/audit-trail
func (s *HTTPServer) handleEvidenceAuditTrail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract evidence ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/compliance/evidence/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[1] != "audit-trail" {
		writeJSONError(w, http.StatusBadRequest, "invalid path, expected /evidence/{id}/audit-trail")
		return
	}
	evidenceID := parts[0]
	if evidenceID == "" {
		writeJSONError(w, http.StatusBadRequest, "evidence ID is required")
		return
	}

	evidenceAuditStore.RLock()
	events := evidenceAuditStore.data[evidenceID]
	result := make([]evidenceAuditEvent, len(events))
	copy(result, events)
	evidenceAuditStore.RUnlock()

	// Build summary
	eventCounts := map[string]int{}
	actors := map[string]bool{}
	for _, e := range result {
		eventCounts[e.EventType]++
		actors[e.Actor] = true
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"evidence_id":   evidenceID,
		"events":        result,
		"total_events":  len(result),
		"event_summary": eventCounts,
		"unique_actors": len(actors),
		"first_event": func() string {
			if len(result) > 0 {
				return result[0].Timestamp
			}
			return ""
		}(),
		"last_event": func() string {
			if len(result) > 0 {
				return result[len(result)-1].Timestamp
			}
			return ""
		}(),
	})
}
