package server

import (
	"testing"
)

func TestEmailRepo_NilPool(t *testing.T) {
	repo := newEmailRepo(nil)
	if repo.config.Provider != "smtp" { t.Error("default provider should be smtp") }
	if repo.config.Port != 587 { t.Error("default port should be 587") }
}

func TestRenderTemplate_EmailVerification(t *testing.T) {
	repo := newEmailRepo(nil)
	body := repo.renderTemplate(&EmailMessage{
		Template: "email_verification",
		Data: map[string]string{"base_url": "https://app.ggid.io", "token": "abc123"},
	})
	if !contains(body, "verify-email") { t.Error("should contain verify link") }
	if !contains(body, "abc123") { t.Error("should contain token") }
}

func TestRenderTemplate_PasswordReset(t *testing.T) {
	repo := newEmailRepo(nil)
	body := repo.renderTemplate(&EmailMessage{
		Template: "password_reset",
		Data: map[string]string{"base_url": "https://app.ggid.io", "token": "xyz789"},
	})
	if !contains(body, "reset-password") { t.Error("should contain reset link") }
	if !contains(body, "xyz789") { t.Error("should contain token") }
}

func TestRenderTemplate_BreachNotification(t *testing.T) {
	repo := newEmailRepo(nil)
	body := repo.renderTemplate(&EmailMessage{
		Template: "breach_notification",
		Data: map[string]string{"message": "Suspicious login detected"},
	})
	if !contains(body, "Suspicious login") { t.Error("should contain breach message") }
}

func TestRenderTemplate_Default(t *testing.T) {
	repo := newEmailRepo(nil)
	body := repo.renderTemplate(&EmailMessage{
		Data: map[string]string{"body": "Hello world"},
	})
	if !contains(body, "Hello world") { t.Error("should contain body") }
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr { return true }
	}
	return false
}
