// Package server implements the HTTP handler for the Identity Service.
package server

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	ggiderrors "github.com/ggid/ggid/pkg/errors"
	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/ggid/ggid/services/identity/internal/idpconfig"
	"github.com/ggid/ggid/services/identity/internal/scim"
	"github.com/ggid/ggid/services/identity/internal/service"
	"github.com/google/uuid"
)

// compile-time interface assertion
var _ http.Handler = (*HTTPHandler)(nil)

// HTTPHandler is the HTTP handler for the Identity Service REST API.
type HTTPHandler struct {
	svc              *service.IdentityService
	mux              *http.ServeMux
	brandingStore    *service.BrandingStore
	accessRequestSvc *service.AccessRequestService
	idpConfigSvc     *idpconfig.Service
}

// NewHTTPHandler creates a new HTTP handler with all routes registered.
func NewHTTPHandler(svc *service.IdentityService) *HTTPHandler {
	h := &HTTPHandler{
		svc:              svc,
		brandingStore:    service.NewBrandingStore(),
		accessRequestSvc: service.NewAccessRequestService(service.NewMemoryAccessRequestStore()),
		idpConfigSvc:     idpconfig.NewService(idpconfig.NewMemoryStore()),
	}
	h.registerRoutes()
	return h
}

func (h *HTTPHandler) registerRoutes() {
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/healthz", h.healthz)
	h.mux.HandleFunc("/readyz", h.readyz)
	h.mux.Handle("/metrics", promhttp.Handler())
	h.mux.HandleFunc("/api/v1/users", h.handleUsers)
	h.mux.HandleFunc("/api/v1/users/", h.handleUserByID)
	h.mux.HandleFunc("/api/v1/users/import", h.handleImportCSV)
	h.mux.HandleFunc("/api/v1/users/export", h.handleExportUsers)
	h.mux.HandleFunc("/api/v1/users/link", h.handleLinkAccount)
	h.mux.HandleFunc("/api/v1/users/unlink", h.handleUnlinkAccount)
	h.mux.HandleFunc("/api/v1/users/import/validate", h.handleImportValidate)
	h.mux.HandleFunc("/api/v1/users/bulk/status", h.handleBulkStatus)

	// Branding endpoints
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	// Access request (IGA workflow) endpoints
	h.mux.HandleFunc("/api/v1/access-requests", h.handleAccessRequests)
	h.mux.HandleFunc("/api/v1/access-requests/", h.handleAccessRequests)

	// SCIM 2.0 endpoints
	scimHandler := scim.NewHandler(h.svc)
	scimHandler.RegisterRoutes(h.mux)
	// SCIM Groups also accessible via /api/v1/scim/ prefix (gateway-compatible)
	h.mux.HandleFunc("/api/v1/scim/Groups", scimHandler.HandleGroupsCollectionPublic)
	h.mux.HandleFunc("/api/v1/scim/Groups/", scimHandler.HandleGroupResourcePublic)

	// Impersonation audit trail
	h.mux.HandleFunc("/api/v1/audit/impersonation", h.handleImpersonationAudit)

	// Enhanced user search
	h.mux.HandleFunc("/api/v1/users/search", h.handleUserSearch)

	// JIT provisioning
	h.mux.HandleFunc("/api/v1/users/jit-provision", func(w http.ResponseWriter, r *http.Request) {
		h.handleJITProvision(r.Context(), w, r)
	})
	h.mux.HandleFunc("/api/v1/users/by-attribute", h.handleUserByAttribute)
	h.mux.HandleFunc("/api/v1/users/attribute-history", func(w http.ResponseWriter, r *http.Request) {
		uid := uuid.MustParse(r.URL.Query().Get("user_id"))
		h.handleAttributeHistory(r.Context(), uid, w, r)
	})

	// User lifecycle automation
	h.mux.HandleFunc("/api/v1/users/lifecycle/rules", h.handleLifecycleRules)
}

