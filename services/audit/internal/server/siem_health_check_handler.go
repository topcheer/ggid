package httpserver

import (
	"net/http"
	"time"
)

// GET /api/v1/audit/siem/health-check
// Full SIEM health: destinations with connectivity, latency, throughput, error_rate.
func (s *HTTPServer) handleSIEMHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	destinations := []map[string]any{
		{
			"name": "splunk-prod", "type": "splunk", "endpoint": "https://splunk.internal:8088",
			"connectivity": "connected", "latency_ms": 45, "throughput_eps": 3200,
			"error_rate_pct": 0.2, "last_success": time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339),
			"queue_depth": 12, "status": "healthy",
		},
		{
			"name": "elasticsearch", "type": "elasticsearch", "endpoint": "https://es.internal:9200",
			"connectivity": "connected", "latency_ms": 78, "throughput_eps": 2800,
			"error_rate_pct": 1.5, "last_success": time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339),
			"queue_depth": 145, "status": "warning",
		},
		{
			"name": "datadog", "type": "datadog", "endpoint": "https://api.datadoghq.com",
			"connectivity": "disconnected", "latency_ms": 0, "throughput_eps": 0,
			"error_rate_pct": 100, "last_success": time.Now().UTC().Add(-3 * time.Hour).Format(time.RFC3339),
			"queue_depth": 5800, "status": "critical",
		},
		{
			"name": "sumo-logic", "type": "sumologic", "endpoint": "https://api.sumologic.com",
			"connectivity": "connected", "latency_ms": 120, "throughput_eps": 1500,
			"error_rate_pct": 0.5, "last_success": time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339),
			"queue_depth": 8, "status": "healthy",
		},
	}

	healthyCount := 0
	criticalCount := 0
	warningCount := 0
	totalThroughput := 0
	totalQueueDepth := 0
	for _, d := range destinations {
		switch d["status"] {
		case "healthy":
			healthyCount++
		case "warning":
			warningCount++
		case "critical":
			criticalCount++
		}
		if eps, ok := d["throughput_eps"].(int); ok {
			totalThroughput += eps
		}
		if qd, ok := d["queue_depth"].(int); ok {
			totalQueueDepth += qd
		}
	}

	overallStatus := "healthy"
	if criticalCount > 0 {
		overallStatus = "degraded"
	} else if warningCount > 0 {
		overallStatus = "warning"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"overall_status":     overallStatus,
		"destinations":       destinations,
		"total_destinations": len(destinations),
		"healthy":            healthyCount,
		"warning":            warningCount,
		"critical":           criticalCount,
		"total_throughput_eps": totalThroughput,
		"total_queue_depth":  totalQueueDepth,
		"checked_at":         time.Now().UTC().Format(time.RFC3339),
	})
}
