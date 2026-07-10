package scim

import "testing"

func TestParseExternalIdFilter_Simple(t *testing.T) {
	result := parseExternalIdFilter(`externalId eq "abc123"`)
	if result != "abc123" {
		t.Errorf("expected abc123, got %s", result)
	}
}

func TestParseExternalIdFilter_Compound(t *testing.T) {
	result := parseExternalIdFilter(`externalId eq "ext-456" and userName eq "john"`)
	if result != "ext-456" {
		t.Errorf("expected ext-456, got %s", result)
	}
}

func TestParseExternalIdFilter_NoMatch(t *testing.T) {
	result := parseExternalIdFilter(`userName eq "john"`)
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestParseExternalIdFilter_CaseInsensitive(t *testing.T) {
	result := parseExternalIdFilter(`EXTERNALID EQ "upper-789"`)
	if result != "upper-789" {
		t.Errorf("expected upper-789, got %s", result)
	}
}
