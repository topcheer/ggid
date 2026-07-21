package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateOpenAPISpec(t *testing.T) {
	spec := GenerateOpenAPISpec()
	if spec.OpenAPI != "3.1.0" { t.Error("should be OpenAPI 3.1.0") }
	if spec.Info.Title != "GGID Platform API" { t.Error("title mismatch") }
	if len(spec.Paths) < 30 { t.Errorf("expected >=30 paths, got %d", len(spec.Paths)) }
	// Check security schemes.
	if spec.Components.SecuritySchemes["bearerAuth"].Scheme != "bearer" { t.Error("missing bearer auth") }
	if spec.Components.SecuritySchemes["apiKey"].Type != "apiKey" { t.Error("missing apiKey") }
	if spec.Components.SecuritySchemes["dpop"].Scheme != "DPoP" { t.Error("missing DPoP") }
	if spec.Components.SecuritySchemes["mtls"].Type != "mutualTLS" { t.Error("missing mTLS") }
}

func TestOpenAPISpecHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/swagger.json", nil)
	w := httptest.NewRecorder()
	OpenAPISpecHandler().ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("expected 200, got %d", w.Code) }
	var spec map[string]any
	json.Unmarshal(w.Body.Bytes(), &spec)
	if spec["openapi"] != "3.1.0" { t.Error("spec should be valid JSON with openapi version") }
}

func TestSwaggerUIHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/docs", nil)
	w := httptest.NewRecorder()
	SwaggerUIHandler().ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Errorf("expected 200, got %d", w.Code) }
	if w.Header().Get("Content-Type") != "text/html" { t.Error("should be HTML") }
	body := w.Body.String()
	if !containsHTML(body, "swagger-ui") { t.Error("should contain swagger-ui div") }
}

func TestOpenAPIPaths_Coverage(t *testing.T) {
	spec := GenerateOpenAPISpec()
	required := []string{
		"/api/v1/auth/verify", "/api/v1/auth/register",
		"/api/v1/identity/users", "/api/v1/oauth/token",
		"/api/v1/policy/authorize", "/api/v1/risk/evaluate",
		"/api/v1/audit/events", "/api/v1/mdm/connectors",
	}
	for _, path := range required {
		if _, ok := spec.Paths[path]; !ok {
			t.Errorf("missing path: %s", path)
		}
	}
}

func containsHTML(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr { return true }
	}
	return false
}
