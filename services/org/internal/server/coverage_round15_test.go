package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrgServer_AccessMatrix(t *testing.T) {
	mux := newTestOrgMux()

	t.Run("GET returns access matrix", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/550e8400-e29b-41d4-a716-446655440001/access-matrix", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("POST returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/550e8400-e29b-41d4-a716-446655440001/access-matrix", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rr.Code)
		}
	})
}

func TestOrgServer_TeamsExport(t *testing.T) {
	mux := newTestOrgMux()

	t.Run("GET returns JSON teams", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/550e8400-e29b-41d4-a716-446655440001/teams/export", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("GET CSV returns CSV", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/550e8400-e29b-41d4-a716-446655440001/teams/export?format=csv", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		ct := rr.Header().Get("Content-Type")
		if ct != "text/csv" {
			t.Fatalf("expected text/csv, got %s", ct)
		}
	})

	t.Run("POST returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/550e8400-e29b-41d4-a716-446655440001/teams/export", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rr.Code)
		}
	})
}

func TestOrgServer_MembershipTrends(t *testing.T) {
	mux := newTestOrgMux()

	t.Run("GET returns trends", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/org/stats/membership-trends", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}
