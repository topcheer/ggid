package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RegistrationConfig controls per-tenant self-registration behavior.
type RegistrationConfig struct {
	TenantID      string   `json:"tenant_id"`
	Enabled       bool     `json:"enabled"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	DefaultRole   string   `json:"default_role,omitempty"`
	RequireVerification bool `json:"require_email_verification"`
}

// registrationConfigRepo manages per-tenant registration settings in PG.
type registrationConfigRepo struct {
	pool *pgxpool.Pool
}

// NewRegistrationConfigRepo creates a registration config repo from a pool.
func NewRegistrationConfigRepo(pool *pgxpool.Pool) *registrationConfigRepo {
	return &registrationConfigRepo{pool: pool}
}

func (r *registrationConfigRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tenant_registration_config (
			tenant_id TEXT PRIMARY KEY,
			enabled BOOLEAN NOT NULL DEFAULT false,
			allowed_domains TEXT[] NOT NULL DEFAULT '{}',
			default_role TEXT NOT NULL DEFAULT 'viewer',
			require_email_verification BOOLEAN NOT NULL DEFAULT true,
			updated_at TIMESTAMPTZ DEFAULT now()
		);
	`)
	return err
}

func (r *registrationConfigRepo) Get(ctx context.Context, tenantID string) (*RegistrationConfig, error) {
	if r.pool == nil {
		return nil, nil
	}
	cfg := &RegistrationConfig{TenantID: tenantID}
	err := r.pool.QueryRow(ctx,
		`SELECT enabled, allowed_domains, default_role, require_email_verification
		 FROM tenant_registration_config WHERE tenant_id = $1`,
		tenantID).Scan(&cfg.Enabled, &cfg.AllowedDomains, &cfg.DefaultRole, &cfg.RequireVerification)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (r *registrationConfigRepo) Upsert(ctx context.Context, cfg *RegistrationConfig) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tenant_registration_config (tenant_id, enabled, allowed_domains, default_role, require_email_verification, updated_at)
		 VALUES ($1, $2, $3, $4, $5, now())
		 ON CONFLICT (tenant_id) DO UPDATE SET
		   enabled = EXCLUDED.enabled,
		   allowed_domains = EXCLUDED.allowed_domains,
		   default_role = EXCLUDED.default_role,
		   require_email_verification = EXCLUDED.require_email_verification,
		   updated_at = now()`,
		cfg.TenantID, cfg.Enabled, cfg.AllowedDomains, cfg.DefaultRole, cfg.RequireVerification)
	return err
}

// IsDomainAllowed checks if the email domain is in the allowed list.
// If allowed_domains is empty, all domains are permitted.
func (c *RegistrationConfig) IsDomainAllowed(email string) bool {
	if len(c.AllowedDomains) == 0 {
		return true
	}
	// Extract domain from email
	at := -1
	for i := len(email) - 1; i >= 0; i-- {
		if email[i] == '@' {
			at = i
			break
		}
	}
	if at == -1 || at == len(email)-1 {
		return false
	}
	domain := email[at+1:]
	for _, allowed := range c.AllowedDomains {
		if domain == allowed {
			return true
		}
	}
	return false
}

// SetRegistrationConfigRepo wires the registration config repository.
func (h *Handler) SetRegistrationConfigRepo(repo *registrationConfigRepo) {
	h.registrationConfigRepo = repo
}

// GET  /api/v1/auth/registration/config?tenant_id=X
// PUT  /api/v1/auth/registration/config
// POST /api/v1/auth/registration/config  (alias for PUT)
func (h *Handler) handleRegistrationConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getRegistrationConfig(w, r)
	case http.MethodPut, http.MethodPost:
		h.updateRegistrationConfig(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) getRegistrationConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		// Try from JWT context
		claims, err := h.parseTokenFromHeader(r)
		if err == nil {
			if tid, ok := claims["tenant_id"].(string); ok {
				tenantID = tid
			}
		}
	}
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	if h.registrationConfigRepo == nil {
		// Default: registration disabled
		writeJSON(w, http.StatusOK, &RegistrationConfig{
			TenantID: tenantID, Enabled: false, DefaultRole: "viewer",
			RequireVerification: true,
		})
		return
	}

	cfg, err := h.registrationConfigRepo.Get(r.Context(), tenantID)
	if err != nil || cfg == nil {
		// Return default config (registration disabled for security)
		writeJSON(w, http.StatusOK, &RegistrationConfig{
			TenantID: tenantID, Enabled: false, DefaultRole: "viewer",
			RequireVerification: true,
		})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *Handler) updateRegistrationConfig(w http.ResponseWriter, r *http.Request) {
	var req RegistrationConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.TenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}
	if req.DefaultRole == "" {
		req.DefaultRole = "viewer"
	}

	if h.registrationConfigRepo != nil {
		if err := h.registrationConfigRepo.Upsert(r.Context(), &req); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save registration config")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "updated",
		"config":   &req,
	})
}
