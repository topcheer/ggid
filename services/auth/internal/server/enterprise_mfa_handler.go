package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ggid/ggid/pkg/audit"
	"github.com/google/uuid"
)

// GET /api/v1/auth/mfa/methods — returns enabled MFA methods for frontend display
func (h *Handler) handleMFAMethods(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	resp := map[string]any{
		"totp_enabled":     true, // TOTP always available
		"webauthn_enabled": true, // WebAuthn always available
		"radius_enabled":   false,
		"yubikey_enabled":  false,
	}

	if h.pool != nil {
		// Check RADIUS config
		var radiusJSON []byte
		if err := h.pool.QueryRow(r.Context(), `SELECT value::text FROM sys_config WHERE key = 'radius_config'`).Scan(&radiusJSON); err == nil {
			var cfg struct {
				Enabled bool `json:"enabled"`
			}
			if json.Unmarshal(radiusJSON, &cfg) == nil {
				resp["radius_enabled"] = cfg.Enabled
			}
		}

		// Check Yubico config
		var yubicoJSON []byte
		if err := h.pool.QueryRow(r.Context(), `SELECT value::text FROM sys_config WHERE key = 'yubico_config'`).Scan(&yubicoJSON); err == nil {
			var cfg struct {
				Enabled bool `json:"enabled"`
			}
			if json.Unmarshal(yubicoJSON, &cfg) == nil {
				resp["yubikey_enabled"] = cfg.Enabled
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- RADIUS MFA Verification ---
// POST /api/v1/auth/mfa/radius/verify
// Body: {"user_id":"...", "passcode":"...", "username":"..."}
// Forwards the passcode to a configured RADIUS server (SecurID, Duo, etc.)

type radiusVerifyRequest struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Passcode string `json:"passcode"`
}

func (h *Handler) handleMFARadiusVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req radiusVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Passcode == "" || req.Username == "" {
		writeError(w, http.StatusBadRequest, "username and passcode are required")
		return
	}

	// Read RADIUS config from sys_config
	if h.pool == nil {
		writeError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	var configJSON []byte
	err := h.pool.QueryRow(r.Context(), `SELECT value::text FROM sys_config WHERE key = 'radius_config'`).Scan(&configJSON)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "RADIUS not configured")
		return
	}

	var radiusCfg struct {
		Server  string `json:"server"`
		Secret  string `json:"secret"`
		Port    int    `json:"port"`
		Timeout int    `json:"timeout"`
		Enabled bool   `json:"enabled"`
	}
	if json.Unmarshal(configJSON, &radiusCfg) != nil {
		writeError(w, http.StatusServiceUnavailable, "invalid RADIUS config")
		return
	}

	if !radiusCfg.Enabled || radiusCfg.Server == "" {
		writeError(w, http.StatusServiceUnavailable, "RADIUS MFA is not enabled")
		return
	}

	// Forward to RADIUS server
	// In production: use github.com/layeh/radius or alexbrainman/radius
	// For now: HTTP-based proxy to RADIUS gateway (Duo Auth API, etc.)
	timeout := time.Duration(radiusCfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	// Simplified RADIUS verification via HTTP gateway
	// Real implementation would use RADIUS protocol directly
	verified := h.verifyRadiusPasscode(ctx, radiusCfg.Server, radiusCfg.Port, radiusCfg.Secret, req.Username, req.Passcode)

	// Audit
	if tid, ok := tenantCtxFromHeader(r); ok {
		event := audit.NewEvent("mfa.radius.verify", "failed", tid, uuid.Nil)
		event.ActorName = req.Username
		event.IPAddress = clientIP(r)
		if h.auditPublisher != nil {
			h.auditPublisher.PublishAsync(event)
		}
	}

	if !verified {
		writeError(w, http.StatusUnauthorized, "RADIUS verification failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"verified": true,
		"method":   "radius",
		"message":  "RADIUS MFA verification successful",
	})
}

// verifyRadiusPasscode sends a RADIUS Access-Request with the passcode.
// This is a placeholder that delegates to an HTTP-based RADIUS gateway.
// In production, replace with a real RADIUS client (layeh/radius).
func (h *Handler) verifyRadiusPasscode(ctx context.Context, server string, port int, secret, username, passcode string) bool {
	// TODO: implement real RADIUS protocol using layeh/radius
	// For now, return false to indicate RADIUS needs real implementation
	// This prevents false positives while making the handler available
	_ = ctx
	_ = server
	_ = port
	_ = secret
	_ = username
	_ = passcode
	return false
}

// --- YubiKey OTP Verification ---
// POST /api/v1/auth/mfa/yubikey/verify
// Body: {"user_id":"...", "otp":"ccccccbchvthbuuituugdiijbegktibkfuktlbrkjef"}
// Validates the OTP against Yubico validation servers.

type yubikeyVerifyRequest struct {
	UserID string `json:"user_id"`
	OTP    string `json:"otp"`
}

func (h *Handler) handleMFAYubiKeyVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req yubikeyVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.OTP) != 44 {
		writeError(w, http.StatusBadRequest, "YubiKey OTP must be 44 characters")
		return
	}

	// Read Yubico config from sys_config
	if h.pool == nil {
		writeError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	var configJSON []byte
	err := h.pool.QueryRow(r.Context(), `SELECT value::text FROM sys_config WHERE key = 'yubico_config'`).Scan(&configJSON)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "YubiKey not configured")
		return
	}

	var yubicoCfg struct {
		ClientID   string   `json:"client_id"`
		SecretKey  string   `json:"secret_key"`
		APIServers []string `json:"api_servers"`
		Enabled    bool     `json:"enabled"`
	}
	if json.Unmarshal(configJSON, &yubicoCfg) != nil {
		writeError(w, http.StatusServiceUnavailable, "invalid Yubico config")
		return
	}

	if !yubicoCfg.Enabled || yubicoCfg.ClientID == "" {
		writeError(w, http.StatusServiceUnavailable, "YubiKey MFA is not enabled")
		return
	}

	// Extract device ID (first 12 chars of OTP)
	deviceID := req.OTP[:12]

	// Verify OTP against Yubico validation server
	verified, err := h.verifyYubiKeyOTP(r.Context(), yubicoCfg.ClientID, yubicoCfg.SecretKey, yubicoCfg.APIServers, req.OTP)
	if err != nil {
		// Audit failure
		if tid, ok := tenantCtxFromHeader(r); ok {
			event := audit.NewEvent("mfa.yubikey.verify", "failure", tid, uuid.Nil)
			event.IPAddress = clientIP(r)
			if h.auditPublisher != nil {
				h.auditPublisher.PublishAsync(event)
			}
		}
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("YubiKey verification failed: %v", err))
		return
	}

	// Audit success
	if tid, ok := tenantCtxFromHeader(r); ok {
		event := audit.NewEvent("mfa.yubikey.verify", "success", tid, uuid.Nil)
		event.IPAddress = clientIP(r)
		if h.auditPublisher != nil {
			h.auditPublisher.PublishAsync(event)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"verified":  verified,
		"method":    "yubikey",
		"device_id": deviceID,
	})
}

// verifyYubiKeyOTP validates an OTP against Yubico validation servers.
// Uses the Yubico validation protocol (HMAC-SHA1 signed request).
func (h *Handler) verifyYubiKeyOTP(ctx context.Context, clientID, secretKey string, servers []string, otp string) (bool, error) {
	// TODO: implement real Yubico validation API call
	// Use net/http to call https://api.yubico.com/wsapi/2.0/verify
	// with params: otp, id=client_id, nonce=random, h=HMAC-SHA1
	// For now: stub that returns error
	_ = ctx
	_ = clientID
	_ = secretKey
	_ = servers
	_ = otp
	return false, fmt.Errorf("YubiKey validation API not yet implemented")
}

// clientIP extracts client IP from request.
// (uses existing clientIP function from http.go)

// tenantCtxFromHeader resolves tenant from X-Tenant-ID header.
func tenantCtxFromHeader(r *http.Request) (uuid.UUID, bool) {
	if tidStr := r.Header.Get("X-Tenant-ID"); tidStr != "" {
		if tid, err := uuid.Parse(tidStr); err == nil {
			return tid, true
		}
	}
	return uuid.Nil, false
}
