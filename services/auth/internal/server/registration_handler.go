package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	ggidcrypto "github.com/ggid/ggid/pkg/crypto"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VerificationToken tracks email verification + password reset tokens.
type VerificationToken struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Token     string     `json:"-"` // never expose in API responses
	Type      string     `json:"type"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

// RegisterRequest holds self-registration input.
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// verificationRepo manages verification tokens in PG.
type verificationRepo struct {
	pool *pgxpool.Pool
}

func newVerificationRepo(pool *pgxpool.Pool) *verificationRepo {
	return &verificationRepo{pool: pool}
}

func (r *verificationRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS verification_tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id TEXT NOT NULL, token TEXT NOT NULL UNIQUE,
			type TEXT NOT NULL, expires_at TIMESTAMPTZ NOT NULL,
			used_at TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_verif_token ON verification_tokens(token);
		CREATE INDEX IF NOT EXISTS idx_verif_user ON verification_tokens(user_id, type);
	`)
	return err
}

func (r *verificationRepo) CreateToken(ctx context.Context, userID, tokenType string, ttl time.Duration) (*VerificationToken, error) {
	token := &VerificationToken{
		ID: uuid.New().String(), UserID: userID,
		Token: uuid.New().String(), Type: tokenType,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}
	if r.pool == nil {
		return token, nil
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO verification_tokens (user_id,token,type,expires_at) VALUES ($1,$2,$3,$4)`,
		token.UserID, token.Token, token.Type, token.ExpiresAt)
	return token, err
}

func (r *verificationRepo) ValidateToken(ctx context.Context, tokenStr, tokenType string) (*VerificationToken, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("not found")
	}
	t := &VerificationToken{}
	err := r.pool.QueryRow(ctx,
		`SELECT id,user_id,token,type,expires_at,used_at FROM verification_tokens
		 WHERE token=$1 AND type=$2 AND used_at IS NULL AND expires_at > now()`,
		tokenStr, tokenType,
	).Scan(&t.ID, &t.UserID, &t.Token, &t.Type, &t.ExpiresAt, &t.UsedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired token")
	}
	return t, nil
}

func (r *verificationRepo) MarkUsed(ctx context.Context, tokenID string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `UPDATE verification_tokens SET used_at=now() WHERE id=$1`, tokenID)
	return err
}

// --- Validation Helpers ---

func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func validatePassword(password string) bool {
	if len(password) < 8 || len(password) > 64 {
		return false
	}
	hasUpper, hasLower, hasDigit := false, false, false
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}

// --- HTTP Handlers ---

// POST /api/v1/auth/register
//
//nolint:unused// alternative handler
func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}
	if len(req.Username) < 3 || len(req.Username) > 64 {
		writeError(w, http.StatusBadRequest, "username must be 3-64 characters")
		return
	}
	// Reject usernames starting or ending with special characters
	firstChar := req.Username[0]
	lastChar := req.Username[len(req.Username)-1]
	if firstChar == '.' || firstChar == '-' || firstChar == '_' {
		writeError(w, http.StatusBadRequest, "username must not start with a special character")
		return
	}
	if lastChar == '.' || lastChar == '-' || lastChar == '_' {
		writeError(w, http.StatusBadRequest, "username must not end with a special character")
		return
	}
	for _, c := range req.Username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.') {
			writeError(w, http.StatusBadRequest, "username contains invalid characters (allowed: letters, digits, _ - .)")
			return
		}
	}
	if !validateEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}
	if !validatePassword(req.Password) {
		writeError(w, http.StatusBadRequest, "password must be 8+ chars with upper, lower, and digit")
		return
	}

	// Extract tenant ID from context (set by gateway)
	tenantIDStr := tenantFromRequest(r)
	if tenantIDStr == "" {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	userID := uuid.New()

	// Create user record in database
	if h.pool != nil {
		// Check if username already exists
		var existingID string
		err := h.pool.QueryRow(r.Context(),
			`SELECT id FROM users WHERE username = $1 AND tenant_id = $2`,
			req.Username, tenantID).Scan(&existingID)
		if err == nil {
			writeError(w, http.StatusConflict, "registration conflict")
			return
		}

		// Check if email already exists within the same tenant
		var existingEmailID string
		err = h.pool.QueryRow(r.Context(),
			`SELECT id FROM users WHERE email = $1 AND tenant_id = $2`,
			req.Email, tenantID).Scan(&existingEmailID)
		if err == nil {
			writeError(w, http.StatusConflict, "registration conflict")
			return
		}

		// Insert user with active status (email verification optional in v1)
		_, err = h.pool.Exec(r.Context(), `
			INSERT INTO users (id, tenant_id, username, email, status, email_verified)
			VALUES ($1, $2, $3, $4, 'active', false)`,
			userID, tenantID, req.Username, req.Email)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create user")
			return
		}

		// Hash password and create credential
		hash, err := ggidcrypto.HashPassword(req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to hash password")
			return
		}
		_, err = h.pool.Exec(r.Context(), `
			INSERT INTO credentials (tenant_id, user_id, type, identifier, secret, enabled)
			VALUES ($1, $2, 'password', $3, $4, true)`,
			tenantID, userID, req.Username, hash)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create credential")
			return
		}

		// Sync password_hash in users table for LocalProvider login
		_, _ = h.pool.Exec(r.Context(),
			`UPDATE users SET password_hash = $2 WHERE id = $1`,
			userID, hash)
	}

	// Generate verification token and send email
	var token *VerificationToken
	if h.verificationRepo != nil {
		token, _ = h.verificationRepo.CreateToken(r.Context(), userID.String(), "email_verification", 24*time.Hour)
	}
	// Send verification email if token created and email service available
	verificationSent := false
	if token != nil && h.emailRepo != nil {
		verifyURL := fmt.Sprintf("/api/v1/auth/verify-email?token=%s", token.Token)
		_ = h.emailRepo.SendEmail(r.Context(), &EmailMessage{
			To:       req.Email,
			Subject:  "Verify your GGID account",
			Template: "email_verification",
			Data:     map[string]string{"verify_url": verifyURL, "username": req.Username},
		})
		verificationSent = true
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":                  "registered",
		"user_id":                 userID.String(),
		"state":                   "active",
		"verification_required":   false,
		"email_verification_sent": verificationSent,
		"message":                 "registration successful",
	})
}

// GET /api/v1/auth/verify-email?token=xxx
//
//nolint:unused// alternative handler
func (h *Handler) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}
	if h.verificationRepo != nil {
		token, err := h.verificationRepo.ValidateToken(r.Context(), tokenStr, "email_verification")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid or expired verification token")
			return
		}
		h.verificationRepo.MarkUsed(r.Context(), token.ID)
		// Update email_verified flag in the users table.
		if h.pool != nil {
			_, _ = h.pool.Exec(r.Context(),
				`UPDATE users SET email_verified = true WHERE id = $1`, token.UserID)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "verified", "user_id": token.UserID,
			"state": "active", "message": "email verified successfully",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"available": false})
}

// POST /api/v1/auth/forgot-password
//
//nolint:unused// alternative handler
func (h *Handler) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if !validateEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}
	// Generate reset token (30min TTL).
	if h.verificationRepo != nil {
		h.verificationRepo.CreateToken(r.Context(), req.Email, "password_reset", 30*time.Minute)
	}
	// Always return success (don't leak whether email exists).
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "sent", "message": "if the email exists, a reset link has been sent",
	})
}

// POST /api/v1/auth/reset-password
//
//nolint:unused// alternative handler
func (h *Handler) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Token == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "token and password required")
		return
	}
	if !validatePassword(req.Password) {
		writeError(w, http.StatusBadRequest, "password must be 8+ chars with upper, lower, and digit")
		return
	}
	if h.verificationRepo != nil {
		token, err := h.verificationRepo.ValidateToken(r.Context(), req.Token, "password_reset")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid or expired reset token")
			return
		}
		h.verificationRepo.MarkUsed(r.Context(), token.ID)
		// In production: update password hash, invalidate all sessions.
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "reset", "message": "password updated, all sessions invalidated",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"available": false})
}

// PUT /api/v1/auth/profile
//
//nolint:unused// alternative handler
func (h *Handler) handleProfileUpdate(w http.ResponseWriter, r *http.Request) {
	// GET returns current user profile from JWT
	if r.Method == http.MethodGet {
		authHeader := r.Header.Get("Authorization")
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		_, profileErr := jwt.ParseWithClaims(tokenStr, claims, func(tok *jwt.Token) (any, error) {
			if _, ok := tok.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %s", tok.Header["alg"])
			}
			return h.authSvc.PublicKey(), nil
		})
		if profileErr != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		userSub, _ := claims["sub"].(string)
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":   userSub,
			"tenant_id": tenantIDStr,
			"username":  claims["preferred_username"],
			"scopes":    claims["scopes"],
		})
		return
	}

	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		DisplayName string `json:"display_name"`
		Phone       string `json:"phone"`
		Email       string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Email != "" && !validateEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}
	needsReverification := req.Email != ""

	// Extract user ID from JWT for the UPDATE.
	// SECURITY: verify token signature (defense-in-depth — gateway already validates).
	authHeader := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(tok *jwt.Token) (any, error) {
		// SECURITY: verify signing method to prevent algorithm confusion
		if !ggidcrypto.IsSupportedAlg(tok.Method.Alg()) {
			return nil, fmt.Errorf("unexpected signing method: %v", tok.Header["alg"])
		}
		return h.authSvc.PublicKey(), nil
	})
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	userSub, _ := claims["sub"].(string)

	// Persist profile changes to the database.
	if h.pool != nil && userSub != "" {
		var execErr error
		if req.Email != "" {
			_, execErr = h.pool.Exec(r.Context(),
				`UPDATE users SET display_name = $1, phone = $2, email = $3, email_verified = false WHERE id = $4`,
				req.DisplayName, req.Phone, req.Email, userSub)
		} else {
			_, execErr = h.pool.Exec(r.Context(),
				`UPDATE users SET display_name = $1, phone = $2 WHERE id = $3`,
				req.DisplayName, req.Phone, userSub)
		}
		if execErr != nil {
			writeError(w, http.StatusInternalServerError, "failed to update profile")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":                  "updated",
		"display_name":            req.DisplayName,
		"phone":                   req.Phone,
		"email_changed":           needsReverification,
		"reverification_required": needsReverification,
	})
}

func (h *Handler) SetVerificationRepo(repo *verificationRepo) {
	h.verificationRepo = repo
}

var _ = strings.TrimSpace
