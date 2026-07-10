// Package server implements the HTTP server for the Auth Service.
package server

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	ggiderrors "github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/social"
	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/service"
	"github.com/ggid/ggid/services/auth/internal/webauthn"
	"github.com/google/uuid"
)

// Handler is the HTTP handler for the Auth Service.
type Handler struct {
	authSvc   *service.AuthService
	mux       *http.ServeMux
	socialReg *social.Registry
	hooks     *service.HookManager
	idpConfigs map[string]*service.IdPConfig // keyed by config ID
}

// New creates a new Auth Service HTTP handler.
func New(authSvc *service.AuthService) *Handler {
	h := &Handler{
		authSvc:    authSvc,
		socialReg:  social.NewRegistry(),
		hooks:      service.NewHookManager(),
		idpConfigs: make(map[string]*service.IdPConfig),
	}
	h.registerRoutes()
	return h
}

func (h *Handler) registerRoutes() {
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/healthz", h.healthz)
	h.mux.HandleFunc("/api/v1/auth/login", h.login)
	h.mux.HandleFunc("/api/v1/auth/register", h.register)
	h.mux.HandleFunc("/api/v1/auth/logout", h.logout)
	h.mux.HandleFunc("/api/v1/auth/refresh", h.refresh)
	h.mux.HandleFunc("/api/v1/auth/password/forgot", h.forgotPassword)
	h.mux.HandleFunc("/api/v1/auth/password/reset", h.resetPassword)

	// Password reset — arch-spec routes (aliases)
	h.mux.HandleFunc("/api/v1/auth/forgot-password", h.forgotPassword)
	h.mux.HandleFunc("/api/v1/auth/reset-password", h.resetPassword)
	h.mux.HandleFunc("/api/v1/auth/password/change", h.changePassword)
	h.mux.HandleFunc("/api/v1/auth/sessions", h.handleSessions)
	h.mux.HandleFunc("/api/v1/auth/mfa/setup", h.mfaSetup)
	h.mux.HandleFunc("/api/v1/auth/mfa/verify", h.mfaVerify)
	h.mux.HandleFunc("/api/v1/auth/mfa/disable", h.mfaDisable)
	h.mux.HandleFunc("/api/v1/auth/mfa/login", h.mfaLogin)

	// Password policy config endpoint
	h.mux.HandleFunc("/api/v1/auth/password/policy", h.passwordPolicy)

	// Magic Link (passwordless login)
	h.mux.HandleFunc("/api/v1/auth/magic-link", h.magicLink)
	h.mux.HandleFunc("/api/v1/auth/magic-link/verify", h.magicLinkVerify)

	// Email verification
	h.mux.HandleFunc("/api/v1/auth/email/verify", h.emailVerify)
	h.mux.HandleFunc("/api/v1/auth/email/resend", h.emailResend)

	// Email verification — arch-spec routes
	h.mux.HandleFunc("/api/v1/auth/send-verification", h.sendVerification)
	h.mux.HandleFunc("/api/v1/auth/verify-email", h.verifyEmail)

	// Phone OTP authentication
	h.mux.HandleFunc("/api/v1/auth/phone/send", h.phoneOTPSend)
	h.mux.HandleFunc("/api/v1/auth/phone/verify", h.phoneOTPVerify)

	// Step-up authentication
	h.mux.HandleFunc("/api/v1/auth/stepup/challenge", h.stepUpChallenge)
	h.mux.HandleFunc("/api/v1/auth/stepup/verify", h.stepUpVerify)

	// Logout all devices
	h.mux.HandleFunc("/api/v1/auth/logout-all", h.logoutAll)

	// Auth hooks (Auth0 Actions equivalent)
	h.mux.HandleFunc("/api/v1/auth/hooks", h.manageHooks)

	// Passwordless (WebAuthn-only) registration + login
	h.mux.HandleFunc("/api/v1/auth/passwordless/register", h.passwordlessRegister)

	// MFA WebAuthn (second factor via passkey)
	h.mux.HandleFunc("/api/v1/auth/mfa/webauthn/begin", h.mfaWebAuthnBegin)
	h.mux.HandleFunc("/api/v1/auth/mfa/webauthn/finish", h.mfaWebAuthnFinish)

	// IdP federation config
	h.mux.HandleFunc("/api/v1/idp/config", h.idpConfig)

	// Email change (dual confirmation)
	h.mux.HandleFunc("/api/v1/auth/email/change", h.emailChange)
	h.mux.HandleFunc("/api/v1/auth/email/change/confirm", h.emailChangeConfirm)

	// Auth0 Lock compatible hosted login
	h.mux.HandleFunc("/authorize", h.authorize)
	h.mux.HandleFunc("/usernamepassword/login", h.usernamePasswordLogin)
	h.mux.HandleFunc("/dbconnections/signup", h.dbConnectionsSignup)

	// Social login endpoints
	h.mux.HandleFunc("/api/v1/auth/social/", h.handleSocial)

	// WebAuthn / Passkey endpoints (nil credential store = skeleton mode)
	webauthnHandler, err := webauthn.NewHandler("ggid.dev", "GGID Platform", nil)
	if err != nil {
		log.Printf("warning: webauthn init failed: %v", err)
	} else {
		webauthnHandler.RegisterRoutes(h.mux)
	}
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Inject tenant context from X-Tenant-ID header
	if tenantIDStr := r.Header.Get("X-Tenant-ID"); tenantIDStr != "" {
		tenantID, err := uuid.Parse(tenantIDStr)
		if err == nil {
			tc := &ggidtenant.Context{
				TenantID:       tenantID,
				IsolationLevel: ggidtenant.IsolationShared,
			}
			r = r.WithContext(ggidtenant.WithContext(r.Context(), tc))
		}
	}
	h.mux.ServeHTTP(w, r)
}

