package httpserver

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// usageSummary represents per-tenant aggregated API usage.
type usageSummary struct {
	TenantID       string `json:"tenant_id"`
	TotalRequests  int    `json:"total_requests"`
	ErrorCount     int    `json:"error_count"`
	AvgLatencyMs   int    `json:"avg_latency_ms"`
	MaxLatencyMs   int    `json:"max_latency_ms"`
	UniquePaths    int    `json:"unique_paths"`
}

// pathUsage represents per-path usage within a tenant.
type pathUsage struct {
	Path          string `json:"path"`
	Method        string `json:"method"`
	Count         int    `json:"count"`
	AvgLatencyMs  int    `json:"avg_latency_ms"`
	ErrorCount    int    `json:"error_count"`
}

// handleUsage handles GET /api/v1/audit/usage — returns aggregated API
// usage metrics per tenant, with optional filters.
//
// Query params:
//   tenant_id  — filter to a specific tenant (platform admin only for cross-tenant)
//   hours      — time window in hours (default 24, max 168/7d)
//   detail     — if "paths", return per-path breakdown instead of per-tenant summary
func (s *HTTPServer) handleUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.pool == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	ctx := r.Context()

	// Parse query params.
	tenantID := r.URL.Query().Get("tenant_id")
	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if v, err := strconv.Atoi(h); err == nil && v > 0 && v <= 168 {
			hours = v
		}
	}
	detail := r.URL.Query().Get("detail") == "paths"

	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	if detail {
		// Per-path breakdown for a specific tenant.
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required when detail=paths")
			return
		}
		query := `
			SELECT path, method, COUNT(*) AS count,
			       COALESCE(AVG(latency_ms)::int, 0) AS avg_latency,
			       COUNT(*) FILTER (WHERE status_code >= 400) AS error_count
			FROM api_usage_log
			WHERE tenant_id = $1 AND created_at >= $2
			GROUP BY path, method
			ORDER BY count DESC
			LIMIT 100`
		rows, err := s.pool.Query(ctx, query, tenantID, since)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("query failed: %v", err))
			return
		}
		defer rows.Close()

		results := make([]pathUsage, 0)
		for rows.Next() {
			var pu pathUsage
			if err := rows.Scan(&pu.Path, &pu.Method, &pu.Count, &pu.AvgLatencyMs, &pu.ErrorCount); err != nil {
				continue
			}
			results = append(results, pu)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"tenant_id": tenantID,
			"hours":     hours,
			"paths":     results,
			"total":     len(results),
		})
		return
	}

	// Per-tenant summary (optionally filtered to one tenant).
	if tenantID != "" {
		query := `
			SELECT tenant_id,
			       COUNT(*) AS total,
			       COUNT(*) FILTER (WHERE status_code >= 400) AS errors,
			       COALESCE(AVG(latency_ms)::int, 0) AS avg_lat,
			       COALESCE(MAX(latency_ms), 0) AS max_lat,
			       COUNT(DISTINCT path) AS paths
			FROM api_usage_log
			WHERE tenant_id = $1 AND created_at >= $2
			GROUP BY tenant_id`
		rows, err := s.pool.Query(ctx, query, tenantID, since)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("query failed: %v", err))
			return
		}
		defer rows.Close()

		summaries := make([]usageSummary, 0)
		for rows.Next() {
			var us usageSummary
			if err := rows.Scan(&us.TenantID, &us.TotalRequests, &us.ErrorCount, &us.AvgLatencyMs, &us.MaxLatencyMs, &us.UniquePaths); err != nil {
				continue
			}
			summaries = append(summaries, us)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"hours":  hours,
			"usage":  summaries,
			"total":  len(summaries),
		})
		return
	}

	// All tenants summary.
	query := `
		SELECT tenant_id,
		       COUNT(*) AS total,
		       COUNT(*) FILTER (WHERE status_code >= 400) AS errors,
		       COALESCE(AVG(latency_ms)::int, 0) AS avg_lat,
		       COALESCE(MAX(latency_ms), 0) AS max_lat,
		       COUNT(DISTINCT path) AS paths
		FROM api_usage_log
		WHERE created_at >= $1
		GROUP BY tenant_id
		ORDER BY total DESC`
	rows, err := s.pool.Query(ctx, query, since)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("query failed: %v", err))
		return
	}
	defer rows.Close()

	summaries := make([]usageSummary, 0)
	for rows.Next() {
		var us usageSummary
		if err := rows.Scan(&us.TenantID, &us.TotalRequests, &us.ErrorCount, &us.AvgLatencyMs, &us.MaxLatencyMs, &us.UniquePaths); err != nil {
			continue
		}
		// Mask tenant IDs that are empty (unauthenticated requests).
		if us.TenantID == "" {
			us.TenantID = "(anonymous)"
		}
		summaries = append(summaries, us)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"hours": hours,
		"usage": summaries,
		"total": len(summaries),
	})
}

// Suppress unused import warnings — strings may be used in future filtering.
var _ = strings.ToLower
