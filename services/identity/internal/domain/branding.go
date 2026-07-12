package domain

import "time"

// TenantBranding represents per-tenant branding configuration.
// This allows each tenant to customize logos, colors, domains, and email templates.
type TenantBranding struct {
	TenantID       string    `json:"tenant_id"`
	LogoURL        string    `json:"logo_url"`
	PrimaryColor   string    `json:"primary_color"`
	SecondaryColor string    `json:"secondary_color"`
	CustomDomain   string    `json:"custom_domain"`
	EmailTemplate  string    `json:"email_template"`
	UpdatedAt      time.Time `json:"updated_at"`
}
