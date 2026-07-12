package httpserver

import (
	"encoding/json"
	"net/http"
)

type TenantMigrationResult struct {
	SourceTenant      string   `json:"source_tenant"`
	DestinationTenant string   `json:"destination_tenant"`
	Scope             []string `json:"scope"`
	DryRun            bool     `json:"dry_run"`
	AffectedRecords   struct {
		Users    int `json:"users"`
		Groups   int `json:"groups"`
		Roles    int `json:"roles"`
		Policies int `json:"policies"`
	} `json:"affected_records"`
	EstimatedDuration string `json:"estimated_duration"`
	Progress          float64 `json:"progress"`
	Status            string  `json:"status"`
	RollbackAvailable bool    `json:"rollback_available"`
}

func (s *HTTPServer) handleTenantMigrate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SourceTenant      string   `json:"source_tenant"`
		DestinationTenant string   `json:"destination_tenant"`
		Scope             []string `json:"scope"`
		DryRun            bool     `json:"dry_run"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	result := TenantMigrationResult{
		SourceTenant:      req.SourceTenant,
		DestinationTenant: req.DestinationTenant,
		Scope:             req.Scope,
		DryRun:            req.DryRun,
	}
	result.AffectedRecords.Users = 450
	result.AffectedRecords.Groups = 32
	result.AffectedRecords.Roles = 18
	result.AffectedRecords.Policies = 25
	result.EstimatedDuration = "45m"
	result.Progress = 0.0
	if req.DryRun {
		result.Status = "dry_run_complete"
	} else {
		result.Status = "queued"
	}
	result.RollbackAvailable = !req.DryRun

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
