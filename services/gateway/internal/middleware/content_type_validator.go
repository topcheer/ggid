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
				// OAuth endpoints accept application/x-www-form-urlencoded (RFC 6749)
				if strings.HasPrefix(r.URL.Path, "/oauth/") || strings.HasPrefix(r.URL.Path, "/api/v1/oauth/") {
					if !strings.HasPrefix(ct, "application/x-www-form-urlencoded") && !strings.HasPrefix(ct, "application/json") {
						WriteErrorNoRequest(w, http.StatusUnsupportedMediaType, "unsupported_media_type",
							"Content-Type must be application/x-www-form-urlencoded or application/json, got: "+ct)
						return
					}
				} else if strings.HasPrefix(r.URL.Path, "/scim/") {
					// SCIM endpoints accept application/scim+json (RFC 7644 Section 3.1)
					if !strings.HasPrefix(ct, "application/scim+json") && !strings.HasPrefix(ct, "application/json") {
						WriteErrorNoRequest(w, http.StatusUnsupportedMediaType, "unsupported_media_type",
							"Content-Type must be application/scim+json or application/json, got: "+ct)
						return
					}
				} else {
					// Accept application/json and variants like application/json; charset=utf-8
					if !strings.HasPrefix(ct, "application/json") {
						WriteErrorNoRequest(w, http.StatusUnsupportedMediaType, "unsupported_media_type",
							"Content-Type must be application/json, got: "+ct)
						return
					}
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
