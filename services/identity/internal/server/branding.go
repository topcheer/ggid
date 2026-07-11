package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/services/identity/internal/domain"
)

// handleBranding handles GET and PUT /api/v1/tenants/{id}/branding
// Path format: /api/v1/tenants/{tenantID}/branding
func (h *HTTPHandler) handleBranding(w http.ResponseWriter, r *http.Request) {
	// Extract tenant ID from path
	// /api/v1/tenants/{id}/branding → parts: ["api", "v1", "tenants", "{id}", "branding"]
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 5 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "tenants" || parts[4] != "branding" {
		writeError(w, http.StatusBadRequest, "invalid branding path")
		return
	}
	tenantID := parts[3]

	switch r.Method {
	case http.MethodGet:
		branding, err := h.brandingStore.GetBranding(r.Context(), tenantID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(branding)

	case http.MethodPut:
		var req domain.TenantBranding
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		branding, err := h.brandingStore.UpdateBranding(r.Context(), tenantID, &req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(branding)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
