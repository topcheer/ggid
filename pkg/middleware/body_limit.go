package middleware

import (
	"net/http"
)

// MaxBodySize is the default maximum request body size (2MB).
const MaxBodySize int64 = 2 << 20 // 2MB

// BodyLimit wraps an http.Handler, rejecting request bodies larger than maxBytes.
// This is defense-in-depth: even when services sit behind the gateway (which
// enforces its own limit), direct access to service ports should not allow OOM.
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
