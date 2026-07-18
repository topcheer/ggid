package server

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type JITEnrollment struct {
	EnrollmentID string    `json:"enrollment_id"`
	UserID       string    `json:"user_id"`
	FactorType   string    `json:"factor_type"` // totp, sms, email
	Token        string    `json:"token"`       // enrollment token (OTP URL or code)
	QRURL        string    `json:"qr_url"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// POST /api/v1/auth/mfa/jit-enroll — JIT MFA enrollment for high-risk users without MFA
func (h *Handler) handleJITMFAEnroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserID    string `json:"user_id"`
		RiskScore int    `json:"risk_score"`
		FactorType string `json:"factor_type"` // optional, default totp
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.UserID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	factorType := req.FactorType
	if factorType == "" {
		factorType = "totp"
	}

	// Only enroll if risk is high enough
	if req.RiskScore < 50 {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "not_required",
			"user_id":   req.UserID,
			"risk_score": req.RiskScore,
			"message":   "risk score below threshold — JIT enrollment not triggered",
		})
		return
	}

	// Generate enrollment
	enrollmentID := uuid.New().String()
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to generate TOTP secret")
		return
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)
	token := uuid.New().String()

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "enrolled",
		"enrollment_id": enrollmentID,
		"user_id":       req.UserID,
		"factor_type":   factorType,
		"token":         token,
		"secret":        secret,
		"qr_url":        "otpauth://totp/GGID:" + req.UserID + "?secret=" + secret + "&issuer=GGID",
		"expires_at":    time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
		"message":       "JIT MFA enrollment triggered — user must complete within 5 minutes",
	})
}
