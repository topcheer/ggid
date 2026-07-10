package middleware

import (
	"net/http"
	"strconv"
)

// SecurityHeadersConfig defines per-tenant security header policies.
type SecurityHeadersConfig struct {
	// Enabled controls whether security headers are injected.
	Enabled bool
	// ContentTypeNosniff sets X-Content-Type-Options: nosniff.
	ContentTypeNosniff bool
	// FrameDeny sets X-Frame-Options: DENY.
	FrameDeny bool
	// FrameAllowFrom sets X-Frame-Options: ALLOW-FROM <origin>. Ignored if FrameDeny is true.
	FrameAllowFrom string
	// HSTSMaxAge sets Strict-Transport-Security max-age in seconds.
	HSTSMaxAge int
		// HSTSIncludeSubDomains adds includeSubDomains to HSTS.
	HSTSIncludeSubDomains bool
		// CSP defines the Content-Security-Policy header.
	CSP string
		// ReferrerPolicy sets Referrer-Policy header.
	ReferrerPolicy string
		// PerTenantOverrides allows per-tenant configuration keyed by tenant ID.
	PerTenantOverrides map[string]*SecurityHeadersConfig
}

// DefaultSecurityHeadersConfig returns a config with secure defaults.
func DefaultSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		Enabled:               true,
		ContentTypeNosniff:    true,
		FrameDeny:             true,
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubDomains: true,
		CSP:                   "default-src 'self'; frame-ancestors 'none'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
	}
}

// SecurityHeadersConfigurable returns middleware that injects security headers
// with full configuration. Tenant-specific overrides are applied when X-Tenant-ID is present.
func SecurityHeadersConfigurable(cfg *SecurityHeadersConfig) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = DefaultSecurityHeadersConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			effective := cfg
			if tenantID := r.Header.Get("X-Tenant-ID"); tenantID != "" {
				if override, ok := cfg.PerTenantOverrides[tenantID]; ok {
					effective = override
				}
			}

			if !effective.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			h := w.Header()

			if effective.ContentTypeNosniff {
				h.Set("X-Content-Type-Options", "nosniff")
			}

			if effective.FrameDeny {
				h.Set("X-Frame-Options", "DENY")
			} else if effective.FrameAllowFrom != "" {
				h.Set("X-Frame-Options", "ALLOW-FROM "+effective.FrameAllowFrom)
			}

			if effective.HSTSMaxAge > 0 {
				value := "max-age=" + strconv.Itoa(effective.HSTSMaxAge)
				if effective.HSTSIncludeSubDomains {
					value += "; includeSubDomains"
				}
				h.Set("Strict-Transport-Security", value)
			}

			if effective.CSP != "" {
				h.Set("Content-Security-Policy", effective.CSP)
			}

			if effective.ReferrerPolicy != "" {
				h.Set("Referrer-Policy", effective.ReferrerPolicy)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// mergeSecurityHeaders applies tenant overrides onto a base config.
func mergeSecurityHeaders(base, override *SecurityHeadersConfig) *SecurityHeadersConfig {
	if override == nil {
		return base
	}
	if base == nil {
		return override
	}
	merged := *base
	if override.Enabled {
		merged.Enabled = true
	}
	if override.ContentTypeNosniff {
		merged.ContentTypeNosniff = true
	}
	if override.FrameDeny {
		merged.FrameDeny = true
	}
	if override.FrameAllowFrom != "" {
		merged.FrameAllowFrom = override.FrameAllowFrom
	}
	if override.HSTSMaxAge > 0 {
		merged.HSTSMaxAge = override.HSTSMaxAge
	}
	if override.HSTSIncludeSubDomains {
		merged.HSTSIncludeSubDomains = true
	}
	if override.CSP != "" {
		merged.CSP = override.CSP
	}
	if override.ReferrerPolicy != "" {
		merged.ReferrerPolicy = override.ReferrerPolicy
	}
	return &merged
}
