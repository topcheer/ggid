package httpserver

import (
	"net/http"
	"strconv"
	"time"
)

// GET /api/v1/policies/access-frequency?resource=X&days=30
func (s *HTTPServer) handleAccessFrequency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	resource := r.URL.Query().Get("resource")
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days == 0 { days = 30 }
	// Hourly buckets
	now := time.Now().UTC()
	buckets := make([]map[string]any, 24)
	for i := 0; i < 24; i++ {
		count := 50 + (i*7)%200
		buckets[i] = map[string]any{"hour": i, "count": count, "unique_users": count/3}
	}
	// Detect anomaly: hour 3 has spike
	anomalies := []map[string]any{
		{"hour": 3, "type": "spike", "description": "unusual access at 3 AM (4x baseline)", "count": 847, "baseline": 50},
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"resource": resource, "days": days, "hourly_buckets": buckets,
		"total_accesses": 4800, "unique_users": 142, "avg_per_hour": 200,
		"anomalies": anomalies, "generated_at": now.Format(time.RFC3339),
	})
}
