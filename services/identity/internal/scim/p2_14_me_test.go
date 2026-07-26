package scim

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// TestHandleMe_NoAuthHeader verifies that /Me returns 401 when no
// X-User-ID header is present (gateway didn't verify a JWT).
func TestHandleMe_NoAuthHeader(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Me", nil)
	rr := httptest.NewRecorder()
	h.handleMe(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

// TestHandleMe_WrongMethod verifies that non-GET methods are rejected.
func TestHandleMe_WrongMethod(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Me", nil)
	req.Header.Set("X-User-ID", uuid.New().String())
	req.Header.Set("X-Tenant-ID", uuid.New().String())
	rr := httptest.NewRecorder()
	h.handleMe(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

// TestHandleMe_InvalidUserID verifies that malformed X-User-ID is rejected.
func TestHandleMe_InvalidUserID(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Me", nil)
	req.Header.Set("X-User-ID", "not-a-uuid")
	req.Header.Set("X-Tenant-ID", uuid.New().String())
	rr := httptest.NewRecorder()
	h.handleMe(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid user ID, got %d", rr.Code)
	}
}

// TestHandleMe_MissingTenantID verifies that missing X-Tenant-ID is rejected.
func TestHandleMe_MissingTenantID(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Me", nil)
	req.Header.Set("X-User-ID", uuid.New().String())
	rr := httptest.NewRecorder()
	h.handleMe(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing tenant ID, got %d", rr.Code)
	}
}

// TestExtractVerifiedUser_ValidHeaders verifies that valid X-User-ID and
// X-Tenant-ID headers are correctly parsed.
func TestExtractVerifiedUser_ValidHeaders(t *testing.T) {
	userID := uuid.New()
	tenantID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Me", nil)
	req.Header.Set("X-User-ID", userID.String())
	req.Header.Set("X-Tenant-ID", tenantID.String())

	parsedUserID, parsedTenantID, err := extractVerifiedUser(req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if parsedUserID != userID {
		t.Fatalf("expected user ID %s, got %s", userID, parsedUserID)
	}
	if parsedTenantID != tenantID {
		t.Fatalf("expected tenant ID %s, got %s", tenantID, parsedTenantID)
	}
}

// TestExtractVerifiedUser_ForgedUserID verifies that attacker-supplied
// X-User-ID without gateway verification is handled safely (the function
// only parses the header value — gateway must clear it for unauthenticated requests).
func TestExtractVerifiedUser_EmptyUserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Me", nil)
	// No X-User-ID header — should fail
	_, _, err := extractVerifiedUser(req)
	if err == nil {
		t.Fatal("expected error for empty X-User-ID")
	}
}