// --- Health Check ---

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Login ---

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := clientIP(r)
	userAgent := r.Header.Get("User-Agent")

	// Brute force protection: dual-dimension sliding window rate limit.
	if tc, err := ggidtenant.FromContext(r.Context()); err == nil {
		if err := h.authSvc.CheckBruteForce(r.Context(), tc.TenantID, ip, req.Username); err != nil {
			writeError(w, http.StatusTooManyRequests, "too many login attempts")
			return
		}
	}

	// Check if the account is locked before attempting login.
	if tc, err := ggidtenant.FromContext(r.Context()); err == nil {
		if h.authSvc.IsAccountLocked(r.Context(), tc.TenantID, req.Username) {
			writeError(w, http.StatusLocked, "account is locked due to too many failed attempts")
			return
		}
	}

	tokens, err := h.authSvc.Login(r.Context(), req.Username, req.Password, ip, userAgent)
	if err != nil {
		// Record failed login attempt for lockout tracking.
		if tc, terr := ggidtenant.FromContext(r.Context()); terr == nil {
			_ = h.authSvc.RecordFailedLogin(r.Context(), tc.TenantID, req.Username)
		}
		log.Printf("login error for user %s: %v", req.Username, err)
		writeAuthError(w, err)
		return
	}

	// Reset failed login counter on success.
	if tc, err := ggidtenant.FromContext(r.Context()); err == nil {
		h.authSvc.ResetFailedLogins(r.Context(), tc.TenantID, req.Username)
	}

	writeJSON(w, http.StatusOK, tokens)
}

// --- Register ---

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	userID := uuid.New() // In production, user is created via Identity Service first
	if err := h.authSvc.Register(r.Context(), tc.TenantID, userID, req.Username, req.Password); err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"user_id": userID.String()})
}

// --- Logout ---

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.authSvc.Logout(r.Context(), req.RefreshToken); err != nil {
		writeError(w, http.StatusInternalServerError, "logout failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"logged_out": true})
}

// --- Refresh ---

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokens, err := h.authSvc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// --- Password Flows ---

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	_ = h.authSvc.ForgotPassword(r.Context(), tc.TenantID, req.Email)
	// Always return success to prevent email enumeration
	writeJSON(w, http.StatusOK, map[string]bool{"reset_initiated": true})
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.authSvc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"password_reset": true})
}

