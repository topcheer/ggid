package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

type UsagePolicy struct {
	MaxTokensPerDay     int      `json:"max_tokens_per_day"`
	MaxRequestsPerMin   int      `json:"max_requests_per_min"`
	AllowedScopes       []string `json:"allowed_scopes"`
	RateLimitStrategy   string   `json:"rate_limit_strategy"`
}

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
		if mapRepoVar != nil {
			data, err := mapRepoVar.Get(r.Context(), "oauth_usage_policies", clientID)
			if err == nil {
				writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "policy": data})
				return
			}
		}
		p := &UsagePolicy{MaxTokensPerDay: 10000, MaxRequestsPerMin: 100, AllowedScopes: []string{"openid", "profile"}, RateLimitStrategy: "token_bucket"}
		writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "policy": p})
	case http.MethodPut, http.MethodPost:
		var p UsagePolicy
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
			return
		}
		if mapRepoVar != nil {
			b, _ := json.Marshal(p)
			var dataMap map[string]any
			json.Unmarshal(b, &dataMap)
			mapRepoVar.Store(r.Context(), "oauth_usage_policies", clientID, dataMap)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "client_id": clientID, "policy": p})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}
