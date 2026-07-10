// Package server implements the HTTP server for the Auth Service.
package server

import (
	"encoding/json"
	stderrors "errors"
	"log"
	"net"
	"net/http"
	"strings"

	ggiderrors "github.com/ggid/ggid/pkg/errors"
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
}

// New creates a new Auth Service HTTP handler.
func New(authSvc *service.AuthService) *Handler {
	h := &Handler{authSvc: authSvc, socialReg: social.NewRegistry()}
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
	h.mux.HandleFunc("/api/v1/auth/password/change", h.changePassword)
	h.mux.HandleFunc("/api/v1/auth/sessions", h.handleSessions)
	h.mux.HandleFunc("/api/v1/auth/mfa/setup", h.mfaSetup)
	h.mux.HandleFunc("/api/v1/auth/mfa/verify", h.mfaVerify)
	h.mux.HandleFunc("/api/v1/auth/mfa/disable", h.mfaDisable)
	h.mux.HandleFunc("/api/v1/auth/mfa/login", h.mfaLogin)

	// Password policy config endpoint
	h.mux.HandleFunc("/api/v1/auth/password/policy", h.passwordPolicy)

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

	tokens, err := h.authSvc.Login(r.Context(), req.Username, req.Password, ip, userAgent)
	if err != nil {
		log.Printf("login error for user %s: %v", req.Username, err)
		writeAuthError(w, err)
		return
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

func (h *Handler) passwordPolicy(w http.ResponseWriter, r *http.Request) {
	policy := h.authSvc.PasswordPolicy()
	writeJSON(w, http.StatusOK, policy)
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
		writeError(w, http.StatusTooManyRequests, "account temporarily locked")
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