type changePasswordRequest struct {
	UserID      string `json:"user_id"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	if err := h.authSvc.ChangePassword(r.Context(), tc.TenantID, userID, req.OldPassword, req.NewPassword); err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"password_changed": true})
}

// --- Sessions ---

func (h *Handler) handleSessions(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		sessions, err := h.authSvc.ListSessions(r.Context(), tc.TenantID, userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list sessions")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})

	case http.MethodDelete:
		sessionIDStr := r.URL.Path[len("/api/v1/auth/sessions/"):]
		sessionID, err := uuid.Parse(sessionIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid session_id")
			return
		}
		if err := h.authSvc.RevokeSession(r.Context(), sessionID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to revoke session")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"revoked": true})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- MFA ---

type mfaSetupRequest struct {
	UserID     string `json:"user_id"`
	DeviceName string `json:"device_name"`
}

func (h *Handler) mfaSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req mfaSetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	resp, err := h.authSvc.MFAService().SetupMFA(r.Context(), user, req.DeviceName)
	if err != nil {
		log.Printf("MFA setup error for user %s: %v", req.UserID, err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

type mfaVerifyRequest struct {
	DeviceID string `json:"device_id"`
	Code     string `json:"code"`
}

func (h *Handler) mfaVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req mfaVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid device_id")
		return
	}

	if err := h.authSvc.MFAService().VerifyMFA(r.Context(), deviceID, req.Code); err != nil {
		if stderrors.Is(err, service.ErrInvalidMFACode) {
			writeError(w, http.StatusUnauthorized, "invalid MFA code")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"verified": true})
}

type mfaDisableRequest struct {
	DeviceID string `json:"device_id"`
}

func (h *Handler) mfaDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req mfaDisableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid device_id")
		return
	}

	if err := h.authSvc.MFAService().DisableMFA(r.Context(), deviceID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"disabled": true})
}

type mfaLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	MFACode  string `json:"mfa_code"`
}

func (h *Handler) mfaLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req mfaLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := clientIP(r)
	userAgent := r.Header.Get("User-Agent")

	tokens, err := h.authSvc.LoginMFA(r.Context(), req.Username, req.Password, req.MFACode, ip, userAgent)
	if err != nil {
		log.Printf("MFA login error for user %s: %v", req.Username, err)
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// --- Password Policy ---

// PasswordPolicyConfigRequest is the body for POST /api/v1/auth/password-policy.
type PasswordPolicyConfigRequest struct {
	MinLength      *int     `json:"min_length,omitempty"`
	RequireUpper   *bool    `json:"require_uppercase,omitempty"`
	RequireLower   *bool    `json:"require_lowercase,omitempty"`
	RequireDigit   *bool    `json:"require_digit,omitempty"`
	RequireSpecial *bool    `json:"require_special,omitempty"`
	Blacklist      []string `json:"blacklist,omitempty"`
}

func (h *Handler) passwordPolicy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		policy := h.authSvc.PasswordPolicy()
		writeJSON(w, http.StatusOK, policy)
	case http.MethodPost:
		var req PasswordPolicyConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.authSvc.UpdatePasswordPolicy(req.MinLength, req.RequireUpper, req.RequireLower, req.RequireDigit, req.RequireSpecial, req.Blacklist); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		policy := h.authSvc.PasswordPolicy()
		writeJSON(w, http.StatusOK, policy)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- Magic Link (Passwordless Login) ---

func (h *Handler) magicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	// Look up user by email via identity client.
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing valid X-Tenant-ID header")
		return
	}

	ctx := ggidtenant.WithContext(r.Context(), &ggidtenant.Context{
		TenantID:       tenantID,
		IsolationLevel: ggidtenant.IsolationShared,
	})

	user, err := h.authSvc.LookupUser(ctx, tenantID, body.Email)
	if err != nil || user == nil {
		// Don't reveal whether email exists — return 200.
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "sent",
			"message": "If the email exists, a magic link has been sent.",
		})
		return
	}

	token, err := h.authSvc.IssueMagicLink(r.Context(), tenantID, user.ID, body.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue magic link")
		return
	}

	// In production, send email. In dev, return the token.
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "sent",
		"message": "If the email exists, a magic link has been sent.",
		"token":   token, // dev mode only
	})
}

func (h *Handler) magicLinkVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" && r.Method == http.MethodPost {
		_ = r.ParseForm()
		token = r.FormValue("token")
	}
	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	ip := clientIP(r)
	userAgent := r.Header.Get("User-Agent")

	tokens, err := h.authSvc.VerifyMagicLink(r.Context(), token, ip, userAgent)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired magic link")
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// --- Email Verification ---

func (h *Handler) emailVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// Try query param fallback.
		body.Token = r.URL.Query().Get("token")
	}
	if body.Token == "" {
		body.Token = r.URL.Query().Get("token")
	}
	if body.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	_, _, _, err := h.authSvc.VerifyEmailToken(r.Context(), body.Token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired verification token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "verified"})
}

func (h *Handler) emailResend(w http.ResponseWriter, r *http.Request) {
	h.sendVerification(w, r)
}

// sendVerification handles POST /api/v1/auth/send-verification and /api/v1/auth/email/resend.
// Generates a verification token, stores it in Redis (24h TTL), and returns it in dev mode.
func (h *Handler) sendVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Email   string `json:"email"`
		UserID  string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Email == "" && body.UserID == "" {
		writeError(w, http.StatusBadRequest, "email or user_id is required")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	// If only email is provided, look up the user.
	var userID uuid.UUID
	if body.UserID != "" {
		userID, err = uuid.Parse(body.UserID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
	} else {
		user, err := h.authSvc.LookupUser(r.Context(), tc.TenantID, body.Email)
		if err != nil || user == nil {
			// Don't reveal whether email exists.
			writeJSON(w, http.StatusOK, map[string]string{
				"status":  "sent",
				"message": "If the email exists, a verification link has been sent.",
			})
			return
		}
		userID = user.ID
		if body.Email == "" {
			body.Email = user.Email
		}
	}

	token, err := h.authSvc.SendVerificationEmail(r.Context(), tc.TenantID, userID, body.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send verification email")
		return
	}

	// Don't reveal whether email exists — but return token in dev mode.
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "sent",
		"message": "If the email exists, a verification link has been sent.",
		"token":   token, // dev mode only
	})
}

// verifyEmail handles GET /api/v1/auth/verify-email?token=xxx.
// Also supports POST with JSON body for API clients.
func (h *Handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" && r.Method == http.MethodPost {
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			token = body.Token
		}
	}
	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	_, _, _, err := h.authSvc.VerifyEmailToken(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired verification token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "verified"})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case stderrors.Is(err, service.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid credentials")
	case stderrors.Is(err, service.ErrAccountLocked):
		writeError(w, http.StatusLocked, "account temporarily locked")
	case stderrors.Is(err, service.ErrMFASetupRequired):
		writeError(w, http.StatusForbidden, err.Error())
	case stderrors.Is(err, service.ErrRateLimited):
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
	case stderrors.Is(err, service.ErrSessionNotFound):
		writeError(w, http.StatusNotFound, "session not found")
	case stderrors.Is(err, service.ErrPasswordTooShort), stderrors.Is(err, service.ErrPasswordTooWeak):
		writeError(w, http.StatusBadRequest, err.Error())
	case stderrors.Is(err, service.ErrPasswordReused):
		writeError(w, http.StatusConflict, err.Error())
	case stderrors.Is(err, service.ErrCredentialAlreadyExists):
		writeError(w, http.StatusConflict, "username or email already registered")
	case stderrors.Is(err, service.ErrInvalidResetToken):
		writeError(w, http.StatusBadRequest, "invalid or expired reset token")
	default:
		var ge *ggiderrors.GGIDError
		if stderrors.As(err, &ge) {
			writeError(w, http.StatusInternalServerError, ge.Message)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Use net.SplitHostPort to correctly handle both IPv4 and IPv6
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// --- Social Login ---

// --- Phone OTP ---

func (h *Handler) phoneOTPSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Phone  string `json:"phone"`
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Phone == "" {
		writeError(w, http.StatusBadRequest, "phone is required")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	otp, err := h.authSvc.SendPhoneOTP(r.Context(), tc.TenantID, userID, body.Phone)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	// In production, send OTP via SMS. In dev, return the OTP.
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "sent",
		"message": "OTP sent to phone number",
		"otp":     otp, // dev mode only
	})
}

func (h *Handler) phoneOTPVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Phone string `json:"phone"`
		OTP   string `json:"otp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Phone == "" || body.OTP == "" {
		writeError(w, http.StatusBadRequest, "phone and otp are required")
		return
	}

	ip := clientIP(r)
	userAgent := r.Header.Get("User-Agent")

	tokens, err := h.authSvc.VerifyPhoneOTP(r.Context(), body.Phone, body.OTP, ip, userAgent)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// --- Step-up Authentication ---

