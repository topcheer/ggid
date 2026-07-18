package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type DeviceTrust struct {
	DeviceID    string    `json:"device_id"`
	Managed     bool      `json:"managed"`
	Encrypted   bool      `json:"encrypted"`
	CompliantOS bool      `json:"compliant_os"`
	Jailbreak   bool      `json:"jailbreak"`
	LastSeen    time.Time `json:"last_seen"`
	TrustScore  int       `json:"trust_score"`
}

func (h *Handler) handleDeviceTrustScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deviceID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/devices/")
	deviceID = strings.TrimSuffix(deviceID, "/trust-score")
	// Try PG first.
	if h.memMapRepo != nil {
		trusts, _ := h.memMapRepo.ListDeviceTrusts(r.Context())
		for _, row := range trusts {
			if did, ok := row["device_id"]; ok && fmt.Sprintf("%v", did) == deviceID {
				writeJSON(w, http.StatusOK, row)
				return
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"device_id": deviceID, "trust_score": 0,
		"managed": false, "encrypted": false,
		"compliant_os": false, "jailbreak": false,
	})
}

func (h *Handler) handleDeviceReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deviceID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/devices/")
	deviceID = strings.TrimSuffix(deviceID, "/report")
	var req struct {
		Managed     bool `json:"managed"`
		Encrypted   bool `json:"encrypted"`
		CompliantOS bool `json:"compliant_os"`
		Jailbreak   bool `json:"jailbreak"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	score := 0
	if req.Managed { score += 30 }
	if req.Encrypted { score += 25 }
	if req.CompliantOS { score += 25 }
	if !req.Jailbreak { score += 20 }
	dt := &DeviceTrust{
		DeviceID: deviceID, Managed: req.Managed, Encrypted: req.Encrypted,
		CompliantOS: req.CompliantOS, Jailbreak: req.Jailbreak,
		LastSeen: time.Now().UTC(), TrustScore: score,
	}
	if h.memMapRepo != nil {
		h.memMapRepo.StoreDeviceTrust(r.Context(), map[string]any{
			"id": deviceID, "device_id": deviceID,
			"managed": dt.Managed, "encrypted": dt.Encrypted,
			"compliant_os": dt.CompliantOS, "jailbreak": dt.Jailbreak,
			"trust_score": dt.TrustScore,
		})
	}
	writeJSON(w, http.StatusOK, dt)
}
