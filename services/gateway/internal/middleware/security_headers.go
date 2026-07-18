package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

// SecurityHeadersConfig holds configurable security header settings.
type SecurityHeadersConfig struct {
	Enabled            bool                          `json:"enabled"`
	FrameDeny          bool                          `json:"frame_deny"`
	FrameAllowFrom     string                        `json:"frame_allow_from,omitempty"`
	CSP                string                        `json:"content_security_policy,omitempty"`
	ContentTypeNosniff bool                          `json:"content_type_nosniff"`
	HSTSMaxAge         int                           `json:"hsts_max_age"`
	PerTenantOverrides map[string]*SecurityHeadersConfig `json:"per_tenant_overrides,omitempty"`
}

// DefaultSecurityHeadersConfig returns the default config.
func DefaultSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		Enabled: true, FrameDeny: true, ContentTypeNosniff: true, HSTSMaxAge: 31536000,
		CSP: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'",
	}
}

// mergeSecurityHeaders merges an override config onto a base config.
func mergeSecurityHeaders(base, override *SecurityHeadersConfig) *SecurityHeadersConfig {
	if base == nil { return override }
	if override == nil { return base }
	result := *base
	if override.CSP != "" { result.CSP = override.CSP }
	if override.FrameDeny { result.FrameDeny = true }
	if override.FrameAllowFrom != "" { result.FrameAllowFrom = override.FrameAllowFrom }
	if override.ContentTypeNosniff { result.ContentTypeNosniff = true }
	if override.HSTSMaxAge > 0 { result.HSTSMaxAge = override.HSTSMaxAge }
	result.Enabled = override.Enabled
	return &result
}

// SecurityHeadersConfigurable returns middleware with configurable security headers.
func SecurityHeadersConfigurable(cfg *SecurityHeadersConfig) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = DefaultSecurityHeadersConfig()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check per-tenant override.
			active := cfg
			if tenantID := r.Header.Get("X-Tenant-ID"); tenantID != "" {
				if override, ok := cfg.PerTenantOverrides[tenantID]; ok {
					active = override
				}
			}
			if !active.Enabled {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("X-Content-Type-Options", "nosniff")
			if active.FrameDeny {
				w.Header().Set("X-Frame-Options", "DENY")
			} else if active.FrameAllowFrom != "" {
				w.Header().Set("X-Frame-Options", "ALLOW-FROM "+active.FrameAllowFrom)
			}
			if active.HSTSMaxAge > 0 {
				w.Header().Set("Strict-Transport-Security", "max-age="+fmt.Sprintf("%d", active.HSTSMaxAge)+"; includeSubDomains")
			}
			csp := active.CSP
			if csp == "" {
				csp = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'"
			}
			w.Header().Set("Content-Security-Policy", csp)
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeadersMiddleware adds security headers to every response (backward compat).
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return SecurityHeadersConfigurable(nil)(next)
}

// CORSConfig defines per-tenant CORS settings.
type TenantCORSConfig struct {
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}

var (
	defaultTenantCORS = TenantCORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Request-ID", "X-Trace-Id", "X-Session-ID"},
		AllowCredentials: true,
		MaxAge:           3600,
	}
	tenantCORSConfigs = map[string]TenantCORSConfig{}
)

func SetTenantCORS(tenantID string, cfg TenantCORSConfig) {
	tenantCORSConfigs[tenantID] = cfg
}

func GetTenantCORS(tenantID string) TenantCORSConfig {
	if cfg, ok := tenantCORSConfigs[tenantID]; ok {
		return cfg
	}
	return defaultTenantCORS
}

func TenantCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		tenantID := r.Header.Get("X-Tenant-ID")
		cfg := GetTenantCORS(tenantID)
		allowed := false
		for _, o := range cfg.AllowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}
		if allowed && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			w.Header().Set("Access-Control-Max-Age", "3600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func HardenCookie(c *http.Cookie) {
	c.Secure = true
	c.HttpOnly = true
	c.SameSite = http.SameSiteStrictMode
}

func SetSecureCookie(w http.ResponseWriter, name, value, path string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: value, Path: path,
		MaxAge: maxAge, Secure: true, HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}
