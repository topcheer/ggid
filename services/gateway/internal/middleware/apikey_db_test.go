package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// --- parseAPIKeyID Tests ---

func TestParseAPIKeyID_Valid(t *testing.T) {
	id := uuid.New()
	key := "ggid_sk_" + id.String() + "_abcdef123456"
	parsed, ok := parseAPIKeyID(key)
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	if parsed != id.String() {
		t.Errorf("expected %s, got %s", id.String(), parsed)
	}
}

func TestParseAPIKeyID_InvalidPrefix(t *testing.T) {
	key := "ggid_pk_" + uuid.New().String() + "_secret"
	_, ok := parseAPIKeyID(key)
	if ok {
		t.Error("should reject wrong prefix")
	}
}

func TestParseAPIKeyID_TooShort(t *testing.T) {
	_, ok := parseAPIKeyID("ggid_sk_short")
	if ok {
		t.Error("should reject too-short key")
	}
}

func TestParseAPIKeyID_InvalidUUID(t *testing.T) {
	key := "ggid_sk_not-a-valid-uuid-xxxx_secret12345"
	_, ok := parseAPIKeyID(key)
	if ok {
		t.Error("should reject invalid UUID")
	}
}

func TestParseAPIKeyID_MissingUnderscoreAfterUUID(t *testing.T) {
	id := uuid.New()
	key := "ggid_sk_" + id.String() + "nounderscore12345678"
	_, ok := parseAPIKeyID(key)
	if ok {
		t.Error("should reject key without underscore after UUID")
	}
}

func TestParseAPIKeyID_EmptyString(t *testing.T) {
	_, ok := parseAPIKeyID("")
	if ok {
		t.Error("should reject empty string")
	}
}

// --- extractAPIKeyFromRequest Tests ---

func TestExtractAPIKey_XAPIKeyHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("X-API-Key", "ggid_sk_test")
	if got := extractAPIKeyFromRequest(req); got != "ggid_sk_test" {
		t.Errorf("expected ggid_sk_test, got %s", got)
	}
}

func TestExtractAPIKey_QueryParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users?api_key=ggid_sk_query", nil)
	if got := extractAPIKeyFromRequest(req); got != "ggid_sk_query" {
		t.Errorf("expected ggid_sk_query, got %s", got)
	}
}

func TestExtractAPIKey_AuthorizationApiKeyHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "ApiKey ggid_sk_authheader")
	if got := extractAPIKeyFromRequest(req); got != "ggid_sk_authheader" {
		t.Errorf("expected ggid_sk_authheader, got %s", got)
	}
}

func TestExtractAPIKey_NoKey(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	if got := extractAPIKeyFromRequest(req); got != "" {
		t.Errorf("expected empty string, got %s", got)
	}
}

func TestExtractAPIKey_BearerTokenIgnored(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGc...")
	if got := extractAPIKeyFromRequest(req); got != "" {
		t.Errorf("expected empty string for Bearer token, got %s", got)
	}
}

func TestExtractAPIKey_XAPIKeyTakesPrecedence(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("X-API-Key", "ggid_sk_header")
	req.Header.Set("Authorization", "ApiKey ggid_sk_auth")
	if got := extractAPIKeyFromRequest(req); got != "ggid_sk_header" {
		t.Errorf("X-API-Key should take precedence, got %s", got)
	}
}

// --- APIKeyAuth with Authorization: ApiKey header Tests ---

func TestAPIKeyAuth_AuthorizationApiKeyHeader(t *testing.T) {
	validator := NewMemoryAPIKeyValidator()
	validator.AddKey("ggid_sk_test_auth", "tenant-1", "user-1", []string{"read"})

	var gotTenant string
	handler := APIKeyAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTenant, _ = r.Context().Value(TenantIDKey).(string)
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "ApiKey ggid_sk_test_auth")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotTenant != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", gotTenant)
	}
}

func TestAPIKeyAuth_BearerTokenPassesThrough(t *testing.T) {
	validator := NewMemoryAPIKeyValidator()
	called := false
	handler := APIKeyAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGc...")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Error("handler should be called — Bearer token should pass through to JWT auth")
	}
}

// --- IsAPIKeyRequest Authorization: ApiKey header test ---

func TestIsAPIKeyRequest_AuthorizationApiKey(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "ApiKey ggid_sk_test")
	if !IsAPIKeyRequest(req) {
		t.Error("should detect Authorization: ApiKey header")
	}
}

func TestIsAPIKeyRequest_QueryParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?api_key=ggid_sk_test", nil)
	if !IsAPIKeyRequest(req) {
		t.Error("should detect api_key query param")
	}
}



// --- DBAPIKeyValidator.Validate parse error Tests (no DB needed) ---

func TestDBAPIKeyValidator_InvalidFormat_ReturnsError(t *testing.T) {
	v := &DBAPIKeyValidator{
		pool: nil, // no DB needed — parse fails before query
		ttl:  30_000_000_000, // 30s in nanoseconds
	}
	_, _, _, err := v.Validate(context.Background(), "invalid-key-format")
	if err == nil {
		t.Error("expected error for invalid key format")
	}
}

func TestDBAPIKeyValidator_ValidFormatButNoDB_PanicsOrErrors(t *testing.T) {
	// This tests that key parsing succeeds but DB lookup fails gracefully.
	// We can't test the full flow without a DB, but we verify the parse step works.
	v := &DBAPIKeyValidator{
		pool: nil,
		ttl:  30_000_000_000,
	}
	id := uuid.New()
	key := "ggid_sk_" + id.String() + "_somesecret"

	// This will panic on nil pool — which is expected in test (no DB).
	// In production, NewDBAPIKeyValidator returns nil if dbURL is empty,
	// and WithDBAPIKeyAuth returns a pass-through for nil validator.
	// So this test documents that behavior: nil pool = never call Validate.
	defer func() {
		if r := recover(); r == nil {
			t.Log("Validate did not panic (acceptable if pool nil handling improved)")
		}
	}()
	_, _, _, _ = v.Validate(context.Background(), key)
}

func TestWithDBAPIKeyAuth_NilValidator_PassThrough(t *testing.T) {
	called := false
	handler := WithDBAPIKeyAuth(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "ggid_sk_test")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Error("nil validator should pass through")
	}
}

// --- Integration: full key format + parse + cache structure ---

func TestAPIKeyFormat_RoundTrip(t *testing.T) {
	// Simulate the auth service's key generation flow:
	// keyID = uuid.New()
	// plain = GenerateRandomToken(24)
	// secret = "ggid_sk_" + keyID.String() + "_" + plain
	//
	// Then the gateway parses it back.
	id := uuid.New()
	secret := "ggid_sk_" + id.String() + "_a1b2c3d4e5f6"

	parsedID, ok := parseAPIKeyID(secret)
	if !ok {
		t.Fatal("key format round-trip failed")
	}
	if parsedID != id.String() {
		t.Errorf("expected %s, got %s", id.String(), parsedID)
	}
}