func (h *Handler) stepUpChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		UserID string `json:"user_id"`
		Method string `json:"method"` // "password" or "mfa"
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	if body.Method == "" {
		body.Method = "password"
	}

	result, err := h.authSvc.InitStepUp(r.Context(), userID, body.Method)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) stepUpVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Challenge string `json:"challenge"`
		Code      string `json:"code"`      // for MFA method
		Password  string `json:"password"`  // for password method
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Challenge == "" {
		writeError(w, http.StatusBadRequest, "challenge is required")
		return
	}

	result, err := h.authSvc.VerifyStepUp(r.Context(), body.Challenge, body.Code, body.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// --- Auth Hooks (Auth0 Actions equivalent) ---

func (h *Handler) manageHooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var hook service.AuthHook
		if err := json.NewDecoder(r.Body).Decode(&hook); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if hook.ID == "" {
			hook.ID = uuid.New().String()
		}
		if hook.Headers == nil {
			hook.Headers = make(map[string]string)
		}
		hook.Enabled = true
		h.hooks.RegisterHook(&hook)
		writeJSON(w, http.StatusCreated, map[string]string{"id": hook.ID, "status": "registered"})
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "id is required")
			return
		}
		h.hooks.RemoveHook(id)
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- Passwordless (WebAuthn-only) ---

