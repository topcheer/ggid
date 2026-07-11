package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContentTypeValidator_PostWithoutContentType(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	})
	mw := ContentTypeValidator(next)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "")
	req.ContentLength = 15
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "missing_content_type") {
		t.Errorf("expected missing_content_type code, got: %s", rr.Body.String())
	}
}

func TestContentTypeValidator_PostWithWrongContentType(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	})
	mw := ContentTypeValidator(next)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "text/plain")
	req.ContentLength = 15
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "unsupported_media_type") {
		t.Errorf("expected unsupported_media_type code, got: %s", rr.Body.String())
	}
}

func TestContentTypeValidator_PostWithJSONPasses(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})
	mw := ContentTypeValidator(next)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 15
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if !called {
		t.Fatal("next should be called for valid JSON")
	}
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestContentTypeValidator_PostJSONWithCharsetPasses(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := ContentTypeValidator(next)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.ContentLength = 15
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if !called {
		t.Fatal("next should be called for application/json; charset=utf-8")
	}
}

func TestContentTypeValidator_GetPassesWithoutContentType(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := ContentTypeValidator(next)

	// GET requests should not require Content-Type
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if !called {
		t.Fatal("GET should not require Content-Type")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestContentTypeValidator_PostWithEmptyBodyPasses(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := ContentTypeValidator(next)

	// POST with no body should pass
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	req.ContentLength = 0
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if !called {
		t.Fatal("POST with empty body should not require Content-Type")
	}
}

func TestContentTypeValidator_PutWithJSONPasses(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := ContentTypeValidator(next)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/123", strings.NewReader(`{"name":"upd"}`))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 14
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if !called {
		t.Fatal("PUT with JSON should pass")
	}
}

func TestContentTypeValidator_PatchWithJSONPasses(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := ContentTypeValidator(next)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/123", strings.NewReader(`{"active":false}`))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 17
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if !called {
		t.Fatal("PATCH with JSON should pass")
	}
}
