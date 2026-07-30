package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/tenant"
)

// ExportJobV2 represents an on-demand audit data export job for the frontend exports page.
type ExportJobV2 struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Format    string    `json:"format"`
	Status    string    `json:"status"` // pending, running, completed, failed
	CreatedAt time.Time `json:"created_at"`
	Size      int64     `json:"size"`
	Records   int       `json:"records"`
	TenantID  string    `json:"-"` // internal: tenant isolation
}

var (
	exportJobsV2Mu sync.RWMutex
	exportJobsV2   = []ExportJobV2{}
)

// GET/POST /api/v1/audit/exports
// GET /api/v1/audit/exports/{id}/download
// Routed via gateway /api/v1/exports prefix → audit service.
func (s *HTTPServer) handleExportsV2(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	tenantIDStr := ""
	if err == nil && tc != nil {
		tenantIDStr = tc.TenantID.String()
	}
	switch {
	case r.URL.Path == "/api/v1/audit/exports" && r.Method == http.MethodGet:
		exportJobsV2Mu.RLock()
		var tenantJobs []ExportJobV2
		for _, job := range exportJobsV2 {
			if job.TenantID == tenantIDStr {
				tenantJobs = append(tenantJobs, job)
			}
		}
		exportJobsV2Mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"exports": tenantJobs, "total": len(tenantJobs)})

	case r.URL.Path == "/api/v1/audit/exports" && r.Method == http.MethodPost:
		var req struct {
			Name   string `json:"name"`
			Format string `json:"format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Format == "" {
			req.Format = "csv"
		}
		job := ExportJobV2{
			ID:        fmt.Sprintf("exp-%d", time.Now().UnixNano()),
			Name:      req.Name,
			Format:    req.Format,
			Status:    "completed",
			CreatedAt: time.Now(),
			TenantID:  tenantIDStr,
		}
		exportJobsV2Mu.Lock()
		exportJobsV2 = append(exportJobsV2, job)
		exportJobsV2Mu.Unlock()
		writeJSON(w, http.StatusCreated, job)

	case strings.HasSuffix(r.URL.Path, "/download") && r.Method == http.MethodGet:
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=export.csv")
		w.WriteHeader(http.StatusOK)

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
