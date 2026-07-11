package middleware

import (
	"net/http"
	"strings"
)

// HostValidationConfig configures the Host header allowlist.
type HostValidationConfig struct {
	// AllowedHosts is the list of permitted Host header values.
	// May include port suffixes (e.g. "example.com:8080").
	AllowedHosts []string
	// AllowPortStripping allows the middleware to strip the port before matching.
	AllowPortStripping bool
	// AllowedPorts is the list of ports allowed in Host header. Empty = any.
	AllowedPorts []string
}

// HostValidation returns middleware that validates the Host header against an allowlist.
// This prevents DNS rebinding attacks where an attacker uses their domain to reach
// internal services that trust localhost.
func HostValidation(cfg HostValidationConfig) func(http.Handler) http.Handler {
	hostSet := make(map[string]bool)
	for _, h := range cfg.AllowedHosts {
		hostSet[strings.ToLower(strings.TrimSpace(h))] = true
	}

	portSet := make(map[string]bool)
	for _, p := range cfg.AllowedPorts {
		portSet[p] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// No allowlist = allow all
			if len(hostSet) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			host := strings.ToLower(r.Host)
			if host == "" {
				WriteError(w, r, http.StatusBadRequest, "bad_request", "missing Host header")
				return
			}

			// Try exact match first
			if hostSet[host] {
				next.ServeHTTP(w, r)
				return
			}

			// Try with port stripped
			if cfg.AllowPortStripping {
				hostname := host
				port := ""
				if idx := strings.LastIndex(host, ":"); idx > 0 {
					hostname = host[:idx]
					port = host[idx+1:]
				}
				if hostSet[hostname] {
					// Check port if configured
					if len(portSet) == 0 || port == "" || portSet[port] {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			WriteError(w, r, http.StatusForbidden, "host_not_allowed", "Host not allowed")
		})
	}
}
