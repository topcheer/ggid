package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

// clientRateLimitConfig holds per-client rate limit settings.
type clientRateLimitConfig struct {
	ClientID       string `json:"client_id"`
	RequestsPerMin int    `json:"requests_per_min"`
	Burst          int    `json:"burst"`
	DailyQuota     int    `json:"daily_quota"`
}

var clientRateLimitStore = struct {
	sync.RWMutex
	configs map[string]*clientRateLimitConfig
}{configs: map[string]*clientRateLimitConfig{
	"web-app":         {ClientID: "web-app", RequestsPerMin: 600, Burst: 100, DailyQuota: 500000},
	"mobile-ios":      {ClientID: "mobile-ios", RequestsPerMin: 300, Burst: 50, DailyQuota: 200000},
	"admin-cli":       {ClientID: "admin-cli", RequestsPerMin: 60, Burst: 10, DailyQuota: 10000},
	"service-backend": {ClientID: "service-backend", RequestsPerMin: 1200, Burst: 200, DailyQuota: 1000000},
}}

// GET /api/v1/oauth/clients/{id}/rate-limits — get rate limit config
// PUT /api/v1/oauth/clients/{id}/rate-limits — update rate limit config
func handleClientRateLimits(w http.ResponseWriter, r *http.Request) {
	// Extract client ID
	clientID := strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/")
	clientID = strings.TrimSuffix(clientID, "/rate-limits")
	clientID = strings.TrimSuffix(clientID, "/")
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id is required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		clientRateLimitStore.RLock()
		cfg, exists := clientRateLimitStore.configs[clientID]
		clientRateLimitStore.RUnlock()

		if !exists {
			cfg = &clientRateLimitConfig{
				ClientID: clientID, RequestsPerMin: 100, Burst: 20, DailyQuota: 50000,
			}
		}

		writeJSON(w, http.StatusOK, cfg)

	case http.MethodPut, http.MethodPost:
		var req struct {
			RequestsPerMin int `json:"requests_per_min"`
			Burst          int `json:"burst"`
			DailyQuota     int `json:"daily_quota"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		if req.RequestsPerMin <= 0 {
			req.RequestsPerMin = 100
		}
		if req.Burst <= 0 {
			req.Burst = 20
		}
		if req.DailyQuota <= 0 {
			req.DailyQuota = 50000
		}

		cfg := &clientRateLimitConfig{
			ClientID:       clientID,
			RequestsPerMin: req.RequestsPerMin,
			Burst:          req.Burst,
			DailyQuota:     req.DailyQuota,
		}

		clientRateLimitStore.Lock()
		clientRateLimitStore.configs[clientID] = cfg
		clientRateLimitStore.Unlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"client_id":       clientID,
			"requests_per_min": req.RequestsPerMin,
			"burst":           req.Burst,
			"daily_quota":     req.DailyQuota,
			"updated":         true,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
