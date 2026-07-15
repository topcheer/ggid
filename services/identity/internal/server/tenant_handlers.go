package server

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// TenantInfo represents a minimal tenant record for resolution.
type TenantInfo struct {
	ID   string `json:"tenant_id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// handleTenantResolve resolves a tenant by slug to its ID.
// GET /api/v1/tenants/resolve?slug=xxx
// This is a public endpoint (no JWT required) used by the login page.
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

	// Query the tenants table directly (shared PG database).
	row := h.svc.Pool().QueryRow(r.Context(),
		`SELECT id::text, name, slug FROM tenants WHERE slug = $1 AND status = 'active'`)

	var t TenantInfo
	if err := row.Scan(&t.ID, &t.Name, &t.Slug); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// handleSystemInitialized checks whether the system has been initialized.
// GET /api/v1/system/initialized
// Returns {initialized, tenant_count, user_count} — public endpoint for onboarding flow.
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

// Ensure json import is used.
var _ = json.Marshal
var _ = uuid.Nil
