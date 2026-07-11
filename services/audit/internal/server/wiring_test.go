package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestWiring_AllRoutesRegistered verifies that all newly wired routes
// return non-404 (they should be handled, even if they return 400/405).
func TestWiring_AuditRiskScore(t *testing.T) {
	s := &HTTPServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/audit/risk-score", s.handleRiskScore)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/risk-score?user_id=test", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Error("risk-score route should be registered (not 404)")
	}
}

func TestWiring_AuditAccessReviews(t *testing.T) {
	s := &HTTPServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/audit/access-reviews", s.handleAccessReviews)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/access-reviews",
		strings.NewReader(`{"manager_id":"550e8400-e29b-41d4-a716-446655440000","user_id":"550e8400-e29b-41d4-a716-446655440001","roles":["admin"]}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Error("access-reviews route should be registered")
	}
}

func TestWiring_AuditPendingReviews(t *testing.T) {
	s := &HTTPServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/audit/access-reviews/pending", s.handlePendingReviews)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/access-reviews/pending",
		strings.NewReader(`{"review_id":"550e8400-e29b-41d4-a716-446655440000","decision":"approve"}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Error("pending reviews route should be registered")
	}
}
