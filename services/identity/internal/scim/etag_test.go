package scim

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

// --- SCIM-15: ETag Tests ---

func TestComputeETag(t *testing.T) {
	u := &domain.User{
		ID:        uuid.New(),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	etag := ComputeETag(u)
	if etag == "" {
		t.Error("ETag should not be empty")
	}
	if etag[:2] != "W/" {
		t.Errorf("ETag should start with W/, got %s", etag[:2])
	}
}

func TestComputeETag_UniquePerTimestamp(t *testing.T) {
	u1 := &domain.User{UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	u2 := &domain.User{UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 1, time.UTC)}
	if ComputeETag(u1) == ComputeETag(u2) {
		t.Error("ETags should differ for different timestamps")
	}
}

func TestCheckIfNoneMatch_Match(t *testing.T) {
	etag := `W/"12345"`
	req := httptest.NewRequest("GET", "/scim/v2/Users/123", nil)
	req.Header.Set("If-None-Match", etag)
	if !CheckIfNoneMatch(req, etag) {
		t.Error("should return true for matching ETag")
	}
}

func TestCheckIfNoneMatch_NoHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/scim/v2/Users/123", nil)
	if CheckIfNoneMatch(req, `W/"12345"`) {
		t.Error("should return false when no If-None-Match header")
	}
}

func TestCheckIfNoneMatch_Wildcard(t *testing.T) {
	req := httptest.NewRequest("GET", "/scim/v2/Users/123", nil)
	req.Header.Set("If-None-Match", "*")
	if !CheckIfNoneMatch(req, `W/"12345"`) {
		t.Error("wildcard should match any resource")
	}
}

func TestCheckIfNoneMatch_NoMatch(t *testing.T) {
	req := httptest.NewRequest("GET", "/scim/v2/Users/123", nil)
	req.Header.Set("If-None-Match", `W/"99999"`)
	if CheckIfNoneMatch(req, `W/"12345"`) {
		t.Error("should return false for non-matching ETag")
	}
}

func TestCheckIfMatch_Match(t *testing.T) {
	etag := `W/"12345"`
	req := httptest.NewRequest("PUT", "/scim/v2/Users/123", nil)
	req.Header.Set("If-Match", etag)
	ok, reason := CheckIfMatch(req, etag)
	if !ok {
		t.Errorf("should succeed for matching ETag: %s", reason)
	}
}

func TestCheckIfMatch_NoHeader(t *testing.T) {
	req := httptest.NewRequest("PUT", "/scim/v2/Users/123", nil)
	ok, _ := CheckIfMatch(req, `W/"12345"`)
	if !ok {
		t.Error("should succeed when no If-Match header")
	}
}

func TestCheckIfMatch_Wildcard(t *testing.T) {
	req := httptest.NewRequest("PUT", "/scim/v2/Users/123", nil)
	req.Header.Set("If-Match", "*")
	ok, _ := CheckIfMatch(req, `W/"12345"`)
	if !ok {
		t.Error("wildcard should succeed if resource exists")
	}
}

func TestCheckIfMatch_Mismatch(t *testing.T) {
	req := httptest.NewRequest("PUT", "/scim/v2/Users/123", nil)
	req.Header.Set("If-Match", `W/"99999"`)
	ok, reason := CheckIfMatch(req, `W/"12345"`)
	if ok {
		t.Error("should fail for mismatched ETag")
	}
	if reason == "" {
		t.Error("should provide failure reason")
	}
}

func TestCheckIfMatch_MultipleETags(t *testing.T) {
	etag := `W/"12345"`
	req := httptest.NewRequest("PUT", "/scim/v2/Users/123", nil)
	req.Header.Set("If-Match", `W/"111", W/"12345", W/"222"`)
	ok, _ := CheckIfMatch(req, etag)
	if !ok {
		t.Error("should match when ETag is in comma-separated list")
	}
}

func TestSetETagHeader(t *testing.T) {
	w := httptest.NewRecorder()
	SetETagHeader(w, `W/"12345"`)
	if w.Header().Get("ETag") != `W/"12345"` {
		t.Errorf("ETag header = %s", w.Header().Get("ETag"))
	}
}
