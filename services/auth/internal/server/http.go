// Package server implements the HTTP server for the Auth Service.
package server

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
	"strings"

	ggiderrors "github.com/ggid/ggid/pkg/errors"
	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/service"
	"github.com/google/uuid"
)

// Handler is the HTTP handler for the Auth Service.
type Handler struct {
	authSvc *service.AuthService
	mux     *http.ServeMux
}

// New creates a new Auth Service HTTP handler.
func New(authSvc *service.AuthService) *Handler {
	h := &Handler{authSvc: authSvc}
	h.registerRoutes()
	return h
}

func (h *Handler) registerRoutes() {
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/auth/login", h.login)
	h.mux.HandleFunc("/api/v1/auth/register", h.register)
	h.mux.HandleFunc("/api/v1/auth/logout", h.logout)
	h.mux.HandleFunc("/api/v1/auth/refresh", h.refresh)
	h.mux.HandleFunc("/api/v1/auth/password/forgot", h.forgotPassword)
	h.mux.HandleFunc("/api/v1/auth/password/reset", h.resetPassword)
	h.mux.HandleFunc("/api/v1/auth/password/change", h.changePassword)
	h.mux.HandleFunc("/api/v1/auth/sessions", h.handleSessions)
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
	// Strip port from RemoteAddr
	idx := strings.LastIndex(r.RemoteAddr, ":")
	if idx > 0 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}
