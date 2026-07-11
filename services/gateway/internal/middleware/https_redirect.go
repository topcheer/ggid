package middleware

import (
	"net/http"
	"strings"
)

// HTTPSRedirectMiddleware redirects HTTP requests to HTTPS.
// When enabled, any non-TLS request receives a 301 redirect to the
// same URL with https:// scheme and the X-Forwarded-Proto or standard
// TLS detection.
//
// Set REDIRECT_HTTPS=true to enable in the gateway config.
func HTTPSRedirectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip redirect if already HTTPS (TLS termination at proxy)
		if r.TLS != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Check X-Forwarded-Proto header (set by load balancers)
		if proto := r.Header.Get("X-Forwarded-Proto"); proto == "https" {
			next.ServeHTTP(w, r)
			return
		}

		// Build the HTTPS URL
		host := r.Host
		if host == "" {
			host = r.URL.Host
		}

		target := "https://" + host + r.URL.RequestURI()

		// Preserve query string
		if r.URL.RawQuery != "" && !strings.Contains(r.URL.RequestURI(), "?") {
			target += "?" + r.URL.RawQuery
		}

		w.Header().Set("Location", target)
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.WriteHeader(http.StatusMovedPermanently)
	})
}