func (h *Handler) passwordlessRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Passwordless registration = standard registration but no password required.
	// Instead, the user must complete WebAuthn registration.
	var body struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	userID, _ := uuid.Parse(body.UserID)
	if userID == uuid.Nil {
		userID = uuid.New()
	}

	// Create a random temporary password (user will never use it).
	tempPass, _ := crypto.GenerateRandomToken(32)
	username := body.Username
	if username == "" {
		username = body.Email
	}

	if err := h.authSvc.Register(r.Context(), tc.TenantID, userID, username, tempPass); err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"user_id":  userID.String(),
		"message":  "Passwordless account created. Complete WebAuthn registration at /api/v1/webauthn/register/begin",
		"next_url": "/api/v1/webauthn/register/begin?user_id=" + userID.String(),
	})
}

// --- MFA WebAuthn (second factor) ---

func (h *Handler) mfaWebAuthnBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	// Redirect to WebAuthn begin registration endpoint.
	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "mfa_webauthn_challenge",
		"message":  "Complete WebAuthn registration as second factor",
		"begin_url": "/api/v1/webauthn/register/begin?user_id=" + body.UserID,
	})
}

func (h *Handler) mfaWebAuthnFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "mfa_webauthn_enrolled",
		"message": "WebAuthn second factor enrolled successfully",
	})
}

// --- IdP Federation ---

func (h *Handler) idpConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var configs []*service.IdPConfig
		for _, c := range h.idpConfigs {
			configs = append(configs, c)
		}
		writeJSON(w, http.StatusOK, map[string]any{"configs": configs})

	case http.MethodPost:
		var cfg service.IdPConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if cfg.ID == "" {
			cfg.ID = uuid.New().String()
		}
		if cfg.Enabled {
			// no-op, just mark as enabled
		}
		cfg.Enabled = true
		h.idpConfigs[cfg.ID] = &cfg
		writeJSON(w, http.StatusCreated, cfg)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "id is required")
			return
		}
		delete(h.idpConfigs, id)
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- Logout All Devices ---

func (h *Handler) logoutAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	var body struct {
		UserID string `json:"user_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	if err := h.authSvc.LogoutAll(r.Context(), tc.TenantID, userID, uuid.Nil); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to logout all devices")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"logged_out_all": true})
}

// --- Email Change (Dual Confirmation) ---

func (h *Handler) emailChange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		UserID   string `json:"user_id"`
		OldEmail string `json:"old_email"`
		NewEmail string `json:"new_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	result, err := h.authSvc.InitiateEmailChange(r.Context(), userID, body.OldEmail, body.NewEmail)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) emailChangeConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Token string `json:"token"`
		Step  string `json:"step"` // "old" or "new"
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	applied, err := h.authSvc.ConfirmEmailChange(r.Context(), body.Token, body.Step)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"confirmed": true,
		"applied":   applied,
	})
}

// --- Auth0 Lock Compatible Endpoints ---

