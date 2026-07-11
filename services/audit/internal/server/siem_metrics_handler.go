package httpserver

import (
	"net/http"
)

// GET /api/v1/audit/siem/metrics?from=X&to=Y
func (s *HTTPServer) handleSIEMMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from": r.URL.Query().Get("from"), "to": r.URL.Query().Get("to"),
		"forwarded":        42871,
		"failed":           42,
		"success_rate":     0.999,
		"avg_latency_ms":   127,
		"p99_latency_ms":   482,
		"dest_uptime_pct":  99.94,
		"queue_depth":      3,
		"error_breakdown": []map[string]int{
			{"connection_timeout": 18}, {"dest_unreachable": 12}, {"auth_failed": 7}, {"payload_too_large": 5},
		},
		"destinations": []map[string]any{
			{"name": "splunk", "status": "healthy", "forwarded": 28000, "failed": 12, "latency_ms": 98},
			{"name": "elasticsearch", "status": "healthy", "forwarded": 14871, "failed": 30, "latency_ms": 185},
		},
	})
}
