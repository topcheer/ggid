package server

import (
	"encoding/json"
	"net/http"
)

type EmailTemplateConfig struct {
	Name      string   `json:"name"`
	Subject   string   `json:"subject"`
	BodyHTML  string   `json:"body_html"`
	Variables []string `json:"variables"`
	Language  string   `json:"language"`
	Enabled   bool     `json:"enabled"`
}

type EmailTemplateConfigResult struct {
	Templates []EmailTemplateConfig `json:"templates"`
}

func (h *Handler) handleEmailTemplateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := EmailTemplateConfigResult{
		Templates: []EmailTemplateConfig{
			{Name: "welcome", Subject: "Welcome to {{app_name}}", BodyHTML: "<h1>Welcome {{user_name}}!</h1>", Variables: []string{"app_name", "user_name", "login_url"}, Language: "en", Enabled: true},
			{Name: "password_reset", Subject: "Reset your password", BodyHTML: "<p>Click <a href=\"{{reset_url}}\">here</a>", Variables: []string{"reset_url", "expiry_hours"}, Language: "en", Enabled: true},
			{Name: "mfa_setup", Subject: "Set up MFA", BodyHTML: "<p>Your TOTP secret: {{totp_secret}}", Variables: []string{"totp_secret", "qr_url"}, Language: "en", Enabled: true},
			{Name: "account_locked", Subject: "Account locked", BodyHTML: "<p>Your account was locked after {{failed_attempts}} attempts.", Variables: []string{"failed_attempts", "unlock_url"}, Language: "en", Enabled: true},
			{Name: "access_granted", Subject: "New access granted", BodyHTML: "<p>You've been granted {{role}} on {{resource}}", Variables: []string{"role", "resource", "granted_by"}, Language: "en", Enabled: true},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
