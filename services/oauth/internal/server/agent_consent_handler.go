package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type PendingConsentRequest struct {
	RequestID        string   `json:"request_id"`
	AgentName        string   `json:"agent_name"`
	RequestedScopes  []string `json:"requested_scopes"`
	Resource         string   `json:"resource"`
	Justification    string   `json:"justification"`
	RequestedAt      string   `json:"requested_at"`
}

type ConsentHistoryEntry struct {
	AgentID    string `json:"agent_id"`
	UserID     string `json:"user_id"`
	Scopes     []string `json:"scopes"`
	GrantedAt  string `json:"granted_at"`
	RevokedAt  string `json:"revoked_at,omitempty"`
}

type AgentConsentResult struct {
	AgentID          string                  `json:"agent_id"`
	PendingRequests  []PendingConsentRequest `json:"pending_requests"`
	ConsentHistory   []ConsentHistoryEntry   `json:"consent_history"`
	AutoExpireHours  int                     `json:"auto_expire_hours"`
}

var agentConsentStore sync.Map

func handleAgentConsent(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	agentID := "unknown"
	if len(parts) >= 5 {
		agentID = parts[4]
	}

	switch r.Method {
	case http.MethodGet:
		result := AgentConsentResult{
			AgentID: agentID,
			PendingRequests: []PendingConsentRequest{
				{RequestID: "cr-001", AgentName: "agent-" + agentID, RequestedScopes: []string{"read:users", "write:audit"}, Resource: "identity-service", Justification: "Batch user provisioning", RequestedAt: "2025-01-15T08:00:00Z"},
				{RequestID: "cr-002", AgentName: "agent-" + agentID, RequestedScopes: []string{"admin:config"}, Resource: "policy-service", Justification: "Emergency policy update", RequestedAt: "2025-01-14T16:00:00Z"},
			},
			ConsentHistory: []ConsentHistoryEntry{
				{AgentID: agentID, UserID: "u-001", Scopes: []string{"read:users"}, GrantedAt: "2025-01-10T10:00:00Z"},
				{AgentID: agentID, UserID: "u-002", Scopes: []string{"read:users", "write:audit"}, GrantedAt: "2025-01-08T14:00:00Z", RevokedAt: "2025-01-12T09:00:00Z"},
				{AgentID: agentID, UserID: "u-003", Scopes: []string{"read:policies"}, GrantedAt: "2025-01-05T11:00:00Z"},
			},
			AutoExpireHours: 168,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPost:
		var req struct {
			Action   string   `json:"action"`
			RequestID string  `json:"request_id"`
			UserID   string   `json:"user_id"`
			Scopes   []string `json:"scopes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		entry := ConsentHistoryEntry{
			AgentID: agentID, UserID: req.UserID, Scopes: req.Scopes,
			GrantedAt: time.Now().UTC().Format(time.RFC3339),
		}
		key := fmt.Sprintf("%s:%s:%s", agentID, req.RequestID, req.UserID)
		agentConsentStore.Store(key, entry)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(entry)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
