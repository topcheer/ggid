package server

import (
	"net/http"
	"strings"
	"time"
)

// GET /api/v1/auth/sessions/{id}/fingerprint
func (h *Handler) handleSessionFingerprint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/sessions/")
	sessionID = strings.TrimSuffix(sessionID, "/fingerprint")
	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": sessionID,
		"fingerprint": map[string]any{
			"user_agent_hash": "a1b2c3d4e5f6", "user_agent": "Mozilla/5.0 (Macintosh)",
			"screen": "1920x1080", "timezone": "America/Los_Angeles",
			"plugins_hash": "f7e8d9c0b1a2", "language": "en-US", "platform": "MacIntel",
		},
		"first_seen": "2026-07-01T08:00:00Z",
		"last_seen": time.Now().UTC().Format(time.RFC3339),
		"matches_baseline": true,
		"hijack_indicators": []string{},
	})
}
