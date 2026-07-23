// Package httpserver — Tenant API usage query endpoint.
// Returns aggregated usage metrics per tenant for Console dashboards.
package httpserver

import (
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

	usageStore.Add(payload.Events)
	w.WriteHeader(http.StatusAccepted)
}

// handleUsageQuery handles GET /api/v1/audit/usage?tenant_id=xxx&days=30
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

	cutoff := time.Now().AddDate(0, 0, -days)

	// Aggregate from in-memory store
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

		// Per-endpoint aggregation
		key := aggKey{rec.TenantID, rec.Path, rec.Method}
		if _, ok := aggregates[key]; !ok {
			aggregates[key] = &EndpointUsage{Path: rec.Path, Method: rec.Method}
		}
		aggregates[key].Requests++
		if rec.Status >= 500 {
			aggregates[key].Errors++
		}
		aggregates[key].AvgLatency = (aggregates[key].AvgLatency*float64(aggregates[key].Requests-1) + rec.Duration) / float64(aggregates[key].Requests)

		// Per-tenant aggregation
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

	// Build response
	result := make(map[string]interface{})
	if tenantID != "" {
		// Single tenant — include endpoint breakdown
		summary := tenantCounts[tenantID]
		if summary == nil {
			summary = &UsageSummary{TenantID: tenantID}
		}
		for _, a := range aggregates {
			if a != nil && (tenantID == "" || true) {
				summary.TopEndpoints = append(summary.TopEndpoints, *a)
			}
		}
		result["tenant"] = summary
	} else {
		// All tenants — summary list
		summaries := make([]UsageSummary, 0, len(tenantCounts))
		for _, tc := range tenantCounts {
			summaries = append(summaries, *tc)
		}
		result["tenants"] = summaries
	}
	result["days"] = days

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
