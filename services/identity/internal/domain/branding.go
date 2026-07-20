package domain

import "time"

// TenantBranding represents per-tenant branding configuration.
// This allows each tenant to customize logos, colors, fonts, and email templates.
type TenantBranding struct {
	TenantID       string    `json:"tenant_id"`
	LogoURL        string    `json:"logo_url"`
	FaviconURL     string    `json:"favicon_url"`
	PrimaryColor   string    `json:"primary_color"`
	AccentColor    string    `json:"accent_color"`
	SecondaryColor string    `json:"secondary_color"`
	FontFamily     string    `json:"font_family"`
	BorderRadius   int       `json:"border_radius"`
	DefaultMode    string    `json:"default_mode"` // "light" or "dark"
	CustomDomain   string    `json:"custom_domain"`
	EmailTemplate  string    `json:"email_template"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// DefaultBranding returns the default branding configuration.
func DefaultBranding(tenantID string) *TenantBranding {
	return &TenantBranding{
		TenantID:       tenantID,
		PrimaryColor:   "#2563eb",
		AccentColor:    "#1e40af",
		SecondaryColor: "#1e40af",
		FontFamily:     "Inter",
		BorderRadius:   8,
		DefaultMode:    "light",
		EmailTemplate:  "default",
	}
}
