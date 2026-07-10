// Package server implements the HTTP handler for the Identity Service.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"

	ggiderrors "github.com/ggid/ggid/pkg/errors"
	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/ggid/ggid/services/identity/internal/scim"
	"github.com/ggid/ggid/services/identity/internal/service"
	"github.com/google/uuid"
)

// compile-time interface assertion
var _ http.Handler = (*HTTPHandler)(nil)

// HTTPHandler is the HTTP handler for the Identity Service REST API.
type HTTPHandler struct {
	svc *service.IdentityService
	mux *http.ServeMux
}

// NewHTTPHandler creates a new HTTP handler with all routes registered.
func NewHTTPHandler(svc *service.IdentityService) *HTTPHandler {
	h := &HTTPHandler{svc: svc}
	h.registerRoutes()
	return h
}

func (h *HTTPHandler) registerRoutes() {
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/healthz", h.healthz)
	h.mux.HandleFunc("/api/v1/users", h.handleUsers)
	h.mux.HandleFunc("/api/v1/users/", h.handleUserByID)
	h.mux.HandleFunc("/api/v1/users/import", h.handleImportCSV)

	// SCIM 2.0 endpoints
	scimHandler := scim.NewHandler(h.svc)
	scimHandler.RegisterRoutes(h.mux)
}

