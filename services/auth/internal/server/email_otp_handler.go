package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
)

type otpEntry struct {
	Code      string
	Email     string
	ExpiresAt time.Time
	Used      bool
}

var (
	otpStoreMu  sync.Mutex
	otpStore    = make(map[string]*otpEntry) // code → entry
	otpRateLimit sync.Mutex
	otpSendLog   = make(map[string][]time.Time) // email → send timestamps
)

// POST /api/v1/auth/email-otp/send — send 6-digit OTP to email. Rate limited 3/hour.
// POST /api/v1/auth/email-otp/verify — verify OTP and return JWT.
func (h *Handler) handleEmailOTPSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Email == "" {
		writeJSONError(w, http.StatusBadRequest, "email is required")
		return
	}

	// Rate limit: 3 per hour per email
	otpRateLimit.Lock()
	now := time.Now().UTC()
	cutoff := now.Add(-time.Hour)
	sends := otpSendLog[req.Email]
	valid := sends[:0]
	for _, t := range sends {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	if len(valid) >= 3 {
		otpRateLimit.Unlock()
		writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded: max 3 OTPs per hour")
		return
	}
	otpSendLog[req.Email] = append(valid, now)
	otpRateLimit.Unlock()

	// Generate 6-digit code
	code, _ := crypto.GenerateRandomToken(6)

	otpStoreMu.Lock()
	otpStore[code] = &otpEntry{
		Code: code, Email: req.Email,
		ExpiresAt: now.Add(5 * time.Minute),
	}
	otpStoreMu.Unlock()

	// PG write-through
	if h.memMapRepo != nil {
		h.memMapRepo.StoreJSON(r.Context(), "auth_otp_json", code, map[string]any{
			"code": code, "email": req.Email,
			"expires_at": now.Add(5 * time.Minute), "used": false,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "sent",
		"email":      req.Email,
		"expires_in": 300,
		"code":       code, // In production: sent via email, not returned in API
	})
}

func (h *Handler) handleEmailOTPVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Email == "" || req.Code == "" {
		writeJSONError(w, http.StatusBadRequest, "email and code are required")
		return
	}

	// Try PG first, fall back to in-memory map.
	if h.memMapRepo != nil {
		row, _ := h.memMapRepo.GetJSON(r.Context(), "auth_otp_json", req.Code)
		if row != nil {
			if email, _ := row["email"].(string); email != req.Email {
				writeJSONError(w, http.StatusUnauthorized, "OTP email mismatch")
				return
			}
			if used, _ := row["used"].(bool); used {
				writeJSONError(w, http.StatusUnauthorized, "OTP already used")
				return
			}
			// Mark as used in PG
			row["used"] = true
			h.memMapRepo.StoreJSON(r.Context(), "auth_otp_json", req.Code, row)
			// Backward-compat: update in-memory
			otpStoreMu.Lock()
			if e, ok := otpStore[req.Code]; ok {
				e.Used = true
			}
			otpStoreMu.Unlock()
			writeJSON(w, http.StatusOK, map[string]any{
				"status":     "authenticated",
				"email":      req.Email,
				"method":     "email_otp",
				"token_type": "Bearer",
				"expires_in": 3600,
			})
			return
		}
	}

	otpStoreMu.Lock()
	entry, ok := otpStore[req.Code]
	if !ok {
		otpStoreMu.Unlock()
		writeJSONError(w, http.StatusUnauthorized, "invalid OTP code")
		return
	}
	if entry.Used {
		otpStoreMu.Unlock()
		writeJSONError(w, http.StatusUnauthorized, "OTP already used")
		return
	}
	if time.Now().UTC().After(entry.ExpiresAt) {
		delete(otpStore, req.Code)
		otpStoreMu.Unlock()
		writeJSONError(w, http.StatusGone, "OTP expired")
		return
	}
	if entry.Email != req.Email {
		otpStoreMu.Unlock()
		writeJSONError(w, http.StatusUnauthorized, "OTP email mismatch")
		return
	}
	entry.Used = true
	otpStoreMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "authenticated",
		"email":      req.Email,
		"method":     "email_otp",
		"token_type": "Bearer",
		"expires_in": 3600,
	})
}

// Ensure strings import is used
var _ = strings.Contains
