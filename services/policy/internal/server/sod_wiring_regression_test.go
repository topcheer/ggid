package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSoDWiringRegression_RouteRegistered verifies the SoD HTTP endpoint
// is registered via RegisterRoutes and is callable (not 404).
func TestSoDWiringRegression_RouteRegistered(t *testing.T) {
	srv := NewHTTPServer(nil, nil, nil)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	body := `{"user_id":"00000000-0000-0000-0000-000000000001","roles":["admin","auditor"]}`
	resp, err := ts.Client().Post(ts.URL+"/api/v1/policies/sod/check", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("SoD check request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		t.Fatal("SoD route returned 404 — feature not wired")
	}
	// Any non-404 status proves the route is registered and handler runs
}

// TestSoDWiringRegression_GETRejects verifies GET returns non-404 (route exists).
func TestSoDWiringRegression_GETRejects(t *testing.T) {
	srv := NewHTTPServer(nil, nil, nil)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/api/v1/policies/sod/check")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		t.Fatal("SoD route returned 404 — route not registered")
	}
}
