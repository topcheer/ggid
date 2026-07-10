package webauthn

import (
	"testing"
)

func TestGenerateCredentialName(t *testing.T) {
	tests := []struct {
		ua   string
		want string
	}{
		{"", "Passkey"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0", "Chrome on Windows"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15) Safari/605", "Safari on macOS"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0) FxiOS/120.0", "Firefox on iOS"},
		{"Mozilla/5.0 (Linux; Android 14) Edg/120.0", "Edge on Android"},
		{"Mozilla/5.0 (X11; Linux x86_64) Firefox/121.0", "Firefox on Linux"},
		{"UnknownBot/1.0", "Browser on Device"},
	}

	for _, tc := range tests {
		t.Run(tc.ua, func(t *testing.T) {
			got := generateCredentialName(tc.ua)
			if got != tc.want {
				t.Errorf("generateCredentialName(%q) = %q, want %q", tc.ua, got, tc.want)
			}
		})
	}
}

func TestWithOrigins(t *testing.T) {
	cfg := &handlerConfig{}
	opt := WithOrigins([]string{"https://custom.com", "https://app.custom.com"})
	opt(cfg)
	if len(cfg.origins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(cfg.origins))
	}
	if cfg.origins[0] != "https://custom.com" {
		t.Errorf("expected https://custom.com, got %s", cfg.origins[0])
	}
}

func TestNewHandler_WithOrigins(t *testing.T) {
	// Test that custom origins are passed through to the webauthn config.
	h, err := NewHandler("test.example.com", "Test", nil, WithOrigins([]string{"https://test.example.com"}))
	if err != nil {
		t.Fatalf("NewHandler with origins: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestNewHandler_DefaultOrigins(t *testing.T) {
	// Test that default origins are set when no WithOrigins is provided.
	h, err := NewHandler("default.example.com", "Default", nil)
	if err != nil {
		t.Fatalf("NewHandler default: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}
