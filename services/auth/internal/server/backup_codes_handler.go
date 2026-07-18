package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ggid/ggid/pkg/errors"
	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// --- Backup Codes (MFA Recovery Codes) ---

// backupCodesGenerateRequest is the body for POST /api/v1/auth/mfa/backup-codes/generate.
type backupCodesGenerateRequest struct {
	UserID string `json:"user_id"`
}

// backupCodesGenerate handles POST /api/v1/auth/mfa/backup-codes/generate.
// Generates 10 new single-use backup codes for the user. Existing codes are replaced.
// The plaintext codes are returned only once — the caller must store them securely.
func (h *Handler) backupCodesGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req backupCodesGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Extract user_id from JWT if not provided in body.
	if req.UserID == "" {
		authHeader := r.Header.Get("Authorization")
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(tokenStr, claims, func(tok *jwt.Token) (any, error) {
			return h.authSvc.PublicKey(), nil
		})
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		req.UserID, _ = claims["sub"].(string)
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	bcs := h.authSvc.BackupCodeService()
	if bcs == nil {
		writeJSON(w, http.StatusOK, map[string]any{"codes": []any{}, "count": 0})
		return
	}

	codes, err := bcs.GenerateBackupCodes(r.Context(), userID)
	if err != nil {
		slog.Error("backup code generation error", "user_id", req.UserID, "error", err)
		writeInternalError(w, "backup_codes_generate", err)
		return
	}

	// Audit: backup codes generated
	if tc, terr := ggidtenant.FromContext(r.Context()); terr == nil {
		h.publishAuditEvent("user.mfa.backup_codes.generated", "success", tc.TenantID, userID)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"codes":      codes,
		"count":      len(codes),
		"warning":    "Store these codes securely. They will not be shown again.",
		"expires_in": "until regenerated",
	})
}

// backupCodesVerifyRequest is the body for POST /api/v1/auth/mfa/backup-codes/verify.
type backupCodesVerifyRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	BackupCode string `json:"backup_code"`
}

// backupCodesVerify handles POST /api/v1/auth/mfa/backup-codes/verify.
// Authenticates the user with password + backup code (alternative to TOTP during MFA login).
// The backup code is consumed (single-use) upon successful verification.
func (h *Handler) backupCodesVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req backupCodesVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.BackupCode == "" {
		writeError(w, http.StatusBadRequest, "backup_code is required")
		return
	}

	ip := clientIP(r)
	userAgent := r.Header.Get("User-Agent")

	tokens, err := h.authSvc.LoginWithBackupCode(r.Context(), req.Username, req.Password, req.BackupCode, ip, userAgent)
	if err != nil {
		slog.Error("backup code verify error", "username", req.Username, "error", err)
		// Map backup code error to 401.
		if strings.Contains(err.Error(), "invalid or used backup code") {
			errors.WriteSimpleAPIError(w, http.StatusUnauthorized, string(errors.ErrUnauthenticated), "invalid or used backup code")
			return
		}
		writeAuthError(w, err)
		return
	}

	// Audit: backup code login success
	if tc, terr := ggidtenant.FromContext(r.Context()); terr == nil {
		h.publishAuditEvent("user.mfa.backup_codes.used", "success", tc.TenantID, uuid.Nil)
	}

	writeJSON(w, http.StatusOK, tokens)
}

// backupCodesRemaining handles GET /api/v1/auth/mfa/backup-codes/remaining?user_id=xxx.
// Returns the count of unused backup codes remaining for the user.
func (h *Handler) backupCodesRemaining(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		// Extract user_id from JWT token in Authorization header.
		authHeader := r.Header.Get("Authorization")
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		_, parseErr := jwt.ParseWithClaims(tokenStr, claims, func(tok *jwt.Token) (any, error) {
			return h.authSvc.PublicKey(), nil
		})
		if parseErr != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		userIDStr, _ = claims["sub"].(string)
	}

	if userIDStr == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	bcs := h.authSvc.BackupCodeService()
	if bcs == nil {
		writeJSON(w, http.StatusOK, map[string]any{"codes": []any{}, "count": 0})
		return
	}

	remaining, err := bcs.RemainingBackupCodes(r.Context(), tc.TenantID, userID)
	if err != nil {
		writeInternalError(w, "backup_codes_remaining", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":   userID.String(),
		"remaining": remaining,
		"total":     10,
	})
}
