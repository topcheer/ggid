package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// RequestIDContextKey is the context key for request IDs.
type RequestIDContextKey struct{}

// RequestIDMiddleware ensures every request has an X-Request-ID header.
// If missing, a new UUID is generated. The ID is stored in context and
// added to the response headers.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), RequestIDContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from an HTTP context.
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(RequestIDContextKey{}).(string); ok {
		return v
	}
	return ""
}

// WithRequestID sets a request ID in context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestIDContextKey{}, id)
}
