// Package config defines the API Gateway configuration.
package config

import (
	"os"
	"time"
)

// Config is the root configuration for the API Gateway.
type Config struct {
	Addr            string            `yaml:"addr"`
	DomainSuffix    string            `yaml:"domain_suffix"`
	JWKSURL         string            `yaml:"jwks_url"`
	JWTIssuer       string            `yaml:"jwt_issuer"`
	JWTAudience     string            `yaml:"jwt_audience"`
	PublicKeyPath   string            `yaml:"public_key_path"`
	Routes          map[string]string `yaml:"routes"` // path_prefix -> backend URL
	ReadTimeout     time.Duration     `yaml:"read_timeout"`
	WriteTimeout    time.Duration     `yaml:"write_timeout"`
}

// Default returns the default gateway configuration.
func Default() *Config {
	return &Config{
		Addr:          ":8080",
		DomainSuffix:  "",
		JWKSURL:       "", // empty = use local public key
		JWTIssuer:     "ggid-auth",
		JWTAudience:   "ggid",
		PublicKeyPath: "configs/rsa_public.pem",
		Routes: map[string]string{
			"/api/v1/auth":     "http://localhost:9001",
			"/api/v1/users":    "http://localhost:8081",
			"/api/v1/roles":    "http://localhost:8070",
			"/api/v1/policies": "http://localhost:8070",
			"/api/v1/orgs":     "http://localhost:8071",
			"/api/v1/audit":    "http://localhost:8072",
			"/oauth":           "http://localhost:9005",
			"/saml":            "http://localhost:9005",
		},
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
}

// LoadFromEnv overrides config from environment variables.
// All service URLs can be overridden for Docker deployment.
func LoadFromEnv(cfg *Config) *Config {
	if v := os.Getenv("GATEWAY_ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv("GATEWAY_DOMAIN_SUFFIX"); v != "" {
		cfg.DomainSuffix = v
	}
	if v := os.Getenv("GATEWAY_JWKS_URL"); v != "" {
		cfg.JWKSURL = v
	}
	if v := os.Getenv("GATEWAY_JWT_ISSUER"); v != "" {
		cfg.JWTIssuer = v
	}
	if v := os.Getenv("JWT_PUBLIC_KEY_PATH"); v != "" {
		cfg.PublicKeyPath = v
	}

	// Service URL overrides — each maps a path prefix to a backend URL
	serviceEnvs := map[string]string{
		"AUTH_SERVICE_URL":     "/api/v1/auth",
		"IDENTITY_SERVICE_URL": "/api/v1/users",
		"POLICY_SERVICE_URL":   "/api/v1/policies",
		"ROLES_SERVICE_URL":    "/api/v1/roles",
		"ORG_SERVICE_URL":      "/api/v1/orgs",
		"AUDIT_SERVICE_URL":    "/api/v1/audit",
		"OAUTH_SERVICE_URL":    "/oauth",
		"SAML_SERVICE_URL":     "/saml",
	}
	for envKey, route := range serviceEnvs {
		if v := os.Getenv(envKey); v != "" {
			cfg.Routes[route] = v
		}
	}

	return cfg
}
