package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/ggid/ggid/services/audit/internal/service"
)

func newIntegrationTestServer() *HTTPServer {
	repo := &mockRepo{events: []*domain.AuditEvent{}}
	return NewHTTPServer(service.NewAuditService(repo))
}

func newIntegrationTestMux() *http.ServeMux {
	mux := http.NewServeMux()
	newIntegrationTestServer().RegisterRoutes(mux)
	return mux
}

func TestServer_RetentionGet(t *testing.T) {
	mux := newIntegrationTestMux()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/retention", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "retention_days") {
		t.Error("response should contain retention_days")
	}
	if !strings.Contains(rr.Body.String(), "enabled") {
		t.Error("response should contain enabled")
	}
}

func TestServer_RetentionUpdate(t *testing.T) {
	mux := newIntegrationTestMux()

	body := `{"retention_days": 30, "enabled": true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/audit/retention", strings.NewReader(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "30") {
		t.Error("response should reflect 30 days")
	}
}

func TestServer_RetentionCleanup(t *testing.T) {
	mux := newIntegrationTestMux()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/retention?days=30", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// Cleanup may return 200 or 500 depending on DB availability, but handler should respond
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "deleted") {
		t.Error("response should contain deleted count")
	}
}

func TestServer_RetentionMethodNotAllowed(t *testing.T) {
	mux := newIntegrationTestMux()

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/audit/retention", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestServer_ComplianceReportV2SOC2(t *testing.T) {
	mux := newIntegrationTestMux()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/compliance-report?type=soc2&tenant_id=550e8400-e29b-41d4-a716-446655440000", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "soc2") {
		t.Error("response should contain soc2 type")
	}
	if !strings.Contains(rr.Body.String(), "summary") {
		t.Error("response should contain summary section")
	}
}

func TestServer_ComplianceReportV2HIPAA(t *testing.T) {
	mux := newIntegrationTestMux()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/compliance-report?type=hipaa&tenant_id=550e8400-e29b-41d4-a716-446655440000", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "hipaa") {
		t.Error("response should contain hipaa type")
	}
}

func TestServer_ComplianceReportV2InvalidType(t *testing.T) {
	mux := newIntegrationTestMux()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/compliance-report?type=invalid", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid type, got %d", rr.Code)
	}
}

func TestServer_ComplianceReportV2MethodNotAllowed(t *testing.T) {
	mux := newIntegrationTestMux()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/compliance-report", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestServer_AlertConfigGet(t *testing.T) {
	mux := newIntegrationTestMux()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/alerts/config", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestServer_AlertConfigUpdate(t *testing.T) {
	mux := newIntegrationTestMux()

	body := `{"enabled": true, "webhook_url": "https://example.com/webhook"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/alerts/config", strings.NewReader(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestServer_AlertTest(t *testing.T) {
	mux := newIntegrationTestMux()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/alerts/test", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// Test endpoint should return 200 even without real webhook configured
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestServer_ComplianceReportV2WithDateRange(t *testing.T) {
	mux := newIntegrationTestMux()

	url := "/api/v1/audit/compliance-report?type=gdpr&tenant_id=550e8400-e29b-41d4-a716-446655440000&from=2025-01-01T00:00:00Z&to=2025-07-01T00:00:00Z"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "gdpr") {
		t.Error("response should contain gdpr type")
	}
	if !strings.Contains(rr.Body.String(), "sections") {
		t.Error("response should contain sections")
	}
}

func TestServer_EvidencePackageSOC2(t *testing.T) {
	mux := newIntegrationTestMux()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/compliance/evidence-package?framework=soc2&tenant_id=550e8400-e29b-41d4-a716-446655440000", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "soc2") {
		t.Error("response should contain soc2 framework")
	}
	if !strings.Contains(rr.Body.String(), "controls") {
		t.Error("response should contain controls array")
	}
	if !strings.Contains(rr.Body.String(), "evidence") {
		t.Error("response should contain evidence section")
	}
}

func TestServer_EvidencePackageInvalidFramework(t *testing.T) {
	mux := newIntegrationTestMux()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/compliance/evidence-package?framework=invalid", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestServer_EvidencePackageGDPR(t *testing.T) {
	mux := newIntegrationTestMux()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/compliance/evidence-package?framework=gdpr&tenant_id=550e8400-e29b-41d4-a716-446655440000", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "gdpr") {
		t.Error("response should contain gdpr framework")
	}
}
