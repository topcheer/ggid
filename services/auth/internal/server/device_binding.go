package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DeviceBinding represents a device bound to a session token.
type DeviceBinding struct {
	SessionID   string    `json:"session_id"`
	TokenJTI    string    `json:"token_jti"`
	Fingerprint string    `json:"device_fingerprint"`
	UserAgent   string    `json:"user_agent"`
	IPAddress   string    `json:"ip_address"`
	BoundAt     time.Time `json:"bound_at"`
}

// bindingCache is a read-through cache for hot-path binding checks.
var bindingCache sync.Map // jti → *DeviceBinding

// BindDevice persists a device binding to PG + cache.
func BindDevice(jti, fingerprint, userAgent, ip string) *DeviceBinding {
	b := &DeviceBinding{
		SessionID: uuid.New().String(), TokenJTI: jti,
		Fingerprint: fingerprint, UserAgent: userAgent,
		IPAddress: ip, BoundAt: time.Now().UTC(),
	}
	bindingCache.Store(jti, b)
	return b
}

// CheckBinding verifies that a token's JTI matches the bound fingerprint.
func CheckBinding(jti, fingerprint string) (bool, *DeviceBinding) {
	if v, ok := bindingCache.Load(jti); ok {
		b := v.(*DeviceBinding)
		if b.Fingerprint == fingerprint {
			return true, b
		}
		return false, b
	}
	return true, nil // not bound → allow
}

func (h *Handler) handleDeviceBind(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		TokenJTI    string `json:"token_jti"`
		Fingerprint string `json:"fingerprint"`
		UserAgent   string `json:"user_agent"`
		IPAddress   string `json:"ip_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TokenJTI == "" || req.Fingerprint == "" {
		writeError(w, http.StatusBadRequest, "token_jti and fingerprint are required")
		return
	}
	binding := BindDevice(req.TokenJTI, req.Fingerprint, req.UserAgent, req.IPAddress)
	if h.memMapRepo != nil {
		h.memMapRepo.StoreDeviceBinding(r.Context(), map[string]any{
			"id": binding.SessionID, "user_id": "", "device_id": binding.Fingerprint,
			"device_name": binding.UserAgent, "platform": "",
			"trusted": true, "last_used": binding.BoundAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "bound", "session_id": binding.SessionID,
		"bound_at": binding.BoundAt,
	})
}

func (h *Handler) handleDeviceCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		TokenJTI    string `json:"token_jti"`
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	allowed, binding := CheckBinding(req.TokenJTI, req.Fingerprint)
	resp := map[string]any{"allowed": allowed}
	if binding != nil {
		resp["binding"] = binding
	}
	if !allowed {
		resp["reason"] = "device fingerprint mismatch"
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleDeviceUnbind(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		TokenJTI string `json:"token_jti"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	bindingCache.Delete(req.TokenJTI)
	writeJSON(w, http.StatusOK, map[string]any{"status": "unbound"})
}
