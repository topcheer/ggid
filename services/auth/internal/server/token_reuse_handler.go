package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// tokenReuseEvent records a suspicious token reuse attempt.
type tokenReuseEvent struct {
	ID         string `json:"id"`
	TokenID    string `json:"token_id"`
	TokenType  string `json:"token_type"` // access, refresh, rotated
	UserID     string `json:"user_id"`
	IPAddress  string `json:"ip_address"`
	UserAgent  string `json:"user_agent"`
	EventType  string `json:"event_type"` // revoked_reuse, rotated_reuse, expired_reuse
	DetectedAt string `json:"detected_at"`
}

var tokenReuseStore = struct {
	sync.RWMutex
	events []tokenReuseEvent
}{events: []tokenReuseEvent{}}

// GET /api/v1/auth/token-reuse-check?user_id=X&hours=24
// POST /api/v1/auth/token-reuse-check — record a reuse event for testing
func (h *Handler) handleTokenReuseCheck(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		hoursStr := r.URL.Query().Get("hours")
		hours := 24
		if hoursStr != "" {
			var n int
			fmt.Sscanf(hoursStr, "%d", &n)
			if n > 0 {
				hours = n
			}
		}

		userFilter := r.URL.Query().Get("user_id")

		cutoff := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)

		tokenReuseStore.RLock()
		suspicious := []tokenReuseEvent{}
		uniqueUsers := map[string]bool{}
		uniqueIPs := map[string]bool{}

		for _, e := range tokenReuseStore.events {
			if e.DetectedAt == "" {
				continue
			}
			t, err := time.Parse(time.RFC3339, e.DetectedAt)
			if err != nil {
				continue
			}
			if !t.After(cutoff) {
				continue
			}
			if userFilter != "" && e.UserID != userFilter {
				continue
			}
			suspicious = append(suspicious, e)
			uniqueUsers[e.UserID] = true
			if e.IPAddress != "" {
				uniqueIPs[e.IPAddress] = true
			}
		}
		tokenReuseStore.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"suspicious_reuses":    suspicious,
			"total_reuses":         len(suspicious),
			"affected_users":       len(uniqueUsers),
			"unique_ips":           len(uniqueIPs),
			"hours_analyzed":       hours,
			"risk_level": func() string {
				if len(suspicious) >= 10 {
					return "critical"
				} else if len(suspicious) >= 5 {
					return "high"
				} else if len(suspicious) >= 1 {
					return "medium"
				}
				return "low"
			}(),
			"checked_at": time.Now().UTC().Format(time.RFC3339),
		})

	case http.MethodPost:
		var req struct {
			TokenID   string `json:"token_id"`
			TokenType string `json:"token_type"`
			UserID    string `json:"user_id"`
			IPAddress string `json:"ip_address"`
			UserAgent string `json:"user_agent"`
			EventType string `json:"event_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.TokenID == "" || req.UserID == "" {
			writeError(w, http.StatusBadRequest, "token_id and user_id are required")
			return
		}

		event := tokenReuseEvent{
			ID:         uuid.New().String(),
			TokenID:    req.TokenID,
			TokenType:  req.TokenType,
			UserID:     req.UserID,
			IPAddress:  req.IPAddress,
			UserAgent:  req.UserAgent,
			EventType:  req.EventType,
			DetectedAt: time.Now().UTC().Format(time.RFC3339),
		}

		tokenReuseStore.Lock()
		tokenReuseStore.events = append(tokenReuseStore.events, event)
		// Keep last 500
		if len(tokenReuseStore.events) > 500 {
			tokenReuseStore.events = tokenReuseStore.events[len(tokenReuseStore.events)-500:]
		}
		tokenReuseStore.Unlock()

		writeJSON(w, http.StatusCreated, event)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
