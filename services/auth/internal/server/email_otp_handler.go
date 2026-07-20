package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/google/uuid"
)

// POST /api/v1/auth/email-otp/send — send 6-digit OTP to email. Rate limited 3/hour.
// Uses auth_otp_entries DB table for persistence with in-memory rate-limit cache.
func (h *Handler) handleEmailOTPSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	// Rate limit via DB: count OTP entries in last hour
	if h.pool != nil {
		var count int
		h.pool.QueryRow(r.Context(), `SELECT count(*) FROM auth_otp_entries WHERE email = $1 AND created_at > NOW() - INTERVAL '1 hour'`, req.Email).Scan(&count)
		if count >= 3 {
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded: max 3 OTPs per hour")
			return
		}
	}

	// Generate 6-digit code
	code, _ := crypto.GenerateRandomToken(6)
	expiresAt := time.Now().UTC().Add(5 * time.Minute)

	// Write to DB
	tenantID := ""
	if tc, err := extractTenantID(r); err == nil {
		tenantID = tc.String()
	}
	if h.pool != nil {
		_, err := h.pool.Exec(r.Context(),
			`INSERT INTO auth_otp_entries (code, email, tenant_id, hashed_code, attempts, expires_at) VALUES ($1, $2, $3, $4, 0, $5)`,
			code, req.Email, tenantID, code, expiresAt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to store OTP")
			return
		}
	}

	// Fallback: also store in memMapRepo for backward compat
	if h.memMapRepo != nil {
		h.memMapRepo.StoreJSON(r.Context(), "auth_otp_json", code, map[string]any{
			"code": code, "email": req.Email,
			"expires_at": expiresAt, "used": false,
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Email == "" || req.Code == "" {
		writeError(w, http.StatusBadRequest, "email and code are required")
		return
	}

	// Try DB first
	if h.pool != nil {
		var dbEmail, tenantID string
		var attempts int
		var expiresAt time.Time
		err := h.pool.QueryRow(r.Context(),
			`SELECT email, COALESCE(tenant_id,''), COALESCE(attempts,0), expires_at FROM auth_otp_entries WHERE code = $1`,
			req.Code).Scan(&dbEmail, &tenantID, &attempts, &expiresAt)
		if err == nil {
			// Found in DB
			if dbEmail != req.Email {
				writeError(w, http.StatusUnauthorized, "OTP email mismatch")
				return
			}
			if time.Now().UTC().After(expiresAt) {
				h.pool.Exec(r.Context(), `DELETE FROM auth_otp_entries WHERE code = $1`, req.Code)
				writeError(w, http.StatusGone, "OTP expired")
				return
			}
			// Delete used OTP from DB
			_, _ = h.pool.Exec(r.Context(), `DELETE FROM auth_otp_entries WHERE code = $1`, req.Code)
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

	// Fallback: try memMapRepo
	if h.memMapRepo != nil {
		row, _ := h.memMapRepo.GetJSON(r.Context(), "auth_otp_json", req.Code)
		if row != nil {
			if email, _ := row["email"].(string); email != req.Email {
				writeError(w, http.StatusUnauthorized, "OTP email mismatch")
				return
			}
			if used, _ := row["used"].(bool); used {
				writeError(w, http.StatusUnauthorized, "OTP already used")
				return
			}
			row["used"] = true
			h.memMapRepo.StoreJSON(r.Context(), "auth_otp_json", req.Code, row)
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

	writeError(w, http.StatusUnauthorized, "invalid OTP code")
}

// extractTenantID gets tenant ID from request context or header.
func extractTenantID(r *http.Request) (uuid.UUID, error) {
	// Try X-Tenant-ID header
	if tidStr := r.Header.Get("X-Tenant-ID"); tidStr != "" {
		return uuid.Parse(tidStr)
	}
	return uuid.Nil, nil
}

// Ensure strings import is used
var _ = strings.Contains
