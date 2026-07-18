package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type BlockedIP struct {
	IP        string    `json:"ip"`
	Reason    string    `json:"reason"`
	BlockedAt time.Time `json:"blocked_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Auto      bool      `json:"auto"`
}

var (
	credStuffingMu sync.RWMutex
	credStuffingIPs = make(map[string]*BlockedIP)
	autoBlockRules = []map[string]any{
		{"rule": "failed_logins_threshold", "value": 10, "window_minutes": 5, "action": "block_1h"},
		{"rule": "unique_user_attempts", "value": 15, "window_minutes": 10, "action": "block_24h"},
		{"rule": "known_credential_stuffing_pattern", "action": "block_permanent"},
	}
)

// handleDetectCredentialStuffing is an alias for the detection endpoint
func (h *Handler) handleDetectCredentialStuffing(w http.ResponseWriter, r *http.Request) {
	h.handleCredentialStuffing(w, r)
}

// POST /api/v1/auth/credential-stuffing/block — manually block IP
// GET /api/v1/auth/credential-stuffing/blocked — list blocked IPs + rules
// POST /api/v1/auth/detect-credential-stuffing — detect credential stuffing attempts
func (h *Handler) handleCredentialStuffing(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			IP       string `json:"ip"`
			Reason   string `json:"reason"`
			Duration string `json:"duration"` // e.g. "1h", "24h", "permanent"
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.IP == "" {
			writeJSONError(w, http.StatusBadRequest, "ip required")
			return
		}

		now := time.Now().UTC()
		expiry := now.Add(1 * time.Hour)
		switch req.Duration {
		case "24h":
			expiry = now.Add(24 * time.Hour)
		case "permanent":
			expiry = time.Time{} // zero = no expiry
		}
		blocked := &BlockedIP{IP: req.IP, Reason: req.Reason, BlockedAt: now, ExpiresAt: expiry, Auto: false}
		credStuffingMu.Lock()
		credStuffingIPs[req.IP] = blocked
		credStuffingMu.Unlock()
		// PG write-through
		if h.memMapRepo != nil {
			h.memMapRepo.StoreJSON(r.Context(), "auth_cred_stuffing_json", req.IP, map[string]any{
				"ip": req.IP, "reason": req.Reason,
				"blocked_at": now, "expires_at": expiry, "auto": false,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "blocked", "ip": req.IP, "blocked_at": now})
		return
	}

		if r.Method == http.MethodGet {
		// Try PG first, fall back to in-memory map
		if h.memMapRepo != nil {
			rows, _ := h.memMapRepo.ListJSON(r.Context(), "auth_cred_stuffing_json")
			if len(rows) > 0 {
				writeJSON(w, http.StatusOK, map[string]any{
					"blocked_ips": rows, "total_blocked": len(rows),
					"auto_block_rules": autoBlockRules,
				})
				return
			}
		}
		credStuffingMu.RLock()
		blocked := make([]*BlockedIP, 0, len(credStuffingIPs))
		for _, b := range credStuffingIPs {
			blocked = append(blocked, b)
		}
		credStuffingMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"blocked_ips":      blocked,
			"total_blocked":    len(blocked),
			"auto_block_rules": autoBlockRules,
		})
		return
	}

	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}
