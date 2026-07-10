package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadFromEnv_ServiceURLs(t *testing.T) {
	os.Setenv("AUTH_SERVICE_URL", "http://auth-env:9001")
	os.Setenv("IDENTITY_SERVICE_URL", "http://identity-env:8080")
	os.Setenv("AUDIT_SERVICE_URL", "http://audit-env:8072")
	os.Setenv("GATEWAY_ADDR", ":9999")
	os.Setenv("GATEWAY_JWKS_URL", "http://jwks-env:8080/.well-known/jwks.json")
	os.Setenv("GATEWAY_JWT_ISSUER", "env-issuer")
	os.Setenv("JWT_PUBLIC_KEY_PATH", "/tmp/env-test.pem")
	os.Setenv("GATEWAY_DOMAIN_SUFFIX", ".env.example.com")
	defer func() {
		os.Unsetenv("AUTH_SERVICE_URL")
		os.Unsetenv("IDENTITY_SERVICE_URL")
		os.Unsetenv("AUDIT_SERVICE_URL")
		os.Unsetenv("GATEWAY_ADDR")
		os.Unsetenv("GATEWAY_JWKS_URL")
		os.Unsetenv("GATEWAY_JWT_ISSUER")
		os.Unsetenv("JWT_PUBLIC_KEY_PATH")
		os.Unsetenv("GATEWAY_DOMAIN_SUFFIX")
	}()

	cfg := Default()
	cfg = LoadFromEnv(cfg)

	if cfg.Addr != ":9999" {
		t.Errorf("expected Addr :9999, got %s", cfg.Addr)
	}
	if cfg.JWKSURL != "http://jwks-env:8080/.well-known/jwks.json" {
		t.Errorf("expected JWKS URL override, got %s", cfg.JWKSURL)
	}
	if cfg.JWTIssuer != "env-issuer" {
		t.Errorf("expected JWTIssuer env-issuer, got %s", cfg.JWTIssuer)
	}
	if cfg.PublicKeyPath != "/tmp/env-test.pem" {
		t.Errorf("expected PublicKeyPath, got %s", cfg.PublicKeyPath)
	}
	if cfg.DomainSuffix != ".env.example.com" {
		t.Errorf("expected DomainSuffix, got %s", cfg.DomainSuffix)
	}
	if cfg.Routes["/api/v1/auth"] != "http://auth-env:9001" {
		t.Errorf("expected auth route override, got %s", cfg.Routes["/api/v1/auth"])
	}
	if cfg.Routes["/api/v1/users"] != "http://identity-env:8080" {
		t.Errorf("expected identity route override, got %s", cfg.Routes["/api/v1/users"])
	}
	if cfg.Routes["/api/v1/audit"] != "http://audit-env:8072" {
		t.Errorf("expected audit route override, got %s", cfg.Routes["/api/v1/audit"])
	}
}

func TestLoadFromEnv_AllServiceURLs(t *testing.T) {
	// Test every service URL env var
	os.Setenv("ROLES_SERVICE_URL", "http://roles-env:8070")
	os.Setenv("PERMISSIONS_SERVICE_URL", "http://perms-env:8070")
	os.Setenv("POLICY_SERVICE_URL", "http://policy-env:8070")
	os.Setenv("ORG_SERVICE_URL", "http://org-env:8071")
	os.Setenv("OAUTH_SERVICE_URL", "http://oauth-env:9005")
	os.Setenv("SAML_SERVICE_URL", "http://saml-env:9006")
	defer func() {
		os.Unsetenv("ROLES_SERVICE_URL")
		os.Unsetenv("PERMISSIONS_SERVICE_URL")
		os.Unsetenv("POLICY_SERVICE_URL")
		os.Unsetenv("ORG_SERVICE_URL")
		os.Unsetenv("OAUTH_SERVICE_URL")
		os.Unsetenv("SAML_SERVICE_URL")
	}()

	cfg := Default()
	cfg = LoadFromEnv(cfg)

	if cfg.Routes["/api/v1/roles"] != "http://roles-env:8070" {
		t.Errorf("expected roles route override, got %s", cfg.Routes["/api/v1/roles"])
	}
	if cfg.Routes["/api/v1/permissions"] != "http://perms-env:8070" {
		t.Errorf("expected permissions route override, got %s", cfg.Routes["/api/v1/permissions"])
	}
	if cfg.Routes["/api/v1/policies"] != "http://policy-env:8070" {
		t.Errorf("expected policies route override, got %s", cfg.Routes["/api/v1/policies"])
	}
	if cfg.Routes["/api/v1/orgs"] != "http://org-env:8071" {
		t.Errorf("expected orgs route override, got %s", cfg.Routes["/api/v1/orgs"])
	}
	if cfg.Routes["/oauth"] != "http://oauth-env:9005" {
		t.Errorf("expected oauth route override, got %s", cfg.Routes["/oauth"])
	}
	if cfg.Routes["/saml"] != "http://saml-env:9006" {
		t.Errorf("expected saml route override, got %s", cfg.Routes["/saml"])
	}
}

func TestGetRouteTimeout_ZeroDefaults(t *testing.T) {
	// Config with zero global defaults
	cfg := &Config{
		ReadTimeout:  0,
		WriteTimeout: 0,
	}
	rt := cfg.GetRouteTimeout("/api/v1/test")
	// Should fall back to hardcoded defaults
	if rt.Read != 15*time.Second {
		t.Errorf("expected default Read 15s, got %v", rt.Read)
	}
	if rt.Write != 15*time.Second {
		t.Errorf("expected default Write 15s, got %v", rt.Write)
	}
	if rt.Idle != 90*time.Second {
		t.Errorf("expected default Idle 90s, got %v", rt.Idle)
	}
	if rt.Dial != 5*time.Second {
		t.Errorf("expected default Dial 5s, got %v", rt.Dial)
	}
}

func TestGetRouteTimeout_WithExplicitRouteConfig(t *testing.T) {
	cfg := &Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
		RouteConfigs: map[string]RouteConfig{
			"/api/v1/slow": {
				Timeout: RouteTimeout{
					Read:  60 * time.Second,
					Write: 60 * time.Second,
					Idle:  120 * time.Second,
					Dial:  10 * time.Second,
				},
			},
		},
	}
	rt := cfg.GetRouteTimeout("/api/v1/slow")
	if rt.Read != 60*time.Second {
		t.Errorf("expected 60s, got %v", rt.Read)
	}
	if rt.Dial != 10*time.Second {
		t.Errorf("expected 10s, got %v", rt.Dial)
	}
}

func TestGetRouteTimeout_PartialFallback_Dial(t *testing.T) {
	// Route config with Dial=0 should fall back to global
	cfg := &Config{
		ReadTimeout:  12 * time.Second,
		WriteTimeout: 12 * time.Second,
		RouteConfigs: map[string]RouteConfig{
			"/api/v1/partial": {
				Timeout: RouteTimeout{
					Read:  30 * time.Second,
					Write: 30 * time.Second,
					Idle:  90 * time.Second,
					Dial:  0, // should fallback to 5s default
				},
			},
		},
	}
	rt := cfg.GetRouteTimeout("/api/v1/partial")
	if rt.Dial != 5*time.Second {
		t.Errorf("expected Dial 5s fallback, got %v", rt.Dial)
	}
}
