package server

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type TrustedDevice struct {
	DeviceID   string    `json:"device_id"`
	UserID     string    `json:"user_id"`
	Name       string    `json:"name"`
	Fingerprint string   `json:"fingerprint"`
	TrustedAt  time.Time `json:"trusted_at"`
	SkipMFA    bool      `json:"skip_mfa"`
}

var (
	trustedDeviceMu sync.RWMutex
	trustedDevices  = make(map[string]*TrustedDevice)
)

// GET /api/v1/auth/devices/trusted?user_id=X
// DELETE /api/v1/auth/devices/trusted/{device_id}
func (h *Handler) handleTrustedDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		userID := r.URL.Query().Get("user_id")
		// Try PG first, fall back to in-memory map
		if h.memMapRepo != nil {
			rows, _ := h.memMapRepo.ListJSON(r.Context(), "auth_trusted_devices_json")
			if len(rows) > 0 {
				var devices []map[string]any
				for _, row := range rows {
					uid, _ := row["user_id"].(string)
					if userID == "" || uid == userID {
						devices = append(devices, row)
					}
				}
				writeJSON(w, http.StatusOK, map[string]any{"trusted_devices": devices, "count": len(devices)})
				return
			}
		}
		trustedDeviceMu.RLock()
		var devices []*TrustedDevice
		for _, d := range trustedDevices {
			if userID == "" || d.UserID == userID {
				devices = append(devices, d)
			}
		}
		trustedDeviceMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"trusted_devices": devices, "count": len(devices)})
		return
	}

	if r.Method == http.MethodDelete {
		deviceID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/devices/trusted/")
		if deviceID == "" || deviceID == "/api/v1/auth/devices/trusted" {
			writeError(w, http.StatusBadRequest, "device_id required")
			return
		}
		trustedDeviceMu.Lock()
		delete(trustedDevices, deviceID)
		trustedDeviceMu.Unlock()
		// PG delete
		if h.memMapRepo != nil {
			h.memMapRepo.DeleteJSON(r.Context(), "auth_trusted_devices_json", deviceID)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "revoked", "device_id": deviceID, "message": "device trust removed — MFA required on next login"})
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
