package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeadersMiddleware adds security headers to every response.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		next.ServeHTTP(w, r)
	})
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

func GetTenantCORS(tenantID string) CORSConfig {
	if cfg, ok := tenantCORSConfigs[tenantID]; ok {
		return cfg
	}
	return defaultTenantCORS
}

func TenantCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		tenantID := r.Header.Get("X-Tenant-ID")
		cfg := GetCORSConfig(tenantID)
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
