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

	// Iterate the binding cache (sync.Map) for active bindings.
	statuses := []map[string]any{}
	bindingCache.Range(func(key, value any) bool {
		b, ok := value.(*DeviceBinding)
		if !ok {
			return true
		}
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
		return true
	})

	uniqueDevices := map[string]bool{}
	for _, s := range statuses {
		if fp, ok := s["device_fingerprint"].(string); ok {
			uniqueDevices[fp] = true
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessions":       statuses,
		"total_sessions": len(statuses),
		"unique_devices": len(uniqueDevices),
		"checked_at":     time.Now().UTC().Format(time.RFC3339),
	})
}
