package server

import (
	"net/http"
	"strings"
)

// handleAccessKeys is an alias handler for /api/v1/auth/access-keys.
// The frontend "Access Keys" page calls /api/v1/access-keys, which the gateway
// rewrites to /api/v1/auth/access-keys. This handler maps those requests to the
// existing api-keys handler by rewriting the path internally.
func (h *Handler) handleAccessKeys(w http.ResponseWriter, r *http.Request) {
	// Rewrite access-keys path → api-keys path, then delegate
	r2 := r.Clone(r.Context())
	r2.URL.Path = strings.Replace(r.URL.Path, "/api/v1/auth/access-keys", "/api/v1/auth/api-keys", 1)
	h.handleAPIKeys(w, r2)
}
