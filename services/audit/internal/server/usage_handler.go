// Package httpserver — Tenant API usage query endpoint.
// Returns aggregated usage metrics per tenant for Console dashboards.
package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// UsageSummary represents aggregated API usage for a tenant.
type UsageSummary struct {
	TenantID     string  `json:"tenant_id"`
	TotalRequests int64  `json:"total_requests"`
	TotalErrors   int64  `json:"total_errors"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	TopEndpoints []EndpointUsage `json:"top_endpoints,omitempty"`
}

// EndpointUsage shows per-endpoint breakdown.
type EndpointUsage struct {
	Path        string `json:"path"`
	Method      string `json:"method"`
	Requests    int64  `json:"requests"`
	Errors      int64  `json:"errors"`
	AvgLatency  float64 `json:"avg_latency_ms"`
}

// In-memory usage store (production would use PG).
// The gateway flushes batches here via POST /api/v1/audit/usage.
var (
	usageStore = newUsageStore()
)

type usageStoreType struct {
	records []UsageRecord
	maxSize int
}

type UsageRecord struct {
	TenantID  string    `json:"tenant_id"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	Duration  float64   `json:"duration_ms"`
	Timestamp time.Time `json:"timestamp"`
}

func newUsageStore() *usageStoreType {
	return &usageStoreType{maxSize: 100000}
}

func (s *usageStoreType) Add(batch []UsageRecord) {
	s.records = append(s.records, batch...)
	// Trim if too large (ring buffer behavior)
	if len(s.records) > s.maxSize {
		s.records = s.records[len(s.records)-s.maxSize:]
	}
}

