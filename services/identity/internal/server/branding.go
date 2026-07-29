package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

// handleBranding handles GET and PUT /api/v1/tenants/{id}/branding
// Path format: /api/v1/tenants/{tenantID}/branding
func (h *HTTPHandler) handleBranding(w http.ResponseWriter, r *http.Request) {
	// Extract tenant ID from path
	// /api/v1/tenants/{id}/branding → parts: ["api", "v1", "tenants", "{id}", "branding"]
	// /api/v1/tenants/{id}/idp-config → dispatch to IdP config handler
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) >= 5 && parts[4] == "idp-config" {
		h.handleIdPConfig(w, r)
		return
	}
	if len(parts) >= 5 && parts[4] == "scim-config" {
		h.handleSCIMConfig(w, r)
		return
	}
	if len(parts) >= 5 && parts[4] == "saml-config" {
		h.handleSystemConfig(w, r)
		return
	}
	if len(parts) < 5 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "tenants" || parts[4] != "branding" {
		writeJSONError(w, http.StatusBadRequest, "invalid branding path")
		return
	}
	tenantID := parts[3]

	// SECURITY: verify the requesting user belongs to this tenant.
	// SECURITY: Fail-closed tenant check. Check context first, then X-Tenant-ID header.
	tc, tcErr := ggidtenant.FromContext(r.Context())
	if tcErr != nil || tc.TenantID == uuid.Nil {
		// Fallback: use X-Tenant-ID header (set by gateway from JWT).
		headerTID := r.Header.Get("X-Tenant-ID")
		if headerTID == "" {
			writeJSONError(w, http.StatusForbidden, "tenant context required")
			return
		}
		if parsed, err := uuid.Parse(headerTID); err == nil {
			tc = &ggidtenant.Context{TenantID: parsed}
		} else {
			writeJSONError(w, http.StatusBadRequest, "invalid tenant context")
			return
		}
	}
	if tc.TenantID.String() != tenantID {
		// Check if user has platform:admin scope
		scopes := strings.Split(r.Header.Get("X-User-Scopes"), ",")
		isPlatformAdmin := false
		for _, sc := range scopes {
			if strings.TrimSpace(sc) == "platform:admin" {
				isPlatformAdmin = true
				break
			}
		}
		if !isPlatformAdmin {
			writeJSONError(w, http.StatusForbidden, "cannot access another tenant's branding")
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		branding, err := h.brandingStore.GetBranding(r.Context(), tenantID)
		if err != nil {
			slog.Error("branding get error", "err", err)
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(branding)

	case http.MethodPut:
		var req domain.TenantBranding
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		branding, err := h.brandingStore.UpdateBranding(r.Context(), tenantID, &req)
		if err != nil {
			slog.Error("branding update error", "err", err)
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(branding)

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleTenantImport handles tenant import sub-path (stub — not yet implemented).
func (h *HTTPHandler) handleTenantImport(w http.ResponseWriter, r *http.Request) {
	writeJSONError(w, http.StatusNotFound, "tenant import not implemented")
}

// handleTenantSessionPolicy handles tenant session-policy sub-path (stub — not yet implemented).
func (h *HTTPHandler) handleTenantSessionPolicy(w http.ResponseWriter, r *http.Request) {
	writeJSONError(w, http.StatusNotFound, "tenant session policy not implemented")
}
