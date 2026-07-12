package server

import (
	"net/http"
	"sync"
	"time"
)

// consentHistoryEntry represents a consent grant or revocation event.
type consentHistoryEntry struct {
	ID         string   `json:"id"`
	ClientID   string   `json:"client_id"`
	UserID     string   `json:"user_id"`
	Scopes     []string `json:"scopes"`
	Action     string   `json:"action"` // granted, revoked, modified
	Timestamp  string   `json:"timestamp"`
	IPAddress  string   `json:"ip_address"`
}

var consentHistoryStore = struct {
	sync.RWMutex
	entries []consentHistoryEntry
}{entries: []consentHistoryEntry{
	{ID: "ch-1", ClientID: "web-app", UserID: "user-001", Scopes: []string{"openid", "profile", "email"}, Action: "granted", Timestamp: time.Now().UTC().Add(-30*24*time.Hour).Format(time.RFC3339), IPAddress: "192.168.1.10"},
	{ID: "ch-2", ClientID: "analytics-3p", UserID: "user-001", Scopes: []string{"read:audit"}, Action: "granted", Timestamp: time.Now().UTC().Add(-15*24*time.Hour).Format(time.RFC3339), IPAddress: "192.168.1.10"},
	{ID: "ch-3", ClientID: "analytics-3p", UserID: "user-001", Scopes: []string{"read:audit"}, Action: "revoked", Timestamp: time.Now().UTC().Add(-5*24*time.Hour).Format(time.RFC3339), IPAddress: "10.0.0.22"},
	{ID: "ch-4", ClientID: "mobile-ios", UserID: "user-003", Scopes: []string{"openid", "profile", "email", "offline_access"}, Action: "granted", Timestamp: time.Now().UTC().Add(-10*24*time.Hour).Format(time.RFC3339), IPAddress: "10.0.0.5"},
	{ID: "ch-5", ClientID: "web-app", UserID: "user-003", Scopes: []string{"openid", "profile", "email"}, Action: "modified", Timestamp: time.Now().UTC().Add(-3*24*time.Hour).Format(time.RFC3339), IPAddress: "192.168.1.30"},
	{ID: "ch-6", ClientID: "admin-cli", UserID: "user-005", Scopes: []string{"openid", "admin", "read:users"}, Action: "granted", Timestamp: time.Now().UTC().Add(-7*24*time.Hour).Format(time.RFC3339), IPAddress: "172.16.0.8"},
	{ID: "ch-7", ClientID: "web-app", UserID: "user-005", Scopes: []string{"openid", "profile"}, Action: "revoked", Timestamp: time.Now().UTC().Add(-1*24*time.Hour).Format(time.RFC3339), IPAddress: "172.16.0.8"},
	{ID: "ch-8", ClientID: "service-backend", UserID: "user-007", Scopes: []string{"read:users", "write:users"}, Action: "granted", Timestamp: time.Now().UTC().Add(-2*24*time.Hour).Format(time.RFC3339), IPAddress: "10.0.0.50"},
}}

// GET /api/v1/oauth/consents/history?client_id=X&user_id=Y&action=granted
func handleConsentsHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	clientFilter := r.URL.Query().Get("client_id")
	userFilter := r.URL.Query().Get("user_id")
	actionFilter := r.URL.Query().Get("action")

	consentHistoryStore.RLock()
	result := []consentHistoryEntry{}
	for _, e := range consentHistoryStore.entries {
		if clientFilter != "" && e.ClientID != clientFilter {
			continue
		}
		if userFilter != "" && e.UserID != userFilter {
			continue
		}
		if actionFilter != "" && e.Action != actionFilter {
			continue
		}
		result = append(result, e)
	}
	consentHistoryStore.RUnlock()

	// Summary
	byAction := map[string]int{}
	byClient := map[string]int{}
	for _, e := range result {
		byAction[e.Action]++
		byClient[e.ClientID]++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"history":      result,
		"total":        len(result),
		"by_action":    byAction,
		"by_client":    byClient,
		"checked_at":   time.Now().UTC().Format(time.RFC3339),
	})
}
