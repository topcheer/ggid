package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

// TenantInfo represents a minimal tenant record for resolution.
type TenantInfo struct {
	ID   string `json:"tenant_id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// BootstrapRequest is the request body for POST /api/v1/system/bootstrap.
type BootstrapRequest struct {
	TenantName    string   `json:"tenant_name"`
	TenantSlug    string   `json:"tenant_slug"`
	AdminUsername string   `json:"admin_username"`
	AdminEmail    string   `json:"admin_email"`
	AdminPassword string   `json:"admin_password"`
	WebAuthnRPID      string   `json:"webauthn_rp_id"`
	WebAuthnRPOrigins []string `json:"webauthn_rp_origins"`
}

// BootstrapResponse is returned on successful bootstrap.
type BootstrapResponse struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Success  bool   `json:"success"`
}

// handleTenantResolve resolves a tenant by slug to its ID.
// GET /api/v1/tenants/resolve?slug=xxx
func (h *HTTPHandler) handleTenantResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug parameter is required"})
		return
	}

	row := h.svc.Pool().QueryRow(r.Context(),
		`SELECT id::text, name, slug FROM tenants WHERE slug = $1 AND status = 'active'`, slug)

	var t TenantInfo
	if err := row.Scan(&t.ID, &t.Name, &t.Slug); err != nil {
		slog.Error("tenant resolve: scan error", "slug", slug, "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// handleSystemInitialized checks whether the system has been initialized.
// GET /api/v1/system/initialized
func (h *HTTPHandler) handleSystemInitialized(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	ctx := r.Context()
	var tenantCount, userCount int
	if err := h.svc.Pool().QueryRow(ctx, `SELECT count(*) FROM tenants`).Scan(&tenantCount); err != nil {
		tenantCount = 0
	}
	if err := h.svc.Pool().QueryRow(ctx, `SELECT count(*) FROM users WHERE deleted_at IS NULL`).Scan(&userCount); err != nil {
		userCount = 0
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"initialized":  tenantCount > 0 && userCount > 0,
		"tenant_count": tenantCount,
		"user_count":   userCount,
	})
}

// handleSystemBootstrap performs first-time system initialization.
// POST /api/v1/system/bootstrap
// Security: only works when user_count == 0 (self-disabling after first use).
func (h *HTTPHandler) handleSystemBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	ctx := r.Context()

	// Security check: refuse if system already initialized.
	var userCount int
	if err := h.svc.Pool().QueryRow(ctx, `SELECT count(*) FROM users WHERE deleted_at IS NULL`).Scan(&userCount); err != nil {
		slog.Error("bootstrap: failed to count users", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to check system state"})
		return
	}
	if userCount > 0 {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "system already initialized"})
		return
	}

	// Parse request body.
	var req BootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Validate required fields.
	if req.TenantName == "" || req.TenantSlug == "" || req.AdminUsername == "" ||
		req.AdminEmail == "" || req.AdminPassword == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "all fields are required"})
		return
	}

	// Validate password strength (basic policy).
	if len(req.AdminPassword) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
		return
	}

	// 1. Create tenant (or use existing if already present).
	var tenantIDStr string
	err := h.svc.Pool().QueryRow(ctx,
		`INSERT INTO tenants (name, slug, status, plan) VALUES ($1, $2, 'active', 'enterprise')
		 ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id::text`,
		req.TenantName, req.TenantSlug).Scan(&tenantIDStr)
	if err != nil {
		slog.Error("bootstrap: failed to create tenant", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create tenant"})
		return
	}
	tenantID, _ := uuid.Parse(tenantIDStr)

	// Inject tenant context for CreateUser
	ctx = tenant.WithContext(ctx, &tenant.Context{TenantID: tenantID})

	// 2. Create admin user via IdentityService.
	user, err := h.svc.CreateUser(ctx, &domain.CreateUserInput{
		TenantID:    tenantID,
		Username:    req.AdminUsername,
		Email:       req.AdminEmail,
		Password:    req.AdminPassword,
		DisplayName: req.AdminUsername,
	})
	if err != nil {
		slog.Error("bootstrap: failed to create admin user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to create admin user: %v", err)})
		return
	}

	// 3. Insert credential for admin user (Argon2id hash).
	hash, err := crypto.HashPassword(req.AdminPassword)
	if err != nil {
		slog.Error("bootstrap: failed to hash password", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}
	_, err = h.svc.Pool().Exec(ctx,
		`INSERT INTO credentials (tenant_id, user_id, type, identifier, secret, enabled)
		 VALUES ($1, $2, 'password', $3, $4, true)
		 ON CONFLICT DO NOTHING`,
		tenantID, user.ID, req.AdminUsername, hash)
	if err != nil {
		slog.Error("bootstrap: failed to insert credential", "error", err)
		// Non-fatal — user is created, credential can be set via password reset.
	}

	// 4. Assign admin role via direct SQL (creates role if not exists).
	_, err = h.svc.Pool().Exec(ctx,
		`INSERT INTO roles (tenant_id, key, name, description, system_role)
		 VALUES ($1, 'admin', 'Administrator', 'Full system access', true)
		 ON CONFLICT DO NOTHING`,
		tenantID)
	if err != nil {
		slog.Error("bootstrap: failed to create admin role", "error", err)
	}

	var roleIDStr string
	if err := h.svc.Pool().QueryRow(ctx,
		`SELECT id::text FROM roles WHERE tenant_id = $1 AND key = 'admin'`, tenantID).Scan(&roleIDStr); err != nil {
		slog.Error("bootstrap: failed to get admin role ID", "error", err)
	}

	if roleIDStr != "" {
		roleID, _ := uuid.Parse(roleIDStr)
		_, _ = h.svc.Pool().Exec(ctx,
			`INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, granted_by)
			 VALUES ($1, $2, 'tenant', $3, $4)
			 ON CONFLICT DO NOTHING`,
			user.ID, roleID, tenantID, user.ID)
	}

	// Save WebAuthn config if provided during bootstrap
	if req.WebAuthnRPID != "" || len(req.WebAuthnRPOrigins) > 0 {
		waConfig, _ := json.Marshal(map[string]any{
			"rp_id":           req.WebAuthnRPID,
			"rp_origins":      req.WebAuthnRPOrigins,
			"rp_display_name": "GGID",
		})
		_, _ = h.svc.Pool().Exec(ctx, `
			INSERT INTO sys_config (key, value, updated_by)
			VALUES ('webauthn_config', $1, $2)
			ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = NOW(), updated_by = $2`,
			waConfig, user.ID)
	}

	// Mark system as initialized
	_, _ = h.svc.Pool().Exec(ctx, `
		INSERT INTO sys_config (key, value) VALUES ('system_config', '{"initialized": true, "bootstrap_completed": true}')
		ON CONFLICT (key) DO UPDATE SET value = '{"initialized": true, "bootstrap_completed": true}'`)

	slog.Info("bootstrap: system initialized", "tenant", tenantIDStr, "user", user.ID.String())
	writeJSON(w, http.StatusCreated, BootstrapResponse{
		TenantID: tenantIDStr,
		UserID:   user.ID.String(),
		Success:  true,
	})
}

// unused but kept for reference
var _ = context.Background
var _ = strings.TrimSpace

// --- Tenant CRUD ---

// handleTenantCRUD handles GET /api/v1/tenants (list), POST /api/v1/tenants (create),
// GET /api/v1/tenants/{id} (detail), DELETE /api/v1/tenants/{id} (delete).
func (h *HTTPHandler) handleTenantCRUD(w http.ResponseWriter, r *http.Request) {
	pool := h.svc.Pool()
	if pool == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database not available"})
		return
	}

	// Dispatch: /api/v1/tenants vs /api/v1/tenants/{id}
	path := strings.TrimRight(r.URL.Path, "/")
	if path == "/api/v1/tenants" {
		switch r.Method {
		case http.MethodGet:
			h.tenantList(w, r)
		case http.MethodPost:
			h.tenantCreate(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	}

	// /api/v1/tenants/{id}
	tenantID := strings.TrimPrefix(path, "/api/v1/tenants/")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.tenantDetail(w, r, tenantID)
	case http.MethodDelete:
		h.tenantDelete(w, r, tenantID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *HTTPHandler) tenantList(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.Pool().Query(r.Context(), `
		SELECT id::text, name, slug, plan::text, status::text, max_users,
		       created_at, updated_at
		FROM tenants ORDER BY created_at DESC`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query tenants"})
		return
	}
	defer rows.Close()

	tenants := []map[string]any{}
	for rows.Next() {
		var id, name, slug, plan, status string
		var maxUsers int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &name, &slug, &plan, &status, &maxUsers, &createdAt, &updatedAt); err != nil {
			continue
		}
		tenants = append(tenants, map[string]any{
			"id":         id,
			"tenant_id":  id,
			"name":       name,
			"slug":       slug,
			"plan":       plan,
			"status":     status,
			"max_users":  maxUsers,
			"created_at": createdAt,
			"updated_at": updatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tenants": tenants,
		"total":   len(tenants),
	})
}

func (h *HTTPHandler) tenantCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		DisplayName string `json:"display_name"`
		Plan        string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	// Auto-generate slug from name
	if req.Slug == "" {
		req.Slug = strings.ToLower(strings.TrimSpace(req.Name))
		req.Slug = strings.Map(func(c rune) rune {
			if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
				return c
			}
			return '-'
		}, req.Slug)
		req.Slug = strings.Trim(req.Slug, "-")
	}
	if req.Plan == "" {
		req.Plan = "free"
	}

	var tenantID string
	err := h.svc.Pool().QueryRow(r.Context(), `
		INSERT INTO tenants (name, slug, plan, status) VALUES ($1, $2, $3, 'active')
		RETURNING id::text`, req.Name, req.Slug, req.Plan).Scan(&tenantID)
	if err != nil {
		slog.Error("tenant create: DB error", "error", err)
		writeJSON(w, http.StatusConflict, map[string]string{"error": "tenant slug already exists or invalid"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"tenant_id": tenantID,
		"id":        tenantID,
		"name":      req.Name,
		"slug":      req.Slug,
		"plan":      req.Plan,
		"status":    "active",
	})
}

func (h *HTTPHandler) tenantDetail(w http.ResponseWriter, r *http.Request, tenantID string) {
	var id, name, slug, plan, status string
	var maxUsers int
	var createdAt, updatedAt time.Time
	err := h.svc.Pool().QueryRow(r.Context(), `
		SELECT id::text, name, slug, plan::text, status::text, max_users, created_at, updated_at
		FROM tenants WHERE id::text = $1 OR slug = $1`, tenantID).Scan(
		&id, &name, &slug, &plan, &status, &maxUsers, &createdAt, &updatedAt)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": id,
		"id":        id,
		"name":      name,
		"slug":      slug,
		"plan":      plan,
		"status":    status,
		"max_users": maxUsers,
		"created_at": createdAt,
		"updated_at": updatedAt,
	})
}

func (h *HTTPHandler) tenantDelete(w http.ResponseWriter, r *http.Request, tenantID string) {
	// Prevent deleting the default (bootstrap) tenant
	if tenantID == "default" || tenantID == defaultTenantID().String() {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "cannot delete default tenant"})
		return
	}

	tag, err := h.svc.Pool().Exec(r.Context(), `
		DELETE FROM tenants WHERE id::text = $1 OR slug = $1`, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete tenant"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}
