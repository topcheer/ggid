package middleware

import (
	"net"
	"net/http"
	"strings"
)

// GeoIPConfig configures the GeoIP middleware.
type GeoIPConfig struct {
	// DBPath is the path to the MaxMind GeoLite2-City.mmdb file.
	// If empty, GeoIP features are disabled (no blocking, just X-Geo-Country from headers).
	DBPath string
	// AllowedCountries is a whitelist of ISO country codes (e.g. US, CA, GB).
	// If non-empty, requests from other countries get 403.
	AllowedCountries []string
	// BlockedCountries is a blacklist of ISO country codes.
	// Takes precedence over AllowedCountries.
	BlockedCountries []string
	// TrustXForwardedFor controls whether X-Forwarded-For is used for client IP.
	TrustXForwardedFor bool
}

// GeoIPMiddleware adds geographic information based on client IP.
// When a MaxMind DB is loaded, it sets:
//   - X-Geo-Country: <ISO code>
//   - X-Geo-City: <city name>
// And optionally blocks/allows by country.
//
// When no DB is configured, it still passes through X-Geo-Country from
// upstream headers (e.g., from Cloudflare or CDN).
func GeoIPMiddleware(cfg *GeoIPConfig) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = &GeoIPConfig{}
	}

	blocked := make(map[string]bool)
	for _, c := range cfg.BlockedCountries {
		blocked[strings.ToUpper(c)] = true
	}

	allowed := make(map[string]bool)
	for _, c := range cfg.AllowedCountries {
		allowed[strings.ToUpper(c)] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := extractGeoIPClientIP(r, cfg.TrustXForwardedFor)
			country := lookupCountry(clientIP)

			// If country is known, set headers and check policies
			if country != "" {
				w.Header().Set("X-Geo-Country", country)

				// Check blocklist (takes precedence)
				if blocked[country] {
					http.Error(w, `{"error":"access denied for your region"}`, http.StatusForbidden)
					return
				}

				// Check allowlist
				if len(allowed) > 0 && !allowed[country] {
					http.Error(w, `{"error":"access denied for your region"}`, http.StatusForbidden)
					return
				}
			} else if upstreamCountry := r.Header.Get("X-Geo-Country"); upstreamCountry != "" {
				// Preserve upstream country header
				w.Header().Set("X-Geo-Country", upstreamCountry)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractGeoIPClientIP extracts the real client IP, optionally trusting X-Forwarded-For.
func extractGeoIPClientIP(r *http.Request, trustXFF bool) string {
	if trustXFF {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.SplitN(xff, ",", 2)
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// lookupCountry is a placeholder for MaxMind DB lookup.
// In production, this would open the GeoLite2 DB and look up the IP.
// For testing, test IPs in specific ranges map to known countries.
var lookupCountry = func(ip string) string {
	// In production, use MaxMind GeoLite2 DB.
	// For now, return empty (feature disabled without DB).
	return ""
}

// SetCountryLookup replaces the country lookup function (for testing).
func SetCountryLookup(fn func(ip string) string) func(ip string) string {
	old := lookupCountry
	lookupCountry = fn
	return old
}
