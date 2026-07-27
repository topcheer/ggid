package webhook

import (
	"os"
	"testing"
)

func TestValidateWebhookURL_ValidHTTPS(t *testing.T) {
	err := validateWebhookURL("https://hooks.example.com/webhook")
	if err != nil {
		t.Errorf("expected valid HTTPS URL to pass, got: %v", err)
	}
}

func TestValidateWebhookURL_ValidHTTP(t *testing.T) {
	err := validateWebhookURL("http://hooks.example.com/webhook")
	if err != nil {
		t.Errorf("expected valid HTTP URL to pass, got: %v", err)
	}
}

func TestValidateWebhookURL_InvalidScheme(t *testing.T) {
	tests := []string{
		"ftp://example.com",
		"file:///etc/passwd",
		"gopher://example.com",
		"javascript:alert(1)",
	}
	for _, u := range tests {
		t.Run(u, func(t *testing.T) {
			err := validateWebhookURL(u)
			if err == nil {
				t.Errorf("expected error for scheme %s", u)
			}
		})
	}
}

func TestValidateWebhookURL_MissingHost(t *testing.T) {
	err := validateWebhookURL("https:///webhook")
	if err == nil {
		t.Error("expected error for missing host")
	}
}

func TestValidateWebhookURL_EmptyString(t *testing.T) {
	err := validateWebhookURL("")
	// url.Parse succeeds on empty string, but host check should catch it
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestValidateWebhookURL_LocalhostBlocked(t *testing.T) {
	// Ensure dev mode is off
	oldVal := os.Getenv("GGID_DEV_MODE")
	os.Unsetenv("GGID_DEV_MODE")
	defer os.Setenv("GGID_DEV_MODE", oldVal)

	tests := []string{
		"http://localhost:8080/webhook",
		"http://127.0.0.1:8080/webhook",
		"http://[::1]:8080/webhook",
		"http://0.0.0.0:8080/webhook",
	}
	for _, u := range tests {
		t.Run(u, func(t *testing.T) {
			err := validateWebhookURL(u)
			if err == nil {
				t.Errorf("expected error for localhost URL %s", u)
			}
		})
	}
}

func TestValidateWebhookURL_DevModeBypass(t *testing.T) {
	os.Setenv("GGID_DEV_MODE", "true")
	defer os.Unsetenv("GGID_DEV_MODE")

	// In dev mode, localhost should be allowed
	err := validateWebhookURL("http://localhost:8080/webhook")
	if err != nil {
		t.Errorf("expected dev mode to allow localhost, got: %v", err)
	}
}

func TestValidateWebhookURL_PrivateIPFormat(t *testing.T) {
	// IP literals in URL should be caught by the localhost check or DNS resolution
	// Note: 192.168.x.x won't resolve via DNS but the hostname check catches localhost patterns
	// The real protection is DNS resolution check for hostnames that resolve to private IPs
	err := validateWebhookURL("http://localhost/webhook")
	if err == nil {
		t.Error("expected localhost to be blocked")
	}
}

func TestValidateWebhookURL_NoPort(t *testing.T) {
	err := validateWebhookURL("https://hooks.example.com/webhook")
	if err != nil {
		t.Errorf("expected valid URL without port, got: %v", err)
	}
}

func TestValidateWebhookURL_WithPath(t *testing.T) {
	err := validateWebhookURL("https://hooks.example.com/api/v1/webhooks/abc123")
	if err != nil {
		t.Errorf("expected valid URL with path, got: %v", err)
	}
}

func TestValidateWebhookURL_WithQueryParams(t *testing.T) {
	err := validateWebhookURL("https://hooks.example.com/webhook?token=abc123&type=event")
	if err != nil {
		t.Errorf("expected valid URL with query params, got: %v", err)
	}
}