// ServeHTTP implements http.Handler.
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *HTTPHandler) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *HTTPHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	ctx, ok := injectTenant(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "missing or invalid X-Tenant-ID header")
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.createUser(ctx, w, r)
	case http.MethodGet:
		h.listUsers(ctx, w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) handleUserByID(w http.ResponseWriter, r *http.Request) {
	ctx, ok := injectTenant(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "missing or invalid X-Tenant-ID header")
		return
	}

	// Extract user ID from path /api/v1/users/{id}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/users/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "user ID is required")
		return
	}

	userID, err := uuid.Parse(parts[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// Check for sub-path (e.g. /api/v1/users/{id}/lock)
	action := ""
	if len(parts) > 1 && parts[1] != "" {
		action = parts[1]
	}

	switch {
	case action == "" && r.Method == http.MethodGet:
		h.getUser(ctx, userID, w, r)
	case action == "" && r.Method == http.MethodDelete:
		h.deleteUser(ctx, userID, w, r)
	case action == "" && r.Method == http.MethodPatch:
		h.updateUser(ctx, userID, w, r)
	case action == "lock" && r.Method == http.MethodPost:
		h.lockUser(ctx, userID, w, r)
	case action == "unlock" && r.Method == http.MethodPost:
		h.unlockUser(ctx, userID, w, r)
	case action == "deactivate" && r.Method == http.MethodPost:
		h.deactivateUser(ctx, userID, w, r)
	case action == "activate" && r.Method == http.MethodPost:
		h.activateUser(ctx, userID, w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) createUser(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string `json:"username"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		Phone       string `json:"phone"`
		DisplayName string `json:"display_name"`
		Locale      string `json:"locale"`
		Timezone    string `json:"timezone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}

	user, err := h.svc.CreateUser(ctx, &domain.CreateUserInput{
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		Phone:       req.Phone,
		DisplayName: req.DisplayName,
		Locale:      req.Locale,
		Timezone:    req.Timezone,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, userToJSON(user))
}

func (h *HTTPHandler) getUser(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userToJSON(user))
}

func (h *HTTPHandler) listUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	filter := &domain.ListUsersFilter{
		PageSize: 50,
		Offset:   0,
	}
	if s := r.URL.Query().Get("search"); s != "" {
		filter.Search = s
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		var n int
		fmt.Sscanf(ps, "%d", &n)
		if n > 0 {
			filter.PageSize = n
		}
	}

	result, err := h.svc.ListUsers(ctx, filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	users := make([]map[string]any, len(result.Users))
	for i, u := range result.Users {
		users[i] = userToJSON(u)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"users":       users,
		"total":       result.Total,
		"next_offset": result.NextOffset,
	})
}

func (h *HTTPHandler) deleteUser(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteUser(ctx, userID); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *HTTPHandler) updateUser(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone       *string `json:"phone"`
		DisplayName *string `json:"display_name"`
		Locale      *string `json:"locale"`
		Timezone    *string `json:"timezone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	user, err := h.svc.UpdateUser(ctx, userID, &domain.UpdateUserInput{
		Phone:       req.Phone,
		DisplayName: req.DisplayName,
		Locale:      req.Locale,
		Timezone:    req.Timezone,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userToJSON(user))
}

func (h *HTTPHandler) lockUser(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.LockUser(ctx, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userToJSON(user))
}

func (h *HTTPHandler) unlockUser(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.UnlockUser(ctx, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userToJSON(user))
}

func (h *HTTPHandler) deactivateUser(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.DeactivateUser(ctx, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userToJSON(user))
}

func (h *HTTPHandler) activateUser(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.ActivateUser(ctx, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userToJSON(user))
}

// --- Helpers ---

func injectTenant(r *http.Request) (context.Context, bool) {
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		return nil, false
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, false
	}
	tc := &ggidtenant.Context{
		TenantID:       tenantID,
		IsolationLevel: ggidtenant.IsolationShared,
	}
	return ggidtenant.WithContext(r.Context(), tc), true
}

func userToJSON(u *domain.User) map[string]any {
	m := map[string]any{
		"id":             u.ID.String(),
		"tenant_id":      u.TenantID.String(),
		"username":       u.Username,
		"email":          u.Email,
		"phone":          u.Phone,
		"status":         string(u.Status),
		"email_verified": u.EmailVerified,
		"display_name":   u.DisplayName,
		"locale":         u.Locale,
		"timezone":       u.Timezone,
		"created_at":     u.CreatedAt,
		"updated_at":     u.UpdatedAt,
	}
	if u.PrimaryEmailID != nil {
		m["primary_email_id"] = u.PrimaryEmailID.String()
	}
	return m
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeServiceError(w http.ResponseWriter, err error) {
	if ge, ok := ggiderrors.AsGGIDError(err); ok {
		switch ge.Code {
		case ggiderrors.ErrNotFound:
			writeError(w, http.StatusNotFound, ge.Message)
		case ggiderrors.ErrAlreadyExists:
			writeError(w, http.StatusConflict, ge.Message)
		case ggiderrors.ErrInvalidArgument:
			writeError(w, http.StatusBadRequest, ge.Message)
		case ggiderrors.ErrPermissionDenied:
			writeError(w, http.StatusForbidden, ge.Message)
		case ggiderrors.ErrUnauthenticated:
			writeError(w, http.StatusUnauthorized, ge.Message)
		default:
			writeError(w, http.StatusInternalServerError, ge.Message)
		}
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

// handleImportCSV handles POST /api/v1/users/import
// Accepts CSV body with columns: username,email,password
func (h *HTTPHandler) handleImportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) == 0 {
		writeError(w, http.StatusBadRequest, "empty CSV")
		return
	}

	// Skip header row if it looks like a header.
	startIdx := 0
	if strings.Contains(strings.ToLower(lines[0]), "username") {
		startIdx = 1
	}

	type importResult struct {
		Line    int    `json:"line"`
		Status  string `json:"status"`
		Message string `json:"message,omitempty"`
	}

	var results []importResult
	successCount := 0

	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 3 {
			results = append(results, importResult{Line: i + 1, Status: "error", Message: "need username,email,password"})
			continue
		}

		username := strings.TrimSpace(fields[0])
		email := strings.TrimSpace(fields[1])
		password := strings.TrimSpace(fields[2])

		if username == "" || email == "" || password == "" {
			results = append(results, importResult{Line: i + 1, Status: "error", Message: "empty fields"})
			continue
		}

		// Email format validation.
		if _, err := mail.ParseAddress(email); err != nil {
			results = append(results, importResult{Line: i + 1, Status: "error", Message: "invalid email format"})
			continue
		}

		// Password strength check (min 8 chars, must have upper+lower+digit).
		if len(password) < 8 {
			results = append(results, importResult{Line: i + 1, Status: "error", Message: "password must be at least 8 characters"})
			continue
		}
		var hasUpper, hasLower, hasDigit bool
		for _, ch := range password {
			switch {
			case 'A' <= ch && ch <= 'Z':
				hasUpper = true
			case 'a' <= ch && ch <= 'z':
				hasLower = true
			case '0' <= ch && ch <= '9':
				hasDigit = true
			}
		}
		if !hasUpper || !hasLower || !hasDigit {
			results = append(results, importResult{Line: i + 1, Status: "error", Message: "password must contain uppercase, lowercase, and digit"})
			continue
		}

		_, err := h.svc.CreateUser(r.Context(), &domain.CreateUserInput{
			Username: username,
			Email:    email,
			Password: password,
		})
		if err != nil {
			results = append(results, importResult{Line: i + 1, Status: "error", Message: err.Error()})
			continue
		}

		results = append(results, importResult{Line: i + 1, Status: "created"})
		successCount++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":   len(lines) - startIdx,
		"success": successCount,
		"failed":  len(lines) - startIdx - successCount,
		"results": results,
	})
}
