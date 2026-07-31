package server

import (
	"net/http"
	"sync"
	"time"
)

// audienceMismatch represents a token audience validation failure.
type audienceMismatch struct {
	ID               string `json:"id"`
	TokenID          string `json:"token_id"`
	UserID           string `json:"user_id"`
	ClientID         string `json:"client_id"`
	ExpectedAudience string `json:"expected_audience"`
	ActualAudience   string `json:"actual_audience"`
	Resource         string `json:"resource"`
	Timestamp        string `json:"timestamp"`
	Blocked          bool   `json:"blocked"`
}

var audienceMismatchStore = struct {
	sync.RWMutex
	mismatches []audienceMismatch
}{}

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
