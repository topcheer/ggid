package server

import (
	"encoding/json"
	"net/http"
)

type BrandingConfig struct {
	LogoURL             string `json:"logo_url"`
	PrimaryColor        string `json:"primary_color"`
	SecondaryColor      string `json:"secondary_color"`
	CustomCSS           string `json:"custom_css"`
	LoginPageConfig     struct {
		Title         string `json:"title"`
		Subtitle      string `json:"subtitle"`
		BackgroundURL string `json:"background_url"`
		ShowSignup    bool   `json:"show_signup"`
	} `json:"login_page_config"`
	EmailTemplateConfig struct {
		HeaderLogoURL string `json:"header_logo_url"`
		FooterText    string `json:"footer_text"`
		PrimaryColor  string `json:"primary_color"`
	} `json:"email_template_config"`
	DarkMode      bool   `json:"dark_mode"`
	CustomDomain  string `json:"custom_domain"`
}

func (h *HTTPHandler) handleBrandingConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := BrandingConfig{
		LogoURL:        "https://ggid.example.com/assets/logo.png",
		PrimaryColor:   "#2563EB",
		SecondaryColor: "#1E40AF",
		CustomCSS:      ".btn-primary { border-radius: 8px; }",
		DarkMode:       true,
		CustomDomain:   "auth.example.com",
	}
	result.LoginPageConfig.Title = "Sign in to GGID"
	result.LoginPageConfig.Subtitle = "Enterprise Identity Platform"
	result.LoginPageConfig.BackgroundURL = "https://ggid.example.com/assets/bg.jpg"
	result.LoginPageConfig.ShowSignup = false
	result.EmailTemplateConfig.HeaderLogoURL = "https://ggid.example.com/assets/email-logo.png"
	result.EmailTemplateConfig.FooterText = "© 2025 GGID. All rights reserved."
	result.EmailTemplateConfig.PrimaryColor = "#2563EB"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
