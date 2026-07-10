package service

import "time"

// IdPConfig represents a SAML IdP federation configuration.
// When a user logs in through this IdP, their identity is mapped to a GGID user.
type IdPConfig struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Protocol     string    `json:"protocol"`     // "saml" or "oidc"
	EntityID     string    `json:"entity_id"`    // SAML EntityID
	SSOURL       string    `json:"sso_url"`      // IdP SSO endpoint
	CertPEM      string    `json:"cert_pem"`     // IdP signing certificate
	NameIDFormat string    `json:"name_id_format"` // e.g. "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
	// Attribute mapping: IdP attribute name → GGID field
	AttrMap      map[string]string `json:"attr_map"`
	// Auto-provision: create user if not exists
	AutoProvision bool      `json:"auto_provision"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
}
