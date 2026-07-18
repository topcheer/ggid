package server

import (
	"net/http"
	"strings"
)

// GET /api/v1/auth/sessions/{id}/fingerprint
// Returns 501 Not Implemented — real fingerprint data requires session table
// schema changes to store user_agent_hash, screen, plugins, etc. at login time.
// Returning fake fingerprint data would be dangerous for SOC/hijack detection.
func (h *Handler) handleSessionFingerprint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/sessions/")
	sessionID = strings.TrimSuffix(sessionID, "/fingerprint")
	_ = sessionID // validated when real implementation lands

	writeJSONError(w, http.StatusNotImplemented, "session fingerprint not yet implemented — requires session schema upgrade to store client fingerprint data at login")
}
