package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

type UsagePolicy struct {
	MaxTokensPerDay     int    `json:"max_tokens_per_day"`
	MaxRequestsPerMin   int    `json:"max_requests_per_min"`
	AllowedScopes       []string `json:"allowed_scopes"`
	RateLimitStrategy   string `json:"rate_limit_strategy"`
}

var (
	usagePolicyMu sync.RWMutex
	usagePolicies = make(map[string]*UsagePolicy)
)

// GET/PUT /api/v1/oauth/clients/{id}/usage-policy
func handleUsagePolicy(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/usage-policy") {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	clientID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/"), "/usage-policy")
	if clientID == "" || strings.Contains(clientID, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid client_id"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		usagePolicyMu.RLock()
		p, ok := usagePolicies[clientID]
		usagePolicyMu.RUnlock()
		if !ok {
			p = &UsagePolicy{MaxTokensPerDay: 10000, MaxRequestsPerMin: 100, AllowedScopes: []string{"openid", "profile"}, RateLimitStrategy: "token_bucket"}
		}
		writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "policy": p})
	case http.MethodPut, http.MethodPost:
		var p UsagePolicy
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
			return
		}
		usagePolicyMu.Lock(); usagePolicies[clientID] = &p; usagePolicyMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "client_id": clientID, "policy": p})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}
