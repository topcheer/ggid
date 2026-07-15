package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
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

var (deviceTrustMu sync.RWMutex; deviceTrusts = make(map[string]*DeviceTrust))

func (h *Handler) handleDeviceTrustScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	deviceID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/devices/")
	deviceID = strings.TrimSuffix(deviceID, "/trust-score")
	deviceTrustMu.RLock(); dt, ok := deviceTrusts[deviceID]; deviceTrustMu.RUnlock()
	if !ok { writeJSON(w, http.StatusOK, map[string]any{"device_id": deviceID, "trust_score": 0, "managed": false, "encrypted": false, "compliant_os": false, "jailbreak": false}); return }
	writeJSON(w, http.StatusOK, dt)
}

func (h *Handler) handleDeviceReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	deviceID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/devices/")
	deviceID = strings.TrimSuffix(deviceID, "/report")
	var req struct{ Managed, Encrypted, CompliantOS, Jailbreak bool }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	score := 0
	if req.Managed { score += 30 }
	if req.Encrypted { score += 25 }
	if req.CompliantOS { score += 25 }
	if !req.Jailbreak { score += 20 }
	dt := &DeviceTrust{DeviceID: deviceID, Managed: req.Managed, Encrypted: req.Encrypted, CompliantOS: req.CompliantOS, Jailbreak: req.Jailbreak, LastSeen: time.Now().UTC(), TrustScore: score}
	deviceTrustMu.Lock(); deviceTrusts[deviceID] = dt; deviceTrustMu.Unlock()
	writeJSON(w, http.StatusOK, dt)
}
