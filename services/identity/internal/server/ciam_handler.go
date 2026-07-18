package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
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

	// Create tenant (in production: DB insert + identity service call).
	tenantID := uuid.New()
	adminUserID := uuid.New()

	// Store branding if provided.
	if req.Branding != nil {
		tenantBrandingStore[tenantID] = &TenantBranding{
			PrimaryColor: req.Branding.PrimaryColor,
			LogoURL:      req.Branding.LogoURL,
			CustomDomain: req.CustomDomain,
		}
	}

	slog.Info("CIAM B2B self-register", "org", req.OrgName, "tenant_id", tenantID, "admin", req.Admin.Email)

	writeJSON(w, http.StatusCreated, SelfRegisterResponse{
		TenantID:    tenantID.String(),
		OrgName:     req.OrgName,
		AdminUserID: adminUserID.String(),
		LoginURL:    "/login",
		Status:      "active",
	})
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
		tenantBrandingStore[tenantID] = &branding
		slog.Info("CIAM branding updated", "tenant_id", tenantID, "color", branding.PrimaryColor, "domain", branding.CustomDomain)
		writeJSON(w, http.StatusOK, branding)

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
