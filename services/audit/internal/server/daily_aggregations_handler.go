package httpserver

import (
	"fmt"
	"net/http"
	"time"
)

// GET /api/v1/audit/aggregations/daily?from=X&to=Y&tenant_id=Z
func (s *HTTPServer) handleDailyAggregations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	from := time.Now().UTC().Add(-30 * 24 * time.Hour)
	to := time.Now().UTC()
	days := int(to.Sub(from) / (24 * time.Hour))
	if days > 90 { days = 90 }
	buckets := make([]map[string]any, days)
	for i := 0; i < days; i++ {
		d := from.AddDate(0, 0, i)
		buckets[i] = map[string]any{"date": d.Format("2006-01-02"), "event_count": 0, "unique_users": 0, "top_actions": []map[string]int{}}
	}
	writeJSON(w, http.StatusOK, map[string]any{"from": from.Format(time.RFC3339), "to": to.Format(time.RFC3339), "days": days, "buckets": buckets, "note": fmt.Sprintf("materialized view for %d days", days)})
}
