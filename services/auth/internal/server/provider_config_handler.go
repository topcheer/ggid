package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"os"

	"github.com/ggid/ggid/pkg/hierarchy"
	"github.com/google/uuid"
)

// validateTenantAccess checks that the query param tenant_id matches the
// authenticated X-Tenant-ID header (set by gateway from JWT). Platform admins
// (platform:admin scope) can access any tenant. Returns false if access denied.
func validateTenantAccess(w http.ResponseWriter, r *http.Request, requestedTenantID *uuid.UUID) bool {
	if requestedTenantID == nil {
		// Instance scope — require admin privileges
		if !isAdminScope(r) {
			writeError(w, http.StatusForbidden, "instance-level configuration requires admin privileges")
			return false
		}
		return true
	}
	headerTID := r.Header.Get("X-Tenant-ID")
	if headerTID == "" {
		// No header = not authenticated by gateway — fail closed
		writeError(w, http.StatusForbidden, "missing tenant context")
		return false
	}
	headerUUID, err := uuid.Parse(headerTID)
	if err != nil {
		writeError(w, http.StatusForbidden, "invalid tenant context")
		return false
	}
	if *requestedTenantID == headerUUID {
		return true // match
	}
	// Mismatch — deny unless platform:admin
	if isAdminScope(r) {
		return true
	}
	writeError(w, http.StatusForbidden, "cannot access another tenant's configuration")
	return false
}

// isAdminScope checks if the request has platform:admin or tenant:admin scope.
// The gateway sets X-User-Scopes (plural) from the verified JWT scope claim.
func isAdminScope(r *http.Request) bool {
	scopes := r.Header.Get("X-User-Scopes")
	if scopes == "" {
		scopes = r.Header.Get("X-User-Role") // backward compat fallback
	}
	for _, s := range strings.Split(scopes, " ") {
		switch strings.TrimSpace(s) {
		case "platform:admin", "tenant:admin":
			return true
		}
	}
	return false
}

// handleProviderConfig handles GET/PUT /api/v1/providers/config
//
// GET /api/v1/providers/config?key=sms_provider&scope=instance
// GET /api/v1/providers/config?key=sms_provider&scope=tenant&tenant_id=<uuid>
// GET /api/v1/providers/config?key=sms_provider&scope=app&tenant_id=<uuid>&client_id=<id>
//
// PUT /api/v1/providers/config  (body: config JSON + query params for scope)
//
// Response includes resolved config with source indicator.

func (h *Handler) handleProviderConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleProviderConfigGet(w, r)
	case http.MethodPut:
		h.handleProviderConfigSet(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleProviderConfigGet(w http.ResponseWriter, r *http.Request) {
	pool := h.getPool()
	if pool == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		key = hierarchy.KeySMSProvider // default
	}

	// Parse scope parameters
	scope := r.URL.Query().Get("scope") // instance, tenant, app (default: hierarchical resolve)

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

	// SECURITY: prevent cross-tenant config access (BOLA fix).
	if !validateTenantAccess(w, r, tenantID) {
		return
	}

	if scope == "instance" || scope == "tenant" || scope == "app" {
		// Direct scope query — return the exact config at this scope level
		var st hierarchy.ScopeType
		switch scope {
		case "instance":
			st = hierarchy.ScopeInstance
		case "tenant":
			st = hierarchy.ScopeTenant
		case "app":
			st = hierarchy.ScopeApp
		}

		// Use hierarchy queryConfig directly
		cfg, err := hierarchy.GetConfig(r.Context(), pool, key, tenantID, clientID, nil)
		if err != nil {
			// No config found — return empty with source=default
			writeJSON(w, http.StatusOK, map[string]any{
				"key":           key,
				"configured":    false,
				"source":        "none",
				"provider_type": "",
				"config":        map[string]any{},
				"enabled":       false,
			})
			return
		}

		// Mask secrets in response
		configMap := maskSecrets(cfg.Config)

		writeJSON(w, http.StatusOK, map[string]any{
			"key":           key,
			"configured":    true,
			"source":        cfg.Source,
			"scope":         st,
			"provider_type": cfg.ProviderType,
			"config":        configMap,
			"enabled":       cfg.Enabled,
		})
		return
	}

	// Default: hierarchical resolve (app → tenant → instance → env fallback)
	cfg, err := hierarchy.GetConfig(r.Context(), pool, key, tenantID, clientID, nil)
	if err != nil {
		// Check env vars as fallback
		envConfig := getEnvProviderConfig(key)
		if envConfig != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"key":           key,
				"configured":    true,
				"source":        "env",
				"provider_type": envConfig["provider_type"],
				"config":        envConfig["config"],
				"enabled":       true,
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"key":        key,
			"configured": false,
			"source":     "none",
			"config":     map[string]any{},
		})
		return
	}

	configMap := maskSecrets(cfg.Config)

	writeJSON(w, http.StatusOK, map[string]any{
		"key":           key,
		"configured":    true,
		"source":        cfg.Source,
		"provider_type": cfg.ProviderType,
		"config":        configMap,
		"enabled":       cfg.Enabled,
	})
}

