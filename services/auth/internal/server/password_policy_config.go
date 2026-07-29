package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/pkg/hierarchy"
	"github.com/ggid/ggid/services/auth/internal/service"
	"github.com/google/uuid"
)

// handlePasswordPolicyConfig handles GET/PUT /api/v1/providers/config?key=password_policy
// This extends the provider config API to support password policy via the
// same hierarchical config system.
//
// GET returns the resolved policy (app→tenant→instance→default)
// PUT saves at the specified scope

func (h *Handler) handlePasswordPolicyConfig(w http.ResponseWriter, r *http.Request) {
	pool := h.getPool()
	if pool == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	// SECURITY: Only allow operating on the caller's own tenant (from gateway JWT).
	callerTenant := r.Header.Get("X-Tenant-ID")
	if callerTenant == "" {
		writeError(w, http.StatusForbidden, "tenant context required")
		return
	}
	if _, err := uuid.Parse(callerTenant); err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant context")
		return
	}

	switch r.Method {
	case http.MethodGet:
		// SECURITY: Override tenant_id query param with caller's tenant.
		queryTID := r.URL.Query().Get("tenant_id")
		if queryTID != "" && queryTID != callerTenant {
			scopes := r.Header.Get("X-Scopes")
			if !strings.Contains(scopes, "platform:admin") {
				writeError(w, http.StatusForbidden, "cannot access other tenants' password policy")
				return
			}
		}
		var tenantID *uuid.UUID
		if tid := r.URL.Query().Get("tenant_id"); tid != "" {
			if parsed, err := uuid.Parse(tid); err == nil {
				tenantID = &parsed
			}
		}
		var clientID *string
		if cid := r.URL.Query().Get("client_id"); cid != "" {
			clientID = &cid
		}

		// Default from code
		defaultCfg := service.PasswordPolicyConfig{
			MinLength:    12,
			MaxLength:    64,
			RequireUpper: true,
			RequireLower: true,
			RequireDigit: true,
		}

		cfg := service.GetPasswordPolicyHierarchical(r.Context(), pool, tenantID, clientID, defaultCfg)

		// Check source
		resolved, _ := hierarchy.GetConfig(r.Context(), pool, "password_policy", tenantID, clientID, nil)
		source := "default"
		if resolved != nil {
			source = resolved.Source
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"policy": cfg,
			"source": source,
		})

	case http.MethodPut:
		var body struct {
			Scope    string                       `json:"scope"`
			TenantID string                       `json:"tenant_id"`
			ClientID string                       `json:"client_id"`
			Policy   service.PasswordPolicyConfig `json:"policy"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		scope := hierarchy.ScopeType(body.Scope)
		if scope == "" {
			scope = hierarchy.ScopeInstance
		}

		var tenantID *uuid.UUID
		if body.TenantID != "" {
			// SECURITY: Verify caller's tenant matches requested tenant.
			if body.TenantID != callerTenant {
				scopes := r.Header.Get("X-Scopes")
				if !strings.Contains(scopes, "platform:admin") {
					writeError(w, http.StatusForbidden, "cannot modify other tenants' password policy")
					return
				}
			}
			if parsed, err := uuid.Parse(body.TenantID); err == nil {
				tenantID = &parsed
			}
		}
		var clientID *string
		if body.ClientID != "" {
			clientID = &body.ClientID
		}

		// Validate scope consistency
		if scope == hierarchy.ScopeTenant && tenantID == nil {
			writeError(w, http.StatusBadRequest, "tenant_id required for tenant scope")
			return
		}
		if scope == hierarchy.ScopeApp && (tenantID == nil || clientID == nil) {
			writeError(w, http.StatusBadRequest, "tenant_id and client_id required for app scope")
			return
		}

		if err := service.SetPasswordPolicyHierarchical(r.Context(), pool, scope, tenantID, clientID, body.Policy); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "saved",
			"scope":  scope,
		})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
