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
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/sessions/")
	sessionID = strings.TrimSuffix(sessionID, "/reevaluate")

	var req struct {
		IPAddr    string `json:"ip_address"`
		DeviceFP  string `json:"device_fingerprint"`
		GeoRegion string `json:"geo_region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSONError(w, http.StatusBadRequest, "invalid request body"); return }
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
	// Try PG first for previous state
	var prevPG map[string]any
	if h.memMapRepo != nil {
		prevPG, _ = h.memMapRepo.GetJSON(r.Context(), "auth_session_risks_json", sessionID)
		if prevPG != nil {
			if prevIP, _ := prevPG["ip_address"].(string); prevIP != "" && prevIP != req.IPAddr {
				score += 25
			}
			if score > 100 { score = 100 }
			now := time.Now().UTC()
			sr := &SessionRisk{
				SessionID: sessionID, RiskScore: score,
				IPAddr: req.IPAddr, DeviceFP: req.DeviceFP,
				GeoRegion: req.GeoRegion, LastEvalAt: now,
			}
			sessionRiskMu.Lock()
			sessionRisks[sessionID] = sr
			sessionRiskMu.Unlock()
			// PG write-through
			h.memMapRepo.StoreJSON(r.Context(), "auth_session_risks_json", sessionID, map[string]any{
				"session_id": sessionID, "risk_score": score,
				"ip_address": req.IPAddr, "device_fingerprint": req.DeviceFP,
				"geo_region": req.GeoRegion, "last_eval_at": now,
			})
			ipChanged := false
			if prevIP, _ := prevPG["ip_address"].(string); prevIP != "" && prevIP != req.IPAddr {
				ipChanged = true
			}
			action := "allow"
			if score >= 70 {
				action = "revoke_session"
			} else if score >= 40 {
				action = "require_step_up"
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"session_id": sessionID, "new_risk_score": score, "action": action,
				"ip_changed": ipChanged, "evaluated_at": now.Format(time.RFC3339),
				"factors": map[string]any{
					"unknown_ip": req.IPAddr == "" || req.IPAddr == "0.0.0.0",
					"unknown_device": req.DeviceFP == "unknown",
					"unknown_geo": req.GeoRegion == "unknown",
				},
			})
			return
		}
	}
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
	// PG write-through
	if h.memMapRepo != nil {
		h.memMapRepo.StoreJSON(r.Context(), "auth_session_risks_json", sessionID, map[string]any{
			"session_id": sessionID, "risk_score": score,
			"ip_address": req.IPAddr, "device_fingerprint": req.DeviceFP,
			"geo_region": req.GeoRegion, "last_eval_at": now,
		})
	}

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
