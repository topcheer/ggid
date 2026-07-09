package config

import (
	"testing"
)

func TestConfig_Default(t *testing.T) {
	cfg := Default()
	if cfg.Addr != ":8080" {
		t.Errorf("expected addr :8080, got %s", cfg.Addr)
	}
	if cfg.JWTIssuer != "ggid-auth" {
		t.Errorf("expected issuer ggid-auth, got %s", cfg.JWTIssuer)
	}
	if cfg.JWTAudience != "ggid" {
		t.Errorf("expected audience ggid, got %s", cfg.JWTAudience)
	}
	if cfg.PublicKeyPath != "configs/rsa_public.pem" {
		t.Errorf("unexpected public key path: %s", cfg.PublicKeyPath)
	}
}

func TestConfig_DefaultHasAllServiceRoutes(t *testing.T) {
	cfg := Default()
	expectedRoutes := []string{
		"/api/v1/auth",
		"/api/v1/users",
		"/api/v1/roles",
		"/api/v1/policies",
		"/api/v1/orgs",
		"/api/v1/audit",
		"/oauth",
		"/saml",
	}
	for _, route := range expectedRoutes {
		if _, ok := cfg.Routes[route]; !ok {
			t.Errorf("expected route %s in default config", route)
		}
	}
}

func TestConfig_LoadFromEnv(t *testing.T) {
	t.Setenv("GATEWAY_ADDR", ":9090")
	t.Setenv("JWT_PUBLIC_KEY_PATH", "/custom/path.pem")
	t.Setenv("AUTH_SERVICE_URL", "http://auth:9001")
	t.Setenv("IDENTITY_SERVICE_URL", "http://identity:9002")

	cfg := LoadFromEnv(Default())
	if cfg.Addr != ":9090" {
		t.Errorf("expected :9090, got %s", cfg.Addr)
	}
	if cfg.PublicKeyPath != "/custom/path.pem" {
		t.Errorf("expected /custom/path.pem, got %s", cfg.PublicKeyPath)
	}
	if cfg.Routes["/api/v1/auth"] != "http://auth:9001" {
		t.Errorf("unexpected auth route: %s", cfg.Routes["/api/v1/auth"])
	}
	if cfg.Routes["/api/v1/users"] != "http://identity:9002" {
		t.Errorf("unexpected identity route: %s", cfg.Routes["/api/v1/users"])
	}
}

func TestConfig_LoadFromEnv_NoOverrides(t *testing.T) {
	cfg := LoadFromEnv(Default())
	if cfg.Addr != ":8080" {
		t.Errorf("expected default :8080, got %s", cfg.Addr)
	}
}
