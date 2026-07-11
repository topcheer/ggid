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

// deviceBindingStore holds in-memory device bindings keyed by token JTI.
type deviceBindingStore struct {
	mu        sync.RWMutex
	bindings  map[string]*DeviceBinding // keyed by token JTI
}

var deviceBindings = &deviceBindingStore{bindings: make(map[string]*DeviceBinding)}

// BindDevice associates a device fingerprint with a session token.
// Once bound, any request from a different fingerprint for the same token is rejected.
func (store *deviceBindingStore) Bind(jti, fingerprint, userAgent, ip string) *DeviceBinding {
	store.mu.Lock()
	defer store.mu.Unlock()
	b := &DeviceBinding{
		SessionID:   uuid.New().String(),
		TokenJTI:    jti,
		Fingerprint: fingerprint,
		UserAgent:   userAgent,
		IPAddress:   ip,
		BoundAt:     time.Now().UTC(),
	}
	store.bindings[jti] = b
	return b
}

// CheckBinding verifies that a token's JTI matches the bound fingerprint.
// Returns (true, nil) if not bound (no binding exists), (true, binding) if bound and match,
// (false, binding) if bound but fingerprint mismatch.
func (store *deviceBindingStore) Check(jti, fingerprint string) (bool, *DeviceBinding) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	b, ok := store.bindings[jti]
	if !ok {
		return true, nil // not bound → allow
	}
	if b.Fingerprint == fingerprint {
		return true, b // bound + match → allow
	}
	return false, b // bound + mismatch → deny
}

// Unbind removes a device binding for the given JTI.
func (store *deviceBindingStore) Unbind(jti string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, ok := store.bindings[jti]; !ok {
		return false
	}
	delete(store.bindings, jti)
	return true
}

// ListByToken returns all device bindings (admin view).
func (store *deviceBindingStore) List() []*DeviceBinding {
	store.mu.RLock()
	defer store.mu.RUnlock()
	result := make([]*DeviceBinding, 0, len(store.bindings))
	for _, b := range store.bindings {
		result = append(result, b)
	}
	return result
}

// handleBindDevice handles POST /api/v1/auth/sessions/bind-device.
// Binds a device fingerprint to a session token JTI.
// Subsequent requests from other devices with the same token will be rejected (403).
func (h *Handler) handleBindDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		TokenJTI    string `json:"token_jti"`
		Fingerprint string `json:"device_fingerprint"`
		UserAgent   string `json:"user_agent"`
		IPAddress   string `json:"ip_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.TokenJTI == "" {
		writeError(w, http.StatusBadRequest, "token_jti is required")
		return
	}
	if req.Fingerprint == "" {
		writeError(w, http.StatusBadRequest, "device_fingerprint is required")
		return
	}

	binding := deviceBindings.Bind(req.TokenJTI, req.Fingerprint, req.UserAgent, req.IPAddress)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "bound",
		"binding": binding,
	})
}

// handleCheckDevice handles POST /api/v1/auth/sessions/check-device.
// Verifies whether the requesting device matches the bound fingerprint.
func (h *Handler) handleCheckDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		TokenJTI    string `json:"token_jti"`
		Fingerprint string `json:"device_fingerprint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.TokenJTI == "" || req.Fingerprint == "" {
		writeError(w, http.StatusBadRequest, "token_jti and device_fingerprint are required")
		return
	}

	allowed, binding := deviceBindings.Check(req.TokenJTI, req.Fingerprint)
	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"allowed":    false,
			"reason":     "device fingerprint mismatch",
			"bound_to":   binding.Fingerprint,
			"bound_at":   binding.BoundAt,
		})
		return
	}

	resp := map[string]any{
		"allowed": true,
	}
	if binding != nil {
		resp["binding"] = binding
	} else {
		resp["binding"] = nil
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleUnbindDevice handles DELETE /api/v1/auth/sessions/unbind-device.
func (h *Handler) handleUnbindDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	jti := r.URL.Query().Get("token_jti")
	if jti == "" {
		writeError(w, http.StatusBadRequest, "token_jti is required")
		return
	}

	if !deviceBindings.Unbind(jti) {
		writeError(w, http.StatusNotFound, "no binding found for this token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "unbound"})
}
