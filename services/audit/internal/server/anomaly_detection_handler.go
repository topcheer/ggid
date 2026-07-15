package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AnomalyEvent represents a detected anomaly.
type AnomalyEvent struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Severity   string    `json:"severity"`
	User       string    `json:"user"`
	Timestamp  time.Time `json:"timestamp"`
	Confidence float64   `json:"confidence"`
	Detail     string    `json:"detail"`
	Status     string    `json:"status"` // active, acknowledged, dismissed
}

var (
	anomalyEventsMu sync.RWMutex
	anomalyEvents   = []AnomalyEvent{}
)

// GET/POST /api/v1/audit/anomaly-detection
// PATCH /api/v1/audit/anomaly-detection/{id}
func (s *HTTPServer) handleAnomalyDetectionCRUD(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/v1/audit/anomaly-detection/") && r.Method == http.MethodPatch {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/anomaly-detection/")
		var req struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid body")
			return
		}
		anomalyEventsMu.Lock()
		defer anomalyEventsMu.Unlock()
		for i := range anomalyEvents {
			if anomalyEvents[i].ID == id {
				anomalyEvents[i].Status = req.Status
				writeJSON(w, http.StatusOK, anomalyEvents[i])
				return
			}
		}
		writeJSONError(w, http.StatusNotFound, "anomaly not found")
		return
	}

	if r.Method == http.MethodGet {
		anomalyEventsMu.RLock()
		events := make([]AnomalyEvent, len(anomalyEvents))
		copy(events, anomalyEvents)
		anomalyEventsMu.RUnlock()
		writeJSON(w, http.StatusOK, events)
		return
	}

	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}
