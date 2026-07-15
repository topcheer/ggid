package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeoIP_PrivateIP_NoBlock(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := GeoIPMiddleware(&GeoIPConfig{TrustXForwardedFor: true})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.10:12345"
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// Private IP does not set a country header; it is not blocked.
}

func TestGeoIP_UpstreamHeader_Passthrough(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := GeoIPMiddleware(&GeoIPConfig{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.10:12345"
	req.Header.Set("X-Geo-Country", "US")
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Geo-Country"); got != "US" {
		t.Fatalf("expected upstream X-Geo-Country=US, got %s", got)
	}
}

func TestGeoIP_CountryLookup_Mock(t *testing.T) {
	old := SetCountryLookup(func(_ string) string { return "CA" })
	defer SetCountryLookup(old)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := GeoIPMiddleware(&GeoIPConfig{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Geo-Country"); got != "CA" {
		t.Fatalf("expected X-Geo-Country=CA, got %s", got)
	}
}

func TestGeoIP_Blocklist(t *testing.T) {
	old := SetCountryLookup(func(_ string) string { return "CN" })
	defer SetCountryLookup(old)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	mw := GeoIPMiddleware(&GeoIPConfig{BlockedCountries: []string{"CN"}})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if called {
		t.Fatal("next handler should not be called for blocked country")
	}
}

func TestGeoIP_Allowlist(t *testing.T) {
	old := SetCountryLookup(func(_ string) string { return "US" })
	defer SetCountryLookup(old)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	mw := GeoIPMiddleware(&GeoIPConfig{AllowedCountries: []string{"US", "CA"}})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !called {
		t.Fatal("next handler should be called for allowed country")
	}
}

func TestGeoIP_Allowlist_DeniesOther(t *testing.T) {
	old := SetCountryLookup(func(_ string) string { return "XX" })
	defer SetCountryLookup(old)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for non-allowed country")
	})

	mw := GeoIPMiddleware(&GeoIPConfig{AllowedCountries: []string{"US"}})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
