package hierarchy

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestGetConfig_Fallback(t *testing.T) {
	// Unit test would need a mock pool. For now this is a compile check.
	// Integration tests run against real DB in CI.
	_ = context.Background()
	_ = uuid.New()
	_ = ScopeApp
	_ = KeySMSProvider
}

func TestProviderConfig_Defaults(t *testing.T) {
	cfg := ProviderConfig{
		ConfigKey:    KeySMSProvider,
		ProviderType: "twilio",
		Enabled:      true,
	}
	if cfg.ConfigKey != "sms_provider" {
		t.Errorf("expected sms_provider, got %s", cfg.ConfigKey)
	}
}
