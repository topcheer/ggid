package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

type SessionInfo struct {
	SessionID    string   `json:"session_id"`
	Device       string   `json:"device"`
	IPAddress    string   `json:"ip_address"`
	Location     string   `json:"location"`
	CreatedAt    string   `json:"created_at"`
	LastActive   string   `json:"last_active"`
	MFAVerified  bool     `json:"mfa_verified"`
	Scopes       []string `json:"scopes"`
	TokenExpiry  string   `json:"token_expiry"`
	SessionBinding string `json:"session_binding"`
}

type SessionInspectResult struct {
	UserID   string        `json:"user_id"`
	Sessions []SessionInfo `json:"sessions"`
	ActiveCount int        `json:"active_count"`
	RiskScore  float64     `json:"risk_score"`
}

func (h *Handler) handleSessionInspect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/sessions/")
	userID = strings.TrimSuffix(userID, "/inspect")

	result := SessionInspectResult{
		UserID: userID,
		Sessions: []SessionInfo{
			{SessionID: "sess-001", Device: "Chrome/macOS", IPAddress: "192.168.1.50", Location: "San Francisco, US", CreatedAt: "2025-01-15T08:00:00Z", LastActive: "2025-01-15T09:45:00Z", MFAVerified: true, Scopes: []string{"openid", "profile", "read:users"}, TokenExpiry: "2025-01-15T11:00:00Z", SessionBinding: "DPoP"},
			{SessionID: "sess-002", Device: "Safari/iOS", IPAddress: "10.0.0.22", Location: "San Francisco, US", CreatedAt: "2025-01-14T20:00:00Z", LastActive: "2025-01-15T07:30:00Z", MFAVerified: true, Scopes: []string{"openid", "profile"}, TokenExpiry: "2025-01-15T10:00:00Z", SessionBinding: "none"},
			{SessionID: "sess-003", Device: "Unknown/Linux", IPAddress: "203.0.113.99", Location: "Unknown", CreatedAt: "2025-01-15T03:00:00Z", LastActive: "2025-01-15T03:15:00Z", MFAVerified: false, Scopes: []string{"openid"}, TokenExpiry: "2025-01-15T04:00:00Z", SessionBinding: "none"},
		},
		ActiveCount: 2,
		RiskScore:   0.35,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
