package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/audit"
	"github.com/google/uuid"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

// GET /api/v1/auth/mfa/methods — returns enabled MFA methods for frontend display
func (h *Handler) handleMFAMethods(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	resp := map[string]any{
		"totp_enabled":     true,
		"webauthn_enabled": true,
		"radius_enabled":   false,
		"yubikey_enabled":  false,
		"radius_test_mode":  false,
		"yubikey_test_mode": false,
	}

	if h.pool != nil {
		var radiusJSON []byte
		if err := h.pool.QueryRow(r.Context(), `SELECT value::text FROM sys_config WHERE key = 'radius_config'`).Scan(&radiusJSON); err == nil {
			var cfg struct {
				Enabled  bool `json:"enabled"`
				TestMode bool `json:"test_mode"`
			}
			if json.Unmarshal(radiusJSON, &cfg) == nil {
				resp["radius_enabled"] = cfg.Enabled
				resp["radius_test_mode"] = cfg.TestMode
			}
		}

		var yubicoJSON []byte
		if err := h.pool.QueryRow(r.Context(), `SELECT value::text FROM sys_config WHERE key = 'yubico_config'`).Scan(&yubicoJSON); err == nil {
			var cfg struct {
				Enabled  bool `json:"enabled"`
				TestMode bool `json:"test_mode"`
			}
			if json.Unmarshal(yubicoJSON, &cfg) == nil {
				resp["yubikey_enabled"] = cfg.Enabled
				resp["yubikey_test_mode"] = cfg.TestMode
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- RADIUS MFA Verification ---
// POST /api/v1/auth/mfa/radius/verify

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

	if h.pool == nil {
		writeError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	var configJSON []byte
	if err := h.pool.QueryRow(r.Context(), `SELECT value::text FROM sys_config WHERE key = 'radius_config'`).Scan(&configJSON); err != nil {
		writeError(w, http.StatusServiceUnavailable, "RADIUS not configured")
		return
	}

	var cfg struct {
		Server   string `json:"server"`
		Secret   string `json:"secret"`
		Port     int    `json:"port"`
		Timeout  int    `json:"timeout"`
		Enabled  bool   `json:"enabled"`
		TestMode bool   `json:"test_mode"`
	}
	if json.Unmarshal(configJSON, &cfg) != nil {
		writeError(w, http.StatusServiceUnavailable, "invalid RADIUS config")
		return
	}

	if !cfg.Enabled {
		writeError(w, http.StatusServiceUnavailable, "RADIUS MFA is not enabled")
		return
	}

	// Test mode: accept any non-empty passcode
	if cfg.TestMode {
		verified := req.Passcode != ""
		auditMFAResult(h, r, "radius", req.Username, verified)
		if !verified {
			writeError(w, http.StatusUnauthorized, "RADIUS verification failed (test mode)")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"verified":  true,
			"method":    "radius",
			"test_mode": true,
		})
		return
	}

	// Production: real RADIUS Access-Request
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	verified := verifyRadiusPasscode(ctx, cfg.Server, cfg.Port, cfg.Secret, req.Username, req.Passcode)
	auditMFAResult(h, r, "radius", req.Username, verified)

	if !verified {
		writeError(w, http.StatusUnauthorized, "RADIUS verification failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"verified": true,
		"method":   "radius",
	})
}

// verifyRadiusPasscode sends a RADIUS Access-Request using layeh.com/radius.
func verifyRadiusPasscode(ctx context.Context, server string, port int, secret, username, passcode string) bool {
	if server == "" || secret == "" {
		return false
	}
	hostPort := server
	if port > 0 {
		hostPort = fmt.Sprintf("%s:%d", server, port)
	}
	packet := radius.New(radius.CodeAccessRequest, []byte(secret))
	rfc2865.UserName_SetString(packet, username)
	rfc2865.UserPassword_SetString(packet, passcode)
	resp, err := radius.Exchange(ctx, packet, hostPort)
	if err != nil {
		return false
	}
	return resp.Code == radius.CodeAccessAccept
}

// --- YubiKey OTP Verification ---
// POST /api/v1/auth/mfa/yubikey/verify

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

	if h.pool == nil {
		writeError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	var configJSON []byte
	if err := h.pool.QueryRow(r.Context(), `SELECT value::text FROM sys_config WHERE key = 'yubico_config'`).Scan(&configJSON); err != nil {
		writeError(w, http.StatusServiceUnavailable, "YubiKey not configured")
		return
	}

	var cfg struct {
		ClientID   string   `json:"client_id"`
		SecretKey  string   `json:"secret_key"`
		APIServers []string `json:"api_servers"`
		Enabled    bool     `json:"enabled"`
		TestMode   bool     `json:"test_mode"`
	}
	if json.Unmarshal(configJSON, &cfg) != nil {
		writeError(w, http.StatusServiceUnavailable, "invalid Yubico config")
		return
	}

	if !cfg.Enabled {
		writeError(w, http.StatusServiceUnavailable, "YubiKey MFA is not enabled")
		return
	}

	deviceID := req.OTP[:12]

	// Test mode: validate format only
	if cfg.TestMode {
		verified := isModhex(req.OTP)
		auditMFAResult(h, r, "yubikey", deviceID, verified)
		if !verified {
			writeError(w, http.StatusUnauthorized, "YubiKey OTP format invalid (test mode)")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"verified":  true,
			"method":    "yubikey",
			"device_id": deviceID,
			"test_mode": true,
		})
		return
	}

	// Production: call Yubico validation API
	verified, err := verifyYubiKeyOTP(r.Context(), cfg.ClientID, cfg.SecretKey, cfg.APIServers, req.OTP)
	auditMFAResult(h, r, "yubikey", deviceID, verified)
	if err != nil || !verified {
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("YubiKey verification failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"verified":  true,
		"method":    "yubikey",
		"device_id": deviceID,
	})
}

// verifyYubiKeyOTP validates an OTP against Yubico validation servers.
func verifyYubiKeyOTP(ctx context.Context, clientID, secretKey string, servers []string, otp string) (bool, error) {
	if clientID == "" {
		return false, fmt.Errorf("missing Yubico client_id")
	}
	if len(servers) == 0 {
		servers = []string{"https://api.yubico.com/wsapi/2.0/verify"}
	}

	nonce, err := cryptoRandHex(16)
	if err != nil {
		return false, err
	}

	params := url.Values{}
	params.Set("id", clientID)
	params.Set("otp", otp)
	params.Set("nonce", nonce)

	// HMAC-SHA1 sign the request
	if secretKey != "" {
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var sb strings.Builder
		for i, k := range keys {
			if i > 0 {
				sb.WriteByte('&')
			}
			sb.WriteString(k)
			sb.WriteByte('=')
			sb.WriteString(params.Get(k))
		}
		mac := hmac.New(sha1.New, []byte(secretKey))
		mac.Write([]byte(sb.String()))
		params.Set("h", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	for _, srv := range servers {
		apiReq, _ := http.NewRequestWithContext(ctx, "GET", srv+"?"+params.Encode(), nil)
		resp, err := httpClient.Do(apiReq)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		respParams := parseYubicoResponse(string(body))
		if respParams["status"] == "OK" {
			return true, nil
		}
		return false, fmt.Errorf("Yubico status: %s", respParams["status"])
	}
	return false, fmt.Errorf("all Yubico servers unreachable")
}

func parseYubicoResponse(body string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, "="); idx > 0 {
			result[line[:idx]] = line[idx+1:]
		}
	}
	return result
}

var modhexChars = "cbdefghijklnrtuv"

func isModhex(s string) bool {
	s = strings.ToLower(s)
	for _, c := range s {
		if !strings.ContainsRune(modhexChars, c) {
			return false
		}
	}
	return true
}

func cryptoRandHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// auditMFAResult publishes an audit event for MFA verification.
func auditMFAResult(h *Handler, r *http.Request, method, actor string, verified bool) {
	result := "failure"
	if verified {
		result = "success"
	}
	if tid, ok := tenantCtxFromHeader(r); ok {
		event := audit.NewEvent("mfa."+method+".verify", result, tid, uuid.Nil)
		event.ActorName = actor
		event.IPAddress = clientIP(r)
		if h.auditPublisher != nil {
			h.auditPublisher.PublishAsync(event)
		}
	}
}

// tenantCtxFromHeader resolves tenant from X-Tenant-ID header.
func tenantCtxFromHeader(r *http.Request) (uuid.UUID, bool) {
	if tidStr := r.Header.Get("X-Tenant-ID"); tidStr != "" {
		if tid, err := uuid.Parse(tidStr); err == nil {
			return tid, true
		}
	}
	return uuid.Nil, false
}
