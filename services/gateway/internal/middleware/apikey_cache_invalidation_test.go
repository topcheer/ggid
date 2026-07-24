package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAPIKeyCacheInvalidator_DeleteInvalidates(t *testing.T) {
	validator := &DBAPIKeyValidator{
		pool: nil,
		ttl:  5 * time.Second,
	}
	keyID := uuid.New().String()

	// Seed the cache
	validator.cache.Store(keyID, &cachedKey{
		tenantID: "test-tenant",
		scopes:   []string{"read"},
		status:   "active",
		cachedAt: time.Now(),
	})

	// Verify cached
	if _, ok := validator.cache.Load(keyID); !ok {
		t.Fatal("expected key in cache before invalidation")
	}

	// Simulate a DELETE request that succeeds
	mw := APIKeyCacheInvalidator(validator)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"revoked"}`))
	}))

	req := httptest.NewRequest("DELETE", "/api/v1/auth/api-keys/"+keyID, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Cache should be invalidated
	if _, ok := validator.cache.Load(keyID); ok {
		t.Error("expected cache entry to be invalidated after successful DELETE")
	}
}

func TestAPIKeyCacheInvalidator_NonDeleteDoesNotInvalidate(t *testing.T) {
	validator := &DBAPIKeyValidator{
		pool: nil,
		ttl:  5 * time.Second,
	}
	keyID := uuid.New().String()
	validator.cache.Store(keyID, &cachedKey{
		tenantID: "test",
		status:   "active",
		cachedAt: time.Now(),
	})

	mw := APIKeyCacheInvalidator(validator)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// GET request should not trigger invalidation
	req := httptest.NewRequest("GET", "/api/v1/auth/api-keys/"+keyID, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if _, ok := validator.cache.Load(keyID); !ok {
		t.Error("cache entry should still exist after GET request")
	}
}

func TestAPIKeyCacheInvalidator_FailedDeleteDoesNotInvalidate(t *testing.T) {
	validator := &DBAPIKeyValidator{
		pool: nil,
		ttl:  5 * time.Second,
	}
	keyID := uuid.New().String()
	validator.cache.Store(keyID, &cachedKey{
		tenantID: "test",
		status:   "active",
		cachedAt: time.Now(),
	})

	mw := APIKeyCacheInvalidator(validator)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // failed delete
	}))

	req := httptest.NewRequest("DELETE", "/api/v1/auth/api-keys/"+keyID, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Cache should NOT be invalidated because the delete failed
	if _, ok := validator.cache.Load(keyID); !ok {
		t.Error("cache entry should still exist after failed DELETE")
	}
}

func TestAPIKeyCacheInvalidator_NonAPIKeyPathPassthrough(t *testing.T) {
	validator := &DBAPIKeyValidator{
		pool: nil,
		ttl:  5 * time.Second,
	}

	called := false
	mw := APIKeyCacheInvalidator(validator)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	// Non-api-keys path should pass through without issues
	req := httptest.NewRequest("DELETE", "/api/v1/users/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should have been called")
	}
}

func TestExtractKeyIDFromPath(t *testing.T) {
	id := uuid.New().String()
	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/auth/api-keys/" + id, id},
		{"/api/v1/api-keys/" + id, id},
		{"/api/v1/access-keys/" + id, id},
		{"/api/v1/auth/api-keys/invalid-uuid", ""},
		{"/api/v1/users/" + id, ""},
		{"/api/v1/auth/api-keys/", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := extractKeyIDFromPath(tt.path)
		if got != tt.want {
			t.Errorf("extractKeyIDFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestNilValidator_Passthrough(t *testing.T) {
	mw := APIKeyCacheInvalidator(nil)
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("DELETE", "/api/v1/auth/api-keys/"+uuid.New().String(), nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Error("nil validator should pass through to handler")
	}
}
