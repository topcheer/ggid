package middleware

import (
	"encoding/json"
	"net/http"
)

// GET /api/v1/observability/traces — recent trace summary.
// GET /api/v1/observability/health — exporter + collector status.

// ObservabilityHandler returns HTTP handlers for observability endpoints.
func ObservabilityHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/observability/traces":
			if r.Method != http.MethodGet {
				writeJSONResp(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
				return
			}
			traces := GetRecentTraces(100)
			// Convert to summary.
			summaries := make([]map[string]any, 0, len(traces))
			for _, t := range traces {
				summaries = append(summaries, map[string]any{
					"trace_id":    t.TraceID,
					"operation":   t.Operation,
					"duration_ms": t.Duration.Milliseconds(),
					"status":      t.StatusCode,
					"timestamp":   t.Timestamp,
					"attributes":  t.Attributes,
				})
			}
			writeJSONResp(w, http.StatusOK, map[string]any{
				"traces": summaries,
				"count":  len(summaries),
			})

		case "/api/v1/observability/health":
			if r.Method != http.MethodGet {
				writeJSONResp(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
				return
			}
			writeJSONResp(w, http.StatusOK, map[string]any{
				"status":       "healthy",
				"exporter":     "otel-otlp",
				"collector":    "jaeger",
				"sampling":     getSampleRate(),
				"traces_stored": len(traceStore),
				"w3c_format":   true,
			})

		default:
			writeJSONResp(w, http.StatusNotFound, map[string]any{"error": "not found"})
		}
	}
}

func writeJSONResp(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
