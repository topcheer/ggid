package middleware

import (
	"net/http"
	"strings"
	"time"
)

// DeprecationInfo holds metadata for deprecated API versions.
type DeprecationInfo struct {
	// Version is the deprecated version string (e.g. "1").
	Version string
	// Sunset is the date when the version will be removed (RFC 8594).
	Sunset time.Time
	// Deprecation indicates whether the version is deprecated.
	Deprecation bool
	// Link is a URL to migration documentation.
	Link string
}

// APIVersioningMiddleware wraps the APIVersioning router with deprecation headers.
// When a request targets a deprecated version, the response includes:
//   - Sunset: <date> (RFC 8594)
//   - Deprecation: true (RFC draft)
//   - Link: <migration-url>; rel="deprecation"
func APIVersioningMiddleware(cfg APIVersionConfig, deprecations map[string]*DeprecationInfo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			version := extractAPIVersion(r, cfg)

			// Add deprecation headers if applicable
			if dep, ok := deprecations[version]; ok {
				if !dep.Sunset.IsZero() {
					w.Header().Set("Sunset", dep.Sunset.Format(time.RFC1123))
				}
				if dep.Deprecation {
					w.Header().Set("Deprecation", "true")
				}
				if dep.Link != "" {
					w.Header().Set("Link", dep.Link+"; rel=\"deprecation\"")
				}
			}

			// Add the detected version as a header for downstream
			w.Header().Set("X-API-Version", version)

			next.ServeHTTP(w, r)
		})
	}
}

// ExtractVersionFromPath extracts version from /vN/ URL prefix (e.g. /v2/users → "2").
// Unlike versionFromPath, this works with any path prefix not just /api/.
func ExtractVersionFromPath(path string) string {
	// Try /vN/ or /vN (at end)
	if !strings.HasPrefix(path, "/v") {
		return ""
	}
	rest := path[2:] // skip "/v"
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
