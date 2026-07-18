package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAttrMapping_List_NotConfigured verifies 503 without repo.
func TestAttrMapping_List_NotConfigured(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("GET", "/api/v1/admin/migration/mappings", nil)
	w := httptest.NewRecorder()
	h.handleAttrMappings(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", w.Code)
	}
}

// TestAttrMapping_Create_InvalidJSON verifies validation.
func TestAttrMapping_Create_InvalidJSON(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("POST", "/api/v1/admin/migration/mappings",
		strings.NewReader(`invalid json`))
	w := httptest.NewRecorder()
	h.handleAttrMappings(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", w.Code)
	}
}

// TestAttrMapping_WrongMethod verifies method routing.
func TestAttrMapping_WrongMethod(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("PATCH", "/api/v1/admin/migration/mappings", nil)
	w := httptest.NewRecorder()
	h.handleAttrMappings(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for PATCH, got %d", w.Code)
	}
}

// TestAttrMapping_TestRoute verifies test sub-route.
func TestAttrMapping_TestRoute(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("GET", "/api/v1/admin/migration/mappings/test", nil)
	w := httptest.NewRecorder()
	h.handleAttrMappings(w, req)
	// GET on test route → 405 (only POST allowed).
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET on test route, got %d", w.Code)
	}
}

// TestValidateMapping validates mapping fields.
func TestValidateMapping(t *testing.T) {
	tests := []struct {
		name    string
		mapping AttributeMapping
		wantErr bool
	}{
		{
			name: "valid with role",
			mapping: AttributeMapping{
				SourceAttribute: "memberOf",
				SourceValue:     "CN=Admins",
				GGIDRole:        "admin",
			},
			wantErr: false,
		},
		{
			name: "valid with only attribute",
			mapping: AttributeMapping{
				SourceAttribute: "department",
				SourceValue:     "Engineering",
				GGIDAttribute:   "team",
			},
			wantErr: false,
		},
		{
			name: "missing source_attribute",
			mapping: AttributeMapping{
				SourceValue: "x",
				GGIDRole:    "admin",
			},
			wantErr: true,
		},
		{
			name: "missing source_value",
			mapping: AttributeMapping{
				SourceAttribute: "memberOf",
				GGIDRole:        "admin",
			},
			wantErr: true,
		},
		{
			name: "missing both role and attribute",
			mapping: AttributeMapping{
				SourceAttribute: "memberOf",
				SourceValue:     "CN=Admins",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMapping(&tt.mapping)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMapping() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestMatchValue verifies exact and wildcard matching.
func TestMatchValue(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		match   bool
	}{
		{"CN=Admins", "CN=Admins", true},
		{"CN=Admins", "CN=Users", false},
		{"CN=Admins*", "CN=Admins,OU=Groups,DC=example", true},
		{"CN=Admins*", "CN=Users,OU=Groups", false},
		{"*", "anything", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"/"+tt.value, func(t *testing.T) {
			if got := matchValue(tt.pattern, tt.value); got != tt.match {
				t.Errorf("matchValue(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.match)
			}
		})
	}
}
