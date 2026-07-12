package config

import (
	"testing"
	"time"
)

func TestGetRouteTimeout_NoRouteConfig(t *testing.T) {
	cfg := Default()
	rt := cfg.GetRouteTimeout("/api/v1/unknown")
	if rt.Read != 15*time.Second {
		t.Errorf("Read: want 15s, got %v", rt.Read)
	}
	if rt.Write != 15*time.Second {
		t.Errorf("Write: want 15s, got %v", rt.Write)
	}
	if rt.Idle != 90*time.Second {
		t.Errorf("Idle: want 90s, got %v", rt.Idle)
	}
	if rt.Dial != 5*time.Second {
		t.Errorf("Dial: want 5s, got %v", rt.Dial)
	}
}

func TestGetRouteTimeout_WithExplicitRouteConfig_C3(t *testing.T) {
	cfg := Default()
	// /api/v1/auth has explicit config in Default()
	rt := cfg.GetRouteTimeout("/api/v1/auth")
	if rt.Read != 5*time.Second {
		t.Errorf("Read: want 5s, got %v", rt.Read)
	}
	if rt.Write != 10*time.Second {
		t.Errorf("Write: want 10s, got %v", rt.Write)
	}
	if rt.Dial != 3*time.Second {
		t.Errorf("Dial: want 3s, got %v", rt.Dial)
	}
}

func TestGetRouteTimeout_ZeroGlobalTimeouts(t *testing.T) {
	cfg := &Config{
		RouteConfigs: map[string]RouteConfig{
			"/api/v1/test": {
				Timeout: RouteTimeout{},
			},
		},
	}
	// With zero global ReadTimeout/WriteTimeout, should default to 15s
	rt := cfg.GetRouteTimeout("/api/v1/test")
	if rt.Read != 15*time.Second {
		t.Errorf("Read default: want 15s, got %v", rt.Read)
	}
	if rt.Write != 15*time.Second {
		t.Errorf("Write default: want 15s, got %v", rt.Write)
	}
	if rt.Idle != 90*time.Second {
		t.Errorf("Idle default: want 90s, got %v", rt.Idle)
	}
	if rt.Dial != 5*time.Second {
		t.Errorf("Dial default: want 5s, got %v", rt.Dial)
	}
}

func TestGetRouteTimeout_PartialRouteConfig(t *testing.T) {
	cfg := &Config{
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 20 * time.Second,
		RouteConfigs: map[string]RouteConfig{
			"/api/v1/partial": {
				Timeout: RouteTimeout{
					Read: 30 * time.Second,
					// Write, Idle, Dial are zero
				},
			},
		},
	}
	rt := cfg.GetRouteTimeout("/api/v1/partial")
	if rt.Read != 30*time.Second {
		t.Errorf("Read: want 30s, got %v", rt.Read)
	}
	if rt.Write != 20*time.Second {
		t.Errorf("Write fallback: want 20s, got %v", rt.Write)
	}
	if rt.Idle != 90*time.Second {
		t.Errorf("Idle default: want 90s, got %v", rt.Idle)
	}
	if rt.Dial != 5*time.Second {
		t.Errorf("Dial default: want 5s, got %v", rt.Dial)
	}
}

func TestGetRouteTimeout_NilRouteConfigs(t *testing.T) {
	cfg := &Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	rt := cfg.GetRouteTimeout("/anything")
	if rt.Read != 10*time.Second {
		t.Errorf("Read: want 10s, got %v", rt.Read)
	}
}

func TestLoadFromEnv_AllEnvVars(t *testing.T) {
	t.Setenv("GATEWAY_ADDR", ":9999")
	t.Setenv("GATEWAY_DOMAIN_SUFFIX", ".example.com")
	t.Setenv("GATEWAY_JWKS_URL", "http://jwks:8080")
	t.Setenv("GATEWAY_JWT_ISSUER", "custom-issuer")
	t.Setenv("JWT_PUBLIC_KEY_PATH", "/custom/path.pem")
	t.Setenv("AUTH_SERVICE_URL", "http://auth:9001")
	t.Setenv("IDENTITY_SERVICE_URL", "http://id:8081")
	t.Setenv("OAUTH_SERVICE_URL", "http://oauth:9005")

	cfg := LoadFromEnv(Default())
	if cfg.Addr != ":9999" {
		t.Errorf("Addr: got '%s'", cfg.Addr)
	}
	if cfg.DomainSuffix != ".example.com" {
		t.Errorf("DomainSuffix: got '%s'", cfg.DomainSuffix)
	}
	if cfg.JWKSURL != "http://jwks:8080" {
		t.Errorf("JWKSURL: got '%s'", cfg.JWKSURL)
	}
	if cfg.JWTIssuer != "custom-issuer" {
		t.Errorf("JWTIssuer: got '%s'", cfg.JWTIssuer)
	}
	if cfg.PublicKeyPath != "/custom/path.pem" {
		t.Errorf("PublicKeyPath: got '%s'", cfg.PublicKeyPath)
	}
	if cfg.Routes["/api/v1/auth"] != "http://auth:9001" {
		t.Errorf("Auth route: got '%s'", cfg.Routes["/api/v1/auth"])
	}
	if cfg.Routes["/api/v1/identity"] != "http://id:8081" {
		t.Errorf("Identity route: got '%s'", cfg.Routes["/api/v1/identity"])
	}
	if cfg.Routes["/oauth"] != "http://oauth:9005" {
		t.Errorf("OAuth route: got '%s'", cfg.Routes["/oauth"])
	}
}

func TestLoadFromEnv_NoEnvVars(t *testing.T) {
	// Clear all relevant env vars
	cfg := LoadFromEnv(Default())
	// Should remain unchanged from defaults
	if cfg.Addr != ":8080" {
		t.Errorf("Default Addr should be :8080, got '%s'", cfg.Addr)
	}
}
