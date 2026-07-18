package server

import (
	"encoding/json"
	"net/http"
	"time"
)

type SyncError struct {
	UserID    string `json:"user_id"`
	Error     string `json:"error"`
	Timestamp string `json:"timestamp"`
}

type SCIMSyncHealth struct {
	EndpointURL      string     `json:"endpoint_url"`
	LastSyncAt       string     `json:"last_sync_at"`
	Status           string     `json:"status"`
	ProvisioningErrors []SyncError `json:"provisioning_errors"`
	UsersSynced      int        `json:"users_synced"`
	UsersPending     int        `json:"users_pending"`
	UsersFailed      int        `json:"users_failed"`
	RateLimitRemaining int      `json:"rate_limit_remaining"`
	ThroughputPerMin float64   `json:"throughput_per_min"`
	NextSyncAt       string     `json:"next_sync_at"`
}

func (h *HTTPHandler) handleSCIMSyncHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := SCIMSyncHealth{
		EndpointURL:        "https://scim.ggcode.dev/v2/Users",
		LastSyncAt:         time.Now().UTC().Add(-15 * time.Minute).Format(time.RFC3339),
		Status:             "healthy",
		ProvisioningErrors: []SyncError{
			{UserID: "u-0342", Error: "Attribute mapping failed: department", Timestamp: time.Now().UTC().Add(-20 * time.Minute).Format(time.RFC3339)},
		},
		UsersSynced:        1247,
		UsersPending:       12,
		UsersFailed:        3,
		RateLimitRemaining: 4800,
		ThroughputPerMin:   85.5,
		NextSyncAt:         time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
