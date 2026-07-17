package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ggid/ggid/services/audit/internal/repository"
)

func TestThreatIntelSources_NilRepo(t *testing.T) {
	s := &HTTPServer{threatIntelRepo: nil}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/threat-intel/sources", nil)
	rec := httptest.NewRecorder()
	s.handleThreatIntel(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestThreatIntelCheck_NoMatch(t *testing.T) {
	repo := repository.NewThreatIntelRepository(nil)
	s := &HTTPServer{threatIntelRepo: repo}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/threat-intel/check",
		strings.NewReader(`{"indicator":"1.2.3.4","indicator_type":"ip"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handleThreatIntel(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"matched":false`) {
		t.Fatalf("expected matched:false, got %s", body)
	}
}

func TestThreatIntelCheck_BadJSON(t *testing.T) {
	repo := repository.NewThreatIntelRepository(nil)
	s := &HTTPServer{threatIntelRepo: repo}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/threat-intel/check",
		strings.NewReader(`{invalid`))
	rec := httptest.NewRecorder()
	s.handleThreatIntel(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad JSON, got %d", rec.Code)
	}
}

func TestThreatIntelSources_List(t *testing.T) {
	repo := repository.NewThreatIntelRepository(nil)
	s := &HTTPServer{threatIntelRepo: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/threat-intel/sources", nil)
	rec := httptest.NewRecorder()
	s.handleThreatIntel(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for list, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"sources"`) {
		t.Fatalf("expected sources in response, got %s", body)
	}
}

func TestThreatIntelStats(t *testing.T) {
	repo := repository.NewThreatIntelRepository(nil)
	s := &HTTPServer{threatIntelRepo: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/threat-intel/stats", nil)
	rec := httptest.NewRecorder()
	s.handleThreatIntel(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"sources_enabled"`) {
		t.Fatalf("expected stats fields, got %s", body)
	}
}

func TestGetAdapter(t *testing.T) {
	ipAdapter := getAdapter("ip")
	if _, ok := ipAdapter.(*AbuseIPDBAdapter); !ok {
		t.Fatalf("expected *AbuseIPDBAdapter for 'ip', got %T", ipAdapter)
	}

	domainAdapter := getAdapter("domain")
	if _, ok := domainAdapter.(*OTXAdapter); !ok {
		t.Fatalf("expected *OTXAdapter for 'domain', got %T", domainAdapter)
	}
}
