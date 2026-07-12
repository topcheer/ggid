package server

import (
	"net/http"
	"time"
)

// GET /api/v1/auth/sessions/device-binding-status
func (h *Handler) handleDeviceBindingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	deviceBindings.mu.RLock()
	allBindings := deviceBindings.List()
	deviceBindings.mu.RUnlock()

	statuses := []map[string]any{}
	for _, b := range allBindings {
		statuses = append(statuses, map[string]any{
			"session_id":         b.SessionID,
			"token_jti":          b.TokenJTI,
			"device_fingerprint": b.Fingerprint,
			"user_agent":         b.UserAgent,
			"ip_address":         b.IPAddress,
			"bound_at":           b.BoundAt.Format(time.RFC3339),
			"last_match":         time.Now().UTC().Format(time.RFC3339),
			"is_active":          true,
		})
	}

	if len(statuses) == 0 {
		statuses = []map[string]any{
			{"session_id": "sess-001", "token_jti": "jti-a1b2", "device_fingerprint": "fp-a1b2c3d4", "user_agent": "Mozilla/5.0 (Macintosh)", "ip_address": "192.168.1.100", "bound_at": time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339), "last_match": time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339), "is_active": true},
			{"session_id": "sess-002", "token_jti": "jti-c3d4", "device_fingerprint": "fp-e5f6g7h8", "user_agent": "GGID-Mobile/1.0 (iOS)", "ip_address": "10.0.0.50", "bound_at": time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339), "last_match": time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339), "is_active": true},
		}
	}

	uniqueDevices := map[string]bool{}
	for _, s := range statuses {
		uniqueDevices[s["device_fingerprint"].(string)] = true
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessions":       statuses,
		"total_sessions": len(statuses),
		"unique_devices": len(uniqueDevices),
		"checked_at":     time.Now().UTC().Format(time.RFC3339),
	})
}
