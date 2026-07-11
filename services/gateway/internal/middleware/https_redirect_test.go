package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPSRedirect_HTTPRedirects(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	mw := HTTPSRedirectMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/api/v1/users", nil)
	req.Host = "app.example.com"
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if called {
		t.Fatal("next should not be called for HTTP request")
	}
	if rr.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.HasPrefix(loc, "https://app.example.com") {
		t.Errorf("expected redirect to https://app.example.com, got %s", loc)
	}
	if loc != "https://app.example.com/api/v1/users" {
		t.Errorf("expected full HTTPS URL, got %s", loc)
	}
}

func TestHTTPSRedirect_ForwardedProtoPasses(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := HTTPSRedirectMiddleware(next)

	// Simulate TLS-terminated request via X-Forwarded-Proto
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if !called {
		t.Fatal("next should be called when X-Forwarded-Proto is https")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestHTTPSRedirect_TLSConnectionPasses(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := HTTPSRedirectMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.TLS = &tls.ConnectionState{} // simulate TLS
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if !called {
		t.Fatal("next should be called when TLS is set")
	}
}

func TestHTTPSRedirect_PreservesPath(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach next")
	})
	mw := HTTPSRedirectMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/api/v1/users?name=alice&page=1", nil)
	req.Host = "app.example.com"
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	loc := rr.Header().Get("Location")
	expected := "https://app.example.com/api/v1/users?name=alice&page=1"
	if loc != expected {
		t.Errorf("expected %s, got %s", expected, loc)
	}
}

func TestHTTPSRedirect_SetsHSTSHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	mw := HTTPSRedirectMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/", nil)
	req.Host = "app.example.com"
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	hsts := rr.Header().Get("Strict-Transport-Security")
	if hsts == "" {
		t.Error("expected Strict-Transport-Security header on redirect")
	}
	if !strings.Contains(hsts, "max-age") {
		t.Errorf("expected max-age in HSTS, got %s", hsts)
	}
}
