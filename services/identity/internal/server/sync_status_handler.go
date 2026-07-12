package server

import (
	"net/http"
	"sync"
	"time"
)

// idpSyncStatus tracks external IdP directory sync state.
type idpSyncStatus struct {
	Provider      string                   `json:"provider"`
	LastSync      string                   `json:"last_sync"`
	NextSync      string                   `json:"next_sync"`
	Status        string                   `json:"status"` // success, failed, in_progress, never
	SyncedUsers   int                      `json:"synced_users"`
	TotalUsers    int                      `json:"total_users"`
	Errors        []map[string]any         `json:"errors"`
	Frequency     string                   `json:"frequency"`
}

var idpSyncStore = struct {
	sync.RWMutex
	statuses []idpSyncStatus
}{statuses: []idpSyncStatus{
	{
		Provider: "okta", Status: "success",
		LastSync: time.Now().UTC().Add(-15 * time.Minute).Format(time.RFC3339),
		NextSync: time.Now().UTC().Add(45 * time.Minute).Format(time.RFC3339),
		SyncedUsers: 15420, TotalUsers: 15420,
		Errors: []map[string]any{}, Frequency: "hourly",
	},
	{
		Provider: "azure-ad", Status: "success",
		LastSync: time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
		NextSync: time.Now().UTC().Add(4 * time.Hour).Format(time.RFC3339),
		SyncedUsers: 8930, TotalUsers: 8945,
		Errors: []map[string]any{
			{"user": "user-8931", "error": "duplicate UPN, skipped", "code": "DUPLICATE_UPN"},
			{"user": "user-8942", "error": "missing required field: email", "code": "VALIDATION_ERROR"},
		},
		Frequency: "6h",
	},
	{
		Provider: "ldap", Status: "failed",
		LastSync: time.Now().UTC().Add(-3 * time.Hour).Format(time.RFC3339),
		NextSync: time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
		SyncedUsers: 0, TotalUsers: 3200,
		Errors: []map[string]any{
			{"error": "connection refused", "code": "CONN_REFUSED", "host": "ldap.internal:389"},
		},
		Frequency: "hourly",
	},
	{
		Provider: "scim-endpoint", Status: "never",
		LastSync: "", NextSync: "",
		SyncedUsers: 0, TotalUsers: 0,
		Errors: []map[string]any{
			{"error": "SCIM endpoint not configured", "code": "NOT_CONFIGURED"},
		},
		Frequency: "manual",
	},
}}

// GET /api/v1/identity/sync-status?provider=X
func (h *HTTPHandler) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	providerFilter := r.URL.Query().Get("provider")

	idpSyncStore.RLock()
	result := []idpSyncStatus{}
	for _, s := range idpSyncStore.statuses {
		if providerFilter != "" && s.Provider != providerFilter {
			continue
		}
		result = append(result, s)
	}
	idpSyncStore.RUnlock()

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
		"total_errors":      totalErrors,
		"checked_at":        time.Now().UTC().Format(time.RFC3339),
	})
}
