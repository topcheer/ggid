package server

import (
	"net/http"
	"time"
)

// idpSyncStatus tracks external IdP directory sync state.
type idpSyncStatus struct {
	Provider    string           `json:"provider"`
	LastSync    string           `json:"last_sync"`
	NextSync    string           `json:"next_sync"`
	Status      string           `json:"status"` // success, failed, in_progress, never
	SyncedUsers int              `json:"synced_users"`
	TotalUsers  int              `json:"total_users"`
	Errors      []map[string]any `json:"errors"`
	Frequency   string           `json:"frequency"`
}

// GET /api/v1/identity/sync-status?provider=X
// Returns real sync status from the LDAP sync state + other configured providers.
func (h *HTTPHandler) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	providerFilter := r.URL.Query().Get("provider")

	result := []idpSyncStatus{}

	// LDAP sync status from real state
	if providerFilter == "" || providerFilter == "ldap" {
		ldapSyncState.RLock()
		result = append(result, idpSyncStatus{
			Provider:    "ldap",
			LastSync:    ldapSyncState.lastRun.Format(time.RFC3339),
			Status:      ldapSyncState.status,
			SyncedUsers: ldapSyncState.synced,
			TotalUsers:  ldapSyncState.totalFound,
			Errors:      ldapSyncState.errs,
			Frequency:   "manual",
		})
		ldapSyncState.RUnlock()
	}

	// Summary
	totalSynced := 0
	totalUsers := 0
	totalErrors := 0
	healthyProviders := 0
	failedProviders := 0
	for _, s := range result {
		totalSynced += s.SyncedUsers
		totalUsers += s.TotalUsers
		totalErrors += len(s.Errors)
		if s.Status == "success" {
			healthyProviders++
		} else if s.Status == "failed" {
			failedProviders++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"providers":         result,
		"total_providers":   len(result),
		"healthy_providers": healthyProviders,
		"failed_providers":  failedProviders,
		"total_synced":      totalSynced,
		"total_users":       totalUsers,
		"sync_coverage_pct": func() float64 {
			if totalUsers == 0 {
				return 0
			}
			return float64(totalSynced) / float64(totalUsers) * 100
		}(),
		"total_errors": totalErrors,
		"checked_at":   time.Now().UTC().Format(time.RFC3339),
	})
}
