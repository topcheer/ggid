package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SessionInfo struct {
	SessionID      string   `json:"session_id"`
	Device         string   `json:"device"`
	IPAddress      string   `json:"ip_address"`
	Location       string   `json:"location"`
	CreatedAt      string   `json:"created_at"`
	LastActive     string   `json:"last_active"`
	MFAVerified    bool     `json:"mfa_verified"`
	Scopes         []string `json:"scopes"`
	TokenExpiry    string   `json:"token_expiry"`
	SessionBinding string   `json:"session_binding"`
}

type SessionInspectResult struct {
	UserID      string        `json:"user_id"`
	Sessions    []SessionInfo `json:"sessions"`
	ActiveCount int           `json:"active_count"`
	RiskScore   float64       `json:"risk_score"`
}

// GET /api/v1/auth/sessions/{userID}/inspect
// Returns real session data from the sessions table.
func (h *Handler) handleSessionInspect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userPathID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/sessions/")
	userPathID = strings.TrimSuffix(userPathID, "/inspect")

	result := SessionInspectResult{
		UserID:   userPathID,
		Sessions: []SessionInfo{},
	}

	// Query real sessions from DB via pool
	if h.pool != nil {
		targetUID, parseErr := uuid.Parse(userPathID)
		if parseErr == nil {
			rows, err := h.pool.Query(r.Context(), `
				SELECT id, COALESCE(ip_address,''), COALESCE(user_agent,''),
				       created_at, updated_at, expires_at
				FROM sessions
				WHERE user_id = $1 AND revoked_at IS NULL
				ORDER BY updated_at DESC LIMIT 50
			`, targetUID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var (
						sessID, ip, ua                 string
						createdAt, updatedAt, expiresAt time.Time
					)
					if err := rows.Scan(&sessID, &ip, &ua, &createdAt, &updatedAt, &expiresAt); err != nil {
						continue
					}
					result.Sessions = append(result.Sessions, SessionInfo{
						SessionID:   sessID,
						Device:      ua,
						IPAddress:   ip,
						CreatedAt:   createdAt.UTC().Format(time.RFC3339),
						LastActive:  updatedAt.UTC().Format(time.RFC3339),
						TokenExpiry: expiresAt.UTC().Format(time.RFC3339),
					})
				}
			}
		}
	}
	result.ActiveCount = len(result.Sessions)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