// ServeHTTP implements http.Handler.
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *HTTPHandler) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// readyz checks readiness for serving requests (readiness probe).
func (h *HTTPHandler) readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
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

	// Handle /api/v1/users/me — current user profile.
	if parts[0] == "me" {
		h.handleMe(ctx, w, r)
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
	case action == "restore" && r.Method == http.MethodPost:
		h.restoreUser(ctx, userID, w, r)
	case action == "avatar" && r.Method == http.MethodPost:
		h.uploadAvatar(ctx, userID, w, r)
	case action == "merge" && r.Method == http.MethodPost:
		h.handleMerge(ctx, userID, w, r)
	case action == "lifecycle-preview" && r.Method == http.MethodGet:
		h.handleLifecyclePreview(ctx, userID, w, r)
	case action == "deprovision" && r.Method == http.MethodPost:
		h.handleDeprovision(ctx, userID, w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleMe handles GET/POST /api/v1/users/me — current user profile.
// GET returns the user's full profile. POST updates limited fields.
// Uses X-User-ID header (set by Gateway after JWT verification) to identify the user.
func (h *HTTPHandler) handleMe(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user identity")
		return
	}

	switch r.Method {
	case http.MethodGet:
		user, err := h.svc.GetUser(ctx, userID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, userToJSON(user))

	case http.MethodPost, http.MethodPatch:
		// Limited self-update: only display_name, phone, avatar_url.
		var body struct {
			DisplayName *string `json:"display_name"`
			Phone       *string `json:"phone"`
			AvatarURL   *string `json:"avatar_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		user, err := h.svc.UpdateUser(ctx, userID, &domain.UpdateUserInput{
			DisplayName: body.DisplayName,
			Phone:       body.Phone,
			AvatarURL:   body.AvatarURL,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, userToJSON(user))

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
	q := r.URL.Query()
	if s := q.Get("search"); s != "" {
		filter.Search = s
	}
	if ps := q.Get("page_size"); ps != "" {
		var n int
		fmt.Sscanf(ps, "%d", &n)
		if n > 0 {
			filter.PageSize = n
		}
	}

	// Multi-criteria filtering.
	if st := q.Get("status"); st != "" {
		ws := domain.UserStatus(st)
		if ws.IsValid() {
			filter.Status = &ws
		}
	}
	if ca := q.Get("created_after"); ca != "" {
		if t, err := time.Parse(time.RFC3339, ca); err == nil {
			filter.CreatedAfter = &t
		}
	}
	if cb := q.Get("created_before"); cb != "" {
		if t, err := time.Parse(time.RFC3339, cb); err == nil {
			filter.CreatedBefore = &t
		}
	}
	if la := q.Get("last_login_after"); la != "" {
		if t, err := time.Parse(time.RFC3339, la); err == nil {
			filter.LastLoginAfter = &t
		}
	}
	if oid := q.Get("org_id"); oid != "" {
		if id, err := uuid.Parse(oid); err == nil {
			filter.OrgID = &id
		}
	}
	if rid := q.Get("role_id"); rid != "" {
		if id, err := uuid.Parse(rid); err == nil {
			filter.RoleID = &id
		}
	}

	// Sorting.
	if sb := q.Get("sort_by"); sb != "" {
		filter.SortBy = sb
	}
	if so := q.Get("sort_order"); so == "desc" {
		filter.SortDesc = true
	} else if so == "asc" {
		filter.SortDesc = false
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

func (h *HTTPHandler) restoreUser(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.RestoreUser(ctx, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userToJSON(user))
}

// uploadAvatar handles POST /api/v1/users/{id}/avatar.
// Accepts multipart/form-data with an image file (max 2MB, image/* types).
// Stores the file locally and returns the avatar_url.
func (h *HTTPHandler) uploadAvatar(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	// Limit request body to 2MB.
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20) // 2MB

	if err := r.ParseMultipartForm(2 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid form data (max 2MB)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer file.Close()

	// Validate content type.
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		writeError(w, http.StatusBadRequest, "file must be an image (image/*)")
		return
	}

	// Validate file size (redundant with MaxBytesReader, but explicit).
	if header.Size > 2<<20 {
		writeError(w, http.StatusBadRequest, "file size exceeds 2MB limit")
		return
	}

	// Read the file content.
	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	// Determine file extension from content type.
	ext := ".png"
	switch contentType {
	case "image/jpeg", "image/jpg":
		ext = ".jpg"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}

	// Store file locally (production would use S3/CDN).
	avatarDir := "uploads/avatars"
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create avatar directory")
		return
	}

	filename := userID.String() + ext
	filePath := filepath.Join(avatarDir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save avatar")
		return
	}

	// Update user's avatar_url in the database.
	avatarURL := "/uploads/avatars/" + filename
	_, err = h.svc.UpdateUser(ctx, userID, &domain.UpdateUserInput{
		AvatarURL: &avatarURL,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":     "uploaded",
		"avatar_url": avatarURL,
	})
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
	ggiderrors.WriteSimpleAPIError(w, status, httpStatusToCode(status), msg)
}

// httpStatusToCode maps an HTTP status code to a GGID error code string.
func httpStatusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return string(ggiderrors.ErrInvalidArgument)
	case http.StatusUnauthorized:
		return string(ggiderrors.ErrUnauthenticated)
	case http.StatusForbidden:
		return string(ggiderrors.ErrPermissionDenied)
	case http.StatusNotFound:
		return string(ggiderrors.ErrNotFound)
	case http.StatusConflict:
		return string(ggiderrors.ErrAlreadyExists)
	case http.StatusTooManyRequests:
		return string(ggiderrors.ErrResourceExhausted)
	default:
		return string(ggiderrors.ErrInternal)
	}
}

func writeServiceError(w http.ResponseWriter, err error) {
	ggiderrors.WriteAPIError(w, err, "")
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

// handleExportUsers handles GET /api/v1/users/export?format=csv|json
// Streams the user list in the requested format.
func (h *HTTPHandler) handleExportUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, ok := injectTenant(r)
	if ! ok {
		writeError(w, http.StatusBadRequest, "missing or invalid X-Tenant-ID header")
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "json" {
		writeError(w, http.StatusBadRequest, "format must be csv or json")
		return
	}

	if h.svc == nil {
		writeError(w, http.StatusInternalServerError, "service not initialized")
		return
	}

	result, err := h.svc.ListUsers(ctx, &domain.ListUsersFilter{PageSize: 10000})
	if err != nil || result == nil {
		writeServiceError(w, err)
		return
	}
	users := result.Users

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="users_export.csv"`)
		wr := csv.NewWriter(w)
		wr.Write([]string{"id", "username", "email", "phone", "status", "display_name", "created_at"})
		for _, u := range users {
			wr.Write([]string{
				u.ID.String(),
				u.Username,
				u.Email,
				u.Phone,
				string(u.Status),
				u.DisplayName,
				u.CreatedAt.Format(time.RFC3339),
			})
		}
		wr.Flush()
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="users_export.json"`)
		json.NewEncoder(w).Encode(map[string]any{"users": users})
	}
}
