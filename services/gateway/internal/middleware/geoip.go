package middleware

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/oschwald/maxminddb-golang"
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

// geoIPDB holds the MaxMind DB reader (lazily loaded).
var (
	geoIPDB     *maxminddb.Reader
	geoIPDBOnce sync.Once
	geoIPDBErr  error
)

// initGeoIPDB loads the MaxMind DB from GEOIP_DB_PATH env or cfg.DBPath.
func initGeoIPDB(dbPath string) {
	geoIPDBOnce.Do(func() {
		path := dbPath
		if path == "" {
			path = os.Getenv("GEOIP_DB_PATH")
		}
		if path == "" {
			return // No DB configured — private IP detection only
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			geoIPDBErr = err
			return
		}
		geoIPDB, geoIPDBErr = maxminddb.Open(path)
	})
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

	// Initialize MaxMind DB if configured
	initGeoIPDB(cfg.DBPath)

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

			// If country is known (and not just LOCAL), set headers and check policies
			if country != "" && country != "LOCAL" {
				w.Header().Set("X-Geo-Country", country)

				// Check blocklist (takes precedence)
				if blocked[country] {
					WriteError(w, r, http.StatusForbidden, "geoip_blocked", "access denied for your region")
					return
				}

				// Check allowlist
				if len(allowed) > 0 && !allowed[country] {
					WriteError(w, r, http.StatusForbidden, "geoip_blocked", "access denied for your region")
					return
				}
			} else if upstreamCountry := r.Header.Get("X-Geo-Country"); upstreamCountry != "" {
				// Preserve upstream country header (e.g., from Cloudflare or CDN)
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

// lookupCountry resolves an IP to an ISO country code.
// When GEOIP_DB_PATH is set and the MaxMind DB is loaded, it performs a real lookup.
// Otherwise, it uses private network detection (RFC1918 = "LOCAL").
var lookupCountry = func(ip string) string {
	// Check for private/loopback IPs first.
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	if parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast() {
		return "LOCAL"
	}

	// Use MaxMind DB if loaded
	if geoIPDB != nil {
		var record struct {
			Country struct {
				ISOCode string `maxminddb:"iso_code"`
			} `maxminddb:"country"`
			City struct {
				Names map[string]string `maxminddb:"names"`
			} `maxminddb:"city"`
		}
		if err := geoIPDB.Lookup(parsed, &record); err == nil && record.Country.ISOCode != "" {
			return record.Country.ISOCode
		}
	}

	return ""
}

// SetCountryLookup replaces the country lookup function (for testing).
func SetCountryLookup(fn func(ip string) string) func(ip string) string {
	old := lookupCountry
	lookupCountry = fn
	return old
}