func (h *Handler) handleProviderConfigSet(w http.ResponseWriter, r *http.Request) {
	// Requires auth — admin only
	pool := h.getPool()
	if pool == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		key = hierarchy.KeySMSProvider
	}

	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "instance"
	}

	var body struct {
		ProviderType string          `json:"provider_type"`
		Config       json.RawMessage `json:"config"`
		Enabled      *bool           `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	// Parse scope-specific IDs
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

	// SECURITY: prevent cross-tenant config modification (BOLA fix).
	if !validateTenantAccess(w, r, tenantID) {
		return
	}

	// Validate scope consistency
	var st hierarchy.ScopeType
	switch scope {
	case "instance":
		st = hierarchy.ScopeInstance
		tenantID = nil
		clientID = nil
	case "tenant":
		st = hierarchy.ScopeTenant
		clientID = nil
		if tenantID == nil {
			writeError(w, http.StatusBadRequest, "tenant_id required for tenant scope")
			return
		}
	case "app":
		st = hierarchy.ScopeApp
		if tenantID == nil || clientID == nil {
			writeError(w, http.StatusBadRequest, "tenant_id and client_id required for app scope")
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "invalid scope: must be instance, tenant, or app")
		return
	}

	cfg := &hierarchy.ProviderConfig{
		ConfigKey:    key,
		ScopeType:    st,
		TenantID:     tenantID,
		ClientID:     clientID,
		ProviderType: body.ProviderType,
		Config:       body.Config,
		Enabled:      enabled,
	}

	if err := hierarchy.SetConfig(r.Context(), pool, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save provider config: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "saved",
		"key":           key,
		"scope":         st,
		"provider_type": body.ProviderType,
		"enabled":       enabled,
	})
}

// maskSecrets redacts sensitive fields (auth_token, password, secret_key, api_key)
// from the config before returning to the API caller.
func maskSecrets(rawConfig json.RawMessage) map[string]any {
	var config map[string]any
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return map[string]any{"raw": string(rawConfig)}
	}

	sensitiveFields := []string{"auth_token", "password", "secret_key", "api_key", "client_secret"}
	for _, field := range sensitiveFields {
		if val, exists := config[field]; exists && val != nil && val != "" {
			config[field] = "****"
		}
	}
	return config
}

// getEnvProviderConfig reads provider config from environment variables.
func getEnvProviderConfig(key string) map[string]any {
	switch key {
	case hierarchy.KeySMSProvider:
		provider := os.Getenv("GGID_SMS_PROVIDER")
		if provider == "" || provider == "log" {
			return nil
		}
		return map[string]any{
			"provider_type": provider,
			"config": map[string]any{
				"provider":    provider,
				"account_sid": os.Getenv("TWILIO_ACCOUNT_SID"),
				"from_number": os.Getenv("TWILIO_FROM_NUMBER"),
			},
		}
	case hierarchy.KeyEmailProvider:
		host := os.Getenv("SMTP_HOST")
		if host == "" {
			return nil
		}
		return map[string]any{
			"provider_type": "smtp",
			"config": map[string]any{
				"provider": "smtp",
				"host":     host,
				"port":     os.Getenv("SMTP_PORT"),
				"username": os.Getenv("SMTP_USER"),
				"from":     os.Getenv("SMTP_FROM"),
			},
		}
	}
	return nil
}