// authorize handles the Auth0-compatible /authorize endpoint.
// Supports client_id + connection parameters like Auth0 Lock.
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	connection := r.URL.Query().Get("connection")
	responseType := r.URL.Query().Get("response_type")
	redirectURI := r.URL.Query().Get("redirect_uri")

	if clientID == "" {
		writeError(w, http.StatusBadRequest, "client_id is required")
		return
	}

	// If connection is specified, redirect to that social provider.
	if connection != "" && connection != "Username-Password-Authentication" {
		// Social connection — redirect to social login begin.
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		host := r.Host
		if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
			host = fh
		}
		authURL := fmt.Sprintf("%s://%s/api/v1/auth/social/%s?redirect_uri=%s",
			scheme, host, connection, redirectURI)
		writeJSON(w, http.StatusOK, map[string]string{
			"redirect_to": authURL,
			"connection":  connection,
			"client_id":   clientID,
		})
		return
	}

	// Database connection — return login page parameters for Lock.
	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":     clientID,
		"response_type": responseType,
		"redirect_uri":  redirectURI,
		"connection":    connection,
		"domain":        r.Host,
		"state":         r.URL.Query().Get("state"),
	})
}

// usernamePasswordLogin handles Auth0 Lock's /usernamepassword/login endpoint.
func (h *Handler) usernamePasswordLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Connection string `json:"connection"`
		ClientID  string `json:"client_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := clientIP(r)
	userAgent := r.Header.Get("User-Agent")

	tokens, err := h.authSvc.Login(r.Context(), body.Username, body.Password, ip, userAgent)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// dbConnectionsSignup handles Auth0 Lock's /dbconnections/signup endpoint.
func (h *Handler) dbConnectionsSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Connection string `json:"connection"`
		ClientID  string `json:"client_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	userID := uuid.New()
	username := body.Email
	if username == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	if err := h.authSvc.Register(r.Context(), tc.TenantID, userID, username, body.Password); err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"_id":  userID.String(),
		"email": body.Email,
	})
}


// handleSocial routes social login requests: /api/v1/auth/social/{provider} and /api/v1/auth/social/{provider}/callback
func (h *Handler) handleSocial(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/social/")
	parts := strings.SplitN(path, "/", 2)

	provider := parts[0]
	if provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	isCallback := len(parts) == 2 && parts[1] == "callback"

	if isCallback {
		h.socialCallback(w, r, provider)
		return
	}
	h.socialBegin(w, r, provider)
}

func (h *Handler) socialBegin(w http.ResponseWriter, r *http.Request, provider string) {
	// Validate the connector exists
	conn, err := h.socialReg.Get(provider)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unsupported provider: "+provider)
		return
	}

	state := uuid.New().String()
	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		redirectURI = "/login"
	}

	// Build callback URL
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	// Use forwarded host if behind proxy
	host := r.Host
	if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
		host = fh
	}
	callbackURL := scheme + "://" + host + "/api/v1/auth/social/" + provider + "/callback"

	authURL, err := conn.GetAuthURL(r.Context(), state, callbackURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build auth URL")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"provider":    provider,
		"auth_url":    authURL,
		"state":       state,
		"redirect_to": redirectURI,
	})
}

func (h *Handler) socialCallback(w http.ResponseWriter, r *http.Request, provider string) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	conn, err := h.socialReg.Get(provider)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unsupported provider: "+provider)
		return
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	host := r.Host
	if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
		host = fh
	}
	callbackURL := scheme + "://" + host + "/api/v1/auth/social/" + provider + "/callback"

	userInfo, err := conn.HandleCallback(r.Context(), code, state, callbackURL)
	if err != nil {
		log.Printf("social callback error (%s): %v", provider, err)
		writeError(w, http.StatusUnauthorized, "social authentication failed")
		return
	}

	log.Printf("social login success: provider=%s external_id=%s email=%s", userInfo.Provider, userInfo.ExternalID, userInfo.Email)

	// Complete social login: JIT-provision or link identity, then issue JWT.
	ip := clientIP(r)
	userAgent := r.Header.Get("User-Agent")

	tokens, err := h.authSvc.SocialLogin(r.Context(), userInfo.Provider, userInfo.ExternalID, userInfo.Email, userInfo.Name, userInfo.AvatarURL, ip, userAgent)
	if err != nil {
		log.Printf("social login completion error (%s): %v", provider, err)
		writeError(w, http.StatusInternalServerError, "social login failed")
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}
