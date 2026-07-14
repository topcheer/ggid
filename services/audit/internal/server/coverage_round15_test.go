package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleQueryMetrics(t *testing.T) {
	s := &HTTPServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/audit/query-metrics", s.handleQueryMetrics)

	t.Run("GET returns metrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/query-metrics", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("POST returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/query-metrics", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rr.Code)
		}
	})
}

func TestHandleSIEMHealth(t *testing.T) {
	s := &HTTPServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/audit/siem/health", s.handleSIEMHealth)

	t.Run("GET returns healthy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/siem/health", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("POST returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/siem/health", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rr.Code)
		}
	})
}

func TestHandleDailyAggregations(t *testing.T) {
	s := &HTTPServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/audit/aggregations/daily", s.handleDailyAggregations)

	t.Run("GET returns daily aggregations", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/aggregations/daily", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("POST returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/aggregations/daily", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rr.Code)
		}
	})
}
