package middleware

import (
	"net/http"
	"strings"
	"time"
)

// RouteTimeoutConfig provides configurable per-route timeout overrides.
// This is a higher-level wrapper around TimeoutMiddleware that supports
// pattern-based route matching (prefix and glob) for backend-specific deadlines.
type RouteTimeoutConfig struct {
	// Default is the timeout for routes without a specific override.
	Default time.Duration
	// RouteTimeouts maps path prefixes to their timeout durations.
	// Longest prefix match wins.
	RouteTimeouts map[string]time.Duration
}

// DefaultRouteTimeoutConfig returns production defaults.
// Slow endpoints like SCIM bulk and audit queries get extended deadlines.
func DefaultRouteTimeoutConfig() *RouteTimeoutConfig {
	return &RouteTimeoutConfig{
		Default: 30 * time.Second,
		RouteTimeouts: map[string]time.Duration{
			"/api/v1/auth/verify":     10 * time.Second,
			"/api/v1/auth/register":  15 * time.Second,
			"/api/v1/scim/Bulk":      120 * time.Second,
			"/api/v1/audit":          60 * time.Second,
			"/api/v1/scim":           45 * time.Second,
			"/api/v1/users":          15 * time.Second,
			"/api/v1/oauth":          10 * time.Second,
		},
	}
}

// MatchTimeout returns the timeout for the given path using longest-prefix match.
// Falls back to Default if no route matches.
func (c *RouteTimeoutConfig) MatchTimeout(path string) time.Duration {
	bestMatch := ""
	bestTimeout := c.Default

	for prefix, timeout := range c.RouteTimeouts {
		if strings.HasPrefix(path, prefix) && len(prefix) > len(bestMatch) {
			bestMatch = prefix
			bestTimeout = timeout
		}
	}

	if bestMatch == "" {
		return c.Default
	}
	return bestTimeout
}

// RouteTimeoutMiddleware creates middleware that applies per-route timeouts.
// It wraps TimeoutMiddleware with a dynamic lookup based on the request path.
func RouteTimeoutMiddleware(cfg *RouteTimeoutConfig) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = DefaultRouteTimeoutConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			timeout := cfg.MatchTimeout(r.URL.Path)
			tc := &TimeoutConfig{
				Default:      timeout,
				RouteConfigs: map[string]time.Duration{r.URL.Path: timeout},
			}
			TimeoutMiddleware(tc)(next).ServeHTTP(w, r)
		})
	}
}
