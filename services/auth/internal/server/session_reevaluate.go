package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SessionRisk tracks risk evaluation for active sessions.
type SessionRisk struct {
	SessionID  string  `json:"session_id"`
	RiskScore  int     `json:"risk_score"`
	IPAddr     string  `json:"ip_address"`
	DeviceFP   string  `json:"device_fingerprint"`
	GeoRegion  string  `json:"geo_region"`
	LastEvalAt time.Time `json:"last_eval_at"`
}

var (
	sessionRiskMu sync.RWMutex
	sessionRisks  = make(map[string]*SessionRisk)
)

// POST /api/v1/auth/sessions/{id}/reevaluate — re-evaluate session risk.
func (h *Handler) handleSessionReevaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/sessions/")
	sessionID = strings.TrimSuffix(sessionID, "/reevaluate")

	var req struct {
		IPAddr    string `json:"ip_address"`
		DeviceFP  string `json:"device_fingerprint"`
		GeoRegion string `json:"geo_region"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.IPAddr == "" { req.IPAddr = r.RemoteAddr }
	if req.DeviceFP == "" { req.DeviceFP = "unknown" }
	if req.GeoRegion == "" { req.GeoRegion = "unknown" }

	// Compute risk factors
	score := 10 // baseline
	if req.IPAddr == "" || req.IPAddr == "0.0.0.0" {
		score += 30
	}
	if req.DeviceFP == "unknown" {
		score += 20
	}
	if req.GeoRegion == "unknown" {
		score += 15
	}

	// Check for IP change from previous evaluation
	sessionRiskMu.Lock()
	prev, exists := sessionRisks[sessionID]
	if exists && prev.IPAddr != "" && prev.IPAddr != req.IPAddr {
		score += 25 // IP change = elevated risk
	}
	if score > 100 { score = 100 }

	now := time.Now().UTC()
	sr := &SessionRisk{
		SessionID: sessionID, RiskScore: score,
		IPAddr: req.IPAddr, DeviceFP: req.DeviceFP,
		GeoRegion: req.GeoRegion, LastEvalAt: now,
	}
	sessionRisks[sessionID] = sr
	sessionRiskMu.Unlock()

	action := "allow"
	if score >= 70 {
		action = "revoke_session"
	} else if score >= 40 {
		action = "require_step_up"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id":    sessionID,
		"new_risk_score": score,
		"action":        action,
		"ip_changed":    exists && prev != nil && prev.IPAddr != req.IPAddr,
		"evaluated_at":  now.Format(time.RFC3339),
		"factors": map[string]any{
			"unknown_ip":     req.IPAddr == "" || req.IPAddr == "0.0.0.0",
			"unknown_device": req.DeviceFP == "unknown",
			"unknown_geo":    req.GeoRegion == "unknown",
		},
	})
}
