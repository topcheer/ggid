// Package scim implements SCIM 2.0 ETag support per RFC 7644 Section 3.14.
package scim

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ggid/ggid/services/identity/internal/domain"
)

// ComputeETag generates a weak ETag for a user resource.
// Format: W/"<unix_nano>" per RFC 7232 Section 2.3.
func ComputeETag(u *domain.User) string {
	return fmt.Sprintf("W/\"%d\"", u.UpdatedAt.UnixNano())
}

// CheckIfNoneMatch checks the If-None-Match header for GET requests.
// Returns true if the client's cached ETag matches the current resource
// (i.e., the server should respond with 304 Not Modified).
func CheckIfNoneMatch(r *http.Request, etag string) bool {
	ifMatch := r.Header.Get("If-None-Match")
	if ifMatch == "" {
		return false
	}
	// If-None-Match: * matches any existing resource
	if strings.TrimSpace(ifMatch) == "*" {
		return true
	}
	// Compare ETags (strip whitespace and quotes)
	for _, clientTag := range strings.Split(ifMatch, ",") {
		clientTag = strings.TrimSpace(clientTag)
		if clientTag == etag || clientTag == "*" {
			return true
		}
	}
	return false
}

// CheckIfMatch validates the If-Match header for PUT/PATCH/DELETE requests.
// Returns (true, "") if the precondition is met.
// Returns (false, reason) if the precondition fails → 412 Precondition Failed.
// If If-Match is absent, the check passes (optimistic concurrency not required).
func CheckIfMatch(r *http.Request, etag string) (bool, string) {
	ifMatch := r.Header.Get("If-Match")
	if ifMatch == "" {
		return true, "" // No precondition → allow
	}

	// If-Match: * succeeds if the resource exists
	if strings.TrimSpace(ifMatch) == "*" {
		return true, ""
	}

	// Compare ETags
	for _, clientTag := range strings.Split(ifMatch, ",") {
		clientTag = strings.TrimSpace(clientTag)
		if clientTag == etag {
			return true, ""
		}
	}

	return false, "ETag does not match If-Match precondition"
}

// SetETagHeader sets the ETag response header.
func SetETagHeader(w http.ResponseWriter, etag string) {
	w.Header().Set("ETag", etag)
}
