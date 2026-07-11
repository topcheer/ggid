package middleware

import (
	"net/http"
	"strings"
)

// ContentTypeValidator enforces Content-Type header on POST/PUT/PATCH requests.
// It returns 400 Bad Request if a write-method request is missing or has a
// non-JSON content type when the body is non-empty.
func ContentTypeValidator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only validate write methods with a body
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			if r.ContentLength > 0 {
				ct := r.Header.Get("Content-Type")
				if ct == "" {
					WriteErrorNoRequest(w, http.StatusBadRequest, "missing_content_type",
						"Content-Type header is required for write requests")
					return
				}
				// Accept application/json and variants like application/json; charset=utf-8
				if !strings.HasPrefix(ct, "application/json") {
					WriteErrorNoRequest(w, http.StatusUnsupportedMediaType, "unsupported_media_type",
						"Content-Type must be application/json, got: "+ct)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