// handleUsageIngest handles POST /api/v1/audit/usage from gateway.
// Writes to both in-memory store (for fallback queries) and PostgreSQL
// api_usage_log table (for persistent metering queries).
func (s *HTTPServer) handleUsageIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Events []UsageRecord `json:"events"`
		Type   string        `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Keep in-memory store for backward compatibility.
	usageStore.Add(payload.Events)

	// Persist to PostgreSQL api_usage_log table.
	if s.pool != nil && len(payload.Events) > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		tx, err := s.pool.Begin(ctx)
		if err == nil {
			defer tx.Rollback(ctx) //nolint:errcheck

			for _, rec := range payload.Events {
				_, _ = tx.Exec(ctx, `
					INSERT INTO api_usage_log (tenant_id, method, path, status_code, latency_ms)
					VALUES ($1, $2, $3, $4, $5)`,
					rec.TenantID, rec.Method, rec.Path, rec.Status, int(rec.Duration))
			}
			_ = tx.Commit(ctx)
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

// handleUsageQuery handles GET /api/v1/audit/usage?tenant_id=xxx&days=30
// Queries from the api_usage_log table (populated by gateway metering middleware).
func (s *HTTPServer) handleUsageQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	daysStr := r.URL.Query().Get("days")
	if daysStr == "" {
		daysStr = "30"
	}
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 || days > 365 {
		days = 30
	}

	result := make(map[string]interface{})
	result["days"] = days

	// If pool is available, query from PostgreSQL api_usage_log table.
	if s.pool != nil {
		cutoff := time.Now().AddDate(0, 0, -days)

		if tenantID != "" {
			// Single tenant — include endpoint breakdown.
			query := `
				SELECT path, method, COUNT(*) AS requests,
				       COUNT(*) FILTER (WHERE status_code >= 500) AS errors,
				       COALESCE(AVG(latency_ms), 0) AS avg_lat
				FROM api_usage_log
				WHERE tenant_id = $1 AND created_at >= $2
				GROUP BY path, method
				ORDER BY requests DESC
				LIMIT 50`
			rows, qErr := s.pool.Query(r.Context(), query, tenantID, cutoff)
			if qErr == nil {
				defer rows.Close()
				var endpoints []EndpointUsage
				var totalReq, totalErr int64
				var sumLat float64
				var latCount int64
				for rows.Next() {
					var eu EndpointUsage
					if err := rows.Scan(&eu.Path, &eu.Method, &eu.Requests, &eu.Errors, &eu.AvgLatency); err != nil {
						continue
					}
					endpoints = append(endpoints, eu)
					totalReq += eu.Requests
					totalErr += eu.Errors
					sumLat += eu.AvgLatency * float64(eu.Requests)
					latCount += eu.Requests
				}
				summary := &UsageSummary{
					TenantID:      tenantID,
					TotalRequests: totalReq,
					TotalErrors:   totalErr,
					TopEndpoints:  endpoints,
				}
				if latCount > 0 {
					summary.AvgLatencyMs = sumLat / float64(latCount)
				}
				result["tenant"] = summary
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(result)
				return
			}
		} else {
			// All tenants — summary list.
			query := `
				SELECT tenant_id,
				       COUNT(*) AS total,
				       COUNT(*) FILTER (WHERE status_code >= 500) AS errors,
				       COALESCE(AVG(latency_ms), 0) AS avg_lat
				FROM api_usage_log
				WHERE created_at >= $1
				GROUP BY tenant_id
				ORDER BY total DESC`
			rows, qErr := s.pool.Query(r.Context(), query, cutoff)
			if qErr == nil {
				defer rows.Close()
				var summaries []UsageSummary
				for rows.Next() {
					var us UsageSummary
					if err := rows.Scan(&us.TenantID, &us.TotalRequests, &us.TotalErrors, &us.AvgLatencyMs); err != nil {
						continue
					}
					if us.TenantID == "" {
						us.TenantID = "(anonymous)"
					}
					summaries = append(summaries, us)
				}
				result["tenants"] = summaries
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(result)
				return
			}
		}
	}

	// Fallback: query in-memory store if pool unavailable.
	cutoff := time.Now().AddDate(0, 0, -days)
	type aggKey struct {
		tenant, path, method string
	}
	aggregates := make(map[aggKey]*EndpointUsage)
	tenantCounts := make(map[string]*UsageSummary)

	for _, rec := range usageStore.records {
		if rec.Timestamp.Before(cutoff) {
			continue
		}
		if tenantID != "" && rec.TenantID != tenantID {
			continue
		}

		key := aggKey{rec.TenantID, rec.Path, rec.Method}
		if _, ok := aggregates[key]; !ok {
			aggregates[key] = &EndpointUsage{Path: rec.Path, Method: rec.Method}
		}
		aggregates[key].Requests++
		if rec.Status >= 500 {
			aggregates[key].Errors++
		}
		aggregates[key].AvgLatency = (aggregates[key].AvgLatency*float64(aggregates[key].Requests-1) + rec.Duration) / float64(aggregates[key].Requests)

		if _, ok := tenantCounts[rec.TenantID]; !ok {
			tenantCounts[rec.TenantID] = &UsageSummary{TenantID: rec.TenantID}
		}
		tc := tenantCounts[rec.TenantID]
		tc.TotalRequests++
		if rec.Status >= 500 {
			tc.TotalErrors++
		}
		tc.AvgLatencyMs = (tc.AvgLatencyMs*float64(tc.TotalRequests-1) + rec.Duration) / float64(tc.TotalRequests)
	}

	if tenantID != "" {
		summary := tenantCounts[tenantID]
		if summary == nil {
			summary = &UsageSummary{TenantID: tenantID}
		}
		for _, a := range aggregates {
			if a != nil {
				summary.TopEndpoints = append(summary.TopEndpoints, *a)
			}
		}
		result["tenant"] = summary
	} else {
		summaries := make([]UsageSummary, 0, len(tenantCounts))
		for _, tc := range tenantCounts {
			summaries = append(summaries, *tc)
		}
		result["tenants"] = summaries
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleUsageDispatch routes GET to query and POST to ingest.
func (s *HTTPServer) handleUsageDispatch(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleUsageQuery(w, r)
	case http.MethodPost:
		s.handleUsageIngest(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
