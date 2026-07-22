package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SelfRegisterRequest is the B2B self-registration payload.
type SelfRegisterRequest struct {
	OrgName      string             `json:"org_name"`
	OrgSize      string             `json:"org_size"`
	Industry     string             `json:"industry"`
	Admin        AdminAccount       `json:"admin"`
	Branding     *RegisterBranding  `json:"branding,omitempty"`
	CustomDomain string             `json:"custom_domain,omitempty"`
}

type AdminAccount struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type RegisterBranding struct {
	PrimaryColor string `json:"primary_color"`
	LogoURL      string `json:"logo_url"`
}

// SelfRegisterResponse is returned after successful registration.
type SelfRegisterResponse struct {
	TenantID    string            `json:"tenant_id"`
	OrgName     string            `json:"org_name"`
	AdminUserID string            `json:"admin_user_id"`
	LoginURL    string            `json:"login_url"`
	Status      string            `json:"status"`
}

// CIAMMetrics aggregates CIAM-relevant metrics.
type CIAMMetrics struct {
	TotalTenants     int                    `json:"total_tenants"`
	ActiveTenants    int                    `json:"active_tenants"`
	TotalUsers       int                    `json:"total_users"`
	MAU              int                    `json:"mau"`
	MFACoveragePct   int                    `json:"mfa_coverage_pct"`
	Registrations7d  int                    `json:"registrations_7d"`
	B2BSignups30d    int                    `json:"b2b_signups_30d"`
	GeneratedAt      string                 `json:"generated_at"`
}

// TenantBranding holds per-tenant brand customization.
type TenantBranding struct {
	PrimaryColor string `json:"primary_color"`
	LogoURL      string `json:"logo_url"`
	CustomDomain string `json:"custom_domain,omitempty"`
	CSS          string `json:"css,omitempty"`
}

var tenantBrandingStore = map[uuid.UUID]*TenantBranding{}

// handleSelfRegister handles B2B tenant self-registration.
// POST /api/v1/identity/tenants/self-register
//
// Orchestrates: tenant creation → admin user → credential → roles →
// permissions → OAuth client → email verification token.
// All DB writes are best-effort with ON CONFLICT DO NOTHING to be
// idempotent — duplicate requests won't corrupt data.
func (h *HTTPHandler) handleSelfRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SelfRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.OrgName == "" || req.Admin.Email == "" || req.Admin.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "org_name, admin.email, and admin.password are required")
		return
	}

	pool := h.svc.Pool()
	if pool == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	ctx := r.Context()

	// 1. Create tenant
	slug := strings.ToLower(strings.ReplaceAll(req.OrgName, " ", "-"))
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "tenant-" + uuid.New().String()[:8]
	}
	var tenantIDStr string
	err := pool.QueryRow(ctx,
		`INSERT INTO tenants (name, slug, status, plan)
		 VALUES ($1, $2, 'active', 'starter')
		 ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id::text`,
		req.OrgName, slug).Scan(&tenantIDStr)
	if err != nil {
		slog.Error("B2B register: create tenant failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create tenant")
		return
	}
	tenantID, _ := uuid.Parse(tenantIDStr)
	ctx = tenant.WithContext(ctx, &tenant.Context{TenantID: tenantID})

	// 2. Create admin user via IdentityService
	adminName := req.Admin.Name
	if adminName == "" {
		adminName = req.Admin.Email
	}
	user, err := h.svc.CreateUser(ctx, &domain.CreateUserInput{
		TenantID:    tenantID,
		Username:    req.Admin.Email,
		Email:       req.Admin.Email,
		Password:    req.Admin.Password,
		DisplayName: adminName,
	})
	if err != nil {
		// User might already exist (idempotent retry)
		slog.Warn("B2B register: create admin user", "error", err, "tenant", tenantIDStr)
	}
	if user == nil {
		writeJSONError(w, http.StatusConflict, "admin user already exists")
		return
	}

	// 3. Insert credential (password hash)
	hash, err := crypto.HashPassword(req.Admin.Password)
	if err != nil {
		slog.Error("B2B register: hash password failed", "error", err)
	} else {
		_, _ = pool.Exec(ctx,
			`INSERT INTO credentials (tenant_id, user_id, type, identifier, secret, enabled)
			 VALUES ($1, $2, 'password', $3, $4, true)
			 ON CONFLICT DO NOTHING`,
			tenantID, user.ID, req.Admin.Email, hash)
	}

	// 4. Create default roles + assign admin
	seedDefaultRoles(ctx, pool, tenantID)
	assignAdminRole(ctx, pool, tenantID, user.ID)

	// 5. Create default OAuth client (ggid-console, public)
	clientID := "ggid-console"
	_, _ = pool.Exec(ctx,
		`INSERT INTO oauth_clients (id, tenant_id, client_id, client_secret_hash, name, type, grant_types, response_types, scopes, token_endpoint_auth_method, enabled)
		 VALUES ($1, $2, $3, '', 'GGID Console', 'public',
			 ARRAY['authorization_code','refresh_token','password'],
			 ARRAY['code','token','id_token'],
			 ARRAY['openid','profile','email','offline_access'],
			 'none', true)
		 ON CONFLICT (tenant_id, client_id) DO NOTHING`,
		uuid.New(), tenantID, clientID)

	// 6. Store branding if provided
	if req.Branding != nil {
		_, _ = pool.Exec(ctx,
			`INSERT INTO tenant_branding (tenant_id, primary_color, logo_url, custom_domain)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (tenant_id) DO UPDATE SET primary_color = $2, logo_url = $3, custom_domain = $4`,
			tenantID, req.Branding.PrimaryColor, req.Branding.LogoURL, req.CustomDomain)
	}

	slog.Info("B2B self-register complete", "org", req.OrgName, "tenant_id", tenantIDStr, "admin", req.Admin.Email)

	writeJSON(w, http.StatusCreated, SelfRegisterResponse{
		TenantID:    tenantIDStr,
		OrgName:     req.OrgName,
		AdminUserID: user.ID.String(),
		LoginURL:    "/login",
		Status:      "active",
	})
}

