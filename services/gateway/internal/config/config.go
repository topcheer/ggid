// Package config defines the API Gateway configuration.
package config

import (
	"os"
	"time"
)

// RouteTimeout holds per-route timeout configuration.
// Zero-value fields fall back to the global defaults.
type RouteTimeout struct {
	Read    time.Duration `yaml:"read"`    // time to read the response headers from backend
	Write   time.Duration `yaml:"write"`   // time to write the full request to backend
	Idle    time.Duration `yaml:"idle"`    // idle timeout for keepalive connections
	Dial    time.Duration `yaml:"dial"`    // dial timeout
}

// RouteConfig holds metadata for a single route.
type RouteConfig struct {
	URL     string       `yaml:"url"`
	Timeout RouteTimeout `yaml:"timeout"`
}

// Config is the root configuration for the API Gateway.
type Config struct {
	Addr            string              `yaml:"addr"`
	DomainSuffix    string              `yaml:"domain_suffix"`
	JWKSURL         string              `yaml:"jwks_url"`
	JWTIssuer       string              `yaml:"jwt_issuer"`
	JWTAudience     string              `yaml:"jwt_audience"`
	PublicKeyPath   string              `yaml:"public_key_path"`
	Routes          map[string]string   `yaml:"routes"`      // path_prefix -> backend URL (backward compat)
	RouteConfigs    map[string]RouteConfig `yaml:"route_configs"` // per-route advanced config
	ReadTimeout     time.Duration       `yaml:"read_timeout"`
	WriteTimeout    time.Duration       `yaml:"write_timeout"`
	UpstreamTimeout time.Duration       `yaml:"upstream_timeout"`
}

// Default returns the default gateway configuration.
// Route URLs can be overridden via env vars: AUTH_SERVICE_URL, USERS_SERVICE_URL,
// POLICY_SERVICE_URL, ORG_SERVICE_URL, AUDIT_SERVICE_URL, OAUTH_SERVICE_URL.
// UPSTREAM_TIMEOUT (in seconds) controls the default upstream proxy timeout.
func Default() *Config {
	upstreamTimeout := 30 * time.Second
	if v := os.Getenv("UPSTREAM_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v + "s"); err == nil {
			upstreamTimeout = d
		}
	}

	return &Config{
		Addr:          ":8080",
		DomainSuffix:  "",
		JWKSURL:       "", // empty = use local public key
		JWTIssuer:     "ggid-auth",
		JWTAudience:   "ggid",
		PublicKeyPath: "configs/rsa_public.pem",
		Routes: map[string]string{
			"/api/v1/auth":        envOrDefault("AUTH_SERVICE_URL", "http://localhost:9001"),
			"/api/v1/users":       envOrDefault("USERS_SERVICE_URL", "http://localhost:8081"),
			"/api/v1/roles":       envOrDefault("POLICY_SERVICE_URL", "http://localhost:8070"),
			"/api/v1/permissions": envOrDefault("POLICY_SERVICE_URL", "http://localhost:8070"),
			"/api/v1/policies":    envOrDefault("POLICY_SERVICE_URL", "http://localhost:8070"),
			"/api/v1/orgs":        envOrDefault("ORG_SERVICE_URL", "http://localhost:8071"),
			"/api/v1/audit":       envOrDefault("AUDIT_SERVICE_URL", "http://localhost:8072"),
			"/api/v1/access-requests": envOrDefault("USERS_SERVICE_URL", "http://localhost:8081"),
			"/api/v1/departments":  envOrDefault("USERS_SERVICE_URL", "http://localhost:8081"),
			"/api/v1/teams":       envOrDefault("ORG_SERVICE_URL", "http://localhost:8071"),
			"/api/v1/tenants":     envOrDefault("USERS_SERVICE_URL", "http://localhost:8081"),
			"/api/v1/identity":    envOrDefault("USERS_SERVICE_URL", "http://localhost:8081"),
			"/api/v1/organizations": envOrDefault("ORG_SERVICE_URL", "http://localhost:8071"),
			"/api/v1/notifications":   envOrDefault("AUTH_SERVICE_URL", "http://localhost:9001"),
			"/api/v1/scim":            envOrDefault("USERS_SERVICE_URL", "http://localhost:8081"),
			"/scim/v2":                envOrDefault("USERS_SERVICE_URL", "http://localhost:8081"),
			"/api/v1/idp":             envOrDefault("OAUTH_SERVICE_URL", "http://localhost:9005"),
			"/api/v1/agents":          envOrDefault("OAUTH_SERVICE_URL", "http://localhost:9005"),
			"/api/v1/oauth":           envOrDefault("OAUTH_SERVICE_URL", "http://localhost:9005"),
			"/api/v1/webhooks":        envOrDefault("AUDIT_SERVICE_URL", "http://localhost:8072"),
			"/api/v1/rate-limits":     envOrDefault("POLICY_SERVICE_URL", "http://localhost:8070"),
			"/api/v1/sod":             envOrDefault("POLICY_SERVICE_URL", "http://localhost:8070"),
			"/api/v1/consent":         envOrDefault("AUTH_SERVICE_URL", "http://localhost:9001"),
			"/api/v1/mfa":             envOrDefault("AUTH_SERVICE_URL", "http://localhost:9001"),
			"/api/v1/tokens":          envOrDefault("AUTH_SERVICE_URL", "http://localhost:9001"),
			"/api/v1/introspection":   envOrDefault("AUTH_SERVICE_URL", "http://localhost:9001"),
			"/api/v1/login-security":  envOrDefault("AUTH_SERVICE_URL", "http://localhost:9001"),
			"/api/v1/password-history": envOrDefault("AUTH_SERVICE_URL", "http://localhost:9001"),
			"/api/v1/delegation":      envOrDefault("AUTH_SERVICE_URL", "http://localhost:9001"),
			"/api/v1/account-linking": envOrDefault("AUTH_SERVICE_URL", "http://localhost:9001"),
			"/api/v1/policy-versions": envOrDefault("POLICY_SERVICE_URL", "http://localhost:8070"),
			"/api/v1/device-bindings": envOrDefault("AUTH_SERVICE_URL", "http://localhost:9001"),
			"/api/v1/alerts":          envOrDefault("AUDIT_SERVICE_URL", "http://localhost:8072"),
			"/api/v1/event-correlation": envOrDefault("AUDIT_SERVICE_URL", "http://localhost:8072"),
			"/api/v1/role-templates":  envOrDefault("POLICY_SERVICE_URL", "http://localhost:8070"),
			"/api/v1/scope-management": envOrDefault("OAUTH_SERVICE_URL", "http://localhost:9005"),
			"/api/v1/siem":            envOrDefault("AUDIT_SERVICE_URL", "http://localhost:8072"),
			"/api/v1/security":        envOrDefault("AUDIT_SERVICE_URL", "http://localhost:8072"),
			"/api/v1/admin":           envOrDefault("AUTH_SERVICE_URL", "http://localhost:9001"),
			"/api/v1/compliance":      envOrDefault("AUDIT_SERVICE_URL", "http://localhost:8072"),
			"/oauth":                              envOrDefault("OAUTH_SERVICE_URL", "http://localhost:9005"),
			"/saml":                               envOrDefault("OAUTH_SERVICE_URL", "http://localhost:9005"),
			"/.well-known/openid-configuration":   envOrDefault("OAUTH_SERVICE_URL", "http://localhost:9005"),
		},
		RouteConfigs: map[string]RouteConfig{
			// Auth needs short timeouts for fast failure on rate-limited requests
			"/api/v1/auth": {
				Timeout: RouteTimeout{
					Read:  5 * time.Second,
					Write: 10 * time.Second,
					Idle:  60 * time.Second,
					Dial:  3 * time.Second,
				},
			},
			// Audit may return large datasets
			"/api/v1/audit": {
				Timeout: RouteTimeout{
					Read:  30 * time.Second,
					Write: 30 * time.Second,
					Idle:  90 * time.Second,
					Dial:  5 * time.Second,
				},
			},
		},
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		UpstreamTimeout: upstreamTimeout,
	}
}

