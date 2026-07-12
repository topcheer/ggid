package httpserver

import (
	"net/http"
)

// These handlers provide top-level route aliases for endpoints that are also
// available under /api/v1/policies/...  They follow the same patterns as the
// existing handlers and return empty arrays with 200 when no data exists.

// GET /api/v1/rate-limits — alias for handleRateLimits
func (s *HTTPServer) handleRateLimitsAlias(w http.ResponseWriter, r *http.Request) {
	s.handleRateLimits(w, r)
}

// GET /api/v1/permissions/tree — alias for handlePermissionTree
func (s *HTTPServer) handlePermissionTreeAlias(w http.ResponseWriter, r *http.Request) {
	s.handlePermissionTree(w, r)
}

// GET /api/v1/sod/rules — alias for handleSoDRules
func (s *HTTPServer) handleSoDRulesAlias(w http.ResponseWriter, r *http.Request) {
	s.handleSoDRules(w, r)
}