package server

import (
	"net/http"
)

// GET /api/v1/oauth/grant-flows
func handleGrantFlows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"}); return
	}
	writeJSON(w, http.StatusOK, map[string]any{"flows": []map[string]any{
		{"flow": "authorization_code", "count": 32145, "success_rate": 0.967, "avg_duration_ms": 245, "error_count": 1063},
		{"flow": "client_credentials", "count": 8921, "success_rate": 0.994, "avg_duration_ms": 89, "error_count": 54},
		{"flow": "refresh_token", "count": 15632, "success_rate": 0.982, "avg_duration_ms": 42, "error_count": 281},
		{"flow": "device_code", "count": 487, "success_rate": 0.912, "avg_duration_ms": 5230, "error_count": 43},
		{"flow": "password", "count": 1207, "success_rate": 0.945, "avg_duration_ms": 312, "error_count": 66},
	}, "total_grants": 58392, "overall_success_rate": 0.971})
}