// envOrDefault returns the env var value if set and non-empty, otherwise the fallback.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
		"AUTH_SERVICE_URL":         "/api/v1/auth",
		"IDENTITY_SERVICE_URL":     "/api/v1/identity",
		"USERS_SERVICE_URL":        "/api/v1/users",
		"ROLES_SERVICE_URL":        "/api/v1/roles",
		"PERMISSIONS_SERVICE_URL":  "/api/v1/permissions",
		"POLICY_SERVICE_URL":       "/api/v1/policies",
		"ORG_SERVICE_URL":          "/api/v1/orgs",
		"AUDIT_SERVICE_URL":        "/api/v1/audit",
		"OAUTH_SERVICE_URL":        "/oauth",
		"SAML_SERVICE_URL":         "/saml",
	}
	for envKey, route := range serviceEnvs {
		if v := os.Getenv(envKey); v != "" {
			cfg.Routes[route] = v
		}
	}

	return cfg
}

// GetRouteTimeout returns the timeout configuration for a given route prefix.
// Falls back to global defaults (ReadTimeout / WriteTimeout) when the route
// has no explicit timeout configured or a field is zero.
func (c *Config) GetRouteTimeout(prefix string) RouteTimeout {
	defaultRead := c.ReadTimeout
	if defaultRead == 0 {
		defaultRead = 15 * time.Second
	}
	defaultWrite := c.WriteTimeout
	if defaultWrite == 0 {
		defaultWrite = 15 * time.Second
	}

	rc, ok := c.RouteConfigs[prefix]
	if !ok {
		return RouteTimeout{
			Read:  defaultRead,
			Write: defaultWrite,
			Idle:  90 * time.Second,
			Dial:  5 * time.Second,
		}
	}
	t := rc.Timeout
	if t.Read == 0 {
		t.Read = defaultRead
	}
	if t.Write == 0 {
		t.Write = defaultWrite
	}
	if t.Idle == 0 {
		t.Idle = 90 * time.Second
	}
	if t.Dial == 0 {
		t.Dial = 5 * time.Second
	}
	return t
}
