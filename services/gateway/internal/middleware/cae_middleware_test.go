package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestIsPrivilegedPath(t *testing.T) {
	if !isPrivilegedPath("/api/v1/admin/users") {
		t.Error("admin path should be privileged")
	}
	if !isPrivilegedPath("/api/v1/crypto/fields") {
		t.Error("crypto path should be privileged")
	}
	if isPrivilegedPath("/api/v1/users/profile") {
		t.Error("profile path should not be privileged")
	}
}

func TestCAEMiddleware_NilEvalFn(t *testing.T) {
	mw := CAEMiddleware(nil)
	if mw == nil {
		t.Fatal("middleware should not be nil")
	}
	// Should pass through.
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)
	if !called {
		t.Error("next handler should be called with nil eval fn")
	}
}

func TestCAEMiddleware_BlockDecision(t *testing.T) {
	blockFn := func(_ context.Context, _, _ string) (int, string) {
		return 90, "block"
	}
	mw := CAEMiddleware(blockFn)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req.Header.Set("X-Session-ID", "sess-block")
	// Set a valid user ID in context (same key as auth middleware).
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, uuid.New().String()))

	w := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach next handler on block")
	})).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCAEMiddleware_AllowDecision(t *testing.T) {
	allowFn := func(_ context.Context, _, _ string) (int, string) {
		return 10, "allow"
	}
	mw := CAEMiddleware(allowFn)

	called := false
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req.Header.Set("X-Session-ID", "sess-allow")
	// Set a valid user ID in context (same key as auth middleware).
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, uuid.New().String()))

	w := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, req)

	if !called {
		t.Error("next handler should be called on allow")
	}
}
