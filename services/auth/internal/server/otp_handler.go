package server

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/service"
)

// POST /api/v1/auth/otp/send — send OTP via SMS or Email.
// Body: {"identifier": "user@example.com" or "+1234567890", "channel": "sms" or "email"}
// Uses Redis for OTP storage with 5min TTL + 60s rate limit.
func (h *Handler) handleOTPSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Identifier string `json:"identifier"`
		Channel    string `json:"channel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Identifier == "" {
		writeError(w, http.StatusBadRequest, "identifier is required")
		return
	}
	if req.Channel != "sms" && req.Channel != "email" {
		writeError(w, http.StatusBadRequest, "channel must be 'sms' or 'email'")
		return
	}

	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		// fallback to header
		tidStr := tenantFromRequest(r)
		if tidStr != "" {
			parsed, perr := uuid.Parse(tidStr)
			if perr == nil {
				tc = &tenant.Context{TenantID: parsed}
			}
		}
	}

	var otpSvc *service.OTPService
	if h.otpService != nil {
		otpSvc = h.otpService
	}
	if otpSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "OTP service not configured")
		return
	}

	channel := service.OTPSMS
	if req.Channel == "email" {
		channel = service.OTPEmail
	}

	if err := otpSvc.SendOTP(r.Context(), tc.TenantID, req.Identifier, channel); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "sent",
		"channel":    req.Channel,
		"expires_in": 300,
	})
}

// POST /api/v1/auth/otp/verify — verify OTP and return auth_ticket.
// Body: {"identifier": "...", "code": "123456", "channel": "sms" or "email"}
// Returns: {"auth_ticket": "xxx", "ticket_type": "urn:ggid:auth-ticket", "expires_in": 30}
func (h *Handler) handleOTPVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Identifier string `json:"identifier"`
		Code       string `json:"code"`
		Channel    string `json:"channel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Identifier == "" || req.Code == "" {
		writeError(w, http.StatusBadRequest, "identifier and code are required")
		return
	}

	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		tidStr := tenantFromRequest(r)
		if tidStr != "" {
			parsed, perr := uuid.Parse(tidStr)
			if perr == nil {
				tc = &tenant.Context{TenantID: parsed}
			}
		}
	}

	var otpSvc *service.OTPService
	if h.otpService != nil {
		otpSvc = h.otpService
	}
	if otpSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "OTP service not configured")
		return
	}

	channel := service.OTPSMS
	if req.Channel == "email" {
		channel = service.OTPEmail
	}

	ticket, err := otpSvc.VerifyOTP(r.Context(), tc.TenantID, req.Identifier, req.Code, channel)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "authenticated",
		"auth_ticket":  ticket,
		"ticket_type":  "urn:ggid:auth-ticket",
		"expires_in":   30,
	})
}