// seedDefaultRoles creates the standard role hierarchy for a new tenant.
func seedDefaultRoles(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) {
	roles := []struct{ key, name, desc string }{
		{"admin", "Administrator", "Full system access"},
		{"tenant:admin", "Tenant Administrator", "Manage tenant users and settings"},
		{"user", "User", "Standard user access"},
		{"viewer", "Viewer", "Read-only access"},
	}
	for _, role := range roles {
		_, _ = pool.Exec(ctx,
			`INSERT INTO roles (tenant_id, key, name, description, system_role)
			 VALUES ($1, $2, $3, $4, true)
			 ON CONFLICT DO NOTHING`,
			tenantID, role.key, role.name, role.desc)
	}
}

// assignAdminRole assigns the admin role to a user within a tenant.
func assignAdminRole(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) {
	var roleIDStr string
	if err := pool.QueryRow(ctx,
		`SELECT id::text FROM roles WHERE tenant_id = $1 AND key = 'admin'`, tenantID).Scan(&roleIDStr); err != nil {
		slog.Error("B2B register: get admin role failed", "error", err)
		return
	}
	roleID, _ := uuid.Parse(roleIDStr)
	_, _ = pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, granted_by)
		 VALUES ($1, $2, 'tenant', $3, $1)
		 ON CONFLICT DO NOTHING`,
		userID, roleID, tenantID)
}

// handleCIAMMetrics returns aggregated CIAM metrics.
// GET /api/v1/identity/ciam/metrics
func (h *HTTPHandler) handleCIAMMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// In production: query tenant count from DB, user count, MFA enrollment, etc.
	// For now returns a structured response ready for DB wiring.
	writeJSON(w, http.StatusOK, CIAMMetrics{
		TotalTenants:    len(tenantBrandingStore),
		ActiveTenants:   len(tenantBrandingStore),
		TotalUsers:      0,
		MAU:             0,
		MFACoveragePct:  0,
		Registrations7d: 0,
		B2BSignups30d:   0,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
	})
}

// handleTenantBranding handles brand customization CRUD.
// GET /api/v1/identity/tenants/branding
// PUT /api/v1/identity/tenants/branding
func (h *HTTPHandler) handleTenantBranding(w http.ResponseWriter, r *http.Request) {
	// Extract tenant ID from header (gateway injects X-Tenant-ID).
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "valid X-Tenant-ID header required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Try DB first
		if pool := h.svc.Pool(); pool != nil {
			var b TenantBranding
			err := pool.QueryRow(r.Context(),
				`SELECT COALESCE(primary_color, '#6366f1'), COALESCE(logo_url, ''), COALESCE(custom_domain, '') FROM tenant_branding WHERE tenant_id = $1`,
				tenantID).Scan(&b.PrimaryColor, &b.LogoURL, &b.CustomDomain)
			if err == nil {
				writeJSON(w, http.StatusOK, b)
				return
			}
		}
		// Fallback: in-memory or default
		branding, ok := tenantBrandingStore[tenantID]
		if !ok {
			branding = &TenantBranding{
				PrimaryColor: "#6366f1",
				LogoURL:      "",
			}
		}
		writeJSON(w, http.StatusOK, branding)

	case http.MethodPut:
		var branding TenantBranding
		if err := json.NewDecoder(r.Body).Decode(&branding); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		// Try DB first
		if pool := h.svc.Pool(); pool != nil {
			_, err := pool.Exec(r.Context(), `
				INSERT INTO tenant_branding (tenant_id, primary_color, logo_url, custom_domain)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (tenant_id) DO UPDATE SET primary_color = $2, logo_url = $3, custom_domain = $4`,
				tenantID, branding.PrimaryColor, branding.LogoURL, branding.CustomDomain)
			if err != nil {
				slog.Error("CIAM branding update failed", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "branding update failed")
				return
			}
		} else {
			tenantBrandingStore[tenantID] = &branding
		}
		slog.Info("CIAM branding updated", "tenant_id", tenantID, "color", branding.PrimaryColor, "domain", branding.CustomDomain)
		writeJSON(w, http.StatusOK, branding)

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
