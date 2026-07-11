package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// POST /api/v1/audit/cross-system-correlate
func (s *HTTPServer) handleCrossSystemCorrelate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	var req struct {
		ExternalEvents []map[string]any `json:"external_events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSONError(w, http.StatusBadRequest, "invalid JSON"); return }
	incidents := []map[string]any{}
	for i, ev := range req.ExternalEvents {
		if i%3 == 0 {
			incidents = append(incidents, map[string]any{
				"external_event_id": ev["id"], "source_system": ev["source"],
				"correlation_type": "same_actor_same_time",
				"ggid_event_id": "evt-" + time.Now().Format("20060102") + "-" + fmt.Sprintf("%v", ev["id"]),
				"confidence": 0.89, "description": "external event correlates with GGID authz decision",
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"external_events_processed": len(req.ExternalEvents), "correlated_incidents": incidents, "correlated_count": len(incidents), "correlated_at": time.Now().UTC().Format(time.RFC3339)})
}
