package server

import (
	"net/http"
	"sync"
	"time"
)

// audienceMismatch represents a token audience validation failure.
type audienceMismatch struct {
	ID              string `json:"id"`
	TokenID         string `json:"token_id"`
	UserID          string `json:"user_id"`
	ClientID        string `json:"client_id"`
	ExpectedAudience string `json:"expected_audience"`
	ActualAudience   string `json:"actual_audience"`
	Resource        string `json:"resource"`
	Timestamp       string `json:"timestamp"`
	Blocked         bool   `json:"blocked"`
}

var audienceMismatchStore = struct {
	sync.RWMutex
	mismatches []audienceMismatch
}{mismatches: []audienceMismatch{
	{ID: "am-1", TokenID: "tok-001", UserID: "user-005", ClientID: "web-app", ExpectedAudience: "api.ggid.dev", ActualAudience: "legacy.api.ggid.dev", Resource: "/api/v1/users", Timestamp: time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339), Blocked: true},
	{ID: "am-2", TokenID: "tok-002", UserID: "user-012", ClientID: "mobile-ios", ExpectedAudience: "api.ggid.dev", ActualAudience: "admin.ggid.dev", Resource: "/api/v1/admin/config", Timestamp: time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339), Blocked: true},
	{ID: "am-3", TokenID: "tok-003", UserID: "user-008", ClientID: "service-backend", ExpectedAudience: "internal.ggid.dev", ActualAudience: "api.ggid.dev", Resource: "/api/v1/audit/events", Timestamp: time.Now().UTC().Add(-4 * time.Hour).Format(time.RFC3339), Blocked: false},
	{ID: "am-4", TokenID: "tok-004", UserID: "user-015", ClientID: "web-app", ExpectedAudience: "api.ggid.dev", ActualAudience: "third-party.example.com", Resource: "/api/v1/users/profile", Timestamp: time.Now().UTC().Add(-6 * time.Hour).Format(time.RFC3339), Blocked: true},
	{ID: "am-5", TokenID: "tok-005", UserID: "user-021", ClientID: "admin-cli", ExpectedAudience: "admin.ggid.dev", ActualAudience: "api.ggid.dev", Resource: "/api/v1/policies", Timestamp: time.Now().UTC().Add(-12 * time.Hour).Format(time.RFC3339), Blocked: true},
}}

// GET /api/v1/oauth/audience-mismatches?hours=24&blocked_only=true
func handleAudienceMismatches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if hoursStr != "" {
		var n int
		for _, c := range hoursStr {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		if n > 0 && n <= 720 {
			hours = n
		}
	}
	blockedOnly := r.URL.Query().Get("blocked_only") == "true"

	cutoff := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)

	audienceMismatchStore.RLock()
	result := []audienceMismatch{}
	byClient := map[string]int{}
	blockedCount := 0
	for _, m := range audienceMismatchStore.mismatches {
		t, _ := time.Parse(time.RFC3339, m.Timestamp)
		if !t.After(cutoff) {
			continue
		}
		if blockedOnly && !m.Blocked {
			continue
		}
		result = append(result, m)
		byClient[m.ClientID]++
		if m.Blocked {
			blockedCount++
		}
	}
	audienceMismatchStore.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"mismatches":     result,
		"total":          len(result),
		"blocked_count":  blockedCount,
		"hours_analyzed": hours,
		"by_client":      byClient,
		"checked_at":     time.Now().UTC().Format(time.RFC3339),
	})
}
