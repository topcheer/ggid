package server

import (
	"encoding/json"
	"net/http"
	"os"
)

// getMFAConfigForTenant returns the MFA config for a given tenant.
// Currently uses global config (all tenants share). Future: per-tenant override.
func getMFAConfigForTenant(tenantID string) *MFAConfig {
	return globalMFAConfig
}

type MFAConfig struct {
	EnforcedRoles    []string         `json:"enforced_roles"`
	EnforcedForAll   bool             `json:"enforced_for_all"`
	AllowedFactors   []string         `json:"allowed_factors"`
	TOTPSettings     TOTPSettings     `json:"totp_settings"`
	WebAuthnSettings WebAuthnSettings `json:"webauthn_settings"`
	PushSettings     PushSettings     `json:"push_settings"`
	BackupCodes      BackupCodeConfig `json:"backup_codes"`
}

type TOTPSettings struct {
	Issuer string `json:"issuer"`
	Digits int    `json:"digits"`
	Period int    `json:"period"`
}

type WebAuthnSettings struct {
	RPID    string `json:"rp_id"`
	Origin  string `json:"origin"`
	Timeout int    `json:"timeout_ms"`
}

type PushSettings struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
}

type BackupCodeConfig struct {
	Enabled              bool `json:"enabled"`
	Count                int  `json:"count"`
	RegenerationCooldown int  `json:"regeneration_cooldown_hours"`
}

var globalMFAConfig = &MFAConfig{
	EnforcedRoles:  []string{"admin", "security_admin", "compliance_officer"},
	EnforcedForAll: false,
	AllowedFactors: []string{"totp", "webauthn", "push", "sms"},
	TOTPSettings: TOTPSettings{
		Issuer: "GGID",
		Digits: 6,
		Period: 30,
	},
	WebAuthnSettings: WebAuthnSettings{
		RPID:    os.Getenv("WEBAUTHN_RP_ID"),
		Origin:  "https://auth.ggid.example",
		Timeout: 60000,
	},
	PushSettings: PushSettings{
		Provider: "fcm",
		APIKey:   "",
	},
	BackupCodes: BackupCodeConfig{
		Enabled:              true,
		Count:                10,
		RegenerationCooldown: 24,
	},
}

func (h *Handler) handleMFAConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// SECURITY: Return per-tenant config using tenant_id from context/header
		tenantKey := r.Header.Get("X-Tenant-ID")
		if tenantKey == "" {
			writeError(w, http.StatusForbidden, "tenant context required")
			return
		}
		cfg := getMFAConfigForTenant(tenantKey)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	case http.MethodPut:
		// SECURITY: Only platform:admin can change MFA enforcement globally.
		// Not tenant:admin — this is a global setting.
		if !hasPlatformAdminScope(r) {
			writeError(w, http.StatusForbidden, "platform:admin scope required to modify MFA config")
			return
		}
		var cfg MFAConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		globalMFAConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
