package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

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
	TenantName   string `json:"tenant_name"`
	TenantSlug   string `json:"tenant_slug"`
	AdminUsername string `json:"admin_username"`
	AdminEmail   string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
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
		log.Printf("tenant resolve: scan error for slug=%q: %v", slug, err)
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
	_ = h.svc.Pool().QueryRow(ctx, `SELECT count(*) FROM tenants`).Scan(&tenantCount)
	_ = h.svc.Pool().QueryRow(ctx, `SELECT count(*) FROM users WHERE deleted_at IS NULL`).Scan(&userCount)

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
		log.Printf("bootstrap: failed to count users: %v", err)
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
		log.Printf("bootstrap: failed to create tenant: %v", err)
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
		log.Printf("bootstrap: failed to create admin user: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to create admin user: %v", err)})
		return
	}

	// 3. Insert credential for admin user (Argon2id hash).
	hash, err := crypto.HashPassword(req.AdminPassword)
	if err != nil {
		log.Printf("bootstrap: failed to hash password: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}
	_, err = h.svc.Pool().Exec(ctx,
		`INSERT INTO credentials (tenant_id, user_id, type, identifier, secret, enabled)
		 VALUES ($1, $2, 'password', $3, $4, true)
		 ON CONFLICT DO NOTHING`,
		tenantID, user.ID, req.AdminUsername, hash)
	if err != nil {
		log.Printf("bootstrap: failed to insert credential: %v", err)
		// Non-fatal — user is created, credential can be set via password reset.
	}

	// 4. Assign admin role via direct SQL (creates role if not exists).
	_, err = h.svc.Pool().Exec(ctx,
		`INSERT INTO roles (tenant_id, key, name, description, system_role)
		 VALUES ($1, 'admin', 'Administrator', 'Full system access', true)
		 ON CONFLICT DO NOTHING`,
		tenantID)
	if err != nil {
		log.Printf("bootstrap: failed to create admin role: %v", err)
	}

	var roleIDStr string
	_ = h.svc.Pool().QueryRow(ctx,
		`SELECT id::text FROM roles WHERE tenant_id = $1 AND key = 'admin'`, tenantID).Scan(&roleIDStr)

	if roleIDStr != "" {
		roleID, _ := uuid.Parse(roleIDStr)
		_, _ = h.svc.Pool().Exec(ctx,
			`INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, granted_by)
			 VALUES ($1, $2, 'tenant', $3, $4)
			 ON CONFLICT DO NOTHING`,
			user.ID, roleID, tenantID, user.ID)
	}

	log.Printf("bootstrap: system initialized — tenant=%s user=%s", tenantIDStr, user.ID.String())
	writeJSON(w, http.StatusCreated, BootstrapResponse{
		TenantID: tenantIDStr,
		UserID:   user.ID.String(),
		Success:  true,
	})
}

// unused but kept for reference
var _ = context.Background
var _ = strings.TrimSpace
