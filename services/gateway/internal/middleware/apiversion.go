// Package middleware implements HTTP middleware for the API Gateway.
package middleware

import (
	"net/http"
	"strings"
)

// APIVersionConfig holds per-version backend routing.
// Keys are version strings ("1", "2", "3") and values are backend URLs.
type APIVersionConfig struct {
	DefaultVersion string            // default version when not specified ("1")
	Backends       map[string]string // version → backend URL
	StripPrefix    bool              // strip the version prefix before forwarding
	HeaderName     string            // custom header name (default: Api-Version)
}

// DefaultAPIVersionConfig returns a default version config.
func DefaultAPIVersionConfig() APIVersionConfig {
	return APIVersionConfig{
		DefaultVersion: "1",
		HeaderName:     "Api-Version",
		Backends:       make(map[string]string),
	}
}

// APIVersioning is middleware that routes requests to different backend
// versions based on either:
//   - URL path prefix: /api/v2/users → version 2
//   - HTTP header: Api-Version: 2
//
// The URL path takes precedence over the header.
// If neither is specified, the default version is used.
func APIVersioning(cfg APIVersionConfig, nextByVersion func(version string) http.Handler) http.Handler {
	if cfg.HeaderName == "" {
		cfg.HeaderName = "Api-Version"
	}
	if cfg.DefaultVersion == "" {
		cfg.DefaultVersion = "1"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := extractAPIVersion(r, cfg)

		// Check if a backend exists for this version
		if len(cfg.Backends) > 0 {
			if _, ok := cfg.Backends[version]; !ok {
				// Fall back to default version
				version = cfg.DefaultVersion
			}
		}

		// Rewrite the path if needed (strip /api/vN/ → /api/)
		if cfg.StripPrefix {
			r = stripVersionPrefix(r)
		}

		handler := nextByVersion(version)
		if handler == nil {
			WriteError(w, r, http.StatusBadGateway, "bad_gateway", "no backend for API version")
			return
		}
		handler.ServeHTTP(w, r)
	})
}

// extractAPIVersion determines the API version from the request.
// URL path /api/vN/ takes precedence over the Api-Version header.
func extractAPIVersion(r *http.Request, cfg APIVersionConfig) string {
	// 1. Check URL path: /api/v2/...
	if v := versionFromPath(r.URL.Path); v != "" {
		return v
	}

	// 2. Check header
	if cfg.HeaderName != "" {
		if v := r.Header.Get(cfg.HeaderName); v != "" {
			return strings.TrimSpace(v)
		}
	}

	// 3. Check query parameter
	if v := r.URL.Query().Get("api_version"); v != "" {
		return v
	}

	// 4. Default
	return cfg.DefaultVersion
}

// versionFromPath extracts version from URL path like /api/v2/users → "2".
func versionFromPath(path string) string {
	// Match /api/vN/ or /api/vN (at end)
	if !strings.HasPrefix(path, "/api/v") {
		return ""
	}
	rest := path[6:] // skip "/api/v"
	// Extract digits
	version := ""
	for _, ch := range rest {
		if ch >= '0' && ch <= '9' {
			version += string(ch)
		} else {
			break
		}
	}
	return version
}

// stripVersionPrefix removes the /api/vN part from the path, rewriting
// /api/v2/users → /api/users so the backend doesn't need to handle versioning.
func stripVersionPrefix(r *http.Request) *http.Request {
	path := r.URL.Path
	if !strings.HasPrefix(path, "/api/v") {
		return r
	}
	rest := path[6:] // skip "/api/v"
	// Skip version digits
	for len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
		rest = rest[1:]
	}
	newPath := "/api" + rest
	r2 := r.Clone(r.Context())
	r2.URL.Path = newPath
	r2.URL.RawPath = ""
	return r2
}

// VersionFromPath is exported for testing.
func VersionFromPath(path string) string {
	return versionFromPath(path)
}

// ExtractAPIVersion is exported for testing.
func ExtractAPIVersion(r *http.Request, cfg APIVersionConfig) string {
	return extractAPIVersion(r, cfg)
}
