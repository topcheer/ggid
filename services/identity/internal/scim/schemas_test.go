package scim

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSchemasCollection verifies RFC 7643 §4 /Schemas discovery endpoint.
func TestSchemasCollection(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Schemas", nil)
	rec := httptest.NewRecorder()
	h.handleSchemasCollection(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/scim+json" {
		t.Errorf("Content-Type = %q, want application/scim+json", ct)
	}
	var schemas []schemaResource
	if err := json.Unmarshal(rec.Body.Bytes(), &schemas); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(schemas) != 3 {
		t.Fatalf("got %d schemas, want 3 (User, Group, EnterpriseUser)", len(schemas))
	}
	ids := map[string]bool{}
	for _, s := range schemas {
		ids[s.ID] = true
		if len(s.Attributes) == 0 {
			t.Errorf("schema %s has no attributes", s.ID)
		}
		if s.Meta["resourceType"] != "Schema" {
			t.Errorf("schema %s meta.resourceType = %q", s.ID, s.Meta["resourceType"])
		}
	}
	for _, want := range []string{userSchemaURN, groupSchemaURN, entUserSchemaURN} {
		if !ids[want] {
			t.Errorf("missing schema %s", want)
		}
	}
}

// TestSchemaResource verifies per-URN schema lookup and 404 for unknown URNs.
func TestSchemaResource(t *testing.T) {
	h := NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Schemas/"+userSchemaURN, nil)
	rec := httptest.NewRecorder()
	h.handleSchemaResource(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("user schema status = %d, want 200", rec.Code)
	}
	var s schemaResource
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if s.ID != userSchemaURN {
		t.Errorf("id = %q, want %q", s.ID, userSchemaURN)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/scim/v2/Schemas/urn:example:nonexistent", nil)
	rec2 := httptest.NewRecorder()
	h.handleSchemaResource(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Errorf("unknown schema status = %d, want 404", rec2.Code)
	}
}

// TestServiceProviderConfigETag verifies etag is advertised as supported
// (matching the actual ETag/If-Match implementation in etag.go).
func TestServiceProviderConfigETag(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/ServiceProviderConfig", nil)
	rec := httptest.NewRecorder()
	h.handleServiceProviderConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var cfg map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	etag, ok := cfg["etag"].(map[string]any)
	if !ok {
		t.Fatal("missing etag config")
	}
	if etag["supported"] != true {
		t.Errorf("etag.supported = %v, want true (ETag is implemented)", etag["supported"])
	}
}
